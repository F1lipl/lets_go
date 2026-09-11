package model

import "github.com/zeromicro/go-zero/core/stores/sqlx"

var _ PostRevisionModel = (*customPostRevisionModel)(nil)

type (
	// PostRevisionModel is an interface to be customized, add more methods here,
	// and implement the added methods in customPostRevisionModel.
	PostRevisionModel interface {
		postRevisionModel
		withSession(session sqlx.Session) PostRevisionModel
	}

	customPostRevisionModel struct {
		*defaultPostRevisionModel
	}
)

// NewPostRevisionModel returns a model for the database table.
func NewPostRevisionModel(conn sqlx.SqlConn) PostRevisionModel {
	return &customPostRevisionModel{
		defaultPostRevisionModel: newPostRevisionModel(conn),
	}
}

func (m *customPostRevisionModel) withSession(session sqlx.Session) PostRevisionModel {
	return NewPostRevisionModel(sqlx.NewSqlConnFromSession(session))
}
