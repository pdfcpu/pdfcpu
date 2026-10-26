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

// Package streamlimit provides shared decoded-stream limit calculations.
package streamlimit

import "math"

// DecodeAllowance returns the cumulative decoded-stream allowance for a resolved per-stream limit.
func DecodeAllowance(maxDecodeBytes int64) int64 {
	if maxDecodeBytes < 0 {
		return -1
	}
	if maxDecodeBytes <= 1<<29 {
		return 1 << 30
	}
	if maxDecodeBytes > math.MaxInt64/2 {
		return math.MaxInt64
	}
	return 2 * maxDecodeBytes
}
