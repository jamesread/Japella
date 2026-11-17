package utils

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jamesread/armature-health/errorlog"
	"github.com/mattn/go-isatty"
	log "github.com/sirupsen/logrus"
)

const DefaultLogFile = "japella.log"

var (
	logBaseTime = time.Now()
	errLog      = errorlog.New()
)

// ErrorLog returns the process-wide error log buffer for the logs.recent_errors health check.
func ErrorLog() *errorlog.ErrorLog {
	return errLog
}

// BracketFormatter emits plain logrus TTY-style lines (INFO[0306] message fields)
// without ANSI color codes, for log file output.
type BracketFormatter struct{}

func (f *BracketFormatter) Format(entry *log.Entry) ([]byte, error) {
	level := strings.ToUpper(entry.Level.String())
	if len(level) > 4 {
		level = level[:4]
	}

	elapsed := int(entry.Time.Sub(logBaseTime) / time.Second)
	message := strings.TrimSuffix(entry.Message, "\n")

	keys := make([]string, 0, len(entry.Data))
	for k := range entry.Data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b bytes.Buffer
	fmt.Fprintf(&b, "%s[%04d] %-44s", level, elapsed, message)
	for _, k := range keys {
		fmt.Fprintf(&b, " %s=", k)
		appendLogValue(&b, entry.Data[k])
	}
	b.WriteByte('\n')
	return b.Bytes(), nil
}

func appendLogValue(b *bytes.Buffer, value any) {
	switch v := value.(type) {
	case string:
		if needsQuoting(v) {
			b.WriteByte('"')
			b.WriteString(v)
			b.WriteByte('"')
		} else {
			b.WriteString(v)
		}
	default:
		fmt.Fprint(b, v)
	}
}

func needsQuoting(text string) bool {
	if len(text) == 0 {
		return true
	}
	for _, ch := range text {
		if !isUnquotedLogRune(ch) {
			return true
		}
	}
	return false
}

func isUnquotedLogRune(ch rune) bool {
	return isAlnumLogRune(ch) || isExtraUnquotedLogRune(ch)
}

func isAlnumLogRune(ch rune) bool {
	return isAZ(ch) || isAZUpper(ch) || isDigit(ch)
}

func isExtraUnquotedLogRune(ch rune) bool {
	return ch == '-' || ch == '.' || ch == '_'
}

func isAZ(ch rune) bool      { return ch >= 'a' && ch <= 'z' }
func isAZUpper(ch rune) bool { return ch >= 'A' && ch <= 'Z' }
func isDigit(ch rune) bool   { return ch >= '0' && ch <= '9' }

func openLogFile(logPath string) (*os.File, bool) {
	if dir := filepath.Dir(logPath); dir != "" && dir != "." {
		if err := os.MkdirAll(dir, 0o750); err != nil { //nolint:gosec // log path is chosen by the operator
			fmt.Fprintf(os.Stderr, "japella: failed to create log directory %s: %v\n", dir, err)
			return nil, false
		}
	}
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600) //nolint:gosec // log path is chosen by the operator
	if err != nil {
		fmt.Fprintf(os.Stderr, "japella: failed to open log file %s: %v\n", logPath, err)
		return nil, false
	}
	return file, true
}

type fileLogHook struct {
	writer    io.Writer
	formatter log.Formatter
}

func (h *fileLogHook) Levels() []log.Level {
	return log.AllLevels
}

func (h *fileLogHook) Fire(entry *log.Entry) error {
	line, err := h.formatter.Format(entry)
	if err != nil {
		return err
	}
	_, err = h.writer.Write(line)
	return err
}

func configureConsoleFormatter() {
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: false,
		ForceColors:   isatty.IsTerminal(os.Stdout.Fd()),
	})
}

// ConfigureLogger applies the shared formatter, level, and output from the
// standard logrus logger. Use for any log.New() instances that must remain separate.
func ConfigureLogger(l *log.Logger) {
	if l == nil {
		return
	}
	l.SetFormatter(log.StandardLogger().Formatter)
	l.SetOutput(log.StandardLogger().Out)
	l.SetLevel(log.StandardLogger().GetLevel())
}

func SetupLogging() {
	configureConsoleFormatter()
	log.SetOutput(os.Stdout)
	log.AddHook(errLog)

	logPath := os.Getenv("JAPELLA_LOG_FILE")
	if logPath == "" {
		logPath = DefaultLogFile
	}
	file, ok := openLogFile(logPath)
	if !ok {
		return
	}

	log.AddHook(&fileLogHook{
		writer:    file,
		formatter: &BracketFormatter{},
	})

	log.Infof("logging to %s", logPath)
}
