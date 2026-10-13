package server

import (
	"errors"
	"log/slog"

	"github.com/labstack/echo/v5"

	"github.com/takuma0403/osagari_web/server/internal/api"
	"github.com/takuma0403/osagari_web/server/internal/apperr"
)

// errorHandler はすべてのエラーを OpenAPI の Error スキーマ（code・message）で返す（API-1）。
// 5xx の原因はログにだけ出し、応答には既定のメッセージだけを入れる。
func errorHandler(logger *slog.Logger) echo.HTTPErrorHandler {
	return func(c *echo.Context, err error) {
		if r, _ := echo.UnwrapResponse(c.Response()); r != nil && r.Committed {
			return
		}
		ae := toAppError(err)
		if ae.Status >= 500 {
			logger.ErrorContext(c.Request().Context(), "サーバーエラー",
				"request_id", requestID(c), "route", routeOf(c), "err", err)
		}
		if werr := c.JSON(ae.Status, api.Error{Code: string(ae.Code), Message: ae.Message}); werr != nil {
			logger.ErrorContext(c.Request().Context(), "エラー応答を書けません", "request_id", requestID(c), "err", werr)
		}
	}
}

// toAppError は err を *apperr.Error に変換する。Echo のルーティング・バインドのエラーはステータスから共通コードにし、
// それ以外（panic を含む）は internal_error にする。
func toAppError(err error) *apperr.Error {
	var ae *apperr.Error
	if errors.As(err, &ae) {
		return ae
	}
	if status := echo.StatusCode(err); status != 0 {
		return apperr.FromStatus(status)
	}
	return apperr.Internal(err)
}
