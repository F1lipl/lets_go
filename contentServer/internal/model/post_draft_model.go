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
		updateWithVersion(ctx context.Context, post *PostDraft, version int64) error
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

func (m *customPostDraftModel) updateWithVersion(ctx context.Context, post *PostDraft, version int64) error {
	query := fmt.Sprintf(
		`update %s set %s where post_id=? AND draft_version=?`, m.table, postRowsWithPlaceHolder,
	)
	_, err := m.conn.ExecCtx(ctx, query, post.PostId, post.DraftVersion, version)
	if err != nil {
		return err
	}
	return nil
}
