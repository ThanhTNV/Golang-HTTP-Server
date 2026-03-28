package logs

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/logger"
	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
)

const (
	// BaseDir is the root for daily log files. Month subfolders are created under it.
	BaseDir = "logs/log-files"
)

// dailyWriter appends JSON log lines to a file named by calendar day, under a folder per month.
type dailyWriter struct {
	mu   sync.Mutex
	file *os.File
	day  string // e.g. 27-Mar-2026 — used to detect midnight rotation
}

func (w *dailyWriter) currentPaths(now time.Time) (dir, filePath string) {
	monthDir := now.Format("Jan-2006")   // e.g. Mar-2026
	dayName := now.Format("02-Jan-2006")   // e.g. 27-Mar-2026
	dir = filepath.Join(BaseDir, monthDir)
	filePath = filepath.Join(dir, dayName+".txt")
	return dir, filePath
}

func (w *dailyWriter) ensureOpen(now time.Time) error {
	_, path := w.currentPaths(now)
	dayKey := now.Format("02-Jan-2006")

	if w.file != nil && w.day == dayKey {
		return nil
	}

	if w.file != nil {
		_ = w.file.Close()
		w.file = nil
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	w.file = f
	w.day = dayKey
	return nil
}

func (w *dailyWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if err := w.ensureOpen(time.Now()); err != nil {
		return 0, err
	}
	return w.file.Write(p)
}

func (w *dailyWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.file == nil {
		return nil
	}
	err := w.file.Close()
	w.file = nil
	w.day = ""
	return err
}

var defaultWriter *dailyWriter

// Init configures the global zerolog logger to write JSON lines to disk (one .txt per day, folder per month).
// Time values use RFC3339, e.g. "2023-07-12T11:57:28+02:00".
func Init() error {
	zerolog.TimeFieldFormat = time.RFC3339

	defaultWriter = &dailyWriter{}
	zlog.Logger = zerolog.New(defaultWriter).
		With().
		Timestamp().
		Logger()
	return nil
}

// Close releases the current log file. Call from main via defer after Init.
func Close() error {
	if defaultWriter == nil {
		return nil
	}
	return defaultWriter.Close()
}

// FiberMiddleware returns a Fiber logger middleware config that writes
// request logs through zerolog in JSON format to the daily log files.
func FiberMiddleware() logger.Config {
	return logger.Config{
		DisableColors: true,
		LoggerFunc: func(c fiber.Ctx, data *logger.Data, _ *logger.Config) error {
			latency := data.Stop.Sub(data.Start)
			status := c.Response().StatusCode()

			evt := zlog.Info()
			if status >= fiber.StatusInternalServerError {
				evt = zlog.Error()
			} else if status >= fiber.StatusBadRequest {
				evt = zlog.Warn()
			}

			evt.
				Str("ip", c.IP()).
				Str("method", c.Method()).
				Str("path", c.Path()).
				Int("status", status).
				Dur("latency", latency)

			if data.ChainErr != nil {
				evt.Err(data.ChainErr)
			}

			evt.Msg("request")
			return nil
		},
	}
}
