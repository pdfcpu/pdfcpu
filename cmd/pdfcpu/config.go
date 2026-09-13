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
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/spf13/cobra"
)

func configCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Initialize, list, inspect, validate, reset configuration",
		Long:  usageLongConfig,
	}

	cmd.AddCommand(
		&cobra.Command{
			Use:   "init",
			Short: "Create missing configuration resources",
			Long:  usageLongConfigInit,
			Args:  cobra.NoArgs,
			RunE:  initializeConfiguration,
		},
		&cobra.Command{
			Use:   "list",
			Short: "Print the stored configuration",
			Long:  usageLongConfigList,
			RunE:  wrapHandler(printConfiguration),
		},
		configInspectCmd(),
		&cobra.Command{
			Use:   "validate",
			Short: "Validate schema and values without modifying configuration",
			Long:  usageLongConfigValidate,
			Args:  cobra.NoArgs,
			RunE:  validateConfiguration,
		},
		configResetCmd(),
	)

	return cmd
}

func initializeConfiguration(_ *cobra.Command, _ []string) error {
	if conf == "disable" {
		return fmt.Errorf("initialize configuration: --conf disable selects stateless mode")
	}
	result, err := api.InitializeConfigurationWithOptions(api.ConfigurationOptions{Root: conf})
	if err != nil {
		return commandError(fmt.Errorf("initialize configuration: %w", err))
	}
	loaded := result.Configuration
	fontDir, _ := loaded.UserFontStore()
	certificateDir, _ := loaded.TrustedCertificateStore()
	root := filepath.Dir(filepath.Dir(loaded.Path))
	var b strings.Builder
	if result.Created {
		fmt.Fprintln(&b, "configuration initialized")
	} else {
		fmt.Fprintln(&b, "configuration already initialized")
	}
	fmt.Fprintf(&b, "root: %s\n", root)
	fmt.Fprintf(&b, "config: %s\n", loaded.Path)
	fmt.Fprintf(&b, "fonts: %s\n", fontDir)
	fmt.Fprintf(&b, "certificates: %s\n", certificateDir)
	fmt.Fprintf(&b, "schema version: %d\n", loaded.SchemaVersion)
	_, err = io.WriteString(os.Stdout, b.String())
	if err != nil {
		return fmt.Errorf("write configuration initialization result: %w", err)
	}
	return nil
}

func configResetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reset",
		Short: "Replace configuration with built-in defaults",
		Long:  usageLongConfigReset,
		Args:  cobra.NoArgs,
		RunE:  resetConfiguration,
	}
}

func configurationResetOptions() api.ConfigurationOptions {
	if conf == "disable" {
		return api.ConfigurationOptions{Mode: api.ConfigurationModeStateless}
	}
	return api.ConfigurationOptions{Root: conf}
}

func confirmConfigurationReset(r io.Reader, w io.Writer) (bool, error) {
	reader := bufio.NewReader(r)
	for {
		if _, err := io.WriteString(w, "Reset the selected configuration to built-in defaults? (yes/no): "); err != nil {
			return false, err
		}
		input, err := reader.ReadString('\n')
		if err != nil {
			return false, err
		}
		switch strings.TrimSpace(strings.ToLower(input)) {
		case "yes":
			return true, nil
		case "no":
			return false, nil
		default:
			if _, err := io.WriteString(w, "Please type yes or no.\n"); err != nil {
				return false, err
			}
		}
	}
}

func writeConfigurationResetResult(w io.Writer, loaded *model.Configuration) error {
	var b strings.Builder
	fmt.Fprintln(&b, "configuration reset")
	fmt.Fprintf(&b, "config: %s\n", loaded.Path)
	fmt.Fprintf(&b, "schema version: %d\n", loaded.SchemaVersion)
	_, err := io.WriteString(w, b.String())
	return err
}

func resetConfiguration(_ *cobra.Command, _ []string) error {
	options := configurationResetOptions()
	if options.Mode != api.ConfigurationModeAuto {
		return fmt.Errorf("reset configuration: %w: mode %s", api.ErrConfigurationNotWritable, options.Mode)
	}
	if !force {
		confirmed, err := confirmConfigurationReset(os.Stdin, os.Stdout)
		if err != nil {
			return fmt.Errorf("reset configuration: confirmation required; use --force: %w", err)
		}
		if !confirmed {
			_, err := io.WriteString(os.Stdout, "configuration reset canceled\n")
			return err
		}
	}
	loaded, err := api.ResetConfigurationWithOptions(options)
	if err != nil {
		return fmt.Errorf("reset configuration: %w", err)
	}
	if err := writeConfigurationResetResult(os.Stdout, loaded); err != nil {
		return fmt.Errorf("write configuration reset result: %w", err)
	}
	return nil
}

func configInspectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "inspect",
		Short: "Inspect selected paths and configured policy without modifying configuration",
		Long:  usageLongConfigInspect,
		Args:  cobra.NoArgs,
		RunE:  inspectConfiguration,
	}
	cmd.Flags().Bool("json", false, "write JSON output")
	return cmd
}

func configurationInspectionOptions() api.ConfigurationOptions {
	if conf == "disable" {
		return api.ConfigurationOptions{Mode: api.ConfigurationModeStateless}
	}
	return api.ConfigurationOptions{Root: conf}
}

func inspectionDisplayValue(value string) string {
	if value == "" {
		return "(none)"
	}
	return value
}

func formatInputByteLimit(value int64) string {
	if value == 0 {
		return "unlimited"
	}
	return formatByteLimit(value)
}

func formatByteLimit(value int64) string {
	const (
		kilobyte = int64(1 << 10)
		megabyte = int64(1 << 20)
		gigabyte = int64(1 << 30)
	)

	switch {
	case value >= gigabyte && value%gigabyte == 0:
		return fmt.Sprintf("%d GB", value/gigabyte)
	case value >= megabyte && value%megabyte == 0:
		return fmt.Sprintf("%d MB", value/megabyte)
	case value >= kilobyte && value%kilobyte == 0:
		return fmt.Sprintf("%d KB", value/kilobyte)
	default:
		return fmt.Sprintf("%d", value)
	}
}

func formatPixelLimit(value int64) string {
	const megapixel = int64(1_000_000)
	if value >= megapixel && value%megapixel == 0 {
		return fmt.Sprintf("%d MP", value/megapixel)
	}
	return fmt.Sprintf("%d", value)
}

func writeConfigurationPathInspection(
	b *strings.Builder,
	label string,
	inspection api.ConfigurationPathInspection,
) {
	fmt.Fprintf(b, "  %s:\n", label)
	fmt.Fprintf(b, "    path: %s\n", inspectionDisplayValue(inspection.Path))
	fmt.Fprintf(b, "    available: %t\n", inspection.Available)
	fmt.Fprintf(b, "    exists: %t\n", inspection.Exists)
	fmt.Fprintf(b, "    writable: %t\n", inspection.Writable)
}

func writeConfigurationInspection(w io.Writer, inspection *api.ConfigurationInspection) error {
	var b strings.Builder
	fmt.Fprintf(&b, "mode: %s\n", inspection.Mode)
	fmt.Fprintf(&b, "source: %s\n", inspection.Source)
	fmt.Fprintf(&b, "default: %t\n", inspection.Default)
	fmt.Fprintf(&b, "stateless: %t\n", inspection.Stateless)
	fmt.Fprintf(&b, "write capable: %t\n", inspection.WriteCapable)
	writeConfigurationPathInspection(&b, "root", inspection.Root)
	writeConfigurationPathInspection(&b, "config", inspection.Paths.Config)
	writeConfigurationPathInspection(&b, "fonts", inspection.Paths.Fonts)
	writeConfigurationPathInspection(&b, "certificates", inspection.Paths.Certificates)
	fmt.Fprintln(&b, "schema version:")
	fmt.Fprintf(&b, "  detected: %d\n", inspection.Schema.Detected)
	fmt.Fprintf(&b, "  minimum supported: %d\n", inspection.Schema.MinimumSupported)
	fmt.Fprintf(&b, "  maximum supported: %d\n", inspection.Schema.MaximumSupported)
	fmt.Fprintln(&b, "network:")
	fmt.Fprintf(&b, "  offline: %t\n", inspection.Network.Offline)
	fmt.Fprintf(&b, "  HTTP timeout seconds: %d\n", inspection.Network.HTTPTimeoutSeconds)
	fmt.Fprintf(&b, "  CRL timeout seconds: %d\n", inspection.Network.CRLTimeoutSeconds)
	fmt.Fprintf(&b, "  OCSP timeout seconds: %d\n", inspection.Network.OCSPTimeoutSeconds)
	fmt.Fprintf(&b, "  preferred revocation checker: %s\n", inspection.Network.PreferredRevocationChecker)
	hosts := strings.Join(inspection.Network.AllowedRevocationHosts, ", ")
	fmt.Fprintf(&b, "  allowed revocation hosts: %s\n", inspectionDisplayValue(hosts))
	fmt.Fprintln(&b, "limits:")
	fmt.Fprintf(&b, "  max input bytes: %s\n", formatInputByteLimit(inspection.Limits.MaxInputBytes))
	fmt.Fprintf(&b, "  max object bytes: %s\n", formatByteLimit(inspection.Limits.MaxObjectBytes))
	fmt.Fprintf(&b, "  max stream bytes: %s\n", formatByteLimit(inspection.Limits.MaxStreamBytes))
	fmt.Fprintf(&b, "  max decode bytes: %s\n", formatByteLimit(inspection.Limits.MaxDecodeBytes))
	fmt.Fprintf(&b, "  max image pixels: %s\n", formatPixelLimit(inspection.Limits.MaxImagePixels))
	fmt.Fprintf(&b, "  max image bytes: %s\n", formatByteLimit(inspection.Limits.MaxImageBytes))
	fmt.Fprintf(&b, "  max object count: %d\n", inspection.Limits.MaxObjectCount)
	fmt.Fprintf(&b, "  max object stream count: %d\n", inspection.Limits.MaxObjectStreamCount)
	fmt.Fprintf(&b, "  max object stream first: %s\n", formatByteLimit(inspection.Limits.MaxObjectStreamFirst))
	fmt.Fprintf(&b, "  max xref entries: %d\n", inspection.Limits.MaxXRefEntries)
	fmt.Fprintf(&b, "  max recursion depth: %d\n", inspection.Limits.MaxRecursionDepth)
	_, err := io.WriteString(w, b.String())
	return err
}

func writeConfigurationInspectionJSON(w io.Writer, inspection *api.ConfigurationInspection) error {
	encoder := json.NewEncoder(w)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "\t")
	return encoder.Encode(inspection)
}

func inspectConfiguration(cmd *cobra.Command, _ []string) error {
	inspection, err := api.InspectConfiguration(configurationInspectionOptions())
	if err != nil {
		return commandError(fmt.Errorf("inspect configuration: %w", err))
	}
	jsonOutput, err := cmd.Flags().GetBool("json")
	if err != nil {
		return fmt.Errorf("read JSON output flag: %w", err)
	}
	if jsonOutput {
		return writeConfigurationInspectionJSON(os.Stdout, inspection)
	}
	return writeConfigurationInspection(os.Stdout, inspection)
}

func validateConfiguration(_ *cobra.Command, _ []string) error {
	options := api.ConfigurationOptions{Root: conf, Mode: api.ConfigurationModeReadOnly}
	if conf == "disable" {
		options = api.ConfigurationOptions{Mode: api.ConfigurationModeStateless}
	}
	loaded, err := api.LoadConfiguration(options)
	if err != nil {
		return commandError(fmt.Errorf("validate configuration: %w", err))
	}
	path := loaded.Path
	if path == "" {
		path = "built-in defaults"
	}
	fmt.Fprintf(os.Stdout, "configuration valid\nconfig: %s\nschema version: %d\n", path, loaded.SchemaVersion)
	return nil
}

func completionCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "completion [bash|zsh|fish|powershell]",
		Short: "Generate shell completion script",
		Long: `Generate shell completion script for pdfcpu.

To load completions:

Bash:

  $ source <(pdfcpu completion bash)

  # To load completions for each session, execute once:
  # Linux:
  $ pdfcpu completion bash > /etc/bash_completion.d/pdfcpu
  # macOS:
  $ pdfcpu completion bash > $(brew --prefix)/etc/bash_completion.d/pdfcpu

Zsh:

  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:

  $ echo "autoload -U compinit; compinit" >> ~/.zshrc

  # To load completions for each session, execute once:
  $ pdfcpu completion zsh > "${fpath[1]}/_pdfcpu"

  # You will need to start a new shell for this setup to take effect.

Fish:

  $ pdfcpu completion fish | source

  # To load completions for each session, execute once:
  $ pdfcpu completion fish > ~/.config/fish/completions/pdfcpu.fish

PowerShell:

  PS> pdfcpu completion powershell | Out-String | Invoke-Expression

  # To load completions for every new session, run:
  PS> pdfcpu completion powershell > pdfcpu.ps1
  # and source this file from your PowerShell profile.
`,
		DisableFlagsInUseLine: true,
		ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
		Args:                  cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return cmd.Root().GenBashCompletion(os.Stdout)
			case "zsh":
				return cmd.Root().GenZshCompletion(os.Stdout)
			case "fish":
				return cmd.Root().GenFishCompletion(os.Stdout, true)
			case "powershell":
				return cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
			}
			return nil
		},
	}
	return cmd
}

func paperCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "paper",
		Short: "Print list of supported paper sizes",
		Long:  usageLongPaper,
		RunE:  wrapHandler(printPaperSizes),
	}
}

func selectedpagesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "selectedpages",
		Short: "Print definition of the -pages flag",
		Long:  usageLongSelectedPages,
		RunE:  wrapHandler(printSelectedPages),
	}
}

func versionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version",
		Long:  usageLongVersion,
		Args:  cobra.NoArgs,
		RunE:  printVersion,
	}
}

func printConfiguration(conf *model.Configuration, args []string) error {
	fmt.Fprintf(os.Stdout, "config: %s\n", conf.Path)
	f, err := os.Open(conf.Path)
	if err != nil {
		return fmt.Errorf("can't open %s", conf.Path)
	}
	defer f.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, f); err != nil {
		return fmt.Errorf("can't read %s", conf.Path)
	}

	fmt.Print(string(buf.String()))
	return nil
}

func confirmed() bool {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Print("(yes/no): ")
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading input. Please try again.")
			continue
		}

		input = strings.TrimSpace(strings.ToLower(input))

		switch input {
		case "yes":
			return true
		case "no":
			return false
		default:
			fmt.Println("Invalid input. Please type 'yes' or 'no'.")
		}
	}
}

func printPaperSizes(conf *model.Configuration, args []string) error {
	fmt.Fprintln(os.Stdout, paperSizes)
	return nil
}

func printSelectedPages(conf *model.Configuration, args []string) error {
	fmt.Fprintln(os.Stdout, usagePageSelection)
	return nil
}

func printVersion(_ *cobra.Command, _ []string) error {
	updateVersionInfoFromBuildInfo()
	writeVersionInfo(os.Stdout)
	return nil
}

func writeVersionInfo(w io.Writer) {
	fmt.Fprintf(w, "version: %s\n", version)
	fmt.Fprintf(w, " commit: %s\n", commit)
	fmt.Fprintf(w, "   date: %s\n", formatVersionDate(date))
	fmt.Fprintf(w, "     go: %s\n", runtime.Version())
}

func formatVersionDate(s string) string {
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return s
	}
	return t.UTC().Format("2006-01-02 15:04:05 MST")
}
