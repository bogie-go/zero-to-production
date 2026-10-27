package models

import "time"

// Ops is what an operator looks at to see whether the system is healthy: how
// far the schema is, and what the job queue is doing. Not a model, nothing
// is stored; it is read off the database.
type Ops struct {
	// SchemaVersion is the newest migration applied.
	SchemaVersion int64
	// Jobs counts the queue's jobs by state (available, running, retryable,
	// scheduled, completed, cancelled, discarded).
	Jobs map[string]int64
	// OldestWaiting is how long the longest-waiting available job has
	// waited: the queue's lag. Zero when nothing is waiting.
	OldestWaiting time.Duration
}
