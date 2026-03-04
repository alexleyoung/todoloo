package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var serverURL = "http://127.0.0.1:8080"

type Todo struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Category    string `json:"category,omitempty"`
	DueDate     string `json:"due_date,omitempty"`
	DueTime     string `json:"due_time,omitempty"`
	Urgency     int    `json:"urgency"`
	Status      string `json:"status"`
	Tags        string `json:"tags,omitempty"`
	Location    string `json:"location,omitempty"`
}

type Job struct {
	ID          int64     `json:"id"`
	RawText     string    `json:"raw_text"`
	SubmittedAt time.Time `json:"submitted_at"`
	Status      string    `json:"status"`
	Attempts    int       `json:"attempts"`
}

type RawResponse struct {
	JobID   int64  `json:"job_id"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

func getClient() *http.Client {
	return &http.Client{Timeout: 10 * time.Second}
}

var addCmd = &cobra.Command{
	Use:   "add [text]",
	Short: "Add a new todo",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		text := strings.Join(args, " ")
		reqBody := fmt.Sprintf(`{"text": %q}`, text)
		resp, err := getClient().Post(serverURL+"/todos/raw", "application/json", strings.NewReader(reqBody))
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusAccepted {
			body, _ := io.ReadAll(resp.Body)
			fmt.Fprintf(os.Stderr, "Error: %s\n", body)
			os.Exit(1)
		}

		var result RawResponse
		json.NewDecoder(resp.Body).Decode(&result)
		fmt.Printf("Added todo (job_id=%d, status=%s)\n", result.JobID, result.Status)
	},
}

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List todos",
	Run: func(cmd *cobra.Command, args []string) {
		url := serverURL + "/todos?"
		if cat, _ := cmd.Flags().GetString("category"); cat != "" {
			url += "category=" + cat + "&"
		}
		if status, _ := cmd.Flags().GetString("status"); status != "" {
			url += "status=" + status + "&"
		}
		if dueBefore, _ := cmd.Flags().GetString("due-before"); dueBefore != "" {
			url += "due_before=" + dueBefore + "&"
		}
		if urgencyMin, _ := cmd.Flags().GetInt("urgency-min"); urgencyMin > 0 {
			url += fmt.Sprintf("urgency_min=%d&", urgencyMin)
		}
		if limit, _ := cmd.Flags().GetInt("limit"); limit > 0 {
			url += fmt.Sprintf("limit=%d&", limit)
		}

		resp, err := getClient().Get(url)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		var todos []Todo
		json.NewDecoder(resp.Body).Decode(&todos)

		if len(todos) == 0 {
			fmt.Println("No todos found")
			return
		}

		fmt.Printf("%-4s %-6s %-30s %-10s %-5s\n", "ID", "Status", "Title", "Due", "Urg")
		fmt.Println(strings.Repeat("-", 70))
		for _, t := range todos {
			due := ""
			if t.DueDate != "" {
				due = t.DueDate
				if t.DueTime != "" {
					due += " " + t.DueTime
				}
			}
			statusIcon := "○"
			if t.Status == "done" {
				statusIcon = "✓"
			}
			urgencyIcon := ""
			for i := 1; i < t.Urgency; i++ {
				urgencyIcon += "!"
			}
			fmt.Printf("%-4d %s %-30s %-10s %s%d\n", t.ID, statusIcon, truncate(t.Title, 28), due, urgencyIcon, t.Urgency)
		}
	},
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}

var doneCmd = &cobra.Command{
	Use:   "done [id]",
	Short: "Mark a todo as done",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		resp, err := getClient().Post(serverURL+"/todos/"+id+"/done", "application/json", nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			fmt.Fprintf(os.Stderr, "Error: %s\n", body)
			os.Exit(1)
		}

		fmt.Printf("Marked todo %s as done\n", id)
	},
}

var editCmd = &cobra.Command{
	Use:   "edit [id]",
	Short: "Edit a todo",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]

		fields := []string{}
		if title, _ := cmd.Flags().GetString("title"); title != "" {
			fields = append(fields, fmt.Sprintf(`"title": %q`, title))
		}
		if desc, _ := cmd.Flags().GetString("description"); desc != "" {
			fields = append(fields, fmt.Sprintf(`"description": %q`, desc))
		}
		if cat, _ := cmd.Flags().GetString("category"); cat != "" {
			fields = append(fields, fmt.Sprintf(`"category": %q`, cat))
		}
		if urgency, _ := cmd.Flags().GetInt("urgency"); urgency > 0 {
			fields = append(fields, fmt.Sprintf(`"urgency": %d`, urgency))
		}
		if dueDate, _ := cmd.Flags().GetString("due-date"); dueDate != "" {
			fields = append(fields, fmt.Sprintf(`"due_date": %q`, dueDate))
		}
		if dueTime, _ := cmd.Flags().GetString("due-time"); dueTime != "" {
			fields = append(fields, fmt.Sprintf(`"due_time": %q`, dueTime))
		}

		if len(fields) == 0 {
			fmt.Println("No fields to update")
			return
		}

		reqBody := "{" + strings.Join(fields, ",") + "}"
		req, _ := http.NewRequestWithContext(context.Background(), "PATCH", serverURL+"/todos/"+id, strings.NewReader(reqBody))
		req.Header.Set("Content-Type", "application/json")

		resp, err := getClient().Do(req)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			fmt.Fprintf(os.Stderr, "Error: %s\n", body)
			os.Exit(1)
		}

		fmt.Printf("Updated todo %s\n", id)
	},
}

var queueCmd = &cobra.Command{
	Use:   "queue",
	Short: "Show queue status",
	Run: func(cmd *cobra.Command, args []string) {
		resp, err := getClient().Get(serverURL + "/queue/status")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		var jobs []Job
		json.NewDecoder(resp.Body).Decode(&jobs)

		if len(jobs) == 0 {
			fmt.Println("Queue is empty")
			return
		}

		fmt.Printf("%-4s %-12s %-30s %s\n", "ID", "Status", "Text", "Attempts")
		fmt.Println(strings.Repeat("-", 70))
		for _, j := range jobs {
			fmt.Printf("%-4d %-12s %-30s %d\n", j.ID, j.Status, truncate(j.RawText, 28), j.Attempts)
		}
	},
}

var retryCmd = &cobra.Command{
	Use:   "retry [job-id]",
	Short: "Retry a failed job",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		id := args[0]
		resp, err := getClient().Post(serverURL+"/queue/"+id+"/retry", "application/json", nil)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			fmt.Fprintf(os.Stderr, "Error: %s\n", body)
			os.Exit(1)
		}

		fmt.Printf("Retrying job %s\n", id)
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&serverURL, "server", "http://127.0.0.1:8080", "Server URL")

	rootCmd.AddCommand(addCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(doneCmd)
	rootCmd.AddCommand(editCmd)
	rootCmd.AddCommand(queueCmd)
	rootCmd.AddCommand(retryCmd)

	listCmd.Flags().String("category", "", "Filter by category")
	listCmd.Flags().String("status", "", "Filter by status")
	listCmd.Flags().String("due-before", "", "Filter by due date")
	listCmd.Flags().Int("urgency-min", 0, "Filter by minimum urgency")
	listCmd.Flags().Int("limit", 0, "Limit results")

	editCmd.Flags().String("title", "", "New title")
	editCmd.Flags().String("description", "", "New description")
	editCmd.Flags().String("category", "", "New category")
	editCmd.Flags().Int("urgency", 0, "New urgency (1-5)")
	editCmd.Flags().String("due-date", "", "New due date (YYYY-MM-DD)")
	editCmd.Flags().String("due-time", "", "New due time (HH:MM)")
}

var rootCmd = &cobra.Command{
	Use:   "tdl",
	Short: "todoloo CLI",
}

func main() {
	if envServer := os.Getenv("TODOLOO_SERVER"); envServer != "" {
		serverURL = envServer
	}
	rootCmd.Execute()
}

func init() {
	_ = strconv.Atoi
}
