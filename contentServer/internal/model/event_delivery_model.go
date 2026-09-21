package model

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

type EventDelivery struct {
	EventId      string         `db:"event_id"`
	ConsumerName string         `db:"consumer_name"`
	Status       uint64         `db:"status"`
	AttemptCount uint64         `db:"attempt_count"`
	MaxAttempts  uint64         `db:"max_attempts"`
	NextAttempt  time.Time      `db:"next_attempt_at"`
	ClaimToken   sql.NullString `db:"claim_token"`
	LockedUntil  sql.NullTime   `db:"locked_until"`
	LastError    string         `db:"last_error"`
	CompletedAt  sql.NullTime   `db:"completed_at"`
	CreatedAt    time.Time      `db:"created_at"`
	UpdatedAt    time.Time      `db:"updated_at"`
}

type EventDeliveryKey struct {
	EventId      string `db:"event_id"`
	ConsumerName string `db:"consumer_name"`
}

type EventDeliveryClaimState struct {
	Status     uint64         `db:"status"`
	ClaimToken sql.NullString `db:"claim_token"`
	LeaseValid bool           `db:"lease_valid"`
}

type EventDeliveryRetryState struct {
	AttemptCount uint64 `db:"attempt_count"`
	MaxAttempts  uint64 `db:"max_attempts"`
}

type EventDeliveryRecoveryState struct {
	EventId      string `db:"event_id"`
	ConsumerName string `db:"consumer_name"`
	ClaimToken   string `db:"claim_token"`
	AttemptCount uint64 `db:"attempt_count"`
	MaxAttempts  uint64 `db:"max_attempts"`
}

type EventDeliveryModel interface {
	InsertPending(ctx context.Context, eventId, consumerName string) (sql.Result, error)
	FindReadyForUpdate(ctx context.Context, limit int) ([]EventDeliveryKey, error)
	MarkProcessing(ctx context.Context, eventId, consumerName, claimToken string, lease time.Duration) (time.Time, error)
	FindClaimStateForUpdate(ctx context.Context, eventId, consumerName string) (*EventDeliveryClaimState, error)
	FindRetryStateForUpdate(ctx context.Context, eventId, consumerName, claimToken string) (*EventDeliveryRetryState, error)
	FindExpiredForUpdate(ctx context.Context, limit int) ([]EventDeliveryRecoveryState, error)
	MarkTerminal(ctx context.Context, eventId, consumerName, claimToken string, status uint64) (sql.Result, error)
	MarkFailed(ctx context.Context, eventId, consumerName, claimToken, lastError string) (sql.Result, error)
	MarkPending(ctx context.Context, eventId, consumerName, claimToken, lastError string, delay time.Duration) (sql.Result, error)
	MarkExpiredFailed(ctx context.Context, eventId, consumerName, claimToken, lastError string) (sql.Result, error)
	MarkExpiredPending(ctx context.Context, eventId, consumerName, claimToken, lastError string) (sql.Result, error)
	withSession(session sqlx.Session) EventDeliveryModel
}

func (m *defaultEventDeliveryModel) InsertPending(ctx context.Context, eventId, consumerName string) (sql.Result, error) {
	query := fmt.Sprintf("INSERT INTO %s (event_id,consumer_name) VALUES (?,?)", m.table)
	return m.conn.ExecCtx(ctx, query, eventId, consumerName)
}

type defaultEventDeliveryModel struct {
	conn  sqlx.SqlConn
	table string
}

func NewEventDeliveryModel(conn sqlx.SqlConn) EventDeliveryModel {
	return &defaultEventDeliveryModel{
		conn:  conn,
		table: "`event_delivery`",
	}
}

func (m *defaultEventDeliveryModel) FindReadyForUpdate(ctx context.Context, limit int) ([]EventDeliveryKey, error) {
	query := fmt.Sprintf(`
SELECT event_id, consumer_name
FROM %s
WHERE status = 1
  AND next_attempt_at <= NOW(3)
  AND attempt_count < max_attempts
ORDER BY next_attempt_at, event_id, consumer_name
LIMIT ?
FOR UPDATE SKIP LOCKED`, m.table)
	var rows []EventDeliveryKey
	if err := m.conn.QueryRowsCtx(ctx, &rows, query, limit); err != nil {
		return nil, err
	}
	return rows, nil
}

func (m *defaultEventDeliveryModel) MarkProcessing(
	ctx context.Context,
	eventId string,
	consumerName string,
	claimToken string,
	lease time.Duration,
) (time.Time, error) {
	query := fmt.Sprintf(`
UPDATE %s
SET status = 2,
    attempt_count = attempt_count + 1,
    claim_token = ?,
    locked_until = TIMESTAMPADD(MICROSECOND, ?, NOW(3)),
    completed_at = NULL
WHERE event_id = ?
  AND consumer_name = ?
  AND status = 1
  AND next_attempt_at <= NOW(3)
  AND attempt_count < max_attempts`, m.table)
	result, err := m.conn.ExecCtx(ctx, query, claimToken, lease.Microseconds(), eventId, consumerName)
	if err != nil {
		return time.Time{}, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return time.Time{}, err
	}
	if affected != 1 {
		return time.Time{}, ErrNotFound
	}

	var row struct {
		LockedUntil time.Time `db:"locked_until"`
	}
	query = fmt.Sprintf("SELECT locked_until FROM %s WHERE event_id = ? AND consumer_name = ?", m.table)
	if err := m.conn.QueryRowCtx(ctx, &row, query, eventId, consumerName); err != nil {
		return time.Time{}, err
	}
	return row.LockedUntil, nil
}

func (m *defaultEventDeliveryModel) FindClaimStateForUpdate(ctx context.Context, eventId, consumerName string) (*EventDeliveryClaimState, error) {
	query := fmt.Sprintf("SELECT status,claim_token,COALESCE(locked_until>NOW(3),0) AS lease_valid FROM %s WHERE event_id=? AND consumer_name=? FOR UPDATE", m.table)
	var row EventDeliveryClaimState
	if err := m.conn.QueryRowCtx(ctx, &row, query, eventId, consumerName); err != nil {
		if err == sqlx.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &row, nil
}

func (m *defaultEventDeliveryModel) FindRetryStateForUpdate(ctx context.Context, eventId, consumerName, claimToken string) (*EventDeliveryRetryState, error) {
	query := fmt.Sprintf("SELECT attempt_count,max_attempts FROM %s WHERE event_id=? AND consumer_name=? AND status=2 AND claim_token=? FOR UPDATE", m.table)
	var row EventDeliveryRetryState
	if err := m.conn.QueryRowCtx(ctx, &row, query, eventId, consumerName, claimToken); err != nil {
		if err == sqlx.ErrNotFound {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &row, nil
}

func (m *defaultEventDeliveryModel) FindExpiredForUpdate(ctx context.Context, limit int) ([]EventDeliveryRecoveryState, error) {
	query := fmt.Sprintf(`
SELECT event_id,consumer_name,claim_token,attempt_count,max_attempts
FROM %s
WHERE status=2
  AND locked_until<=NOW(3)
ORDER BY locked_until,event_id,consumer_name
LIMIT ?
FOR UPDATE SKIP LOCKED`, m.table)
	var rows []EventDeliveryRecoveryState
	if err := m.conn.QueryRowsCtx(ctx, &rows, query, limit); err != nil {
		return nil, err
	}
	return rows, nil
}

func (m *defaultEventDeliveryModel) MarkTerminal(ctx context.Context, eventId, consumerName, claimToken string, status uint64) (sql.Result, error) {
	query := fmt.Sprintf("UPDATE %s SET status=?,claim_token=NULL,locked_until=NULL,completed_at=NOW(3),last_error='' WHERE event_id=? AND consumer_name=? AND status=2 AND claim_token=?", m.table)
	return m.conn.ExecCtx(ctx, query, status, eventId, consumerName, claimToken)
}

func (m *defaultEventDeliveryModel) MarkFailed(ctx context.Context, eventId, consumerName, claimToken, lastError string) (sql.Result, error) {
	query := fmt.Sprintf("UPDATE %s SET status=5,claim_token=NULL,locked_until=NULL,completed_at=NOW(3),last_error=? WHERE event_id=? AND consumer_name=? AND status=2 AND claim_token=?", m.table)
	return m.conn.ExecCtx(ctx, query, lastError, eventId, consumerName, claimToken)
}

func (m *defaultEventDeliveryModel) MarkPending(ctx context.Context, eventId, consumerName, claimToken, lastError string, delay time.Duration) (sql.Result, error) {
	query := fmt.Sprintf("UPDATE %s SET status=1,claim_token=NULL,locked_until=NULL,completed_at=NULL,next_attempt_at=TIMESTAMPADD(MICROSECOND,?,NOW(3)),last_error=? WHERE event_id=? AND consumer_name=? AND status=2 AND claim_token=?", m.table)
	return m.conn.ExecCtx(ctx, query, delay.Microseconds(), lastError, eventId, consumerName, claimToken)
}

func (m *defaultEventDeliveryModel) MarkExpiredFailed(ctx context.Context, eventId, consumerName, claimToken, lastError string) (sql.Result, error) {
	query := fmt.Sprintf("UPDATE %s SET status=5,claim_token=NULL,locked_until=NULL,completed_at=NOW(3),last_error=? WHERE event_id=? AND consumer_name=? AND status=2 AND claim_token=? AND locked_until<=NOW(3)", m.table)
	return m.conn.ExecCtx(ctx, query, lastError, eventId, consumerName, claimToken)
}

func (m *defaultEventDeliveryModel) MarkExpiredPending(ctx context.Context, eventId, consumerName, claimToken, lastError string) (sql.Result, error) {
	query := fmt.Sprintf("UPDATE %s SET status=1,claim_token=NULL,locked_until=NULL,completed_at=NULL,next_attempt_at=NOW(3),last_error=? WHERE event_id=? AND consumer_name=? AND status=2 AND claim_token=? AND locked_until<=NOW(3)", m.table)
	return m.conn.ExecCtx(ctx, query, lastError, eventId, consumerName, claimToken)
}

func (m *defaultEventDeliveryModel) withSession(session sqlx.Session) EventDeliveryModel {
	return NewEventDeliveryModel(sqlx.NewSqlConnFromSession(session))
}
