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

package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/pdfcpu/pdfcpu/internal/contextutil"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/spf13/cobra"
)

func addPasswordFileFlag(cmd *cobra.Command, name string, persistent bool) {
	flags := cmd.Flags()
	if persistent {
		flags = cmd.PersistentFlags()
	}
	flags.String(name+"-file", "", "read password from file; one trailing line ending (LF or CRLF) is allowed and ignored")
	cmd.MarkFlagsMutuallyExclusive(name, name+"-file")
}

func addPasswordFileFlags(cmd *cobra.Command, persistent bool) {
	addPasswordFileFlag(cmd, "upw", persistent)
	addPasswordFileFlag(cmd, "opw", persistent)
}

func passwordChangeFileNames(cmd *cobra.Command) (string, string) {
	prefix := strings.TrimPrefix(cmd.Name(), "change")
	return prefix + "old-file", prefix + "new-file"
}

func addPasswordChangeFlags(cmd *cobra.Command) {
	oldName, newName := passwordChangeFileNames(cmd)
	cmd.Flags().String(oldName, "", "read old password from file; requires --"+newName)
	cmd.Flags().String(newName, "", "read new password from file; requires --"+oldName)
	cmd.MarkFlagsRequiredTogether(oldName, newName)
}

func readPasswordFile(cmd *cobra.Command, name string) (string, error) {
	path, err := cmd.Flags().GetString(name)
	if err != nil {
		return "", err
	}
	if path == "" || path == "-" {
		return "", fmt.Errorf("--%s requires a filesystem path", name)
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("--%s: %w", name, err)
	}
	value := string(b)
	if before, ok := strings.CutSuffix(value, "\r\n"); ok {
		return before, nil
	}
	return strings.TrimSuffix(value, "\n"), nil
}

func commandPasswords(cmd *cobra.Command) (map[string]string, error) {
	passwords := make(map[string]string)
	for _, name := range []string{"upw", "opw"} {
		if !cmd.Flags().Changed(name + "-file") {
			continue
		}
		if cmd.Flags().Changed(name) {
			return nil, fmt.Errorf("--%s and --%s-file are mutually exclusive", name, name)
		}
		value, err := readPasswordFile(cmd, name+"-file")
		if err != nil {
			return nil, err
		}
		passwords[name] = value
	}
	return passwords, nil
}

func applyCommandPasswords(conf *model.Configuration, passwords map[string]string) {
	if value, ok := passwords["upw"]; ok {
		conf.UserPW = value
	}
	if value, ok := passwords["opw"]; ok {
		conf.OwnerPW = value
	}
}

func passwordChangeCommandArgs(cmd *cobra.Command, args []string) error {
	oldName, newName := passwordChangeFileNames(cmd)
	oldSet := cmd.Flags().Changed(oldName)
	newSet := cmd.Flags().Changed(newName)
	if oldSet != newSet {
		return fmt.Errorf("--%s and --%s must be provided together", oldName, newName)
	}
	if !oldSet {
		return cobra.RangeArgs(3, 4)(cmd, args)
	}
	if len(args) < 1 || len(args) > 2 {
		return fmt.Errorf("password files require inFile [outFile], without positional passwords")
	}
	return nil
}

func wrapPasswordChangeHandler(handler func(context.Context, *model.Configuration, []string) error) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if err := contextutil.Check(cmd.Context()); err != nil {
			return err
		}
		if err := passwordChangeCommandArgs(cmd, args); err != nil {
			return err
		}
		oldName, newName := passwordChangeFileNames(cmd)
		if cmd.Flags().Changed(oldName) {
			oldPW, err := readPasswordFile(cmd, oldName)
			if err != nil {
				return err
			}
			newPW, err := readPasswordFile(cmd, newName)
			if err != nil {
				return err
			}
			args = append([]string{args[0], oldPW, newPW}, args[1:]...)
		}
		return wrapContextHandler(handler)(cmd, args)
	}
}
