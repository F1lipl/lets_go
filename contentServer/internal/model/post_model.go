package model

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ PostModel = (*customPostModel)(nil)

type (
	// PostModel is an interface to be customized, add more methods here,
	// and implement the added methods in customPostModel.
	PostModel interface {
		postModel
		withSession(session sqlx.Session) PostModel
		FindOneForUpdate(ctx context.Context, postID string) (*Post, error)
		UpdateForDelete(
			ctx context.Context,
			authorID string,
			postID string,
			expectedVersion uint64,
			deletedAt time.Time,
		) (sql.Result, error)
	}

	customPostModel struct {
		*defaultPostModel
	}
)

// NewPostModel returns a model for the database table.
func NewPostModel(conn sqlx.SqlConn) PostModel {
	return &customPostModel{
		defaultPostModel: newPostModel(conn),
	}
}

func (m *customPostModel) withSession(session sqlx.Session) PostModel {
	return NewPostModel(sqlx.NewSqlConnFromSession(session))
}

func (m *customPostModel) FindOneForUpdate(ctx context.Context, postID string) (*Post, error) {
	query := fmt.Sprintf("select %s from %s where post_id=? limit 1 for update", postRows, m.table)
	var row Post
	if err := m.conn.QueryRowCtx(ctx, &row, query, postID); err != nil {
		if err == sqlx.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &row, nil
}

func (m *customPostModel) UpdateForDelete(
	ctx context.Context,
	authorID string,
	postID string,
	expectedVersion uint64,
	deletedAt time.Time,
) (sql.Result, error) {
	query := fmt.Sprintf(`
        UPDATE %s
        SET lifecycle_status = ?,
            deleted_at = ?,
            post_version = post_version + 1
        WHERE post_id = ?
          AND author_id = ?
          AND post_version = ?
          AND lifecycle_status <> ?
    `, m.table)

	return m.conn.ExecCtx(
		ctx,
		query,
		uint64(1),
		deletedAt,
		postID,
		authorID,
		expectedVersion,
		uint64(4),
	)
}
