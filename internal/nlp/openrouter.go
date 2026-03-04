package nlp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/alexyoung/todoloo/internal/config"
	"github.com/alexyoung/todoloo/internal/models"
)

type OpenRouterParser struct {
	apiKey string
	model  string
	client *http.Client
}

type openaiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openaiRequest struct {
	Model    string          `json:"model"`
	Messages []openaiMessage `json:"messages"`
}

type openaiResponse struct {
	Choices []choice `json:"choices"`
}

type choice struct {
	Message openaiMessage `json:"message"`
}

func NewOpenRouterParser(cfg config.LLMConfig) *OpenRouterParser {
	apiKey := cfg.APIKey
	if apiKey == "" {
		apiKey = os.Getenv("OPENROUTER_API_KEY")
	}
	if apiKey == "" {
		log.Printf("warning: OpenRouter API key not found, set OPENROUTER_API_KEY env var or api_key in config")
	}
	return &OpenRouterParser{
		apiKey: apiKey,
		model:  cfg.Model,
		client: &http.Client{Timeout: time.Duration(cfg.TimeoutSeconds) * time.Second},
	}
}

func (p *OpenRouterParser) Parse(ctx context.Context, raw string, meta ParserMeta) (*models.ParsedTodo, error) {
	prompt := buildPrompt(raw, meta)

	reqBody := openaiRequest{
		Model: p.model,
		Messages: []openaiMessage{
			{Role: "system", Content: "You are a task parser. Return ONLY valid JSON, no preamble."},
			{Role: "user", Content: prompt},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("HTTP-Referer", "https://todoloo.local")
	req.Header.Set("X-Title", "Todoloo")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call OpenRouter: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("OpenRouter returned status %d", resp.StatusCode)
	}

	var openaiResp openaiResponse
	if err := json.NewDecoder(resp.Body).Decode(&openaiResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(openaiResp.Choices) == 0 {
		return nil, fmt.Errorf("no choices in response")
	}

	return p.parseResponse(ctx, openaiResp.Choices[0].Message.Content, meta)
}

func (p *OpenRouterParser) parseResponse(ctx context.Context, response string, meta ParserMeta) (*models.ParsedTodo, error) {
	jsonStr := ExtractJSON(response)
	if jsonStr == "" {
		return nil, fmt.Errorf("no JSON found in response")
	}

	var result struct {
		Title       string   `json:"title"`
		Description string   `json:"description"`
		Category    string   `json:"category"`
		DateExpr    string   `json:"date_expr"`
		Urgency     int      `json:"urgency"`
		Location    string   `json:"location"`
		Tags        []string `json:"tags"`
		Notes       string   `json:"notes"`
		Recurrence  string   `json:"recurrence"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	dueDate, dueTime, err := ResolveDate(ctx, result.DateExpr, meta.Today, p.resolveDateFallback)
	if err != nil {
		return nil, fmt.Errorf("date resolution failed: %w", err)
	}

	return &models.ParsedTodo{
		Title:       result.Title,
		Description: result.Description,
		Category:    result.Category,
		DueDate:     dueDate,
		DueTime:     dueTime,
		Urgency:     result.Urgency,
		Location:    result.Location,
		Tags:        result.Tags,
		Notes:       result.Notes,
		Recurrence:  result.Recurrence,
	}, nil
}

func (p *OpenRouterParser) resolveDateFallback(ctx context.Context, expr string, ref time.Time) (time.Time, error) {
	prompt := fmt.Sprintf(`Given the reference date %s (%s), convert the following natural language date/time expression to an ISO 8601 date (YYYY-MM-DD) and 24-hour time (HH:MM).
Expression: "%s"
Respond with ONLY a JSON object like: {"date": "2026-03-15", "time": "14:30"}`, ref.Format("2006-01-02"), ref.Weekday(), expr)

	reqBody := openaiRequest{
		Model: p.model,
		Messages: []openaiMessage{
			{Role: "system", Content: "Return ONLY valid JSON."},
			{Role: "user", Content: prompt},
		},
	}

	jsonData, _ := json.Marshal(reqBody)
	req, _ := http.NewRequestWithContext(ctx, "POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("HTTP-Referer", "https://todoloo.local")
	req.Header.Set("X-Title", "Todoloo")

	resp, err := p.client.Do(req)
	if err != nil {
		return time.Time{}, err
	}
	defer resp.Body.Close()

	var openaiResp openaiResponse
	if err := json.NewDecoder(resp.Body).Decode(&openaiResp); err != nil {
		return time.Time{}, err
	}

	if len(openaiResp.Choices) == 0 {
		return time.Time{}, fmt.Errorf("no choices in fallback response")
	}

	var dateResult struct {
		Date string `json:"date"`
		Time string `json:"time"`
	}
	if err := json.Unmarshal([]byte(ExtractJSON(openaiResp.Choices[0].Message.Content)), &dateResult); err != nil {
		return time.Time{}, err
	}

	parsed, err := time.Parse("2006-01-02", dateResult.Date)
	if err != nil {
		return time.Time{}, err
	}

	if dateResult.Time != "" {
		if t, err := time.Parse("15:04", dateResult.Time); err == nil {
			parsed = parsed.Add(time.Duration(t.Hour())*time.Hour + time.Duration(t.Minute())*time.Minute)
		}
	}

	return parsed, nil
}
