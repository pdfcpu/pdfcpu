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

// Package main provides the command line for interacting with pdfcpu.
package main

import (
	"errors"
	"fmt"
	"os"
	"runtime/debug"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/fault"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// Set by Goreleaser.
var (
	version = model.VersionStr
	commit  = "?"
	date    = "?"
)

func init() {
	updateVersionInfoFromBuildInfo()
}

func updateVersionInfoFromBuildInfo() {
	if commit != "?" && date != "?" {
		return
	}

	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	setVersionInfoFromBuildSettings(info.Settings)
}

func setVersionInfoFromBuildSettings(settings []debug.BuildSetting) {
	for _, setting := range settings {
		switch setting.Key {
		case "vcs.revision":
			if commit == "?" {
				commit = shortCommit(setting.Value)
			}
		case "vcs.time":
			if date == "?" {
				date = setting.Value
			}
		}
	}
}

func shortCommit(s string) string {
	if len(s) < 8 {
		return s
	}
	return s[:8]
}

func printError(err error) {
	if !needStackTrace {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		return
	}

	fmt.Fprintf(os.Stderr, "Fatal: %v\n", err)
	var p fault.Panic
	if errors.As(err, &p) && len(p.Stack) > 0 {
		fmt.Fprintf(os.Stderr, "\nStack Trace:\n%s", p.Stack)
	}
}

func main() {
	if err := Execute(); err != nil {
		printError(err)
		os.Exit(1)
	}
}
