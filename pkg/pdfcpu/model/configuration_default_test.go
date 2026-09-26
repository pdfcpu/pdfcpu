//go:build !js

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
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/font"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/fault"
)

func isolatedDefaultConfigurationRoot(t *testing.T) string {
	t.Helper()
	preserveConfigurationGlobals(t)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("XDG_CONFIG_HOME", t.TempDir())
	t.Setenv("AppData", t.TempDir())
	root, err := os.UserConfigDir()
	if err != nil {
		t.Fatal(err)
	}
	ConfigPath = "default"
	loadedDefaultConfig = nil
	font.UserFontDir = ""
	TrustedCertDir = ""
	return root
}

func defaultConfigurationResourcesReady(conf *Configuration) error {
	fontDir, fontsAvailable := conf.UserFontStore()
	certDir, certsAvailable := conf.TrustedCertificateStore()
	if !fontsAvailable || !certsAvailable {
		return fmt.Errorf("default configuration resources unavailable")
	}
	for _, path := range []string{filepath.Join(fontDir, "Roboto-Regular.gob"), certDir} {
		if _, err := os.Stat(path); err != nil {
			return err
		}
	}
	return nil
}

func concurrentDefaultConfigurations(t *testing.T) []*Configuration {
	t.Helper()
	const workers = 8
	configs := make([]*Configuration, workers)
	errs := make([]error, workers)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			configs[i] = NewDefaultConfiguration()
			errs[i] = defaultConfigurationResourcesReady(configs[i])
		}()
	}
	close(start)
	wg.Wait()
	seen := make(map[*Configuration]bool)
	for i, conf := range configs {
		if conf == nil || seen[conf] {
			t.Fatalf("worker %d returned a nil or shared configuration", i)
		}
		if errs[i] != nil {
			t.Fatalf("worker %d returned before resources were ready: %v", i, errs[i])
		}
		seen[conf] = true
	}
	return configs
}

// TestNewDefaultConfigurationConcurrentInitialization covers missing and existing configuration files.
func TestNewDefaultConfigurationConcurrentInitialization(t *testing.T) {
	for _, existing := range []bool{false, true} {
		t.Run(fmt.Sprintf("existing=%t", existing), func(t *testing.T) {
			root := isolatedDefaultConfigurationRoot(t)
			path := filepath.Join(root, "pdfcpu", "config.yml")
			if existing {
				if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
					t.Fatal(err)
				}
				bb := bytes.Replace(configFileBytes, []byte("offline: false"), []byte("offline: true"), 1)
				bb = bytes.Replace(bb, []byte("allowedRevocationHosts: []"),
					[]byte("allowedRevocationHosts: [ocsp.example.corp]"), 1)
				if err := os.WriteFile(path, bb, 0600); err != nil {
					t.Fatal(err)
				}
			}
			for range 2 {
				for _, conf := range concurrentDefaultConfigurations(t) {
					if conf.Path != path || conf.Offline != existing {
						t.Fatalf("unexpected configuration: path=%q offline=%t", conf.Path, conf.Offline)
					}
					if existing {
						if len(conf.AllowedRevocationHosts) != 1 || conf.AllowedRevocationHosts[0] != "ocsp.example.corp" {
							t.Fatalf("configuration hosts changed: %v", conf.AllowedRevocationHosts)
						}
						conf.AllowedRevocationHosts[0] = "changed.example.corp"
					}
				}
			}
			if err := ResetConfig(); err != nil {
				t.Fatal(err)
			}
			if conf := NewDefaultConfiguration(); conf.Offline || len(conf.AllowedRevocationHosts) != 0 {
				t.Fatal("reset did not replace the cached configuration")
			}
		})
	}
}

func defaultConfigurationWithError() (conf *Configuration, err error) {
	defer fault.Catch(&err)
	return NewDefaultConfiguration(), nil
}

// TestNewDefaultConfigurationRetriesResourceFailure verifies failed initialization does not publish loader state.
func TestNewDefaultConfigurationRetriesResourceFailure(t *testing.T) {
	root := isolatedDefaultConfigurationRoot(t)
	configDir := filepath.Join(root, "pdfcpu")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	fontPath := filepath.Join(configDir, "fonts")
	if err := os.WriteFile(fontPath, []byte("blocked"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := defaultConfigurationWithError(); err == nil {
		t.Fatal("expected font directory initialization error")
	}
	if loadedDefaultConfig != nil || font.UserFontDir != "" || TrustedCertDir != "" {
		t.Fatal("failed initialization published loader state")
	}
	if err := os.Remove(fontPath); err != nil {
		t.Fatal(err)
	}
	conf, err := defaultConfigurationWithError()
	if err != nil {
		t.Fatalf("retry configuration initialization: %v", err)
	}
	if err := defaultConfigurationResourcesReady(conf); err != nil {
		t.Fatal(err)
	}
}
