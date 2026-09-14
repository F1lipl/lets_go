package handler

import (
	"context"
	"errors"
	"net/http"
	"reflect"

	"contentserver/internal/ecode"
	"contentserver/internal/httpresponse"
)

var errNilBusinessResult = errors.New("logic returned nil data without an error")

func writeInvalidRequest(ctx context.Context, w http.ResponseWriter, err error) {
	httpresponse.WriteError(ctx, w, ecode.Wrap(ecode.InvalidRequest, err))
}

func writeBusinessResponse(ctx context.Context, w http.ResponseWriter, data any, err error) {
	if err != nil {
		httpresponse.WriteError(ctx, w, err)
		return
	}
	if isNil(data) {
		httpresponse.WriteError(ctx, w, errNilBusinessResult)
		return
	}

	httpresponse.WriteSuccess(ctx, w, data)
}

func isNil(value any) bool {
	if value == nil {
		return true
	}

	reflected := reflect.ValueOf(value)
	switch reflected.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map,
		reflect.Pointer, reflect.Slice:
		return reflected.IsNil()
	default:
		return false
	}
}
