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

func ShowMsg(msg string) {
	s := "pdfcpu " + msg
	if log.DebugEnabled() {
		log.Debug.Println(s)
	}
	if log.ReadEnabled() {
		log.Read.Println(s)
	}
	if log.ValidateEnabled() {
		log.Validate.Println(s)
	}
	if log.CLIEnabled() {
		log.CLI.Println(s)
	}
}

func ShowMsgTopic(topic, msg string) {
	msg = topic + ": " + msg
	ShowMsg(msg)
}

func ShowRepaired(msg string) {
	ShowMsgTopic("repaired", msg)
}

func ShowSkipped(msg string) {
	ShowMsgTopic("skipped", msg)
}

func ShowDigestedSpecViolation(msg string) {
	ShowMsgTopic("digested", msg)
}

func ShowDigestedSpecViolationError(xRefTable *XRefTable, err error) {
	msg := fmt.Sprintf("spec violation around obj#(%d): %v\n", xRefTable.CurObj, err)
	ShowMsgTopic("digested", msg)
}
