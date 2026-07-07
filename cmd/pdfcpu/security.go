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
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/cli"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/spf13/cobra"
)

type encryptOptions struct {
	mode string
	key  string
	perm string
}

func changeopwCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "changeopw inFile opwOld opwNew [ outFile ]",
		Short: "Change owner password",
		Long:  usageLongChangeOwnerPW,
		Args:  cobra.RangeArgs(3, 4),
		RunE:  wrapHandler(handleChangeOwnerPasswordCommand),
	}

	cmd.Flags().StringVar(&upw, "upw", "", "user password")

	return cmd
}

func changeupwCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "changeupw inFile upwOld upwNew [ outFile ]",
		Short: "Change user password",
		Long:  usageLongChangeUserPW,
		Args:  cobra.RangeArgs(3, 4),
		RunE:  wrapHandler(handleChangeUserPasswordCommand),
	}

	cmd.Flags().StringVar(&opw, "opw", "", "owner password")

	return cmd
}

func decryptCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "decrypt inFile [ outFile ]",
		Short: "Remove password protection",
		Long:  usageLongDecrypt,
		Args:  cobra.RangeArgs(1, 2),
		RunE:  wrapHandler(handleDecryptCommand),
	}
	addPasswordFlags(cmd)

	return cmd
}

func encryptCmd() *cobra.Command {
	opts := &encryptOptions{
		mode: "aes",
		perm: "none",
	}

	cmd := &cobra.Command{
		Use:   "encrypt inFile [ outFile ]",
		Short: "Set password protection",
		Long:  usageLongEncrypt,
		Args:  cobra.RangeArgs(1, 2),
		RunE: wrapHandler(func(conf *model.Configuration, args []string) error {
			return handleEncryptCommand(conf, args, opts)
		}),
	}
	addPasswordFlags(cmd)
	cmd.MarkFlagRequired("opw")
	cmd.Flags().StringVarP(&opts.mode, "mode", "m", opts.mode, "algorithm: rc4|aes")
	cmd.Flags().StringVarP(&opts.key, "key", "k", opts.key, "key length in bits: 40|128|256")
	cmd.Flags().StringVar(&opts.perm, "perm", opts.perm, "user access permissions: none|print|all")

	return cmd
}

func permissionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "permissions",
		Short: "List, set user access permissions",
		Long:  usageLongPerm,
	}
	addPersistentPasswordFlags(cmd)

	setCmd := &cobra.Command{
		Use:   "set inFile [ outFile ]",
		Short: "Set permissions",
		Args:  cobra.RangeArgs(1, 2),
		RunE:  wrapHandler(handleSetPermissionsCommand),
	}
	setCmd.Flags().StringVar(&perm, "perm", "none", "user access permissions")

	cmd.AddCommand(
		&cobra.Command{
			Use:   "list inFile...",
			Short: "List permissions",
			Args:  cobra.MinimumNArgs(1),
			RunE:  wrapHandler(handleListPermissionsCommand),
		},
		setCmd,
	)

	return cmd
}

func handleListPermissionsCommand(conf *model.Configuration, args []string) error {
	if conf == nil {
		return api.ErrMissingConfiguration
	}
	if len(args) == 0 {
		return api.ErrMissingPDFInput
	}

	inFiles := []string{}
	for i, arg := range args {
		if arg == "" {
			return fmt.Errorf("list permissions: input %d: %w", i+1, api.ErrMissingPDFInput)
		}
		if strings.ContainsAny(arg, "*?[") {
			matches, err := filepath.Glob(arg)
			if err != nil {
				return fmt.Errorf("list permissions: expand input pattern %q: %w", arg, err)
			}
			if len(matches) == 0 {
				return fmt.Errorf("list permissions: input pattern %q matched no files", arg)
			}
			for _, match := range matches {
				if err := inputPDFArg(conf, match); err != nil {
					return fmt.Errorf("list permissions: parse arguments: %w", err)
				}
			}
			inFiles = append(inFiles, matches...)
			continue
		}
		if err := inputPDFArg(conf, arg); err != nil {
			return fmt.Errorf("list permissions: parse arguments: %w", err)
		}
		inFiles = append(inFiles, arg)
	}

	return runCommand(cli.ListPermissionsCommand(inFiles, conf))
}

func permCompletion(permPrefix string) string {
	for _, perm := range []string{"none", "print", "all"} {
		if !strings.HasPrefix(perm, permPrefix) {
			continue
		}
		return perm
	}

	return permPrefix
}

func isBinary(s string) bool {
	_, err := strconv.ParseUint(s, 2, 12)
	return err == nil
}

func isHex(s string) bool {
	if len(s) < 2 || s[0] != 'x' {
		return false
	}
	s = s[1:]
	_, err := strconv.ParseUint(s, 16, 16)
	return err == nil
}

func configPerm(perm string, conf *model.Configuration) {
	if perm != "" {
		switch perm {
		case "none":
			conf.Permissions = model.PermissionsNone
		case "print":
			conf.Permissions = model.PermissionsPrint
		case "all":
			conf.Permissions = model.PermissionsAll
		default:
			var p uint64
			if perm[0] == 'x' {
				p, _ = strconv.ParseUint(perm[1:], 16, 16)
			} else {
				p, _ = strconv.ParseUint(perm, 2, 12)
			}
			conf.Permissions = model.PermissionFlags(p)
		}
	}
}

func validatePerm(perm string) error {
	if perm == "" || perm == "none" || perm == "print" || perm == "all" || isBinary(perm) || isHex(perm) {
		return nil
	}
	return errors.New("perm unless number must be one of: all, none, print")
}

func handleSetPermissionsCommand(conf *model.Configuration, args []string) error {
	if err := validateSecurityCommandArgs("set permissions", conf, args, 1, 2); err != nil {
		return err
	}

	normalizedPerm := perm
	if normalizedPerm != "" {
		normalizedPerm = permCompletion(normalizedPerm)
	}

	if err := validatePerm(normalizedPerm); err != nil {
		return fmt.Errorf("set permissions: validate permissions: %w", err)
	}

	inFile, outFile, err := optionalOutputPDFArgs(conf, args)
	if err != nil {
		return fmt.Errorf("set permissions: parse arguments: %w", err)
	}

	configPerm(normalizedPerm, conf)

	return runCommand(cli.SetPermissionsCommand(inFile, outFile, conf))
}

func handleDecryptCommand(conf *model.Configuration, args []string) error {
	if conf == nil {
		return api.ErrMissingConfiguration
	}
	if err := validateCryptoCommandArgs("decrypt", args); err != nil {
		return err
	}
	inFile, outFile, err := inputOutputPDFArgs(conf, args)
	if err != nil {
		return fmt.Errorf("decrypt: parse arguments: %w", err)
	}
	return runCommand(cli.DecryptCommand(inFile, outFile, conf))
}

func validateEncryptModeFlag(opts *encryptOptions) error {
	if opts == nil {
		return errors.New("missing encrypt options")
	}
	switch opts.mode {
	case "":
		opts.mode = "aes"
	case "rc4", "aes":
	default:
		return errors.New("valid modes: rc4,aes default:aes")
	}

	if opts.mode == "rc4" && opts.key == "" {
		opts.key = "128"
	}
	if opts.mode == "aes" && opts.key == "" {
		opts.key = "256"
	}
	return validateEncryptKeyFlag(opts.mode, opts.key)
}

func validateEncryptKeyFlag(mode, key string) error {
	if mode == "rc4" && key != "40" && key != "128" {
		return errors.New("supported RC4 key lengths: 40,128 default:128")
	}
	if mode == "aes" && key != "40" && key != "128" && key != "256" {
		return errors.New("supported AES key lengths: 40,128,256 default:256")
	}
	return nil
}

func validateEncryptFlags(opts *encryptOptions) error {
	if err := validateEncryptModeFlag(opts); err != nil {
		return err
	}
	if opts.perm != "none" && opts.perm != "print" && opts.perm != "all" && opts.perm != "" {
		return errors.New("supported permissions: none,print,all default:none (viewing always allowed!)")
	}
	return nil
}

func handleEncryptCommand(conf *model.Configuration, args []string, opts *encryptOptions) error {
	if conf == nil {
		return api.ErrMissingConfiguration
	}
	if err := validateCryptoCommandArgs("encrypt", args); err != nil {
		return err
	}
	if opts == nil {
		return errors.New("encrypt: missing options")
	}
	if opts.perm != "" {
		opts.perm = permCompletion(opts.perm)
	}

	if conf.OwnerPW == "" {
		return fmt.Errorf("encrypt: owner password must not be empty (use --opw): %w", pdfcpu.ErrOwnerPasswordRequired)
	}

	if err := validateEncryptFlags(opts); err != nil {
		return fmt.Errorf("encrypt: validate flags: %w", err)
	}

	conf.EncryptUsingAES = opts.mode != "rc4"

	kl, _ := strconv.Atoi(opts.key)
	conf.EncryptKeyLength = kl

	if opts.perm == "all" {
		conf.Permissions = model.PermissionsAll
	}

	if opts.perm == "print" {
		conf.Permissions = model.PermissionsPrint
	}

	inFile, outFile, err := inputOutputPDFArgs(conf, args)
	if err != nil {
		return fmt.Errorf("encrypt: parse arguments: %w", err)
	}

	return runCommand(cli.EncryptCommand(inFile, outFile, conf))
}

func validateCryptoCommandArgs(op string, args []string) error {
	if len(args) == 0 || args[0] == "" {
		return api.ErrMissingPDFInput
	}
	if len(args) > 2 {
		return fmt.Errorf("%s: expected 1 or 2 arguments", op)
	}
	return nil
}

func handleChangeUserPasswordCommand(conf *model.Configuration, args []string) error {
	const op = "change user password"

	inFile, outFile, err := passwordChangePDFArgs(op, conf, args)
	if err != nil {
		return err
	}

	pwOld := args[1]
	pwNew := args[2]

	return runCommand(cli.ChangeUserPWCommand(inFile, outFile, &pwOld, &pwNew, conf))
}

func handleChangeOwnerPasswordCommand(conf *model.Configuration, args []string) error {
	const op = "change owner password"

	inFile, outFile, err := passwordChangePDFArgs(op, conf, args)
	if err != nil {
		return err
	}

	pwOld := args[1]
	pwNew := args[2]
	if pwNew == "" {
		return fmt.Errorf("%s: new owner password must not be empty: %w", op, pdfcpu.ErrOwnerPasswordRequired)
	}

	return runCommand(cli.ChangeOwnerPWCommand(inFile, outFile, &pwOld, &pwNew, conf))
}

func validateSecurityCommandArgs(
	op string,
	conf *model.Configuration,
	args []string,
	minArgs, maxArgs int,
) error {
	if conf == nil {
		return api.ErrMissingConfiguration
	}
	if len(args) == 0 || args[0] == "" {
		return api.ErrMissingPDFInput
	}
	if len(args) < minArgs || len(args) > maxArgs {
		return fmt.Errorf("%s: expected %d or %d arguments", op, minArgs, maxArgs)
	}
	return nil
}

func passwordChangePDFArgs(op string, conf *model.Configuration, args []string) (string, string, error) {
	if err := validateSecurityCommandArgs(op, conf, args, 3, 4); err != nil {
		return "", "", err
	}

	inFile := args[0]
	if conf.CheckFileNameExt && inFile != "-" {
		if err := ensurePDFExtension(inFile); err != nil {
			return "", "", fmt.Errorf("%s: parse arguments: %w", op, err)
		}
	}

	outFile := inFile
	if inFile == "-" {
		outFile = "-"
	}
	if len(args) == 4 {
		outFile = args[3]
		if outFile != "-" {
			if err := ensurePDFExtension(outFile); err != nil {
				return "", "", fmt.Errorf("%s: parse arguments: %w", op, err)
			}
		}
		if err := ensureOutputFileAvailable(outFile); err != nil {
			return "", "", fmt.Errorf("%s: parse arguments: %w", op, err)
		}
	}

	return inFile, outFile, nil
}
