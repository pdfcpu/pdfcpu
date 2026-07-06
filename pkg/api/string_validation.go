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
	"strings"
)

func validateNoEmptyStrings(ss []string, name string) error {
	for _, s := range ss {
		if strings.TrimSpace(s) == "" {
			return fmt.Errorf("%s must not be empty", name)
		}
	}
	return nil
}

func validateAttachmentFileNames(files []string) error {
	for _, file := range files {
		fileName := strings.TrimSpace(strings.SplitN(file, ",", 2)[0])
		if fileName == "" {
			return errors.New("attachment filename must not be empty")
		}
	}
	return nil
}

func validateProperties(properties map[string]string) error {
	for k, v := range properties {
		if strings.TrimSpace(k) == "" {
			return errors.New("property name must not be empty")
		}
		if strings.TrimSpace(v) == "" {
			return errors.New("property value must not be empty")
		}
	}
	return nil
}
