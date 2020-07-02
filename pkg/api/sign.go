/*
Copyright 2020 The pdf Authors.

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
	"os"
	"time"

	pdf "github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
)

// SignFile signs inFile with a digital signature and writes the result to outFile.
func SignFile(inFile, outFile string, conf *pdf.Configuration) (err error) {

	if conf == nil {
		conf = pdf.NewDefaultConfiguration()
	}

	f, err := os.Open(inFile)
	if err != nil {
		return err
	}

	ctx, _, _, _, err := readValidateAndOptimize(f, conf, time.Now())
	if err != nil {
		return err
	}

	return ctx.Sign(outFile)
}
