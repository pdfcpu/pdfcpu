/*
Copyright 2018 The pdfcpu Authors.

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

import "fmt"

type dim struct {
	w, h int
}

func (d dim) String() string {
	return fmt.Sprintf("%dx%d points", d.w, d.h)
}

// PaperSize is a map of known paper sizes.
var PaperSize = map[string]dim{
	// ISO A
	"A0": dim{2384, 3370},
	"A1": dim{1684, 2384},
	"A2": dim{1190, 1684},
	"A3": dim{842, 1190},
	"A4": dim{595, 842},
	"A5": dim{420, 595},
	"A6": dim{298, 420},
	"A7": dim{210, 298},
	"A8": dim{148, 210},
	// ISO B
	// ISO C
	// ISO D
	// ISO RA & SRA
	// American
	"Letter":    dim{612, 792}, // ANSI A
	"Legal":     dim{612, 1008},
	"Ledger":    dim{792, 1224}, // ANSI B
	"Tabloid":   dim{1224, 792}, // ANSI B
	"Executive": dim{522, 756},
	"ANSIC":     dim{1584, 1224},
	"ANSID":     dim{2448, 1584},
	"ANSIE":     dim{3168, 2448},
}
