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

package api

import (
	"errors"
	"fmt"
	"os"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// ConfigurationSource identifies how the configuration root was selected.
type ConfigurationSource string

const (
	// ConfigurationSourceFlag identifies an explicit root or stateless selection, including a command-line flag.
	ConfigurationSourceFlag ConfigurationSource = "flag"

	// ConfigurationSourceEnvironment identifies a root selected by the environment.
	ConfigurationSourceEnvironment ConfigurationSource = "environment"

	// ConfigurationSourceOSDefault identifies the operating system's default configuration root.
	ConfigurationSourceOSDefault ConfigurationSource = "os-default"
)

// ConfigurationInspection describes selected configuration and filesystem state
// without exposing secret-bearing operation state.
type ConfigurationInspection struct {
	// Mode is the selected configuration mode.
	Mode ConfigurationMode `json:"mode"`

	// Source identifies how Root was selected.
	Source ConfigurationSource `json:"source"`

	// Default is true when the operating system's default root was selected.
	Default bool `json:"default"`

	// Stateless is true when built-in configuration is in use without filesystem resources.
	Stateless bool `json:"stateless"`

	// WriteCapable is true when the selected mode permits configuration writes.
	WriteCapable bool `json:"writeCapable"`

	// Root describes the selected configuration root.
	Root ConfigurationPathInspection `json:"root"`

	// Paths contains resolved configuration resource paths.
	Paths ConfigurationPathsInspection `json:"paths"`

	// Schema describes detected and supported configuration schemas.
	Schema ConfigurationSchemaInspection `json:"schema"`

	// Network contains configured outbound-network settings.
	Network ConfigurationNetworkInspection `json:"network"`

	// Limits contains configured input-driven resource limits.
	Limits ConfigurationLimitsInspection `json:"limits"`
}

// ConfigurationPathInspection describes one configuration filesystem path.
type ConfigurationPathInspection struct {
	// Path is the resolved filesystem path. It is empty when the resource is unavailable.
	Path string `json:"path"`

	// Available is true when the selected mode provides this resource.
	Available bool `json:"available"`

	// Exists is true when Path exists.
	Exists bool `json:"exists"`

	// Writable is true when the current process may write to Path.
	Writable bool `json:"writable"`
}

// ConfigurationPathsInspection contains resolved configuration resource paths.
type ConfigurationPathsInspection struct {
	// Config describes config.yml.
	Config ConfigurationPathInspection `json:"config"`

	// Fonts describes the user-font directory.
	Fonts ConfigurationPathInspection `json:"fonts"`

	// Certificates describes the trusted-certificate directory.
	Certificates ConfigurationPathInspection `json:"certificates"`
}

// ConfigurationSchemaInspection describes configuration schema compatibility.
type ConfigurationSchemaInspection struct {
	// Detected is the schema version read from the selected configuration.
	Detected int `json:"detected"`

	// MinimumSupported is the oldest schema version supported by this build.
	MinimumSupported int `json:"minimumSupported"`

	// MaximumSupported is the newest schema version supported by this build.
	MaximumSupported int `json:"maximumSupported"`
}

// ConfigurationNetworkInspection contains configured outbound-network settings.
type ConfigurationNetworkInspection struct {
	// Offline disables outbound network activity.
	Offline bool `json:"offline"`

	// HTTPTimeoutSeconds is the general HTTP timeout in seconds.
	HTTPTimeoutSeconds int `json:"httpTimeoutSeconds"`

	// CRLTimeoutSeconds is the certificate-revocation-list timeout in seconds.
	CRLTimeoutSeconds int `json:"crlTimeoutSeconds"`

	// OCSPTimeoutSeconds is the Online Certificate Status Protocol timeout in seconds.
	OCSPTimeoutSeconds int `json:"ocspTimeoutSeconds"`

	// PreferredRevocationChecker identifies the preferred certificate revocation mechanism.
	PreferredRevocationChecker string `json:"preferredRevocationChecker"`

	// AllowedRevocationHosts lists private hosts permitted for certificate revocation checks.
	AllowedRevocationHosts []string `json:"allowedRevocationHosts"`
}

// ConfigurationLimitsInspection contains configured input-driven resource limits.
type ConfigurationLimitsInspection struct {
	// MaxInputBytes limits each PDF input and stdin spool. Zero means unlimited.
	MaxInputBytes int64 `json:"maxInputBytes"`

	// MaxObjectBytes limits an indirect-object buffer, excluding stream payloads.
	MaxObjectBytes int64 `json:"maxObjectBytes"`

	// MaxStreamBytes limits encoded stream bytes read from a PDF.
	MaxStreamBytes int64 `json:"maxStreamBytes"`

	// MaxDecodeBytes limits decoded stream bytes produced by filters.
	MaxDecodeBytes int64 `json:"maxDecodeBytes"`

	// MaxImagePixels limits decoded or rendered image dimensions.
	MaxImagePixels int64 `json:"maxImagePixels"`

	// MaxImageBytes limits decoded or rendered image buffer sizes.
	MaxImageBytes int64 `json:"maxImageBytes"`

	// MaxObjectCount limits xref stream size expansion.
	MaxObjectCount int `json:"maxObjectCount"`

	// MaxObjectStreamCount limits object stream entry counts.
	MaxObjectStreamCount int `json:"maxObjectStreamCount"`

	// MaxObjectStreamFirst limits object stream prolog bytes.
	MaxObjectStreamFirst int64 `json:"maxObjectStreamFirst"`

	// MaxXRefEntries limits xref stream index expansion.
	MaxXRefEntries int `json:"maxXRefEntries"`

	// MaxRecursionDepth limits recursive parsing and object graph traversal.
	MaxRecursionDepth int `json:"maxRecursionDepth"`
}

func inspectConfigurationPath(path string, available bool) (ConfigurationPathInspection, error) {
	inspection := ConfigurationPathInspection{Path: path, Available: available}
	if !available || path == "" {
		return inspection, nil
	}
	info, err := os.Stat(path)
	if errors.Is(err, os.ErrNotExist) {
		return inspection, nil
	}
	if err != nil {
		return inspection, fmt.Errorf("inspect configuration path %q: %w", path, err)
	}
	inspection.Exists = true
	inspection.Writable = configurationPathWritable(path, info)
	return inspection, nil
}

func inspectConfigurationPaths(root string, conf *model.Configuration) (ConfigurationPathInspection, ConfigurationPathsInspection, error) {
	rootInspection, err := inspectConfigurationPath(root, true)
	if err != nil {
		return ConfigurationPathInspection{}, ConfigurationPathsInspection{}, err
	}
	configInspection, err := inspectConfigurationPath(conf.Path, true)
	if err != nil {
		return ConfigurationPathInspection{}, ConfigurationPathsInspection{}, err
	}
	fontDir, fontsAvailable := conf.UserFontStore()
	fontInspection, err := inspectConfigurationPath(fontDir, fontsAvailable)
	if err != nil {
		return ConfigurationPathInspection{}, ConfigurationPathsInspection{}, err
	}
	certificateDir, certificatesAvailable := conf.TrustedCertificateStore()
	certificateInspection, err := inspectConfigurationPath(certificateDir, certificatesAvailable)
	if err != nil {
		return ConfigurationPathInspection{}, ConfigurationPathsInspection{}, err
	}
	return rootInspection, ConfigurationPathsInspection{
		Config:       configInspection,
		Fonts:        fontInspection,
		Certificates: certificateInspection,
	}, nil
}

func inspectConfigurationLimits(limits model.ResourceLimits) ConfigurationLimitsInspection {
	return ConfigurationLimitsInspection{
		MaxInputBytes:        limits.MaxInputBytes,
		MaxObjectBytes:       limits.MaxObjectBytes,
		MaxStreamBytes:       limits.MaxStreamBytes,
		MaxDecodeBytes:       limits.MaxDecodeBytes,
		MaxImagePixels:       limits.MaxImagePixels,
		MaxImageBytes:        limits.MaxImageBytes,
		MaxObjectCount:       limits.MaxObjectCount,
		MaxObjectStreamCount: limits.MaxObjectStreamCount,
		MaxObjectStreamFirst: limits.MaxObjectStreamFirst,
		MaxXRefEntries:       limits.MaxXRefEntries,
		MaxRecursionDepth:    limits.MaxRecursionDepth,
	}
}

func inspectConfigurationNetwork(conf *model.Configuration) ConfigurationNetworkInspection {
	hosts := append([]string{}, conf.AllowedRevocationHosts...)
	checker := "CRL"
	if conf.PreferredCertRevocationChecker == model.OCSP {
		checker = "OCSP"
	}
	return ConfigurationNetworkInspection{
		Offline:                    conf.Offline,
		HTTPTimeoutSeconds:         conf.Timeout,
		CRLTimeoutSeconds:          conf.TimeoutCRL,
		OCSPTimeoutSeconds:         conf.TimeoutOCSP,
		PreferredRevocationChecker: checker,
		AllowedRevocationHosts:     hosts,
	}
}

func configurationForInspection(options ConfigurationOptions) (*model.Configuration, string, ConfigurationSource, error) {
	if err := validateConfigurationOptions(options); err != nil {
		return nil, "", "", err
	}
	if options.Mode == ConfigurationModeStateless {
		return model.NewStatelessConfiguration(), "", ConfigurationSourceFlag, nil
	}
	root, source, err := resolveConfigurationSelection(options.Root)
	if err != nil {
		return nil, "", "", err
	}
	conf, err := LoadConfiguration(ConfigurationOptions{Root: root, Mode: ConfigurationModeReadOnly})
	if err != nil {
		return nil, "", "", err
	}
	return conf, root, source, nil
}

func buildConfigurationInspection(options ConfigurationOptions, conf *model.Configuration, root string, source ConfigurationSource) (*ConfigurationInspection, error) {
	rootInspection := ConfigurationPathInspection{}
	paths := ConfigurationPathsInspection{}
	var err error
	if options.Mode != ConfigurationModeStateless {
		rootInspection, paths, err = inspectConfigurationPaths(root, conf)
		if err != nil {
			return nil, err
		}
	}
	return &ConfigurationInspection{
		Mode:         options.Mode,
		Source:       source,
		Default:      source == ConfigurationSourceOSDefault,
		Stateless:    options.Mode == ConfigurationModeStateless,
		WriteCapable: options.Mode == ConfigurationModeAuto,
		Root:         rootInspection,
		Paths:        paths,
		Schema: ConfigurationSchemaInspection{
			Detected:         conf.SchemaVersion,
			MinimumSupported: model.ConfigurationSchemaVersionCurrent,
			MaximumSupported: model.ConfigurationSchemaVersionCurrent,
		},
		Network: inspectConfigurationNetwork(conf),
		Limits:  inspectConfigurationLimits(conf.Limits),
	}, nil
}

// InspectConfiguration reports selected configuration and filesystem state without modifying either.
// Automatic mode reads existing state without initializing missing resources.
func InspectConfiguration(options ConfigurationOptions) (*ConfigurationInspection, error) {
	conf, root, source, err := configurationForInspection(options)
	if err != nil {
		return nil, err
	}
	return buildConfigurationInspection(options, conf, root, source)
}
