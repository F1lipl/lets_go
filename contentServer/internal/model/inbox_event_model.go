package model

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type InboxEvent struct {
	ConsumerName string       `db:"consumer_name"`
	EventId      string       `db:"event_id"`
	EventType    string       `db:"event_type"`
	Status       uint64       `db:"status"`
	AttemptCount uint64       `db:"attempt_count"`
	PayloadJson  string       `db:"payload_json"`
	ReceivedAt   time.Time    `db:"received_at"`
	ProcessedAt  sql.NullTime `db:"processed_at"`
	LastError    string       `db:"last_error"`
}

type InboxEventModel interface {
	Insert(ctx context.Context, data *InboxEvent) (sql.Result, error)
	FindOne(ctx context.Context, consumerName, eventId string) (*InboxEvent, error)
	MarkProcessed(ctx context.Context, consumerName, eventId string) (sql.Result, error)
	MarkFailed(ctx context.Context, consumerName, eventId, lastError string) (sql.Result, error)
	withSession(session sqlx.Session) InboxEventModel
}

type defaultInboxEventModel struct {
	conn  sqlx.SqlConn
	table string
}

func NewInboxEventModel(conn sqlx.SqlConn) InboxEventModel {
	return &defaultInboxEventModel{
		conn:  conn,
		table: "`inbox_event`",
	}
}

func (m *defaultInboxEventModel) Insert(ctx context.Context, data *InboxEvent) (sql.Result, error) {
	query := fmt.Sprintf("insert into %s (`consumer_name`,`event_id`,`event_type`,`status`,`attempt_count`,`payload_json`,`processed_at`,`last_error`) values (?,?,?,?,?,?,?,?)", m.table)
	return m.conn.ExecCtx(ctx, query, data.ConsumerName, data.EventId, data.EventType, data.Status, data.AttemptCount, data.PayloadJson, data.ProcessedAt, data.LastError)
}

func (m *defaultInboxEventModel) FindOne(ctx context.Context, consumerName, eventId string) (*InboxEvent, error) {
	query := fmt.Sprintf("select `consumer_name`,`event_id`,`event_type`,`status`,`attempt_count`,`payload_json`,`received_at`,`processed_at`,`last_error` from %s where `consumer_name` = ? and `event_id` = ? limit 1", m.table)
	var row InboxEvent
	if err := m.conn.QueryRowCtx(ctx, &row, query, consumerName, eventId); err != nil {
		if err == sqlx.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &row, nil
}

func (m *defaultInboxEventModel) MarkProcessed(ctx context.Context, consumerName, eventId string) (sql.Result, error) {
	query := fmt.Sprintf("update %s set `status` = 2, `processed_at` = current_timestamp(3), `last_error` = '' where `consumer_name` = ? and `event_id` = ? and `status` = 1", m.table)
	return m.conn.ExecCtx(ctx, query, consumerName, eventId)
}

func (m *defaultInboxEventModel) MarkFailed(ctx context.Context, consumerName, eventId, lastError string) (sql.Result, error) {
	query := fmt.Sprintf("update %s set `status` = 3, `attempt_count` = `attempt_count` + 1, `last_error` = ? where `consumer_name` = ? and `event_id` = ? and `status` = 1", m.table)
	return m.conn.ExecCtx(ctx, query, lastError, consumerName, eventId)
}

func (m *defaultInboxEventModel) withSession(session sqlx.Session) InboxEventModel {
	return NewInboxEventModel(sqlx.NewSqlConnFromSession(session))
}
