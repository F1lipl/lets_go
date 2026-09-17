package domain

import "errors"

// Domain errors describe business concepts without depending on the API
// transport contract. The HTTP boundary maps these errors to stable ecode
// values.
var (
	ErrMediaAssetNotReady      = errors.New("media asset is not ready")
	ErrDraftNotFound           = errors.New("draft not found")
	ErrDraftContentInvalid     = errors.New("draft content invalid")
	ErrPublishNotAllowed       = errors.New("publish not allowed")
	ErrInvalidPost             = errors.New("invalid post")
	ErrDraftPostMismatch       = errors.New("post draft does not belong to post")
	ErrInvalidPostID           = errors.New("invalid post id")
	ErrInvalidUserID           = errors.New("invalid user id")
	ErrInvalidRevisionID       = errors.New("invalid revision id")
	ErrInvalidLifecycleStatus  = errors.New("invalid lifecycle status")
	ErrInvalidVisibility       = errors.New("invalid visibility")
	ErrInvalidAssetID          = errors.New("invalid asset id")
	ErrInvalidVersion          = errors.New("invalid version")
	ErrDraftVersionConflict    = errors.New("draft version conflict")
	ErrPostNotFound            = errors.New("post not found")
	ErrPostAlreadyDeleted      = errors.New("post already deleted")
	ErrPostOperationNotAllowed = errors.New("post operation not allowed")
	ErrPostVersionConflict     = errors.New("post version conflict")
)
