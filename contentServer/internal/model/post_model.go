package model

import "github.com/zeromicro/go-zero/core/stores/sqlx"

var _ PostModel = (*customPostModel)(nil)

type (
	// PostModel is an interface to be customized, add more methods here,
	// and implement the added methods in customPostModel.
	PostModel interface {
		postModel
		withSession(session sqlx.Session) PostModel
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
