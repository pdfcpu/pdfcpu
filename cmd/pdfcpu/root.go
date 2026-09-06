/*
Copyright 2025 The pdfcpu Authors.

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
	"path/filepath"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/font"
	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/spf13/cobra"
)

var (
	conf             string
	force            bool
	kpw              string
	needStackTrace   bool //= true
	offline          bool
	offlineSet       bool
	opw              string
	perm             string
	quiet            bool
	removeEncryption bool
	removeSignatures bool
	selectedPages    string
	unit             string
	upw              string
	verbose          int
)

var rootCmd = &cobra.Command{
	Use:   "pdfcpu",
	Short: "PDF tooling for Go and the command line",
	Long: `pdfcpu provides command-line tools for working with PDF files.
It is built on a Go API for direct PDF control.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute runs the root command.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVarP(&conf, "conf", "c", "", "set or disable config dir: $path | disable")
	rootCmd.PersistentFlags().BoolVar(&force, "force", false, "allow overwriting files and other destructive operations")
	rootCmd.PersistentFlags().BoolVarP(&offline, "offline", "o", false, "disable http traffic")
	rootCmd.PersistentFlags().BoolVarP(&quiet, "quiet", "q", false, "disable output")
	rootCmd.PersistentFlags().CountVarP(&verbose, "verbose", "v", "Increase verbosity. Use -v or -vv.")
	rootCmd.AddCommand(commands()...)
}

func commands() []*cobra.Command {
	return []*cobra.Command{
		// Document commands.
		validateCmd(),
		optimizeCmd(),
		infoCmd(),
		dumpCmd(),
		createCmd(),
		mergeCmd(),
		splitCmd(),
		trimCmd(),
		collectCmd(),

		// Page commands.
		pagesCmd(),
		rotateCmd(),
		nupCmd(),
		gridCmd(),
		bookletCmd(),
		resizeCmd(),
		posterCmd(),
		ndownCmd(),
		cutCmd(),
		cropCmd(),
		zoomCmd(),
		boxesCmd(),

		// Content commands.
		watermarkCmd(),
		stampCmd(),
		annotationsCmd(),
		bookmarksCmd(),
		pagemodeCmd(),
		pagelayoutCmd(),
		viewerprefCmd(),

		// Resource commands.
		importCmd(),
		fontsCmd(),
		imagesCmd(),
		attachmentsCmd(),
		portfolioCmd(),
		keywordsCmd(),
		propertiesCmd(),

		// Extract commands.
		extractCmd(),

		// Form commands.
		formCmd(),

		// Security commands.
		encryptCmd(),
		decryptCmd(),
		changeupwCmd(),
		changeopwCmd(),
		permissionsCmd(),

		// Trust and signature commands.
		certificatesCmd(),
		signaturesCmd(),

		// Support commands.
		completionCmd(),
		configCmd(),
		paperCmd(),
		selectedpagesCmd(),
		versionCmd(),
	}
}

func initConfig() {

	if verbose > 2 {
		verbose = 2
	}

	needStackTrace = verbose > 0

	if quiet {
		return
	}

	log.SetDefaultCLILogger()

	//log.SetDefaultParseLogger()

	if verbose > 0 {
		log.SetDefaultDebugLogger()
		log.SetDefaultInfoLogger()
		log.SetDefaultStatsLogger()
	}

	if verbose == 2 {
		log.SetDefaultTraceLogger()
		log.SetDefaultReadLogger()
		log.SetDefaultValidateLogger()
		log.SetDefaultOptimizeLogger()
		log.SetDefaultWriteLogger()
	}
}

func commandConfigurationOptions() (api.ConfigurationOptions, error) {
	if conf == "disable" {
		return api.ConfigurationOptions{Mode: api.ConfigurationModeStateless}, nil
	}
	options := api.ConfigurationOptions{Root: conf}
	if conf == "" {
		return options, nil
	}

	info, err := os.Stat(conf)
	if err != nil {
		if os.IsNotExist(err) {
			return api.ConfigurationOptions{}, fmt.Errorf("conf: %s does not exist", conf)
		}
		return api.ConfigurationOptions{}, fmt.Errorf("conf: %s %v", conf, err)
	}
	if !info.IsDir() {
		return api.ConfigurationOptions{}, fmt.Errorf("conf: %s not a directory", conf)
	}
	return options, nil
}

func activateCommandConfiguration(c *model.Configuration, options api.ConfigurationOptions) {
	if options.Mode == api.ConfigurationModeStateless {
		model.ConfigPath = "disable"
		font.UserFontDir = ""
		model.TrustedCertDir = ""
		return
	}

	model.ConfigPath = filepath.Dir(filepath.Dir(c.Path))
	if dir, ok := c.UserFontStore(); ok {
		font.UserFontDir = dir
	}
	if dir, ok := c.TrustedCertificateStore(); ok {
		model.TrustedCertDir = dir
	}
}

func loadCommandConfiguration() (*model.Configuration, error) {
	options, err := commandConfigurationOptions()
	if err != nil {
		return nil, err
	}

	// Check if offline flag was explicitly set
	if cmd := rootCmd; cmd != nil {
		if f := cmd.Flag("offline"); f != nil {
			offlineSet = f.Changed
		}
	}

	c, err := api.LoadConfigurationWithOptions(options)
	if err != nil {
		return nil, err
	}
	activateCommandConfiguration(c, options)
	return c, nil
}

func getConfig() (*model.Configuration, error) {
	conf, err := loadCommandConfiguration()
	if err != nil {
		return nil, fmt.Errorf("pdfcpu: %w", err)
	}

	conf.OwnerPW = opw
	conf.UserPW = upw
	conf.PrivateKeyPW = kpw
	conf.RemoveSignatures = removeSignatures
	conf.RemoveEncryption = removeEncryption

	if offlineSet {
		conf.Offline = offline
	}

	return conf, nil
}
