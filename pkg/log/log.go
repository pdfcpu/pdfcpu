/*
Copyright 2018 The pdfcpu Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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

	// Fatalf is equivalent to Printf() followed by a program abort.
	Fatalf(format string, args ...interface{})

	// Fatalln is equivalent to Println() followed by a progam abort.
	Fatalln(args ...interface{})
}

type logger struct {
	log Logger
}

// pdfcpu's loggers.
var (

	// Horizontal loggers
	Debug = &logger{}
	Info  = &logger{}
	Stats = &logger{}
	Trace = &logger{}

	// Vertical loggers
	Parse    = &logger{}
	Read     = &logger{}
	Validate = &logger{}
	Optimize = &logger{}
	Write    = &logger{}
	CLI      = &logger{}
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

// SetTraceLogger sets the trace logger.
func SetTraceLogger(log Logger) {
	Trace.log = log
}

// SetParseLogger sets the parse logger.
func SetParseLogger(log Logger) {
	Parse.log = log
}

// SetReadLogger sets the read logger.
func SetReadLogger(log Logger) {
	Read.log = log
}

// SetValidateLogger sets the validate logger.
func SetValidateLogger(log Logger) {
	Validate.log = log
}

// SetOptimizeLogger sets the optimize logger.
func SetOptimizeLogger(log Logger) {
	Optimize.log = log
}

// SetWriteLogger sets the write logger.
func SetWriteLogger(log Logger) {
	Write.log = log
}

// SetCLILogger sets the api logger.
func SetCLILogger(log Logger) {
	CLI.log = log
}

// SetDefaultDebugLogger sets the default debug logger.
func SetDefaultDebugLogger() {
	SetDebugLogger(log.New(os.Stderr, "DEBUG: ", log.Ldate|log.Ltime))
}

// SetDefaultInfoLogger sets the default info logger.
func SetDefaultInfoLogger() {
	SetInfoLogger(log.New(os.Stderr, " INFO: ", log.Ldate|log.Ltime))
}

// SetDefaultStatsLogger sets the default stats logger.
func SetDefaultStatsLogger() {
	SetStatsLogger(log.New(os.Stderr, "STATS: ", log.Ldate|log.Ltime))
}

// SetDefaultTraceLogger sets the default trace logger.
func SetDefaultTraceLogger() {
	SetTraceLogger(log.New(os.Stderr, "TRACE: ", log.Ldate|log.Ltime))
}

// SetDefaultParseLogger sets the default parse logger.
func SetDefaultParseLogger() {
	SetParseLogger(log.New(os.Stderr, "PARSE: ", log.Ldate|log.Ltime))
}

// SetDefaultReadLogger sets the default read logger.
func SetDefaultReadLogger() {
	SetReadLogger(log.New(os.Stderr, " READ: ", log.Ldate|log.Ltime))
}

// SetDefaultValidateLogger sets the default validate logger.
func SetDefaultValidateLogger() {
	SetValidateLogger(log.New(os.Stderr, "VALID: ", log.Ldate|log.Ltime))
}

// SetDefaultOptimizeLogger sets the default optimize logger.
func SetDefaultOptimizeLogger() {
	SetOptimizeLogger(log.New(os.Stderr, "  OPT: ", log.Ldate|log.Ltime))
}

// SetDefaultWriteLogger sets the default write logger.
func SetDefaultWriteLogger() {
	SetWriteLogger(log.New(os.Stderr, "WRITE: ", log.Ldate|log.Ltime))
}

// SetDefaultCLILogger sets the default cli logger.
func SetDefaultCLILogger() {
	SetCLILogger(log.New(os.Stdout, "", 0))
}

// SetDefaultLoggers sets all loggers to their default logger.
func SetDefaultLoggers() {
	SetDefaultDebugLogger()
	SetDefaultInfoLogger()
	SetDefaultStatsLogger()
	SetDefaultTraceLogger()
	SetDefaultParseLogger()
	SetDefaultReadLogger()
	SetDefaultValidateLogger()
	SetDefaultOptimizeLogger()
	SetDefaultWriteLogger()
	SetDefaultCLILogger()
}

// DisableLoggers turns off all logging.
func DisableLoggers() {
	SetDebugLogger(nil)
	SetInfoLogger(nil)
	SetStatsLogger(nil)
	SetTraceLogger(nil)
	SetParseLogger(nil)
	SetReadLogger(nil)
	SetValidateLogger(nil)
	SetOptimizeLogger(nil)
	SetWriteLogger(nil)
	SetCLILogger(nil)
}

// IsTraceLoggerEnabled returns true if the Trace Logger is enabled.
func IsTraceLoggerEnabled() bool {
	return Trace.log != nil
}

// IsCLILoggerEnabled returns true if the CLI Logger is enabled.
func IsCLILoggerEnabled() bool {
	return CLI.log != nil
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

// Fatalf is equivalent to Printf() followed by a program abort.
func (l *logger) Fatalf(format string, args ...interface{}) {

	if l.log == nil {
		return
	}

	l.log.Fatalf(format, args...)
}

// Fatalf is equivalent to Println() followed by a program abort.
func (l *logger) Fatalln(args ...interface{}) {

	if l.log == nil {
		return
	}

	l.log.Fatalln(args...)
}
