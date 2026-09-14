package handler

import (
	"context"
	"net/http"

	"contentserver/internal/ecode"

	"github.com/zeromicro/go-zero/rest/httpx"
)

func writeInvalidRequest(ctx context.Context, w http.ResponseWriter, err error) {
	httpx.ErrorCtx(ctx, w, ecode.Wrap(ecode.InvalidRequest, err))
}
