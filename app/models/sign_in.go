package models

import "time"

// SignIn is one successful login, as much of it as the "new sign-in" email
// says: who, when, and from where. Not stored: it is carried by the email's
// job from the request that made it.
type SignIn struct {
	UserID    int64
	At        time.Time
	IP        string
	UserAgent string
}
