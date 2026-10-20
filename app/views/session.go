package views

// SignupRequest is the body of POST /users.
type SignupRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// LoginRequest is the body of POST /sessions.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// SessionResponse is a minted bearer token: what POST /users and
// POST /sessions both answer with, the same shape GuestSessionResponse is —
// a signed-in user's token and a guest's are the same kind of thing, just
// minted a different way.
type SessionResponse struct {
	Token string `json:"token"`
}
