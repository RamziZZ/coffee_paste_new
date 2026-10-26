package routes

import (
	"encoding/json"
	"os"
	"sort"
	"strings"
	"testing"

	"coffee_paste_backend/config"

	"github.com/gin-gonic/gin"
)

// TestRoutesMatchAPIDocs memastikan setiap route yang terdaftar di
// routes.go punya pasangan method+path di api.json, dan sebaliknya,
// sehingga spesifikasi OpenAPI tidak menyimpang dari implementasi.
func TestRoutesMatchAPIDocs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	cfg := &config.Config{JWTSecret: "test-secret"}
	SetupRoutes(r, nil, cfg)

	// Kumpulkan route aktual, hilangkan prefix /api/v1 (di api.json
	// didefinisikan lewat servers), ubah :param menjadi {param}.
	actual := map[string]bool{}
	for _, ri := range r.Routes() {
		p := ri.Path
		if !strings.HasPrefix(p, "/api/v1") {
			continue
		}
		p = strings.TrimPrefix(p, "/api/v1")
		if p == "" {
			p = "/"
		}
		for i := 0; i < len(p); i++ {
			if p[i] == ':' {
				end := i + 1
				for end < len(p) && p[end] != '/' {
					end++
				}
				p = p[:i] + "{" + p[i+1:end] + "}" + p[end:]
				i = end + 1
			}
		}
		actual[ri.Method+" "+p] = true
	}

	raw, err := os.ReadFile("../api.json")
	if err != nil {
		t.Fatalf("gagal membaca api.json: %v", err)
	}

	var spec struct {
		Paths map[string]map[string]json.RawMessage `json:"paths"`
	}
	if err := json.Unmarshal(raw, &spec); err != nil {
		t.Fatalf("api.json tidak valid: %v", err)
	}

	documented := map[string]bool{}
	for p, methods := range spec.Paths {
		for m := range methods {
			documented[strings.ToUpper(m)+" "+p] = true
		}
	}

	var missing, extra []string
	for k := range documented {
		if !actual[k] {
			missing = append(missing, "api.json: "+k)
		}
	}
	for k := range actual {
		if !documented[k] {
			extra = append(extra, "routes.go: "+k)
		}
	}
	sort.Strings(missing)
	sort.Strings(extra)

	if len(missing) > 0 || len(extra) > 0 {
		t.Errorf("route dan api.json tidak sinkron:\n  tidak ada di routes.go:\n    %s\n  tidak ada di api.json:\n    %s",
			strings.Join(missing, "\n    "), strings.Join(extra, "\n    "))
	}
}
