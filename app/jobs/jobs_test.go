package jobs

import (
	"io"
	"log/slog"
	"testing"
)

func TestWorkersBuild(t *testing.T) {
	workers, registered := Workers(slog.New(slog.NewTextHandler(io.Discard, nil)), Deps{})
	if workers == nil || !registered {
		t.Fatalf("workers = %v, registered = %v; want a bundle with the email workers", workers, registered)
	}
}
