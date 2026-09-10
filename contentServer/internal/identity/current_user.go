package identity

import (
	"context"
	"errors"
)

var (
	ErrMissingUserID    = errors.New("missing userId in login context")
	ErrMissingSessionID = errors.New("missing sessionId in login context")
)

// CurrentUser contains the identity established by the login middleware.
type CurrentUser struct {
	UserID    string
	SessionID string
}

// FromContext returns the current user identity. Callers must never accept a
// request field as a substitute for this value when deciding resource ownership.
func FromContext(ctx context.Context) (CurrentUser, error) {
	userID, ok := ctx.Value("userId").(string)
	if !ok || userID == "" {
		return CurrentUser{}, ErrMissingUserID
	}

	sessionID, ok := ctx.Value("sessionId").(string)
	if !ok || sessionID == "" {
		return CurrentUser{}, ErrMissingSessionID
	}

	return CurrentUser{
		UserID:    userID,
		SessionID: sessionID,
	}, nil
}
