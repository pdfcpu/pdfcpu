/*
Copyright 2023 The pdfcpu Authors.

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

package format

import (
	"strconv"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// Text returns a string with resolved place holders for pageNr, pageCount, timestamp or pdfcpu version.
func Text(text, timeStampFormat string, pageNr, pageCount int) (string, bool) {
	// replace  %p with pageNr
	//			%P with pageCount
	//			%t with timestamp
	//			%v with pdfcpu version
	var (
		bb         []byte
		hasPercent bool
		unique     bool
	)
	for i := 0; i < len(text); i++ {
		if text[i] == '%' {
			if hasPercent {
				bb = append(bb, '%')
			}
			hasPercent = true
			continue
		}
		if hasPercent {
			hasPercent = false
			if text[i] == 'p' {
				bb = append(bb, strconv.Itoa(pageNr)...)
				unique = true
				continue
			}
			if text[i] == 'P' {
				bb = append(bb, strconv.Itoa(pageCount)...)
				unique = true
				continue
			}
			if text[i] == 't' {
				bb = append(bb, time.Now().Format(timeStampFormat)...)
				unique = true
				continue
			}
			if text[i] == 'v' {
				bb = append(bb, model.VersionStr...)
				unique = true
				continue
			}
		}
		bb = append(bb, text[i])
	}
	return string(bb), unique
}
