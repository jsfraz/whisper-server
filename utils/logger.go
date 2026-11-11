package utils

import (
	"log/slog"
	"os"
)

// InitLogger configures the default structured logger.
// Release logs are JSON for aggregation; debug logs stay human readable.
func InitLogger(ginMode string) {
	var handler slog.Handler = slog.NewTextHandler(os.Stdout, nil)
	if ginMode == "release" {
		handler = slog.NewJSONHandler(os.Stdout, nil)
	}
	slog.SetDefault(slog.New(handler))
}
