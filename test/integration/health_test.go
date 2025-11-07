package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	handlerpkg "github.com/NCUHOME-Y/25-Hack-TimiCat-BE/internal/app/handler"
	servicepkg "github.com/NCUHOME-Y/25-Hack-TimiCat-BE/internal/app/service"
)

func TestHealthz(t *testing.T) {
	svc := servicepkg.NewHealthService()
	h := handlerpkg.NewHealthHandler(svc)
	server := httptest.NewServer(h)
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("get failed: %v", err)
	}
	defer resp.Body.Close()

	var body map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode failed: %v", err)
	}
	if code, ok := body["code"].(float64); !ok || int(code) != 0 {
		t.Fatalf("expected code 0, got: %v", body["code"])
	}
}
