package nlp

import (
	"context"
	"time"

	"github.com/alexyoung/todoloo/internal/models"
)

type ParserMeta struct {
	Today      time.Time
	Categories []string
}

type Parser interface {
	Parse(ctx context.Context, raw string, meta ParserMeta) (*models.ParsedTodo, error)
}
