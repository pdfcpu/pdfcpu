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

package safemath

import (
	"errors"
	"math"
)

// AddInt returns a+b unless either operand is negative or the result would overflow int.
func AddInt(a, b int) (int, error) {
	if a < 0 || b < 0 || a > math.MaxInt-b {
		return 0, errors.New("pdfcpu: integer overflow")
	}
	return a + b, nil
}

// MultiplyInt returns a*b unless either operand is negative or the result would overflow int.
func MultiplyInt(a, b int) (int, error) {
	if a < 0 || b < 0 || a != 0 && b > math.MaxInt/a {
		return 0, errors.New("pdfcpu: integer overflow")
	}
	return a * b, nil
}

// MultiplyInt64 returns a*b unless either operand is negative or the result would overflow int64.
func MultiplyInt64(a, b int64) (int64, error) {
	if a < 0 || b < 0 || a != 0 && b > math.MaxInt64/a {
		return 0, errors.New("pdfcpu: integer overflow")
	}
	return a * b, nil
}
