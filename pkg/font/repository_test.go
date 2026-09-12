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

package font

import (
	"context"
	"errors"
	"slices"
	"testing"
)

func requireRepositoryFont(t *testing.T, repo *Repository, name string, want bool) {
	t.Helper()
	got, err := repo.IsUserFont(t.Context(), name)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("font %s: got %t, want %t", name, got, want)
	}
}

func TestRepositoriesIsolateUserFontDirectories(t *testing.T) {
	dirA := t.TempDir()
	dirB := t.TempDir()
	writeUserFontMetric(t, dirA, "FontA")
	writeUserFontMetric(t, dirB, "FontB")

	repoA := RepositoryForDir(dirA)
	repoB := RepositoryForDir(dirB)
	if repoA == repoB {
		t.Fatal("different font directories share one repository")
	}
	if RepositoryForDir(dirA) != repoA {
		t.Fatal("same font directory did not reuse its repository")
	}

	requireRepositoryFont(t, repoA, "FontA", true)
	requireRepositoryFont(t, repoA, "FontB", false)
	requireRepositoryFont(t, repoB, "FontA", false)
	requireRepositoryFont(t, repoB, "FontB", true)
}

func TestStatelessRepositoryUsesCoreFontsOnly(t *testing.T) {
	repo := RepositoryForDir("")
	supported, err := repo.SupportedFont(t.Context(), "Helvetica")
	if err != nil || !supported {
		t.Fatalf("core font: supported=%t, err=%v", supported, err)
	}
	requireRepositoryFont(t, repo, "CachedElsewhere", false)
	if _, err := repo.Read(t.Context(), "CachedElsewhere"); !errors.Is(err, ErrUnknownFont) {
		t.Fatalf("read stateless user font: got %v, want %v", err, ErrUnknownFont)
	}
	bb, err := repo.TextBoundingBox(t.Context(), "text", "Helvetica", 12)
	if err != nil {
		t.Fatal(err)
	}
	if bb.Width() <= 0 || bb.Height() <= 0 {
		t.Fatalf("invalid core-font text bounding box: %s", bb)
	}
	if _, err := repo.TextBoundingBox(t.Context(), "text", "CachedElsewhere", 12); !errors.Is(err, ErrUnknownFont) {
		t.Fatalf("stateless user-font text bounding box: got %v, want %v", err, ErrUnknownFont)
	}
}

func TestRepositoryReturnsDetachedMetrics(t *testing.T) {
	dir := t.TempDir()
	writeUserFontMetric(t, dir, "Detached")
	repo := RepositoryForDir(dir)

	got, ok, err := repo.UserFont(t.Context(), "Detached")
	if err != nil || !ok {
		t.Fatalf("load detached font: ok=%t err=%v", ok, err)
	}
	got.GlyphWidths[0] = 999

	got, ok, err = repo.UserFont(t.Context(), "Detached")
	if err != nil || !ok {
		t.Fatalf("reload detached font: ok=%t err=%v", ok, err)
	}
	if got.GlyphWidths[0] != 500 {
		t.Fatalf("repository metrics mutated through returned value: %+v", got)
	}
}

func TestRepositoryGlyphIDs(t *testing.T) {
	dir := t.TempDir()
	writeUserFontMetric(t, dir, "Glyphs")
	repo := RepositoryForDir(dir)

	gids, ok, err := repo.GlyphIDs(t.Context(), "ABA", "Glyphs")
	if err != nil || !ok {
		t.Fatalf("glyph IDs: ok=%t err=%v", ok, err)
	}
	if want := []uint16{0, 0}; !slices.Equal(gids, want) {
		t.Fatalf("got %v, want %v", gids, want)
	}
}

func TestRepositoryRetriesCanceledInitialLoad(t *testing.T) {
	dir := t.TempDir()
	writeUserFontMetric(t, dir, "Retry")
	repo := &Repository{dir: dir}
	c, cancel := context.WithCancel(t.Context())
	cancel()
	if _, _, err := repo.UserFont(c, "Retry"); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled load: got %v, want context.Canceled", err)
	}
	_, ok, err := repo.UserFont(t.Context(), "Retry")
	if err != nil || !ok {
		t.Fatalf("retry load: ok=%t err=%v", ok, err)
	}
}

func TestReloadUserFontsInvalidatesDirectoryRepository(t *testing.T) {
	originalDir := UserFontDir
	t.Cleanup(func() {
		UserFontDir = originalDir
		if err := ReloadUserFonts(context.WithoutCancel(t.Context())); err != nil {
			t.Errorf("restore user fonts: %v", err)
		}
	})

	UserFontDir = t.TempDir()
	writeUserFontMetric(t, UserFontDir, "Reloaded")
	before := RepositoryForDir(UserFontDir)
	if err := ReloadUserFonts(t.Context()); err != nil {
		t.Fatal(err)
	}
	after := RepositoryForDir(UserFontDir)
	if after == before {
		t.Fatal("font reload retained stale directory repository")
	}
	requireRepositoryFont(t, after, "Reloaded", true)
}
