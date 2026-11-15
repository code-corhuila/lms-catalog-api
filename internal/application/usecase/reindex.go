package usecase

import (
	"context"

	"go.uber.org/zap"
)

// Reindex is a deliberate stub for ADR-009's third lms-worker
// responsibility. There is no search engine/index to rebuild in this
// project yet — HU-04's catalog search (search_books.go) is a plain SQL
// query against catalog.books, not a separate index. This only confirms
// the route is reachable and safe to call on lms-worker's schedule, so
// that job gets a real 200 instead of a 404. Replace this body once a
// real search index (and something to reindex) actually exists.
type Reindex struct {
	Log *zap.SugaredLogger
}

func NewReindex(log *zap.SugaredLogger) *Reindex {
	return &Reindex{Log: log}
}

func (uc *Reindex) Execute(_ context.Context) error {
	uc.Log.Info("reindex called — stub, no search index exists yet")
	return nil
}
