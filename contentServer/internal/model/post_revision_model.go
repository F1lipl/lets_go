package model

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ PostRevisionModel = (*customPostRevisionModel)(nil)

type (
	// PostRevisionModel is an interface to be customized, add more methods here,
	// and implement the added methods in customPostRevisionModel.
	PostRevisionModel interface {
		postRevisionModel
		withSession(session sqlx.Session) PostRevisionModel
		FindCardSnapshot(ctx context.Context, revisionID string) (*PostRevisionCardSnapshot, error)
	}

	customPostRevisionModel struct {
		*defaultPostRevisionModel
	}
)

// PostRevisionCardSnapshot loads only the fields required for a card. Reading
// the full revision would also load document_json and plain_text.
type PostRevisionCardSnapshot struct {
	RevisionID   string         `db:"revision_id"`
	PostID       string         `db:"post_id"`
	Title        string         `db:"title"`
	Summary      string         `db:"summary"`
	CoverAssetID sql.NullString `db:"cover_asset_id"`
	PublishedAt  time.Time      `db:"published_at"`
}

// NewPostRevisionModel returns a model for the database table.
func NewPostRevisionModel(conn sqlx.SqlConn) PostRevisionModel {
	return &customPostRevisionModel{
		defaultPostRevisionModel: newPostRevisionModel(conn),
	}
}

func (m *customPostRevisionModel) withSession(session sqlx.Session) PostRevisionModel {
	return NewPostRevisionModel(sqlx.NewSqlConnFromSession(session))
}

func (m *customPostRevisionModel) FindCardSnapshot(ctx context.Context, revisionID string) (*PostRevisionCardSnapshot, error) {
	query := fmt.Sprintf("SELECT revision_id,post_id,title,summary,cover_asset_id,published_at FROM %s WHERE revision_id=? LIMIT 1", m.table)
	var row PostRevisionCardSnapshot
	if err := m.conn.QueryRowCtx(ctx, &row, query, revisionID); err != nil {
		if err == sqlx.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &row, nil
}
