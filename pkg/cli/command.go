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

package cli

import (
	"io"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// Command represents an execution context.
type Command struct {
	Mode              model.CommandMode
	InFile            *string
	InFileCert        *string
	InFilePrivateKey  *string
	InFileJSON        *string
	InFiles           []string
	InDir             *string
	OutFile           *string
	OutFileJSON       *string
	OutDir            *string
	PageSelection     []string
	PWOld             *string
	PWNew             *string
	StringVal         string
	IntVal            int
	BoolVal1          bool
	BoolVal2          bool
	BoolVal3          bool
	IntVals           []int
	StringVals        []string
	StringMap         map[string]string
	Input             io.ReadSeeker
	Inputs            []io.ReadSeeker
	Output            io.Writer
	ErrorOutput       io.Writer
	Box               *model.Box
	Import            *pdfcpu.Import
	NUp               *model.NUp
	Cut               *model.Cut
	PageBoundaries    *model.PageBoundaries
	Resize            *model.Resize
	Zoom              *model.Zoom
	Watermark         *model.Watermark
	ViewerPreferences *model.ViewerPreferences
	PageConf          *pdfcpu.PageConfiguration
	Conf              *model.Configuration
}
