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

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func getBaseDir(path string) string {
	i := strings.Index(path, "**")
	basePath := path[:i]
	basePath = filepath.Clean(basePath)
	if basePath == "" {
		return "."
	}
	return basePath
}

func isDir(path string) (bool, error) {
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	return info.IsDir(), nil
}

func expandWildcardsRec(s string, inFiles *[]string, conf *model.Configuration) error {
	s = filepath.Clean(s)
	wantsPdf := strings.HasSuffix(s, ".pdf")
	return filepath.WalkDir(getBaseDir(s), func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if ok := hasPDFExtension(path); ok {
			*inFiles = append(*inFiles, path)
			return nil
		}
		if !wantsPdf && conf.CheckFileNameExt {
			if !quiet {
				fmt.Fprintf(os.Stderr, "%s needs extension \".pdf\".\n", path)
			}
		}
		return nil
	})
}

func expandWildcards(s string, inFiles *[]string, conf *model.Configuration) error {
	paths, err := filepath.Glob(s)
	if err != nil {
		return err
	}
	for _, path := range paths {
		if conf.CheckFileNameExt {
			if !hasPDFExtension(path) {
				if isDir, err := isDir(path); isDir && err == nil {
					continue
				}
				if !quiet {
					fmt.Fprintf(os.Stderr, "%s needs extension \".pdf\".\n", path)
				}
				continue
			}
		}

		*inFiles = append(*inFiles, path)
	}
	return nil
}

func collectInFiles(conf *model.Configuration, args []string) ([]string, error) {
	inFiles := []string{}
	for _, arg := range args {
		if arg == "-" {
			inFiles = append(inFiles, arg)
			continue
		}

		if strings.Contains(arg, "**") {
			// **/			skips files w/o extension "pdf"
			// **/*.pdf
			if err := expandWildcardsRec(arg, &inFiles, conf); err != nil {
				return nil, fmt.Errorf("expand input %q: %w", arg, err)
			}
			continue
		}

		if strings.Contains(arg, "*") {
			// *			skips files w/o extension "pdf"
			// *.pdf
			if err := expandWildcards(arg, &inFiles, conf); err != nil {
				return nil, fmt.Errorf("expand input %q: %w", arg, err)
			}
			continue
		}

		if conf.CheckFileNameExt {
			if !hasPDFExtension(arg) {
				if isDir, err := isDir(arg); isDir && err == nil {
					if err := expandWildcards(arg+"/*", &inFiles, conf); err != nil {
						return nil, fmt.Errorf("expand input %q: %w", arg, err)
					}
					continue
				}
				if !quiet {
					fmt.Fprintf(os.Stderr, "%s needs extension \".pdf\".\n", arg)
				}
				continue
			}
		}

		inFiles = append(inFiles, arg)
	}
	return inFiles, nil
}
