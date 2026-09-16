package respository

import (
	"contentserver/internal/domain"
	"contentserver/internal/model"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type postDraftRepository struct {
}

func getImageCount(draft *domain.PostDraft) uint64 {
	var count uint64
	for _, block := range draft.Document.Blocks {
		count += uint64(len(block.AssetIDs))
	}
	return count
}

func toPostDraft(draft *domain.PostDraft) (*model.PostDraft, error) {
	documentJson, err := json.Marshal(draft.Document)
	if err != nil {
		return nil, err
	}
	tag, err := json.Marshal(draft.TagNames)
	if err != nil {
		return nil, err
	}
	now := time.Now()

	newDraft := &model.PostDraft{
		PostId:                draft.PostID.String(),
		DraftVersion:          draft.Version,
		Title:                 draft.Title,
		Summary:               draft.Summary,
		DocumentSchemaVersion: draft.Document.SchemaVersion,
		DocumentJson:          string(documentJson),
		TagNamesJson:          string(tag),
		PlainText:             "",
		BlockCount:            uint64(len(draft.Document.Blocks)),
		ImageCount:            getImageCount(draft),
		CreatedAt:             now,
		UpdatedAt:             now,
	}
	if draft.Cover != nil {
		newDraft.CoverAssetId = sql.NullString{
			String: draft.Cover.AssetID.String(),
			Valid:  true,
		}
		newDraft.CoverFocusX = sql.NullFloat64{
			Float64: draft.Cover.FocusX,
			Valid:   true,
		}
		newDraft.CoverFocusY = sql.NullFloat64{
			Float64: draft.Cover.FocusY,
			Valid:   true,
		}
		newDraft.CoverCropStyle = draft.Cover.CropStyle
	}
	return newDraft, nil
}

func (postDraft *postDraftRepository) CreatePostDraft(ctx context.Context, conn sqlx.SqlConn, draft *domain.PostDraft) error {
	postDraftModel := model.NewPostDraftModel(conn)
	Postdraft, err := toPostDraft(draft)
	if err != nil {
		return err
	}
	_, err = postDraftModel.Insert(ctx, Postdraft)
	if err != nil {
		return err
	}
	return nil
}

func (postDraft *postDraftRepository) SaveDraft(ctx context.Context, conn sqlx.SqlConn, draft *domain.PostDraft, expectedVersion uint64) error {
	postDraftModel := model.NewPostDraftModel(conn)
	newDraft, err := toPostDraft(draft)
	if err != nil {
		return err
	}
	err = postDraftModel.UpdateWithVersion(ctx, newDraft, expectedVersion)
	if err != nil {
		if errors.Is(err, model.ErrVersionConflict) {
			return domain.ErrDraftVersionConflict
		}
		return err
	}
	draft.Version = expectedVersion + 1
	return nil
	//TODO event
}

func (postDraft *postDraftRepository) DeleteDraft(ctx context.Context, conn sqlx.SqlConn, id domain.PostID) error {
	if id.IsZero() {
		return domain.ErrInvalidPostID
	}
	postDraftModel := model.NewPostDraftModel(conn)
	err := postDraftModel.Delete(ctx, id.String())
	if err != nil {
		return err
	}
	return nil
	//TODO event
}
