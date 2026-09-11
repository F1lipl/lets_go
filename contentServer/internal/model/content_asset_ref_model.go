package model

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type ContentAssetRef struct {
	OwnerType     uint64    `db:"owner_type"`
	OwnerId       string    `db:"owner_id"`
	UsageType     uint64    `db:"usage_type"`
	BlockId       string    `db:"block_id"`
	AssetId       string    `db:"asset_id"`
	SortOrder     uint64    `db:"sort_order"`
	SourceVersion uint64    `db:"source_version"`
	UpdatedAt     time.Time `db:"updated_at"`
}

type ContentAssetRefModel interface {
	Insert(ctx context.Context, data *ContentAssetRef) (sql.Result, error)
	FindByOwner(ctx context.Context, ownerType uint64, ownerId string) ([]ContentAssetRef, error)
	FindByAssetId(ctx context.Context, assetId string) ([]ContentAssetRef, error)
	DeleteByOwner(ctx context.Context, ownerType uint64, ownerId string) error
	withSession(session sqlx.Session) ContentAssetRefModel
}

type defaultContentAssetRefModel struct {
	conn  sqlx.SqlConn
	table string
}

func NewContentAssetRefModel(conn sqlx.SqlConn) ContentAssetRefModel {
	return &defaultContentAssetRefModel{
		conn:  conn,
		table: "`content_asset_ref`",
	}
}

func (m *defaultContentAssetRefModel) Insert(ctx context.Context, data *ContentAssetRef) (sql.Result, error) {
	query := fmt.Sprintf("insert into %s (`owner_type`,`owner_id`,`usage_type`,`block_id`,`asset_id`,`sort_order`,`source_version`) values (?,?,?,?,?,?,?)", m.table)
	return m.conn.ExecCtx(ctx, query, data.OwnerType, data.OwnerId, data.UsageType, data.BlockId, data.AssetId, data.SortOrder, data.SourceVersion)
}

func (m *defaultContentAssetRefModel) FindByOwner(ctx context.Context, ownerType uint64, ownerId string) ([]ContentAssetRef, error) {
	query := fmt.Sprintf("select `owner_type`,`owner_id`,`usage_type`,`block_id`,`asset_id`,`sort_order`,`source_version`,`updated_at` from %s where `owner_type` = ? and `owner_id` = ? order by `usage_type`,`block_id`,`sort_order`", m.table)
	var rows []ContentAssetRef
	if err := m.conn.QueryRowsCtx(ctx, &rows, query, ownerType, ownerId); err != nil {
		return nil, err
	}
	return rows, nil
}

func (m *defaultContentAssetRefModel) FindByAssetId(ctx context.Context, assetId string) ([]ContentAssetRef, error) {
	query := fmt.Sprintf("select `owner_type`,`owner_id`,`usage_type`,`block_id`,`asset_id`,`sort_order`,`source_version`,`updated_at` from %s where `asset_id` = ?", m.table)
	var rows []ContentAssetRef
	if err := m.conn.QueryRowsCtx(ctx, &rows, query, assetId); err != nil {
		return nil, err
	}
	return rows, nil
}

func (m *defaultContentAssetRefModel) DeleteByOwner(ctx context.Context, ownerType uint64, ownerId string) error {
	query := fmt.Sprintf("delete from %s where `owner_type` = ? and `owner_id` = ?", m.table)
	_, err := m.conn.ExecCtx(ctx, query, ownerType, ownerId)
	return err
}

func (m *defaultContentAssetRefModel) withSession(session sqlx.Session) ContentAssetRefModel {
	return NewContentAssetRefModel(sqlx.NewSqlConnFromSession(session))
}
