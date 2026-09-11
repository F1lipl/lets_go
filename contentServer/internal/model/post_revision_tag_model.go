package model

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type PostRevisionTag struct {
	RevisionId      string    `db:"revision_id"`
	TagId           string    `db:"tag_id"`
	TagNameSnapshot string    `db:"tag_name_snapshot"`
	SortOrder       uint64    `db:"sort_order"`
	CreatedAt       time.Time `db:"created_at"`
}

type PostRevisionTagModel interface {
	Insert(ctx context.Context, data *PostRevisionTag) (sql.Result, error)
	FindByRevisionId(ctx context.Context, revisionId string) ([]PostRevisionTag, error)
	withSession(session sqlx.Session) PostRevisionTagModel
}

type defaultPostRevisionTagModel struct {
	conn  sqlx.SqlConn
	table string
}

func NewPostRevisionTagModel(conn sqlx.SqlConn) PostRevisionTagModel {
	return &defaultPostRevisionTagModel{
		conn:  conn,
		table: "`post_revision_tag`",
	}
}

func (m *defaultPostRevisionTagModel) Insert(ctx context.Context, data *PostRevisionTag) (sql.Result, error) {
	query := fmt.Sprintf("insert into %s (`revision_id`,`tag_id`,`tag_name_snapshot`,`sort_order`) values (?,?,?,?)", m.table)
	return m.conn.ExecCtx(ctx, query, data.RevisionId, data.TagId, data.TagNameSnapshot, data.SortOrder)
}

func (m *defaultPostRevisionTagModel) FindByRevisionId(ctx context.Context, revisionId string) ([]PostRevisionTag, error) {
	query := fmt.Sprintf("select `revision_id`,`tag_id`,`tag_name_snapshot`,`sort_order`,`created_at` from %s where `revision_id` = ? order by `sort_order`", m.table)
	var rows []PostRevisionTag
	if err := m.conn.QueryRowsCtx(ctx, &rows, query, revisionId); err != nil {
		return nil, err
	}
	return rows, nil
}

func (m *defaultPostRevisionTagModel) withSession(session sqlx.Session) PostRevisionTagModel {
	return NewPostRevisionTagModel(sqlx.NewSqlConnFromSession(session))
}
