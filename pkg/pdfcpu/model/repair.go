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

func ReportSpecViolation(xRefTable *XRefTable, err error) {
	// TODO Apply across code base.
	pre := fmt.Sprintf("digesting spec violation around obj#(%d)", xRefTable.CurObj)
	if log.DebugEnabled() {
		log.Debug.Printf("%s: %v\n", pre, err)
	}
	if log.ReadEnabled() {
		log.Read.Printf("%s: %v\n", pre, err)
	}
	if log.ValidateEnabled() {
		log.Validate.Printf("%s: %v\n", pre, err)
	}
	if log.CLIEnabled() {
		log.CLI.Printf("%s: %v\n", pre, err)
	}
}

func ShowRepaired(msg string) {
	msg = "repaired: " + msg
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
