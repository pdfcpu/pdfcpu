package model_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/stretchr/testify/require"
)

func TestEnsureDefaultConfigAt(t *testing.T) {
	t.Run("Config is being created if missing", func(t *testing.T) {
		tmpDir := t.TempDir()

		err := model.EnsureDefaultConfigAt(tmpDir, false)

		require.NoError(t, err)
		configFile := filepath.Join(tmpDir, "pdfcpu", "config.yml")
		_, err = os.Stat(configFile)
		require.NoError(t, err)
	})
}
