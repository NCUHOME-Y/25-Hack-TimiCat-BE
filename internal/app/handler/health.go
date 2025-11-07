package handler

import (
	"net/http"

	"github.com/NCUHOME-Y/25-Hack-TimiCat-BE/internal/app/service"
	pkgerr "github.com/NCUHOME-Y/25-Hack-TimiCat-BE/internal/pkg/err"
)

type HealthHandler struct {
	svc *service.HealthService
}

func NewHealthHandler(svc *service.HealthService) http.Handler {
	return &HealthHandler{svc: svc}
}

func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	data := map[string]string{"status": "ok"}
	pkgerr.JSON(w, r, pkgerr.CodeOK, data)
}
