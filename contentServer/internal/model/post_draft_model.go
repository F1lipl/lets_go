package model

import "github.com/zeromicro/go-zero/core/stores/sqlx"

var _ PostDraftModel = (*customPostDraftModel)(nil)

type (
	// PostDraftModel is an interface to be customized, add more methods here,
	// and implement the added methods in customPostDraftModel.
	PostDraftModel interface {
		postDraftModel
		withSession(session sqlx.Session) PostDraftModel
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
