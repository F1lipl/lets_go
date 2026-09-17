package respository

import (
	"contentserver/internal/domain"
	"contentserver/internal/model"
	"database/sql"
	"encoding/json"
	"time"
)

func optionalTime(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	t := value.Time
	return &t
}

func restorePost(row *model.Post) (*domain.Post, error) {
	id, err := domain.ParsePostID(row.PostId)
	if err != nil {
		return nil, err
	}
	author, err := domain.ParseUserID(row.AuthorId)
	if err != nil {
		return nil, err
	}
	post := &domain.Post{
		PostID: id, AuthorID: author,
		LifecycleStatus: domain.LifecycleStatus(row.LifecycleStatus),
		Visibility:      domain.Visibility(row.Visibility), AvailabilityStatus: row.AvailabilityStatus,
		RevisionSequence: row.RevisionSequence, Version: row.PostVersion,
		FirstPublishedAt: optionalTime(row.FirstPublishedAt), LastPublishedAt: optionalTime(row.LastPublishedAt),
		CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt, DeletedAt: optionalTime(row.DeletedAt),
	}
	if row.PublishedRevisionId.Valid {
		revisionID, err := domain.ParseRevisionID(row.PublishedRevisionId.String)
		if err != nil {
			return nil, err
		}
		post.PublishedRevisionID = &revisionID
	}
	return post, nil
}

func restoreDraft(row *model.PostDraft) (*domain.PostDraft, error) {
	id, err := domain.ParsePostID(row.PostId)
	if err != nil {
		return nil, err
	}
	draft := &domain.PostDraft{
		PostID: id, Title: row.Title, Summary: row.Summary, Document: row.DocumentJson,
		DocumentSchemaVersion: row.DocumentSchemaVersion, PlainText: row.PlainText,
		BlockCount: row.BlockCount, ImageCount: row.ImageCount,
		Version: row.DraftVersion, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
	if row.CoverAssetId.Valid {
		assetID, err := domain.ParseAssetID(row.CoverAssetId.String)
		if err != nil {
			return nil, err
		}
		draft.Cover = &domain.Cover{AssetID: assetID, FocusX: row.CoverFocusX.Float64, FocusY: row.CoverFocusY.Float64, CropStyle: row.CoverCropStyle}
	}
	if err := json.Unmarshal([]byte(row.TagNamesJson), &draft.TagNames); err != nil {
		return nil, err
	}
	return draft, nil
}

func revisionRow(revision *domain.PostRevision) *model.PostRevision {
	row := &model.PostRevision{
		RevisionId: revision.RevisionId.String(), PostId: revision.PostId.String(),
		RevisionNumber: revision.RevisionNumber, SourceDraftVersion: revision.SourceDraftVersion,
		Title: revision.Title, Summary: revision.Summary, DocumentJson: revision.Document,
		DocumentSchemaVersion: revision.DocumentSchemaVersion, PlainText: revision.PlainText,
		BlockCount: revision.BlockCount, ImageCount: revision.ImageCount, PublishedAt: revision.PublishedAt,
	}
	if revision.Cover != nil {
		row.CoverAssetId = sql.NullString{String: revision.Cover.AssetID.String(), Valid: true}
		row.CoverFocusX = sql.NullFloat64{Float64: revision.Cover.FocusX, Valid: true}
		row.CoverFocusY = sql.NullFloat64{Float64: revision.Cover.FocusY, Valid: true}
		row.CoverCropStyle = revision.Cover.CropStyle
	}
	return row
}
