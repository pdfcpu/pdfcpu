/*
Copyright 2026 The pdfcpu Authors.

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

package cli

import (
	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func commandWritesPDFToStdout(cmd *Command) bool {
	if cmd == nil {
		return false
	}
	switch cmd.Mode {
	case model.POSTER, model.NDOWN, model.CUT:
		return false
	case model.EXTRACTPAGES:
		return cmd.OutDir != nil && *cmd.OutDir == "-"
	}
	if cmd.OutFile == nil {
		return false
	}
	if *cmd.OutFile == "-" {
		return true
	}
	return *cmd.OutFile == "" && cmd.InFile != nil && *cmd.InFile == "-"
}

func reportCommandProgress(cmd *Command, format string, args ...any) {
	if commandWritesPDFToStdout(cmd) {
		return
	}
	log.CLI.Printf(format, args...)
}

func commandOutputPath(cmd *Command) string {
	if cmd == nil || cmd.OutFile == nil {
		return ""
	}
	if *cmd.OutFile != "" {
		return *cmd.OutFile
	}
	if cmd.InFile == nil {
		return ""
	}
	return *cmd.InFile
}

func reportCommandOutputPath(cmd *Command) {
	reportOutputPath(commandOutputPath(cmd))
}

func reportOutputPath(outFile string) {
	if outFile == "" || outFile == "-" {
		return
	}
	log.CLI.Printf("writing %s...\n", outFile)
}
