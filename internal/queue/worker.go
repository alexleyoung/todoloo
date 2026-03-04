package queue

import (
	"context"
	"log"
	"math"
	"time"

	"github.com/alexyoung/todoloo/internal/config"
	"github.com/alexyoung/todoloo/internal/db"
	"github.com/alexyoung/todoloo/internal/nlp"
)

type Worker struct {
	db     *db.DB
	parser nlp.Parser
	cfg    config.QueueConfig
}

func NewWorker(database *db.DB, parser nlp.Parser, cfg config.QueueConfig) *Worker {
	return &Worker{
		db:     database,
		parser: parser,
		cfg:    cfg,
	}
}

func (w *Worker) Run(ctx context.Context) {
	w.process(ctx)

	ticker := time.NewTicker(time.Duration(w.cfg.PollIntervalSeconds) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			w.process(ctx)
		case <-ctx.Done():
			return
		}
	}
}

func (w *Worker) process(ctx context.Context) {
	jobs, err := w.db.GetActionableJobs(ctx, w.cfg.MaxRetries)
	if err != nil {
		log.Printf("queue poll failed: %v", err)
		return
	}

	for _, job := range jobs {
		w.db.MarkProcessing(ctx, job.ID)

		result, err := w.parser.Parse(ctx, job.RawText, nlp.ParserMeta{
			Today:      time.Now(),
			Categories: w.cfg.Categories,
		})

		if err != nil {
			delay := w.backoffDelay(job.Attempts)
			w.db.MarkFailed(ctx, job.ID, err.Error())
			log.Printf("parse failed, will retry job=%d attempt=%d retry_in=%v err=%v", job.ID, job.Attempts, delay, err)
			continue
		}

		if _, err := w.db.InsertTodo(ctx, result, job.ID); err != nil {
			w.db.MarkFailed(ctx, job.ID, err.Error())
			log.Printf("insert todo failed: %v", err)
			continue
		}

		w.db.MarkDone(ctx, job.ID)
		log.Printf("todo processed job=%d title=%s", job.ID, result.Title)
	}
}

func (w *Worker) backoffDelay(attempts int) time.Duration {
	base := 30 * time.Second
	mult := math.Pow(w.cfg.BackoffMultiplier, float64(attempts))
	return time.Duration(float64(base) * mult)
}
