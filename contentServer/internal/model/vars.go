package model

import (
	"errors"

	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

var (
	ErrNotFound        = sqlx.ErrNotFound
	ErrVersionConflict = errors.New("version conflict")
)
