package nlp

import (
	"context"
	"fmt"
	"time"

	"github.com/olebedev/when"
	"github.com/olebedev/when/rules/common"
	"github.com/olebedev/when/rules/en"
)

var w *when.Parser

func init() {
	w = when.New(nil)
	w.Add(en.All...)
	w.Add(common.All...)
}

type LLMFallback func(ctx context.Context, expr string, ref time.Time) (time.Time, error)

func ResolveDate(ctx context.Context, expr string, ref time.Time, llmFallback LLMFallback) (date, timeOfDay string, err error) {
	if expr == "" {
		return "", "", nil
	}

	result, err := w.Parse(expr, ref)
	if err == nil && result != nil {
		return result.Time.Format("2006-01-02"), result.Time.Format("15:04"), nil
	}

	if llmFallback == nil {
		return "", "", fmt.Errorf("date resolution failed for %q and no fallback available", expr)
	}

	t, err := llmFallback(ctx, expr, ref)
	if err != nil {
		return "", "", fmt.Errorf("date resolution failed for %q: %w", expr, err)
	}
	return t.Format("2006-01-02"), t.Format("15:04"), nil
}
