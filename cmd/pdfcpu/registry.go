/*
Copyright 2019 The pdfcpu Authors.

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

package main

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/hhrutter/pdfcpu/pkg/api"
	"github.com/hhrutter/pdfcpu/pkg/pdfcpu"
)

var (
	errUnknownCmd   = errors.New("Unknown command")
	errAmbiguousCmd = errors.New("Ambiguous command")
)

// Command represents command meta information and command details wrapped into api.Command.
type Command struct {
	handler    func(config *pdfcpu.Configuration) *api.Command
	cmdMap     CommandMap // Optional sub commands.
	usageShort string     // Short command description.
	usageLong  string     // Long command description.
}

func (c Command) String() string {
	return fmt.Sprintf("cmd: <%s> <%s>\n", c.usageShort, c.usageLong)
}

// CommandMap is a command execution engine supporting completion.
type CommandMap map[string]*Command

// NewCommandMap returns an initialized command map.
func NewCommandMap() CommandMap {
	return map[string]*Command{}
}

// Register adds a new command.
func (m CommandMap) Register(cmdStr string, cmd Command) {
	m[cmdStr] = &cmd
}

// Handle applies command completion and if successful
// executes the correct prepare function and processes the resulting command.
func (m CommandMap) Handle(cmdPrefix string, command string, config *pdfcpu.Configuration) (*api.Command, string, error) {

	var cmdStr string

	for k := range m {
		if !strings.HasPrefix(k, cmdPrefix) {
			continue
		}
		if len(cmdStr) > 0 {
			return nil, command, errAmbiguousCmd
		}
		cmdStr = k
	}

	if cmdStr == "" {
		return nil, command, errUnknownCmd
	}

	parseFlags(m[cmdStr])

	if m[cmdStr].handler != nil {
		return m[cmdStr].handler(config), command, nil
	}

	if len(os.Args) == 2 {
		fmt.Fprintln(os.Stderr, m[cmdStr].usageShort)
		os.Exit(1)
	}

	return m[cmdStr].cmdMap.Handle(os.Args[2], cmdStr, config)
}

// HelpString returns documentation for a topic.
func (m CommandMap) HelpString(topic string) string {

	cmd, ok := m[topic]
	if !ok || cmd.usageShort == "" {
		return fmt.Sprintf("Unknown help topic `%s`.  Run 'pdfcpu help'.\n", topic)
	}

	return fmt.Sprintf("%s\n\n%s\n", cmd.usageShort, cmd.usageLong)
}

func (m CommandMap) String() string {

	logStr := []string{}

	for k, v := range m {
		logStr = append(logStr, fmt.Sprintf("%s: %v\n", k, v))
	}

	return strings.Join(logStr, "")
}
