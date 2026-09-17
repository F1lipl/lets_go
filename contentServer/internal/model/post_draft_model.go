package model

import (
	"context"
	"errors"
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
		SelectForUpdate(ctx context.Context, id string, version uint64) (*PostDraft, error)
		FindOneForUpdate(ctx context.Context, id string) (*PostDraft, error)
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

func (m *customPostDraftModel) SelectForUpdate(ctx context.Context, id string, version uint64) (*PostDraft, error) {
	query := fmt.Sprintf(
		`select * from %s where post_id=? AND draft_version=? for update`, m.table,
	)
	var resp PostDraft
	err := m.conn.QueryRowCtx(ctx, &resp, query, id, version)
	switch {
	case err == nil:
		return &resp, nil
	case errors.Is(err, sqlx.ErrNotFound):
		return nil, ErrNotFound
	default:
		return nil, err
	}

}

func (m *customPostDraftModel) FindOneForUpdate(ctx context.Context, id string) (*PostDraft, error) {
	var row PostDraft
	query := fmt.Sprintf("select %s from %s where post_id=? for update", postDraftRows, m.table)
	if err := m.conn.QueryRowCtx(ctx, &row, query, id); err != nil {
		return nil, err
	}
	return &row, nil
}
