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
	"io/ioutil"
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

// pdfcpu's 3 defined loggers.
var (
	Debug = &logger{}
	Info  = &logger{}
	Stats = &logger{}
	Trace = &logger{}
	//Validate = &logger{}
	//Write    = &logger{}
	//Stats    = &logger{}
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

// SetTraceLogger sets the stats logger.
func SetTraceLogger(log Logger) {
	Trace.log = log
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

// SetDefaultTraceLogger sets the default stats logger.
func SetDefaultTraceLogger() {
	SetTraceLogger(log.New(ioutil.Discard, "TRACE: ", log.Ldate|log.Ltime))
}

// SetDefaultLoggers sets all loggers to their default logger.
func SetDefaultLoggers() {
	SetDefaultDebugLogger()
	SetDefaultInfoLogger()
	SetDefaultStatsLogger()
	SetDefaultTraceLogger()
}

// DisableLoggers turns off all logging.
func DisableLoggers() {
	SetDebugLogger(nil)
	SetInfoLogger(nil)
	SetStatsLogger(nil)
	SetTraceLogger(nil)
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

func (l *logger) Fatalf(format string, args ...interface{}) {

	if l.log == nil {
		return
	}

	l.log.Fatalf(format, args)
}

func (l *logger) Fatalln(args ...interface{}) {

	if l.log == nil {
		return
	}

	l.log.Fatalln(args)
}
