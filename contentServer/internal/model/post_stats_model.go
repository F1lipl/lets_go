package model

import "github.com/zeromicro/go-zero/core/stores/sqlx"

var _ PostStatsModel = (*customPostStatsModel)(nil)

type (
	// PostStatsModel is an interface to be customized, add more methods here,
	// and implement the added methods in customPostStatsModel.
	PostStatsModel interface {
		postStatsModel
		withSession(session sqlx.Session) PostStatsModel
	}

	customPostStatsModel struct {
		*defaultPostStatsModel
	}
)

// NewPostStatsModel returns a model for the database table.
func NewPostStatsModel(conn sqlx.SqlConn) PostStatsModel {
	return &customPostStatsModel{
		defaultPostStatsModel: newPostStatsModel(conn),
	}
}

func (m *customPostStatsModel) withSession(session sqlx.Session) PostStatsModel {
	return NewPostStatsModel(sqlx.NewSqlConnFromSession(session))
}
