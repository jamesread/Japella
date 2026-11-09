package utils

import (
	"fmt"
	"os"

	"github.com/jamesread/armature-health/errorlog"
	"github.com/jamesread/golure/pkg/logsetup"
	"github.com/mattn/go-isatty"
	log "github.com/sirupsen/logrus"
)

const DefaultLogFile = "japella.log"

var errLog = errorlog.New()

// ErrorLog returns the process-wide error log buffer for the logs.recent_errors health check.
func ErrorLog() *errorlog.ErrorLog {
	return errLog
}

func configureConsoleFormatter() {
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp: false,
		ForceColors:   isatty.IsTerminal(os.Stdout.Fd()),
	})
}

func SetupLogging() {
	configureConsoleFormatter()
	log.SetOutput(os.Stdout)
	log.AddHook(errLog)

	logPath := os.Getenv("JAPELLA_LOG_FILE")
	if logPath == "" {
		logPath = DefaultLogFile
	}

	if err := logsetup.AddFileHook(logPath, &logsetup.BracketFormatter{}); err != nil {
		fmt.Fprintf(os.Stderr, "japella: %v\n", err)
		return
	}

	log.Infof("logging to %s", logPath)
}
