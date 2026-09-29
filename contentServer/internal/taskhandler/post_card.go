package taskhandler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"contentserver/internal/domain"
	"contentserver/internal/event"
	"contentserver/internal/model"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// PostCardHandler projects the current published revision. The task repository
// owns the transaction and marks the delivery terminal in the same commit.
type PostCardHandler struct{}

var _ event.TaskHandler = PostCardHandler{}

type postPublishedPayload struct {
	SchemaVersion uint64 `json:"schemaVersion"`
	PostID        string `json:"postId"`
	AuthorID      string `json:"authorId"`
	RevisionID    string `json:"revisionId"`
	PostVersion   uint64 `json:"postVersion"`
}

func (PostCardHandler) Handle(ctx context.Context, tx sqlx.Session, task *event.TaskContext) error {
	if task == nil || tx == nil {
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
	if _, err := domain.ParsePostID(payload.PostID); err != nil {
		return event.Permanent(fmt.Errorf("invalid post publication post id: %w", err))
	}
	if _, err := domain.ParseUserID(payload.AuthorID); err != nil {
		return event.Permanent(fmt.Errorf("invalid post publication author id: %w", err))
	}
	if _, err := domain.ParseRevisionID(payload.RevisionID); err != nil {
		return event.Permanent(fmt.Errorf("invalid post publication revision id: %w", err))
	}

	conn := sqlx.NewSqlConnFromSession(tx)
	post, err := model.NewPostModel(conn).FindOneForUpdate(ctx, payload.PostID)
	if errors.Is(err, model.ErrNotFound) {
		return event.Permanent(fmt.Errorf("post %s is missing", payload.PostID))
	}
	if err != nil {
		return err
	}
	if post.AuthorId != payload.AuthorID || post.PostVersion < payload.PostVersion {
		return event.Permanent(errors.New("post publication does not match current post"))
	}
	if post.LifecycleStatus == uint64(domain.LifecycleDeleted) || post.DeletedAt.Valid ||
		!post.PublishedRevisionId.Valid || post.PublishedRevisionId.String != payload.RevisionID {
		return event.ErrSuperseded
	}
	if post.LifecycleStatus != uint64(domain.LifecyclePublished) {
		return event.Permanent(errors.New("post is not published"))
	}

	revision, err := model.NewPostRevisionModel(conn).FindCardSnapshot(ctx, payload.RevisionID)
	if errors.Is(err, model.ErrNotFound) {
		return event.Permanent(fmt.Errorf("post revision %s is missing", payload.RevisionID))
	}
	if err != nil {
		return err
	}
	if revision.PostID != payload.PostID || !revision.CoverAssetID.Valid || revision.CoverAssetID.String == "" {
		return event.Permanent(errors.New("post revision has invalid card data"))
	}

	cards := model.NewPostCardProjectionModel(conn)
	current, err := cards.FindOne(ctx, payload.PostID)
	if err != nil && !errors.Is(err, model.ErrNotFound) {
		return err
	}
	if current != nil && current.SourcePostVersion >= post.PostVersion {
		return nil
	}
	card := &model.PostCardProjection{
		PostId: payload.PostID, RevisionId: revision.RevisionID,
		AuthorId: post.AuthorId, Title: revision.Title, Summary: revision.Summary,
		CoverAssetId: revision.CoverAssetID, Visibility: post.Visibility,
		AvailabilityStatus: post.AvailabilityStatus, PublishedAt: revision.PublishedAt,
		SourcePostVersion: post.PostVersion,
	}
	if current == nil {
		_, err = cards.Insert(ctx, card)
	} else {
		err = cards.Update(ctx, card)
	}
	return err
}
