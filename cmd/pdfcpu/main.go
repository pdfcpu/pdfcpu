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
	"fmt"
	"os"
	"runtime/debug"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// Set by Goreleaser.
var (
	version = model.VersionStr
	commit  = "?"
	date    = "?"
)

func init() {
	// Update version info from build info if not set by Goreleaser
	if date == "?" {
		if info, ok := debug.ReadBuildInfo(); ok {
			for _, setting := range info.Settings {
				if setting.Key == "vcs.revision" {
					commit = setting.Value
					if len(commit) >= 8 {
						commit = commit[:8]
					}
				}
				if setting.Key == "vcs.time" {
					date = setting.Value
				}
			}
		}
	}
}

func main() {
	if err := Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
