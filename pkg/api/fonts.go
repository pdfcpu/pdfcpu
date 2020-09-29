/*
	Copyright 2020 The pdfcpu Authors.

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
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/font"
	"github.com/pdfcpu/pdfcpu/pkg/log"
)

func isSupportedFontFile(filename string) bool {
	return strings.HasSuffix(strings.ToLower(filename), ".gob")
}

// ListFonts returns a list of supported fonts.
func ListFonts() ([]string, error) {

	// Get list of PDF core fonts.
	coreFonts := font.CoreFontNames()
	for i, s := range coreFonts {
		coreFonts[i] = "  " + s
	}
	sort.Strings(coreFonts)

	sscf := []string{"Corefonts:"}
	sscf = append(sscf, coreFonts...)

	// Get installed fonts from pdfcpu config dir in users home dir
	userFonts := font.UserFontNames()
	for i, s := range userFonts {
		userFonts[i] = "  " + s
	}
	sort.Strings(userFonts)

	ssuf := []string{fmt.Sprintf("Userfonts(%s):", font.UserFontDir)}
	ssuf = append(ssuf, userFonts...)

	sscf = append(sscf, "")
	return append(sscf, ssuf...), nil
}

// InstallFonts installs true type fonts for embedding.
func InstallFonts(fileNames []string) error {
	log.CLI.Printf("installing to %s...", font.UserFontDir)
	for _, fn := range fileNames {
		switch filepath.Ext(fn) {
		case ".ttf":
			log.CLI.Println(filepath.Base(fn))
			if err := font.InstallTrueTypeFont(font.UserFontDir, fn); err != nil {
				return err
			}
		}
	}
	return nil
}
