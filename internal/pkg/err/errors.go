package err

import (
	"encoding/json"
	"net/http"

	"context"
)

type Response struct {
	Code      int         `json:"code"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data,omitempty"`
	RequestID string      `json:"request_id,omitempty"`
}

const (
	CodeOK       = 0
	CodeInternal = 1000
	CodeNotFound = 1001
	CodeBadParam = 1002
)

var codeMessage = map[int]string{
	CodeOK:       "ok",
	CodeInternal: "internal_error",
	CodeNotFound: "not_found",
	CodeBadParam: "bad_parameter",
}

// JSON 写入统一的响应格式
func JSON(w http.ResponseWriter, r *http.Request, code int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatusFromCode(code))
	reqID := requestIDFromContext(r.Context())
	resp := Response{
		Code:      code,
		Message:   codeMessage[code],
		Data:      data,
		RequestID: reqID,
	}
	_ = json.NewEncoder(w).Encode(resp)
}

func httpStatusFromCode(code int) int {
	switch code {
	case CodeOK:
		return http.StatusOK
	case CodeBadParam:
		return http.StatusBadRequest
	case CodeNotFound:
		return http.StatusNotFound
	default:
		return http.StatusInternalServerError
	}
}

// 用于请求 ID
type ctxKey string

const requestIDKey ctxKey = "request_id"

func requestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v := ctx.Value(requestIDKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}
