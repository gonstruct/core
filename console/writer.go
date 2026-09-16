package console

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"time"

	"golang.org/x/term"
)

const lineWidth = 84

const (
	colorReset  = "\033[0m"
	colorGray   = "\033[90m"
	colorGreen  = "\033[1;32m"
	colorRed    = "\033[1;31m"
	colorYellow = "\033[1;33m"
	colorCyan   = "\033[1;36m"

	badgeDebug = "\033[97;100m DEBUG \033[0m"
	badgeInfo  = "\033[97;44m INFO \033[0m"
	badgeWarn  = "\033[30;43m WARN \033[0m"
	badgeError = "\033[97;41m ERROR \033[0m"
)

type ConsoleWriter struct {
	out io.Writer
}

func (self *ConsoleWriter) Write(data []byte) (int, error) {
	event := map[string]any{}
	if err := json.Unmarshal(data, &event); err != nil {
		return self.out.Write(data)
	}

	switch {
	case event["task"] == true:
		self.task(event)
	case event["request"] == true:
		self.request(event)
	default:
		self.line(event)
	}

	return len(data), nil
}

func (self *ConsoleWriter) line(event map[string]any) {
	message, _ := event["message"].(string)

	suffix := ""
	if event["section"] == true {
		suffix = "\n"
	}

	fmt.Fprintf(self.out, "\n %s %s%s\n%s", self.badge(event), message, self.fields(event), suffix)
}

func (self *ConsoleWriter) task(event map[string]any) {
	description, _ := event["message"].(string)
	elapsed := self.elapsed(event)

	status := colorGreen + "DONE" + colorReset
	if event["level"] == "error" {
		status = colorRed + "FAIL" + colorReset
	}

	dotCount := max(self.width()-len(description)-len(elapsed)-9, 3)
	dots := strings.Repeat(".", dotCount)
	fmt.Fprintf(self.out, "  %s %s%s %s%s %s\n", description, colorGray, dots, elapsed, colorReset, status)
}

func (self *ConsoleWriter) request(event map[string]any) {
	description, _ := event["message"].(string)
	requestID, _ := event["request_id"].(string)
	if requestID != "" {
		description += " " + requestID
	}
	startedAt := self.timestamp(event)
	elapsed := self.elapsed(event)
	status, _ := event["status"].(float64)
	statusText := fmt.Sprintf("%d", int(status))

	dotCount := max(self.width()-len(startedAt)-len(description)-len(elapsed)-len(statusText)-6, 3)
	dots := strings.Repeat(".", dotCount)
	fmt.Fprintf(
		self.out,
		" %s%s%s %s %s%s %s%s %s%s%s\n",
		colorGray, startedAt, colorReset,
		description,
		colorGray, dots, elapsed, colorReset,
		self.statusColor(int(status)), statusText, colorReset,
	)
}

func (self *ConsoleWriter) fields(event map[string]any) string {
	keys := make([]string, 0, len(event))
	for key := range event {
		switch key {
		case "level", "time", "message", "error", "section":
			continue
		}
		keys = append(keys, key)
	}
	slices.Sort(keys)

	builder := strings.Builder{}
	for _, key := range keys {
		fmt.Fprintf(&builder, " %s%s=%v%s", colorGray, key, event[key], colorReset)
	}

	if errorText, exists := event["error"]; exists {
		fmt.Fprintf(&builder, " %serr=%v%s", colorRed, errorText, colorReset)
	}

	return builder.String()
}

func (*ConsoleWriter) badge(event map[string]any) string {
	switch event["level"] {
	case "debug":
		return badgeDebug
	case "warn":
		return badgeWarn
	case "error", "fatal":
		return badgeError
	default:
		return badgeInfo
	}
}

func (*ConsoleWriter) statusColor(status int) string {
	switch {
	case status >= 500:
		return colorRed
	case status >= 400:
		return colorYellow
	case status >= 300:
		return colorCyan
	default:
		return colorGreen
	}
}

func (*ConsoleWriter) elapsed(event map[string]any) string {
	milliseconds, _ := event["duration"].(float64)
	elapsed := time.Duration(milliseconds * float64(time.Millisecond))

	if elapsed < time.Millisecond {
		return fmt.Sprintf("%dµs", elapsed.Microseconds())
	}

	if elapsed < time.Second {
		return fmt.Sprintf("%.2fms", float64(elapsed.Microseconds())/1000)
	}

	return fmt.Sprintf("%.2fs", elapsed.Seconds())
}

func (*ConsoleWriter) timestamp(event map[string]any) string {
	text, _ := event["time"].(string)
	parsed, err := time.Parse(time.RFC3339, text)
	if err != nil {
		parsed = time.Now()
	}

	return parsed.Format("2006-01-02 15:04:05")
}

func (*ConsoleWriter) width() int {
	width, _, err := term.GetSize(int(os.Stderr.Fd()))
	if err != nil || width < lineWidth {
		return lineWidth
	}

	return width
}
