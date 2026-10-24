package mailer

import "time"

// SetClock fixes the mailer's clock for the external test package.
func SetClock(m *Mailer, now func() time.Time) { m.now = now }
