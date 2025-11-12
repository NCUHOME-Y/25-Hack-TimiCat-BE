package httpx

import (
	"encoding/json"
	"net/http"
)

type ErrorResp struct {
	Code      int    `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func WriteOK(w http.ResponseWriter, v any) {
	WriteJSON(w, http.StatusOK, v)
}

func WriteErr(w http.ResponseWriter, status int, msg, reqID string) {
	WriteJSON(w, status, ErrorResp{Code: status, Message: msg, RequestID: reqID})
}
