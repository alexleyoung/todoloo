package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/alexyoung/todoloo/internal/api"
	"github.com/alexyoung/todoloo/internal/config"
	"github.com/alexyoung/todoloo/internal/db"
	"github.com/alexyoung/todoloo/internal/nlp"
	"github.com/alexyoung/todoloo/internal/queue"
)

var (
	configPath string
	pidFile    string
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: todoloo <start|stop|run>")
		os.Exit(1)
	}

	cmd := os.Args[1]

	switch cmd {
	case "start":
		startServer()
	case "stop":
		stopServer()
	case "run":
		runServer()
	default:
		fmt.Printf("Unknown command: %s\n", cmd)
		os.Exit(1)
	}
}

func getPaths() (string, string) {
	home, _ := os.UserHomeDir()
	if home == "" {
		home = os.Getenv("HOME")
	}
	todolooDir := home + "/.todoloo"
	return todolooDir + "/config.yaml", todolooDir + "/todoloo.pid"
}

func startServer() {
	configPath, pidFile = getPaths()

	if pidFileExists() {
		fmt.Println("Server is already running (PID file exists)")
		os.Exit(1)
	}

	cfg, err := config.LoadPath(configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	database, err := db.Open(cfg.Database.Path)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer database.Close()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Start queue worker
	parser, err := nlp.NewParser(cfg.LLM)
	if err != nil {
		log.Fatalf("failed to create parser: %v", err)
	}

	worker := queue.NewWorker(database, parser, cfg.Queue)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		worker.Run(ctx)
	}()

	// Start HTTP server
	router := api.Router(database)
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)

	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Printf("server starting on %s", addr)
		if err := api.Serve(addr, router); err != nil {
			log.Printf("server error: %v", err)
		}
	}()

	writePIDFile()
	wg.Wait()
	removePIDFile()
	log.Println("shutdown complete")
}

func runServer() {
	configPath, _ = getPaths()

	cfg, err := config.LoadPath(configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	database, err := db.Open(cfg.Database.Path)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer database.Close()

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// Start queue worker
	parser, err := nlp.NewParser(cfg.LLM)
	if err != nil {
		log.Fatalf("failed to create parser: %v", err)
	}

	worker := queue.NewWorker(database, parser, cfg.Queue)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		worker.Run(ctx)
	}()

	// Start HTTP server
	router := api.Router(database)
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)

	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Printf("server starting on %s", addr)
		if err := api.Serve(addr, router); err != nil {
			log.Printf("server error: %v", err)
		}
	}()

	wg.Wait()
	log.Println("shutdown complete")
}

func stopServer() {
	_, pidFile = getPaths()

	if !pidFileExists() {
		fmt.Println("Server is not running (no PID file)")
		os.Exit(1)
	}

	pid, err := readPIDFile()
	if err != nil {
		fmt.Println("Failed to read PID file")
		os.Exit(1)
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		fmt.Printf("Failed to find process: %v\n", err)
		removePIDFile()
		os.Exit(1)
	}

	if err := proc.Kill(); err != nil {
		fmt.Printf("Failed to stop server: %v\n", err)
		os.Exit(1)
	}

	removePIDFile()
	fmt.Println("Server stopped")
}

func pidFileExists() bool {
	_, err := os.Stat(pidFile)
	return err == nil
}

func writePIDFile() {
	pid := os.Getpid()
	os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", pid)), 0644)
}

func readPIDFile() (int, error) {
	data, err := os.ReadFile(pidFile)
	if err != nil {
		return 0, err
	}
	var pid int
	_, err = fmt.Sscanf(string(data), "%d", &pid)
	return pid, err
}

func removePIDFile() {
	os.Remove(pidFile)
}
