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

package test

import (
	"bytes"
	"crypto/sha256"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

type configurationTreeEntry struct {
	mode    fs.FileMode
	modTime time.Time
	size    int64
	digest  [sha256.Size]byte
}

func environmentWithout(names ...string) []string {
	omit := map[string]bool{}
	for _, name := range names {
		omit[name] = true
	}
	var env []string
	for _, entry := range os.Environ() {
		name, _, _ := strings.Cut(entry, "=")
		if !omit[name] {
			env = append(env, entry)
		}
	}
	return env
}

func isolatedCommandEnvironment(configurationRoot, fallbackRoot string) []string {
	env := environmentWithout(
		"APPDATA",
		"HOME",
		"HOMEDRIVE",
		"HOMEPATH",
		"PDFCPU_CONFIG_ROOT",
		"USERPROFILE",
		"XDG_CONFIG_HOME",
	)
	if configurationRoot != "" {
		env = append(env, "PDFCPU_CONFIG_ROOT="+configurationRoot)
	}
	if fallbackRoot != "" {
		env = append(
			env,
			"APPDATA="+fallbackRoot,
			"HOME="+fallbackRoot,
			"USERPROFILE="+fallbackRoot,
			"XDG_CONFIG_HOME="+fallbackRoot,
		)
	}
	return env
}

func initializeConfigurationRoot(t *testing.T, root string) {
	t.Helper()
	if _, stderr, err := runPDFCPUWithConfig(t, root, "config", "init"); err != nil {
		t.Fatalf("initialize configuration root: %v\n%s", err, stderr)
	}
}

func replaceConfigurationSchemaVersion(t *testing.T, root string) {
	t.Helper()
	path := filepath.Join(root, "pdfcpu", "config.yml")
	bb, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	updated := bytes.Replace(bb, []byte("schemaVersion: 1"), []byte("schemaVersion: 2"), 1)
	if bytes.Equal(updated, bb) {
		t.Fatal("configuration does not contain schema version 1")
	}
	if err := os.WriteFile(path, updated, 0600); err != nil {
		t.Fatal(err)
	}
}

func runNormalCommand(t *testing.T, env []string, args ...string) ([]byte, error) {
	t.Helper()
	cmd := exec.Command(pdfcpuBin, args...)
	cmd.Env = env
	return cmd.CombinedOutput()
}

func configurationTreeSnapshot(t *testing.T, root string) map[string]configurationTreeEntry {
	t.Helper()
	entries := map[string]configurationTreeEntry{}
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		info, err := configurationSnapshotInfo(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		treeEntry := configurationTreeEntry{mode: info.Mode(), modTime: info.ModTime(), size: info.Size()}
		if info.Mode().IsRegular() {
			bb, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			treeEntry.digest = sha256.Sum256(bb)
		}
		entries[rel] = treeEntry
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return entries
}

func assertConfigurationTreeUnchanged(
	t *testing.T,
	before, after map[string]configurationTreeEntry,
) {
	t.Helper()
	if len(before) != len(after) {
		t.Fatalf("configuration tree entry count changed: %d -> %d", len(before), len(after))
	}
	paths := make([]string, 0, len(before))
	for path := range before {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		got, ok := after[path]
		if !ok {
			t.Fatalf("configuration tree entry removed: %s", path)
		}
		if want := before[path]; got != want {
			t.Fatalf("configuration tree entry changed: %s\nwant: %+v\ngot:  %+v", path, want, got)
		}
	}
}

func initializeConfigurationTree(t *testing.T, root string) {
	t.Helper()
	stdout, stderr, err := runPDFCPUWithConfig(t, root, "paper")
	if err != nil {
		t.Fatalf("initialize configuration: %v\nstdout:\n%s\nstderr:\n%s", err, stdout, stderr)
	}
}

func rewriteCommandConfiguration(t *testing.T, root string, old, replacement []byte) {
	t.Helper()
	path := filepath.Join(root, "pdfcpu", "config.yml")
	bb, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	bb = bytes.ReplaceAll(bb, []byte("\r\n"), []byte("\n"))
	updated := bytes.Replace(bb, old, replacement, 1)
	if bytes.Equal(updated, bb) {
		t.Fatalf("configuration does not contain %q", old)
	}
	if err := os.WriteFile(path, updated, 0600); err != nil {
		t.Fatal(err)
	}
}

func assertCommandOutputContains(t *testing.T, out []byte, values ...string) {
	t.Helper()
	for _, value := range values {
		if !strings.Contains(string(out), value) {
			t.Fatalf("command output does not contain %q:\n%s", value, out)
		}
	}
}

func runConfigReset(t *testing.T, root, input string, args ...string) ([]byte, []byte, error) {
	t.Helper()
	commandArgs := []string{"--conf", root, "config", "reset"}
	commandArgs = append(commandArgs, args...)
	cmd := exec.Command(pdfcpuBin, commandArgs...)
	cmd.Env = os.Environ()
	cmd.Stdin = strings.NewReader(input)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	return stdout.Bytes(), stderr.Bytes(), err
}

// configurationSnapshotInfo reads metadata from an open handle to avoid stale Windows directory enumeration metadata.
func configurationSnapshotInfo(path string) (fs.FileInfo, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return f.Stat()
}
