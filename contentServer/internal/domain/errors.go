package domain

import "errors"

// Domain errors describe business concepts without depending on the API
// transport contract. The HTTP boundary maps these errors to stable ecode
// values.
var (
	ErrInvalidPostID          = errors.New("invalid post id")
	ErrInvalidUserID          = errors.New("invalid user id")
	ErrInvalidRevisionID      = errors.New("invalid revision id")
	ErrInvalidLifecycleStatus = errors.New("invalid lifecycle status")
	ErrInvalidVisibility      = errors.New("invalid visibility")
	ErrInvalidAssetID         = errors.New("invalid asset id")
)
