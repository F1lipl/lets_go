package taskhandler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"contentserver/internal/domain"
	"contentserver/internal/event"
	"contentserver/internal/respository"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// PostCardHandler validates the event envelope and delegates projection to its
// repository. The task repository owns the transaction and delivery status.
type PostCardHandler struct{ repo PostCardProjector }

type PostCardProjector interface {
	ApplyPublication(context.Context, sqlx.Session, respository.PostCardPublication) error
}

func NewPostCardHandler(repo PostCardProjector) PostCardHandler { return PostCardHandler{repo: repo} }

var _ event.TaskHandler = PostCardHandler{}

type postPublishedPayload struct {
	SchemaVersion uint64 `json:"schemaVersion"`
	PostID        string `json:"postId"`
	AuthorID      string `json:"authorId"`
	RevisionID    string `json:"revisionId"`
	PostVersion   uint64 `json:"postVersion"`
}

func (h PostCardHandler) Handle(ctx context.Context, tx sqlx.Session, task *event.TaskContext) error {
	if task == nil || tx == nil || h.repo == nil {
		return event.Permanent(errors.New("post card task or transaction is missing"))
	}
	if task.EventType != "PostPublished" || task.SchemaVersion != 1 {
		return event.Permanent(fmt.Errorf("unsupported post card event %q version %d", task.EventType, task.SchemaVersion))
	}
	var payload postPublishedPayload
	if err := json.Unmarshal(task.Payload, &payload); err != nil {
		return event.Permanent(fmt.Errorf("decode post publication: %w", err))
	}
	if payload.SchemaVersion != task.SchemaVersion || payload.PostVersion == 0 {
		return event.Permanent(errors.New("invalid post publication version"))
	}
	postID, err := domain.ParsePostID(payload.PostID)
	if err != nil {
		return event.Permanent(fmt.Errorf("invalid post publication post id: %w", err))
	}
	authorID, err := domain.ParseUserID(payload.AuthorID)
	if err != nil {
		return event.Permanent(fmt.Errorf("invalid post publication author id: %w", err))
	}
	revisionID, err := domain.ParseRevisionID(payload.RevisionID)
	if err != nil {
		return event.Permanent(fmt.Errorf("invalid post publication revision id: %w", err))
	}
	err = h.repo.ApplyPublication(ctx, tx, respository.PostCardPublication{
		PostID: postID, AuthorID: authorID, RevisionID: revisionID, PostVersion: payload.PostVersion,
	})
	if errors.Is(err, respository.ErrPostCardSuperseded) {
		return event.ErrSuperseded
	}
	if errors.Is(err, respository.ErrPostCardSourceMissing) || errors.Is(err, respository.ErrPostCardSourceInvalid) {
		return event.Permanent(err)
	}
	return err
}
