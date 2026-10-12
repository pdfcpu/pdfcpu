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

package test

import (
	"sync"
	"testing"

	"github.com/umats/pdfcpu/pkg/api"
	"github.com/umats/pdfcpu/pkg/pdfcpu/model"
)

// Both tests mutate process-wide ConfigPath; keep them out of the parallel test phase.
// TestDisableConfigDir verifies disable config dir.
func TestDisableConfigDir(t *testing.T) {
	api.DisableConfigDir()

	if model.ConfigPath != "disable" {
		t.Errorf("model.ConfigPath != \"disable\" (%s)", model.ConfigPath)
	}
}

// TestDisableConfigDir_Parallel verifies disable config dir parallel.
func TestDisableConfigDir_Parallel(t *testing.T) {
	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			api.DisableConfigDir()
		}()
	}
	wg.Wait()
	t.Log("DisableConfigDir passed")
}
