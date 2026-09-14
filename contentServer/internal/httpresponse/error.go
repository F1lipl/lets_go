package httpresponse

import (
	"context"
	"errors"

	"contentserver/internal/ecode"

	"github.com/zeromicro/go-zero/core/logx"
)

// ErrorBody is the stable failure payload returned by every business API.
type ErrorBody struct {
	ErrorCode int    `json:"errorCode"`
	Message   string `json:"message"`
}

type resultSetter interface {
	SetResult(errorCode int, message string)
}

// SuccessHandler fills the common success fields without wrapping or changing
// the endpoint-specific response body.
func SuccessHandler(_ context.Context, body any) any {
	if response, ok := body.(resultSetter); ok {
		response.SetResult(ecode.Success.Int(), ecode.Success.Message())
	}

	return body
}

// ErrorHandler converts typed application errors into the common API payload.
// Unknown errors are deliberately reduced to InternalError.
func ErrorHandler(ctx context.Context, err error) (int, any) {
	code := ecode.FromError(err)
	if code >= ecode.InternalError {
		logx.WithContext(ctx).Errorw(
			"content api request failed",
			logx.Field("errorCode", code.Int()),
			logx.Field("detail", errorDetail(err)),
		)
	}

	return ecode.HTTPStatus(code), ErrorBody{
		ErrorCode: code.Int(),
		Message:   code.Message(),
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
