package model

import (
	"context"
	"fmt"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ PostDraftModel = (*customPostDraftModel)(nil)

type (
	// PostDraftModel is an interface to be customized, add more methods here,
	// and implement the added methods in customPostDraftModel.
	PostDraftModel interface {
		postDraftModel
		withSession(session sqlx.Session) PostDraftModel
		UpdateWithVersion(ctx context.Context, post *PostDraft, version uint64) error
	}

	customPostDraftModel struct {
		*defaultPostDraftModel
	}
)

// NewPostDraftModel returns a model for the database table.
func NewPostDraftModel(conn sqlx.SqlConn) PostDraftModel {
	return &customPostDraftModel{
		defaultPostDraftModel: newPostDraftModel(conn),
	}
}

func (m *customPostDraftModel) withSession(session sqlx.Session) PostDraftModel {
	return NewPostDraftModel(sqlx.NewSqlConnFromSession(session))
}

func (m *customPostDraftModel) UpdateWithVersion(ctx context.Context, post *PostDraft, version uint64) error {
	query := fmt.Sprintf(
		`update %s set draft_version=draft_version+1, title=?, summary=?, cover_asset_id=?, cover_focus_x=?, cover_focus_y=?, cover_crop_style=?, document_schema_version=?, document_json=?, tag_names_json=?, plain_text=?, block_count=?, image_count=? where post_id=? and draft_version=?`,
		m.table,
	)
	result, err := m.conn.ExecCtx(
		ctx,
		query,
		post.Title,
		post.Summary,
		post.CoverAssetId,
		post.CoverFocusX,
		post.CoverFocusY,
		post.CoverCropStyle,
		post.DocumentSchemaVersion,
		post.DocumentJson,
		post.TagNamesJson,
		post.PlainText,
		post.BlockCount,
		post.ImageCount,
		post.PostId,
		version,
	)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrVersionConflict
	}
	return nil
}
