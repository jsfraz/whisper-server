package utils

import (
	"log/slog"
	"os"
	"strings"
)

// SanitizeLogValue removes line breaks so user-controlled values cannot forge
// extra log lines. Structured handlers already escape fields; this keeps the
// data flow clean for CodeQL and plain text handlers.
func SanitizeLogValue(value string) string {
	return strings.NewReplacer("\n", "", "\r", "").Replace(value)
}

// InitLogger configures the default structured logger.
// Release logs are JSON for aggregation; debug logs stay human readable.
func InitLogger(ginMode string) {
	var handler slog.Handler = slog.NewTextHandler(os.Stdout, nil)
	if ginMode == "release" {
		handler = slog.NewJSONHandler(os.Stdout, nil)
	}
	slog.SetDefault(slog.New(handler))
}
