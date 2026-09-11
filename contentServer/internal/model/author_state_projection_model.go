package model

import "github.com/zeromicro/go-zero/core/stores/sqlx"

var _ AuthorStateProjectionModel = (*customAuthorStateProjectionModel)(nil)

type (
	// AuthorStateProjectionModel is an interface to be customized, add more methods here,
	// and implement the added methods in customAuthorStateProjectionModel.
	AuthorStateProjectionModel interface {
		authorStateProjectionModel
		withSession(session sqlx.Session) AuthorStateProjectionModel
	}

	customAuthorStateProjectionModel struct {
		*defaultAuthorStateProjectionModel
	}
)

// NewAuthorStateProjectionModel returns a model for the database table.
func NewAuthorStateProjectionModel(conn sqlx.SqlConn) AuthorStateProjectionModel {
	return &customAuthorStateProjectionModel{
		defaultAuthorStateProjectionModel: newAuthorStateProjectionModel(conn),
	}
}

func (m *customAuthorStateProjectionModel) withSession(session sqlx.Session) AuthorStateProjectionModel {
	return NewAuthorStateProjectionModel(sqlx.NewSqlConnFromSession(session))
}
