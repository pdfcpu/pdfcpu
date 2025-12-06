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

package main

import (
	"fmt"
	"os"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/spf13/cobra"
)

var (
	selectedPages                string
	upw, opw, perm, unit, conf   string
	verbose, veryVerbose         bool
	quiet, offline               bool
	offlineSet                   bool
	needStackTrace               = true
)

var rootCmd = &cobra.Command{
	Use:   "pdfcpu",
	Short: "A PDF processor written in Go",
	Long: `pdfcpu is a tool for PDF manipulation written in Go.

It supports a wide range of PDF operations including:
- Validation, optimization, and encryption
- Page manipulation (merge, split, rotate, resize, etc.)
- Watermarks and stamps
- Form filling and management
- Attachments and portfolio management
- And much more...`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Persistent flags available to all commands
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "turn on logging")
	rootCmd.PersistentFlags().BoolVar(&veryVerbose, "vv", false, "verbose logging")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "disable output")
	rootCmd.PersistentFlags().StringVarP(&conf, "conf", "c", "", "set or disable config dir: $path|disable")
	rootCmd.PersistentFlags().StringVar(&opw, "opw", "", "owner password")
	rootCmd.PersistentFlags().StringVar(&upw, "upw", "", "user password")
}

func initConfig() {
	needStackTrace = verbose || veryVerbose

	if quiet {
		return
	}

	log.SetDefaultCLILogger()

	if verbose || veryVerbose {
		log.SetDefaultDebugLogger()
		log.SetDefaultInfoLogger()
		log.SetDefaultStatsLogger()
	}

	if veryVerbose {
		log.SetDefaultTraceLogger()
		log.SetDefaultReadLogger()
		log.SetDefaultValidateLogger()
		log.SetDefaultOptimizeLogger()
		log.SetDefaultWriteLogger()
	}
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
		model.ConfigPath = conf
		return
	}
	if conf == "disable" {
		model.ConfigPath = "disable"
	}
}

func ensureDefaultConfig() (*model.Configuration, error) {
	validateConfigDirFlag()

	// Check if offline flag was explicitly set
	if cmd := rootCmd; cmd != nil {
		if f := cmd.Flag("offline"); f != nil {
			offlineSet = f.Changed
		}
	}

	if !types.MemberOf(model.ConfigPath, []string{"default", "disable"}) {
		if err := model.EnsureDefaultConfigAt(model.ConfigPath, false); err != nil {
			return nil, err
		}
	}
	return model.NewDefaultConfiguration(), nil
}

func getConfig() *model.Configuration {
	conf, err := ensureDefaultConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "pdfcpu: %v\n", err)
		os.Exit(1)
	}

	conf.OwnerPW = opw
	conf.UserPW = upw

	if offlineSet {
		conf.Offline = offline
	}

	return conf
}
