package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"jsfraz/whisper-server/routes"
	"jsfraz/whisper-server/utils"
	"net/http/httptest"
	"os"
)

// Writes the OpenAPI document generated from the registered routes to openapi.json.
// Run from the repository root: go run ./cmd/spec
func main() {
	// The spec route is only registered outside release mode.
	utils.GetSingleton().Config.GinMode = "debug"

	router, err := routes.NewRouter()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, httptest.NewRequest("GET", "/api/openapi.json", nil))
	if rec.Code != 200 {
		fmt.Fprintf(os.Stderr, "unexpected status %d: %s\n", rec.Code, rec.Body.String())
		os.Exit(1)
	}

	body := rec.Body.Bytes()
	var out bytes.Buffer
	if err := json.Indent(&out, body, "", "  "); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	out.WriteByte('\n')
	if err := os.WriteFile("openapi.json", out.Bytes(), 0644); err != nil { //nolint:gosec // repository artifact
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
