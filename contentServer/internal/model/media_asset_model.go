package model

import "github.com/zeromicro/go-zero/core/stores/sqlx"

var _ MediaAssetModel = (*customMediaAssetModel)(nil)

type (
	// MediaAssetModel is an interface to be customized, add more methods here,
	// and implement the added methods in customMediaAssetModel.
	MediaAssetModel interface {
		mediaAssetModel
		withSession(session sqlx.Session) MediaAssetModel
	}

	customMediaAssetModel struct {
		*defaultMediaAssetModel
	}
)

// NewMediaAssetModel returns a model for the database table.
func NewMediaAssetModel(conn sqlx.SqlConn) MediaAssetModel {
	return &customMediaAssetModel{
		defaultMediaAssetModel: newMediaAssetModel(conn),
	}
}

func (m *customMediaAssetModel) withSession(session sqlx.Session) MediaAssetModel {
	return NewMediaAssetModel(sqlx.NewSqlConnFromSession(session))
}
