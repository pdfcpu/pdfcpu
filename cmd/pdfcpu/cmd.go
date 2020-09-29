/*
Copyright 2020 The pdfcpu Authors.

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
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

var (
	errUnknownCmd   = errors.New("pdfcpu: unknown command")
	errAmbiguousCmd = errors.New("pdfcpu: ambiguous command")
)

// Command represents command meta information and details.
type command struct {
	handler    func(conf *pdfcpu.Configuration)
	cmdMap     commandMap // Optional map of sub commands.
	usageShort string     // Short command description.
	usageLong  string     // Long command description.
}

func (c command) String() string {
	return fmt.Sprintf("cmd: <%s> <%s>\n", c.usageShort, c.usageLong)
}

type commandMap map[string]*command

func newCommandMap() commandMap {
	return map[string]*command{}
}

func (m commandMap) register(cmdStr string, cmd command) {
	m[cmdStr] = &cmd
}

func parseFlags(cmd *command) {
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

func validateConfigDirFlag() {
	if len(conf) > 0 && conf != "disable" {
		info, err := os.Stat(conf)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Fprintf(os.Stderr, "conf: %s does not exist\n\n", conf)
				os.Exit(1)
			}
			fmt.Fprintf(os.Stderr, "conf: %s %v\n\n", conf, err)
			os.Exit(1)
		}
		if !info.IsDir() {
			fmt.Fprintf(os.Stderr, "conf: %s not a directory\n\n", conf)
			os.Exit(1)
		}
		pdfcpu.ConfigPath = conf
		return
	}
	if conf == "disable" {
		pdfcpu.ConfigPath = "disable"
	}
}

func ensureDefaultConfig() (*pdfcpu.Configuration, error) {
	validateConfigDirFlag()
	//fmt.Printf("conf = %s\n", pdfcpu.ConfigPath)
	if !pdfcpu.MemberOf(pdfcpu.ConfigPath, []string{"default", "disable"}) {
		if err := pdfcpu.EnsureDefaultConfigAt(pdfcpu.ConfigPath); err != nil {
			return nil, err
		}
	}
	return pdfcpu.NewDefaultConfiguration(), nil
}

// process applies command completion and if successful processes the resulting command.
func (m commandMap) process(cmdPrefix string, command string) (string, error) {
	var cmdStr string

	// Support command completion.
	for k := range m {
		if !strings.HasPrefix(k, cmdPrefix) {
			continue
		}
		if len(cmdStr) > 0 {
			return command, errAmbiguousCmd
		}
		cmdStr = k
	}

	if cmdStr == "" {
		return command, errUnknownCmd
	}

	parseFlags(m[cmdStr])

	conf, err := ensureDefaultConfig()
	if err != nil {
		return command, err
	}

	conf.OwnerPW = opw
	conf.UserPW = upw

	if m[cmdStr].handler != nil {
		m[cmdStr].handler(conf)
		return command, nil
	}

	if len(os.Args) == 2 {
		fmt.Fprintln(os.Stderr, m[cmdStr].usageShort)
		os.Exit(1)
	}

	return m[cmdStr].cmdMap.process(os.Args[2], cmdStr)
}

// HelpString returns documentation for a topic.
func (m commandMap) HelpString(topic string) (string, error) {
	topicStr := ""
	for k := range m {
		if !strings.HasPrefix(k, topic) {
			continue
		}
		if len(topicStr) > 0 {
			return topic, errAmbiguousCmd
		}
		topicStr = k
	}

	cmd, ok := m[topicStr]
	if !ok || cmd.usageShort == "" {
		return fmt.Sprintf("Unknown help topic `%s`.  Run 'pdfcpu help'.\n", topic), nil
	}

	return fmt.Sprintf("%s\n\n%s\n", cmd.usageShort, cmd.usageLong), nil
}

func (m commandMap) String() string {
	logStr := []string{}
	for k, v := range m {
		logStr = append(logStr, fmt.Sprintf("%s: %v\n", k, v))
	}
	return strings.Join(logStr, "")
}
