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
)

var (
	fileStats, mode, selectedPages  string
	upw, opw, key, perm, unit, conf string
	verbose, veryVerbose            bool
	quiet, sorted                   bool
	needStackTrace                  = true
	cmdMap                          commandMap
)

// Set by Goreleaser.
var (
	commit = "?"
	date   = "?"
)

func init() {
	initFlags()
	initCommandMap()
}

func main() {
	if len(os.Args) == 1 {
		fmt.Fprintln(os.Stderr, usage)
		os.Exit(0)
	}

	// The first argument is the pdfcpu command string.
	cmdStr := os.Args[1]

	// Process command string for given configuration.
	str, err := cmdMap.process(cmdStr, "")
	if err != nil {
		if len(str) > 0 {
			cmdStr = fmt.Sprintf("%s %s", str, os.Args[2])
		}
		fmt.Fprintf(os.Stderr, "%v \"%s\"\n", err, cmdStr)
		fmt.Fprintln(os.Stderr, "Run 'pdfcpu help' for usage.")
		os.Exit(1)
	}

	os.Exit(0)
}
