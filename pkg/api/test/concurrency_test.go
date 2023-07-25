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

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func TestDisableConfigDir(t *testing.T) {
	t.Parallel()
	api.DisableConfigDir()

	if model.ConfigPath != "disable" {
		t.Errorf("model.ConfigPath != \"disable\" (%s)", model.ConfigPath)
	}
}

func TestDisableConfigDir_Parallel(t *testing.T) {
	t.Parallel()

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
