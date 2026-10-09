package model

import (
	"context"
	"fmt"
	"strings"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var _ MediaAssetModel = (*customMediaAssetModel)(nil)

type (
	// MediaAssetModel is an interface to be customized, add more methods here,
	// and implement the added methods in customMediaAssetModel.
	MediaAssetModel interface {
		mediaAssetModel
		withSession(session sqlx.Session) MediaAssetModel
		FindReadyByIDs(ctx context.Context, ids []string) ([]MediaAsset, error)
		FindByIDsForShare(ctx context.Context, ids []string) ([]MediaAsset, error)
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

func (m *customMediaAssetModel) FindByIDsForShare(ctx context.Context, ids []string) ([]MediaAsset, error) {
	if len(ids) == 0 {
		return []MediaAsset{}, nil
	}
	args := make([]any, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	query := fmt.Sprintf("select %s from %s where asset_id in (%s) order by asset_id for share",
		mediaAssetRows, m.table, strings.TrimSuffix(strings.Repeat("?,", len(ids)), ","))
	var rows []MediaAsset
	if err := m.conn.QueryRowsCtx(ctx, &rows, query, args...); err != nil {
		return nil, err
	}
	return rows, nil
}

func (m *customMediaAssetModel) withSession(session sqlx.Session) MediaAssetModel {
	return NewMediaAssetModel(sqlx.NewSqlConnFromSession(session))
}

// FindReadyByIDs avoids one query per image when a published document contains
// multiple assets. Chunking keeps the number of SQL parameters bounded.
func (m *customMediaAssetModel) FindReadyByIDs(ctx context.Context, ids []string) ([]MediaAsset, error) {
	if len(ids) == 0 {
		return []MediaAsset{}, nil
	}
	rows := make([]MediaAsset, 0, len(ids))
	for start := 0; start < len(ids); start += 100 {
		end := start + 100
		if end > len(ids) {
			end = len(ids)
		}
		args := make([]any, end-start)
		for i, id := range ids[start:end] {
			args[i] = id
		}
		query := fmt.Sprintf("select %s from %s where asset_id in (%s) and status=3 and deleted_at is null",
			mediaAssetRows, m.table, strings.TrimSuffix(strings.Repeat("?,", len(args)), ","))
		var chunk []MediaAsset
		if err := m.conn.QueryRowsCtx(ctx, &chunk, query, args...); err != nil {
			return nil, err
		}
		rows = append(rows, chunk...)
	}
	return rows, nil
}
