package views

// OpsResponse is GET /admin/ops: what an operator checks to see that the
// system is healthy. It is for staff, so it may say what a public probe must
// not.
type OpsResponse struct {
	// Version is the release that is running (Kamal's git sha), or "dev".
	Version string `json:"version"`
	// Env is the behaviour the process runs with.
	Env string `json:"env"`
	// SchemaVersion is the newest migration applied.
	SchemaVersion int64 `json:"schema_version"`
	// Jobs counts the queue's jobs by state.
	Jobs map[string]int64 `json:"jobs"`
	// OldestWaitingSeconds is the queue's lag: how long the job that has
	// waited longest has been waiting. 0 when nothing waits.
	OldestWaitingSeconds float64 `json:"oldest_waiting_seconds"`
}
