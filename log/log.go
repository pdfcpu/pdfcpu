// Package log provides a logging abstraction.
package log

import (
	"log"
	"os"
)

// Logger defines an interface for logging messages.
type Logger interface {

	// Printf logs a formatted string.
	Printf(format string, args ...interface{})

	// Println logs a line.
	Println(args ...interface{})
}

type logger struct {
	log Logger
}

// pdfcpu's 3 defined loggers.
var (
	Debug = &logger{}
	Info  = &logger{}
	Stats = &logger{}
)

// SetDebugLogger sets the debug logger.
func SetDebugLogger(log Logger) {
	Debug.log = log
}

// SetInfoLogger sets the info logger.
func SetInfoLogger(log Logger) {
	Info.log = log
}

// SetStatsLogger sets the stats logger.
func SetStatsLogger(log Logger) {
	Stats.log = log
}

// SetDefaultDebugLogger sets the default debug logger.
func SetDefaultDebugLogger() {
	SetDebugLogger(log.New(os.Stderr, "DEBUG: ", log.Ldate|log.Ltime))
}

// SetDefaultInfoLogger sets the default info logger.
func SetDefaultInfoLogger() {
	SetInfoLogger(log.New(os.Stderr, "INFO: ", log.Ldate|log.Ltime))
}

// SetDefaultStatsLogger sets the default stats logger.
func SetDefaultStatsLogger() {
	SetStatsLogger(log.New(os.Stderr, "STATS: ", log.Ldate|log.Ltime))
}

// SetDefaultLoggers sets all loggers to their default logger.
func SetDefaultLoggers() {
	SetDefaultDebugLogger()
	SetDefaultInfoLogger()
	SetDefaultStatsLogger()
}

// DisableLoggers turns off all logging.
func DisableLoggers() {
	SetDebugLogger(nil)
	SetInfoLogger(nil)
	SetStatsLogger(nil)
}

// Printf writes a formatted message to the log.
func (l *logger) Printf(format string, args ...interface{}) {

	if l.log == nil {
		return
	}

	l.log.Printf(format, args...)
}

// Println writes a line to the log.
func (l *logger) Println(args ...interface{}) {

	if l.log == nil {
		return
	}

	l.log.Println(args...)
}
