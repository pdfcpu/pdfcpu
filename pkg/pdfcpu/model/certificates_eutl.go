//go:build pdfcpu_eutl
// +build pdfcpu_eutl

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

package model

import (
	"embed"
	"os"
	"path/filepath"
)

//go:embed resources/certs/*.p7c
var certFilesEU embed.FS

func installDefaultCertificates() error {
	files, err := certFilesEU.ReadDir("resources/certs")
	if err != nil {
		return err
	}

	euDir := filepath.Join(TrustedCertDir, "eu")
	if err := os.MkdirAll(euDir, 0755); err != nil {
		return err
	}

	for _, file := range files {
		if err := installDefaultCertificate(file.Name(), euDir); err != nil {
			return err
		}
	}

	return nil
}

func installDefaultCertificate(name, dir string) error {
	content, err := certFilesEU.ReadFile("resources/certs/" + name)
	if err != nil {
		return err
	}

	return os.WriteFile(filepath.Join(dir, name), content, 0644)
}
