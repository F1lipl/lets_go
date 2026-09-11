package model

import "github.com/zeromicro/go-zero/core/stores/sqlx"

var _ MediaVariantModel = (*customMediaVariantModel)(nil)

type (
	// MediaVariantModel is an interface to be customized, add more methods here,
	// and implement the added methods in customMediaVariantModel.
	MediaVariantModel interface {
		mediaVariantModel
		withSession(session sqlx.Session) MediaVariantModel
	}

	customMediaVariantModel struct {
		*defaultMediaVariantModel
	}
)

// NewMediaVariantModel returns a model for the database table.
func NewMediaVariantModel(conn sqlx.SqlConn) MediaVariantModel {
	return &customMediaVariantModel{
		defaultMediaVariantModel: newMediaVariantModel(conn),
	}
}

func (m *customMediaVariantModel) withSession(session sqlx.Session) MediaVariantModel {
	return NewMediaVariantModel(sqlx.NewSqlConnFromSession(session))
}
