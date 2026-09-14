package httpresponse

import (
	"context"
	"errors"
	"net/http"

	"contentserver/internal/ecode"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/trace"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// Envelope is the single external response shape used by business APIs.
type Envelope struct {
	ErrorCode int    `json:"errorCode"`
	Message   string `json:"message"`
	Data      any    `json:"data,omitempty"`
	RequestID string `json:"requestId,omitempty"`
}

// WriteSuccess renders business data at the HTTP boundary.
func WriteSuccess(ctx context.Context, w http.ResponseWriter, data any) {
	httpx.WriteJsonCtx(ctx, w, http.StatusOK, Envelope{
		ErrorCode: ecode.Success.Int(),
		Message:   ecode.Success.Message(),
		Data:      data,
		RequestID: trace.TraceIDFromContext(ctx),
	})
}

// WriteError maps an application error to a stable external response. The
// underlying cause is logged for diagnosis but is never returned to clients.
func WriteError(ctx context.Context, w http.ResponseWriter, err error) {
	code := ecode.FromError(err)
	if code >= ecode.InternalError {
		logx.WithContext(ctx).Errorw(
			"content api request failed",
			logx.Field("errorCode", code.Int()),
			logx.Field("detail", errorDetail(err)),
		)
	}

	httpx.WriteJsonCtx(ctx, w, statusFor(code), Envelope{
		ErrorCode: code.Int(),
		Message:   code.Message(),
		RequestID: trace.TraceIDFromContext(ctx),
	})
}

func statusFor(code ecode.Code) int {
	switch code {
	case ecode.Success:
		return http.StatusOK
	case ecode.InvalidRequest, ecode.InvalidCursor, ecode.InvalidPageSize,
		ecode.InvalidPostID, ecode.InvalidRequestID, ecode.DraftContentInvalid,
		ecode.TagNameInvalid, ecode.MediaTypeUnsupported, ecode.MediaSizeExceeded:
		return http.StatusBadRequest
	case ecode.RequestIdentityInvalid:
		return http.StatusUnauthorized
	case ecode.PostNotVisible:
		return http.StatusForbidden
	case ecode.PostNotFound, ecode.DraftNotFound, ecode.RevisionNotFound,
		ecode.MediaAssetNotFound, ecode.TagNotFound, ecode.RouteDraftNotFound:
		return http.StatusNotFound
	case ecode.PostAlreadyDeleted, ecode.PostOperationNotAllowed,
		ecode.PostNotPublished, ecode.DraftVersionConflict,
		ecode.PublishNotAllowed, ecode.MediaAssetNotReady, ecode.MediaAssetInUse,
		ecode.RouteVersionConflict, ecode.PostVersionConflict:
		return http.StatusConflict
	case ecode.DependencyUnavailable:
		return http.StatusServiceUnavailable
	default:
		return http.StatusInternalServerError
	}
}

func errorDetail(err error) string {
	var coded *ecode.Error
	if errors.As(err, &coded) && coded.Cause() != nil {
		return coded.Cause().Error()
	}
	if err != nil {
		return err.Error()
	}

	return "unknown error"
}
