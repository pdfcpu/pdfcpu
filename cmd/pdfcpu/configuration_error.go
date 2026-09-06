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

	"github.com/pdfcpu/pdfcpu/pkg/api"
)

type configurationSchemaCommandError struct {
	err    error
	schema *api.ConfigurationSchemaCompatibilityError
}

func newConfigurationSchemaCommandError(err error) error {
	var schemaErr *api.ConfigurationSchemaCompatibilityError
	if !errors.As(err, &schemaErr) {
		return nil
	}
	return &configurationSchemaCommandError{err: err, schema: schemaErr}
}

func configurationSchemaLabel(version int) string {
	if version == 0 {
		return "legacy"
	}
	return fmt.Sprintf("%d", version)
}

func selectedConfigurationResetCommand() string {
	if conf == "" {
		return "pdfcpu config reset"
	}
	return fmt.Sprintf("pdfcpu --conf %q config reset", conf)
}

func (e *configurationSchemaCommandError) Error() string {
	if errors.Is(e.err, api.ErrConfigurationResetRequired) {
		return fmt.Sprintf(
			"configuration reset required\nconfig: %s\ndetected schema version: %s\nrequired schema version: %d\nrun: %s",
			e.schema.Path,
			configurationSchemaLabel(e.schema.Detected),
			e.schema.Current,
			selectedConfigurationResetCommand(),
		)
	}
	return fmt.Sprintf(
		"configuration schema is newer than this pdfcpu version\n"+
			"config: %s\ndetected schema version: %d\nsupported schema version: %d\n"+
			"upgrade pdfcpu before using this configuration",
		e.schema.Path,
		e.schema.Detected,
		e.schema.Current,
	)
}

func (e *configurationSchemaCommandError) Unwrap() error {
	return e.err
}
