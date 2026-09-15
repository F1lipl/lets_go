// Package ecode defines stable application-level response codes for the
// content service. Codes are part of the public API contract and must not be
// reused with a different meaning.
package ecode

import (
	"errors"
)

// Code is independent of the HTTP status code.
type Code int

// Retired codes 800105, 800109, 800112, 800115, 800701-800706,
// 800802 and 800803 remain reserved.
const (
	Success Code = 0

	InvalidRequest         Code = 800101
	InvalidCursor          Code = 800102
	InvalidPageSize        Code = 800103
	InvalidPostID          Code = 800104
	InvalidUserID          Code = 800106
	InvalidRevisionID      Code = 800107
	InvalidMediaAssetID    Code = 800108
	InvalidVisibility      Code = 800110
	InvalidLifecycleStatus Code = 800111
	InvalidVersion         Code = 800113
	InvalidBatchPostIDs    Code = 800114

	PostNotFound            Code = 800201
	PostAlreadyDeleted      Code = 800202
	PostOperationNotAllowed Code = 800203
	PostNotPublished        Code = 800204
	PostNotVisible          Code = 800205

	DraftNotFound        Code = 800301
	DraftVersionConflict Code = 800302
	DraftContentInvalid  Code = 800303

	PublishNotAllowed    Code = 800401
	RevisionNotFound     Code = 800402
	RevisionCreateFailed Code = 800403

	MediaAssetNotFound    Code = 800501
	MediaAssetNotReady    Code = 800502
	MediaTypeUnsupported  Code = 800503
	MediaSizeExceeded     Code = 800504
	MediaAssetInUse       Code = 800505
	MediaUploadExpired    Code = 800506
	MediaUploadIncomplete Code = 800507
	MediaHashMismatch     Code = 800508
	MediaProcessingFailed Code = 800509

	TagNotFound    Code = 800601
	TagNameInvalid Code = 800602
	InvalidTagID   Code = 800603
	TooManyTags    Code = 800604
	TagUnavailable Code = 800605

	PostVersionConflict    Code = 800801
	RequestIdentityInvalid Code = 800901

	InternalError         Code = 900001
	DatabaseError         Code = 900002
	CacheError            Code = 900003
	DependencyUnavailable Code = 900004
)

var messages = map[Code]string{
	Success:                 "操作成功",
	InvalidRequest:          "请求参数不正确",
	InvalidCursor:           "分页游标不正确",
	InvalidPageSize:         "分页数量不正确",
	InvalidPostID:           "帖子标识不正确",
	InvalidUserID:           "用户标识不正确",
	InvalidRevisionID:       "发布版本标识不正确",
	InvalidMediaAssetID:     "图片资源标识不正确",
	InvalidVisibility:       "可见范围不正确",
	InvalidLifecycleStatus:  "帖子状态不正确",
	InvalidVersion:          "版本号不正确",
	InvalidBatchPostIDs:     "批量帖子标识不正确",
	PostNotFound:            "帖子不存在",
	PostAlreadyDeleted:      "帖子已删除",
	PostOperationNotAllowed: "当前状态不允许执行此操作",
	PostNotPublished:        "帖子尚未发布",
	PostNotVisible:          "帖子当前不可见",
	DraftNotFound:           "草稿不存在",
	DraftVersionConflict:    "草稿已被更新，请刷新后重试",
	DraftContentInvalid:     "草稿内容不符合发布要求",
	PublishNotAllowed:       "当前帖子不能发布",
	RevisionNotFound:        "发布版本不存在",
	RevisionCreateFailed:    "创建发布版本失败",
	MediaAssetNotFound:      "图片资源不存在",
	MediaAssetNotReady:      "图片资源尚未处理完成",
	MediaTypeUnsupported:    "图片格式不受支持",
	MediaSizeExceeded:       "图片大小超过限制",
	MediaAssetInUse:         "图片正在被内容使用",
	MediaUploadExpired:      "图片上传凭据已过期",
	MediaUploadIncomplete:   "图片尚未上传完成",
	MediaHashMismatch:       "图片内容校验不一致",
	MediaProcessingFailed:   "图片处理失败",
	TagNotFound:             "话题不存在",
	TagNameInvalid:          "话题名称不正确",
	InvalidTagID:            "话题标识不正确",
	TooManyTags:             "话题数量超过限制",
	TagUnavailable:          "话题当前不可用",
	PostVersionConflict:     "帖子已被更新，请刷新后重试",
	RequestIdentityInvalid:  "当前请求缺少有效的用户信息",
	InternalError:           "服务暂时不可用",
	DatabaseError:           "数据处理失败",
	CacheError:              "临时数据处理失败",
	DependencyUnavailable:   "依赖服务暂时不可用",
}

// Error carries a stable client-facing code while preserving the underlying
// cause for logging and errors.Is/errors.As checks.
type Error struct {
	code  Code
	cause error
}

func New(code Code) error {
	return &Error{code: normalize(code)}
}

func Wrap(code Code, cause error) error {
	return &Error{code: normalize(code), cause: cause}
}

func (e *Error) Error() string {
	return e.code.Message()
}

func (e *Error) Unwrap() error {
	return e.cause
}

func (e *Error) Code() Code {
	return e.code
}

func (e *Error) Cause() error {
	return e.cause
}

func (c Code) Int() int {
	return int(c)
}

func (c Code) Message() string {
	if message, ok := messages[c]; ok {
		return message
	}

	return messages[InternalError]
}

func (c Code) String() string {
	return c.Message()
}

// FromError extracts an application code. Unknown errors intentionally map to
// InternalError so implementation details are not returned to clients.
func FromError(err error) Code {
	var coded *Error
	if errors.As(err, &coded) {
		return normalize(coded.Code())
	}

	return InternalError
}

func normalize(code Code) Code {
	if _, ok := messages[code]; ok {
		return code
	}

	return InternalError
}
