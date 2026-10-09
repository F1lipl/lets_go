package model

import (
	"context"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ TagModel = (*customTagModel)(nil)

type (
	// TagModel is an interface to be customized, add more methods here,
	// and implement the added methods in customTagModel.
	TagModel interface {
		tagModel
		FindOrCreate(ctx context.Context, id, normalizedName, displayName, authorID string) (*Tag, error)
		withSession(session sqlx.Session) TagModel
	}

	customTagModel struct {
		*defaultTagModel
	}
)

// NewTagModel returns a model for the database table.
func NewTagModel(conn sqlx.SqlConn) TagModel {
	return &customTagModel{
		defaultTagModel: newTagModel(conn),
	}
}

func (m *customTagModel) FindOrCreate(ctx context.Context, id, normalizedName, displayName, authorID string) (*Tag, error) {
	_, err := m.conn.ExecCtx(ctx,
		"INSERT INTO tag (tag_id,normalized_name,display_name,status,created_by) VALUES (?,?,?,1,?) ON DUPLICATE KEY UPDATE tag_id=tag_id",
		id, normalizedName, displayName, authorID)
	if err != nil {
		return nil, err
	}
	var row Tag
	if err := m.conn.QueryRowCtx(ctx, &row,
		"SELECT tag_id,normalized_name,display_name,status,created_by,created_at,updated_at FROM tag WHERE normalized_name=? FOR UPDATE", normalizedName); err != nil {
		return nil, err
	}
	return &row, nil
}

func (m *customTagModel) withSession(session sqlx.Session) TagModel {
	return NewTagModel(sqlx.NewSqlConnFromSession(session))
}
