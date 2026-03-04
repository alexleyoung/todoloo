package nlp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/alexyoung/todoloo/internal/config"
	"github.com/alexyoung/todoloo/internal/models"
)

type OllamaParser struct {
	host   string
	model  string
	client *http.Client
}

type ollamaRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
	Stream bool   `json:"stream"`
}

type ollamaResponse struct {
	Response string `json:"response"`
}

type llmParseResult struct {
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

func NewOllamaParser(cfg config.LLMConfig) *OllamaParser {
	return &OllamaParser{
		host:   cfg.Host,
		model:  cfg.Model,
		client: &http.Client{Timeout: time.Duration(cfg.TimeoutSeconds) * time.Second},
	}
}

func (p *OllamaParser) Parse(ctx context.Context, raw string, meta ParserMeta) (*models.ParsedTodo, error) {
	prompt := buildPrompt(raw, meta)

	reqBody := ollamaRequest{
		Model:  p.model,
		Prompt: prompt,
		Stream: false,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.host+"/api/generate", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call Ollama: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Ollama returned status %d", resp.StatusCode)
	}

	var ollamaResp ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return p.parseResponse(ctx, ollamaResp.Response, meta)
}

func (p *OllamaParser) parseResponse(ctx context.Context, response string, meta ParserMeta) (*models.ParsedTodo, error) {
	jsonStr := extractJSON(response)
	if jsonStr == "" {
		return nil, fmt.Errorf("no JSON found in response")
	}

	var result llmParseResult
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

func (p *OllamaParser) resolveDateFallback(ctx context.Context, expr string, ref time.Time) (time.Time, error) {
	prompt := fmt.Sprintf(`Given the reference date %s (%s), convert the following natural language date/time expression to an ISO 8601 date (YYYY-MM-DD) and 24-hour time (HH:MM).
Expression: "%s"
Respond with ONLY a JSON object like: {"date": "2026-03-15", "time": "14:30"}`, ref.Format("2006-01-02"), ref.Weekday(), expr)

	reqBody := ollamaRequest{
		Model:  p.model,
		Prompt: prompt,
		Stream: false,
	}

	jsonData, _ := json.Marshal(reqBody)
	req, _ := http.NewRequestWithContext(ctx, "POST", p.host+"/api/generate", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return time.Time{}, err
	}
	defer resp.Body.Close()

	var ollamaResp ollamaResponse
	if err := json.NewDecoder(resp.Body).Decode(&ollamaResp); err != nil {
		return time.Time{}, err
	}

	var dateResult struct {
		Date string `json:"date"`
		Time string `json:"time"`
	}
	if err := json.Unmarshal([]byte(extractJSON(ollamaResp.Response)), &dateResult); err != nil {
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

func buildPrompt(raw string, meta ParserMeta) string {
	categories := "personal"
	if len(meta.Categories) > 0 {
		categories = strings.Join(meta.Categories, ", ")
	}

	return fmt.Sprintf(`You are a task parser. Extract structured information from a natural language todo.
Return ONLY a valid JSON object matching this schema — no preamble, no explanation.

Today's date is %s (%s).
Available categories: %s.

Schema:
{
  "title":        string (concise action-oriented title),
  "description":  string (any extra context, or ""),
  "category":     string (one of the available categories, or "personal" if unclear),
  "date_expr":    string (the raw date/time expression verbatim, e.g. "this wednesday at 5:30pm", "tomorrow", "next week", or "" if none),
  "urgency":      int (1=low, 2=minor, 3=normal, 4=high, 5=critical),
  "location":     string (or ""),
  "tags":         array of strings (keywords, or []),
  "notes":        string (anything else useful, or ""),
  "recurrence":   string (JSON recurrence rule, or "")
}

Do NOT resolve dates to ISO format — extract the expression exactly as written.

Input: "%s"`, meta.Today.Format("2006-01-02"), meta.Today.Weekday(), categories, raw)
}

func extractJSON(text string) string {
	start := strings.Index(text, "{")
	if start == -1 {
		return ""
	}
	end := strings.LastIndex(text, "}")
	if end == -1 {
		return ""
	}
	return text[start : end+1]
}
