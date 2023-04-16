package api

import (
	"sync"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func TestDisableConfigDir(t *testing.T) {
	t.Parallel()
	DisableConfigDir()

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
			DisableConfigDir()
		}()
	}
	wg.Wait()
	t.Log("DisableConfigDir is passed")
}
