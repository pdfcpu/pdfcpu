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
	"flag"
	"fmt"
	"os"

	"github.com/pdfcpu/pdfcpu/pkg/cli"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

var (
	fileStats, mode, selectedPages string
	upw, opw, key, perm, units     string
	verbose, veryVerbose           bool
	quiet                          bool
	needStackTrace                 = true
	cmdMap                         CommandMap
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

	// The first argument is the pdfcpu command
	cmdStr := os.Args[1]

	conf := pdfcpu.NewDefaultConfiguration()

	str, err := cmdMap.Handle(cmdStr, "", conf)
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

func parseFlags(cmd *Command) {

	// Execute after command completion.

	i := 2

	// This command uses a subcommand and is therefore a special case => start flag processing after 3rd argument.
	if cmd.handler == nil {
		if len(os.Args) == 2 {
			fmt.Fprintln(os.Stderr, cmd.usageShort)
			os.Exit(1)
		}
		i = 3
	}

	// Parse commandline flags.
	if !flag.CommandLine.Parsed() {

		err := flag.CommandLine.Parse(os.Args[i:])
		if err != nil {
			os.Exit(1)
		}

		initLogging(verbose, veryVerbose)
	}

	return
}

func process(cmd *cli.Command) {

	out, err := cli.Process(cmd)

	if err != nil {
		if needStackTrace {
			fmt.Fprintf(os.Stderr, "Fatal: %+v\n", err)
		} else {
			fmt.Fprintf(os.Stderr, "%v\n", err)
		}
		os.Exit(1)
	}

	if out != nil && !quiet {
		for _, s := range out {
			fmt.Fprintln(os.Stdout, s)
		}
	}

	os.Exit(0)
}
