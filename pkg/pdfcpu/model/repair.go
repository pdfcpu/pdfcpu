/*
Copyright 2024 The pdfcpu Authors.

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

package model

import (
	"fmt"

	"github.com/pdfcpu/pdfcpu/pkg/log"
)

func showMessage(topic, msg string) {
	msg = topic + ": " + msg
	if log.DebugEnabled() {
		log.Debug.Println("pdfcpu " + msg)
	}
	if log.ReadEnabled() {
		log.Read.Println("pdfcpu " + msg)
	}
	if log.ValidateEnabled() {
		log.Validate.Println("pdfcpu " + msg)
	}
	if log.CLIEnabled() {
		log.CLI.Println(msg)
	}
}

func ShowRepaired(msg string) {
	showMessage("repaired", msg)
}

func ShowSkipped(msg string) {
	showMessage("skipped", msg)
}

func ShowDigestedSpecViolation(msg string) {
	showMessage("digested", msg)
}

func ShowDigestedSpecViolationError(xRefTable *XRefTable, err error) {
	msg := fmt.Sprintf("spec violation around obj#(%d): %v\n", xRefTable.CurObj, err)
	showMessage("digested", msg)
}
