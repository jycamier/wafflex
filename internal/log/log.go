package log

import (
	"log/slog"
	"os"
	"time"

	"github.com/lmittmann/tint"
)

func Init(level slog.Level) {
	handler := tint.NewHandler(os.Stderr, &tint.Options{
		Level:      level,
		TimeFormat: time.Kitchen,
	})
	slog.SetDefault(slog.New(handler))
}
