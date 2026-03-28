package logs

import (
	"bufio"
	"encoding/json"
	"os"
	"time"

	"github.com/gofiber/fiber/v3"
)

const cronjobLogFile = "/data/cronjob.log"

type logEntry struct {
	Time    string          `json:"time"`
	RawJSON json.RawMessage `json:"-"`
}

func RegisterRoutes(app *fiber.App) {
	app.Get("/logs/date/:date", GetLogsByDate)
	app.Get("/logs/month/:month", GetLogsByMonth)
}

// GetLogsByDate returns log entries matching a specific date.
// GET /logs/date/2026-03-28
func GetLogsByDate(c fiber.Ctx) error {
	dateParam := c.Params("date")
	if _, err := time.Parse("2006-01-02", dateParam); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid date format, expected YYYY-MM-DD",
		})
	}

	entries, err := filterLogs(func(t time.Time) bool {
		return t.Format("2006-01-02") == dateParam
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"date":  dateParam,
		"count": len(entries),
		"logs":  entries,
	})
}

// GetLogsByMonth returns log entries matching a specific month.
// GET /logs/month/2026-03
func GetLogsByMonth(c fiber.Ctx) error {
	monthParam := c.Params("month")
	if _, err := time.Parse("2006-01", monthParam); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid month format, expected YYYY-MM",
		})
	}

	entries, err := filterLogs(func(t time.Time) bool {
		return t.Format("2006-01") == monthParam
	})
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"month": monthParam,
		"count": len(entries),
		"logs":  entries,
	})
}

// filterLogs reads cronjob.log line by line, parses the timestamp from each
// JSON entry, and returns lines where matchFn returns true.
func filterLogs(matchFn func(time.Time) bool) ([]json.RawMessage, error) {
	file, err := os.Open(cronjobLogFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []json.RawMessage{}, nil
		}
		return nil, err
	}
	defer file.Close()

	var results []json.RawMessage
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var entry logEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}

		if entry.Time == "" {
			continue
		}

		t, err := time.Parse(time.RFC3339, entry.Time)
		if err != nil {
			t, err = time.Parse(time.RFC3339Nano, entry.Time)
			if err != nil {
				continue
			}
		}

		if matchFn(t) {
			lineCopy := make(json.RawMessage, len(line))
			copy(lineCopy, line)
			results = append(results, lineCopy)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	if results == nil {
		results = []json.RawMessage{}
	}
	return results, nil
}

