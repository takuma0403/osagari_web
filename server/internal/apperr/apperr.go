// Package apperr はアプリ共通のエラーを定義する（API-1）。
// service は *Error を返し、handler はそのまま return する。HTTP 応答（OpenAPI の Error スキーマ）への変換は
// internal/server のエラーハンドラーがまとめて行う。
package apperr

import (
	"errors"
	"fmt"
	"net/http"
)

// Code は機械で判定するためのエラーコード（snake_case）。応答の Error.code に入る。
type Code string

// 共通のエラーコード。機能ごとのコードは各 issue で足す（docs/tech/api.md の API-1 に一覧を載せる）。
const (
	CodeBadRequest              Code = "bad_request"
	CodeUnauthorized            Code = "unauthorized"
	CodeForbidden               Code = "forbidden"
	CodeNotFound                Code = "not_found"
	CodeMethodNotAllowed        Code = "method_not_allowed"
	CodeInvalidStatusTransition Code = "invalid_status_transition" // DB: 条件付きUPDATE の更新件数が0
	CodeTooManyRequests         Code = "too_many_requests"
	CodeInternal                Code = "internal_error"
)

type spec struct {
	status  int
	message string
}

// specs は共通コードの HTTP ステータスと既定のメッセージ（利用者に表示できる日本語）。
var specs = map[Code]spec{
	CodeBadRequest:              {http.StatusBadRequest, "入力内容に誤りがあります"},
	CodeUnauthorized:            {http.StatusUnauthorized, "ログインが必要です"},
	CodeForbidden:               {http.StatusForbidden, "この操作は許可されていません"},
	CodeNotFound:                {http.StatusNotFound, "見つかりません"},
	CodeMethodNotAllowed:        {http.StatusMethodNotAllowed, "この操作には対応していません"},
	CodeInvalidStatusTransition: {http.StatusConflict, "状態が変わったため操作できませんでした。画面を更新してください"},
	CodeTooManyRequests:         {http.StatusTooManyRequests, "しばらく時間をおいてからもう一度お試しください"},
	CodeInternal:                {http.StatusInternalServerError, "サーバーでエラーが発生しました。時間をおいてもう一度お試しください"},
}

// Error はアプリ共通のエラー。Message は利用者に表示してよい内容だけを入れ、内部の原因は Err に入れる（応答には出ない）。
type Error struct {
	Status  int
	Code    Code
	Message string
	Err     error
}

func (e *Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

func (e *Error) Unwrap() error { return e.Err }

// StatusCode は HTTP ステータスを返す（echo.HTTPStatusCoder の実装）。
func (e *Error) StatusCode() int { return e.Status }

// Wrap は内部の原因 err を付けた複製を返す。
func (e *Error) Wrap(err error) *Error {
	c := *e
	c.Err = err
	return &c
}

// New は任意のステータス・コード・メッセージでエラーを作る。機能固有のコードに使う。
func New(status int, code Code, message string) *Error {
	return &Error{Status: status, Code: code, Message: message}
}

// newCommon は共通コードのエラーを作る。message が空なら既定のメッセージを使う。
func newCommon(code Code, message string) *Error {
	s := specs[code]
	if message == "" {
		message = s.message
	}
	return &Error{Status: s.status, Code: code, Message: message}
}

// 共通コードのコンストラクタ。message が空なら既定のメッセージになる。
func BadRequest(message string) *Error   { return newCommon(CodeBadRequest, message) }
func Unauthorized(message string) *Error { return newCommon(CodeUnauthorized, message) }
func Forbidden(message string) *Error    { return newCommon(CodeForbidden, message) }
func NotFound(message string) *Error     { return newCommon(CodeNotFound, message) }
func InvalidStatusTransition(message string) *Error {
	return newCommon(CodeInvalidStatusTransition, message)
}
func TooManyRequests(message string) *Error { return newCommon(CodeTooManyRequests, message) }

// Internal は想定外のエラー。err は原因としてログにだけ残り、応答には既定のメッセージだけが出る。
func Internal(err error) *Error { return newCommon(CodeInternal, "").Wrap(err) }

// FromStatus は HTTP ステータスに対応する共通エラーを返す。Echo のルーティングやバインドのエラーの変換に使う。
// 表にない 4xx は status をそのまま使い bad_request、5xx はすべて internal_error（500）にする。
func FromStatus(status int) *Error {
	for code, s := range specs {
		if s.status == status {
			return newCommon(code, "")
		}
	}
	if status >= 400 && status < 500 {
		e := newCommon(CodeBadRequest, "")
		e.Status = status
		return e
	}
	return newCommon(CodeInternal, "")
}

// CodeOf は err に含まれる *Error のコードを返す。含まれなければ空文字。
func CodeOf(err error) Code {
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	return ""
}
