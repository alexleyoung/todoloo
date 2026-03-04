package nlp

import (
	"fmt"

	"github.com/alexyoung/todoloo/internal/config"
)

func NewParser(cfg config.LLMConfig) (Parser, error) {
	switch cfg.Provider {
	case "ollama":
		return NewOllamaParser(cfg), nil
	default:
		return nil, fmt.Errorf("unknown LLM provider: %s", cfg.Provider)
	}
}
