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

package pdfcpu

import (
	"fmt"
	"strings"
)

type parameterMap[T any] map[string]func(string, *T) error

func handleParameter[T any](m map[string]func(string, *T) error, paramPrefix, paramValueStr string, v *T) error {
	param := ""
	prefix := strings.ToLower(paramPrefix)

	for k := range m {
		if !strings.HasPrefix(strings.ToLower(k), prefix) {
			continue
		}
		if param != "" {
			return fmt.Errorf("ambiguous parameter prefix \"%s\"", paramPrefix)
		}
		param = k
	}

	if param == "" {
		return fmt.Errorf("unknown parameter prefix \"%s\"", paramPrefix)
	}

	return m[param](paramValueStr, v)
}
