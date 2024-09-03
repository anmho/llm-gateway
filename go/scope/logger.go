package scope

import "log/slog"

var logger = slog.Default()

func GetLogger() *slog.Logger {
	return logger
}
