package views

// GuestSessionResponse is the body of POST /guest_sessions: a bearer token
// good for any route that requires one, the same as a real login's would
// be.
type GuestSessionResponse struct {
	Token string `json:"token"`
}
