package model

import "github.com/zeromicro/go-zero/core/stores/sqlx"

var _ PostCardProjectionModel = (*customPostCardProjectionModel)(nil)

type (
	// PostCardProjectionModel is an interface to be customized, add more methods here,
	// and implement the added methods in customPostCardProjectionModel.
	PostCardProjectionModel interface {
		postCardProjectionModel
		withSession(session sqlx.Session) PostCardProjectionModel
	}

	customPostCardProjectionModel struct {
		*defaultPostCardProjectionModel
	}
)

// NewPostCardProjectionModel returns a model for the database table.
func NewPostCardProjectionModel(conn sqlx.SqlConn) PostCardProjectionModel {
	return &customPostCardProjectionModel{
		defaultPostCardProjectionModel: newPostCardProjectionModel(conn),
	}
}

func (m *customPostCardProjectionModel) withSession(session sqlx.Session) PostCardProjectionModel {
	return NewPostCardProjectionModel(sqlx.NewSqlConnFromSession(session))
}
