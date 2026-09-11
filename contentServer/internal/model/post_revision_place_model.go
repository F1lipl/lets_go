package model

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type PostRevisionPlace struct {
	RevisionId string    `db:"revision_id"`
	PlaceId    string    `db:"place_id"`
	BlockId    string    `db:"block_id"`
	SortOrder  uint64    `db:"sort_order"`
	CreatedAt  time.Time `db:"created_at"`
}

type PostRevisionPlaceModel interface {
	Insert(ctx context.Context, data *PostRevisionPlace) (sql.Result, error)
	FindByRevisionId(ctx context.Context, revisionId string) ([]PostRevisionPlace, error)
	withSession(session sqlx.Session) PostRevisionPlaceModel
}

type defaultPostRevisionPlaceModel struct {
	conn  sqlx.SqlConn
	table string
}

func NewPostRevisionPlaceModel(conn sqlx.SqlConn) PostRevisionPlaceModel {
	return &defaultPostRevisionPlaceModel{
		conn:  conn,
		table: "`post_revision_place`",
	}
}

func (m *defaultPostRevisionPlaceModel) Insert(ctx context.Context, data *PostRevisionPlace) (sql.Result, error) {
	query := fmt.Sprintf("insert into %s (`revision_id`,`place_id`,`block_id`,`sort_order`) values (?,?,?,?)", m.table)
	return m.conn.ExecCtx(ctx, query, data.RevisionId, data.PlaceId, data.BlockId, data.SortOrder)
}

func (m *defaultPostRevisionPlaceModel) FindByRevisionId(ctx context.Context, revisionId string) ([]PostRevisionPlace, error) {
	query := fmt.Sprintf("select `revision_id`,`place_id`,`block_id`,`sort_order`,`created_at` from %s where `revision_id` = ? order by `sort_order`", m.table)
	var rows []PostRevisionPlace
	if err := m.conn.QueryRowsCtx(ctx, &rows, query, revisionId); err != nil {
		return nil, err
	}
	return rows, nil
}

func (m *defaultPostRevisionPlaceModel) withSession(session sqlx.Session) PostRevisionPlaceModel {
	return NewPostRevisionPlaceModel(sqlx.NewSqlConnFromSession(session))
}
