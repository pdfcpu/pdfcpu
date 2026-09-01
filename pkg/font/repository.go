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
	"fmt"
	"math"
	"path/filepath"
	"sync"

	"github.com/pdfcpu/pdfcpu/internal/corefont/metrics"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// Repository provides immutable, lazily loaded font metrics for one user-font directory.
type Repository struct {
	dir     string
	once    sync.Once
	metrics map[string]TTFLight
	err     error
}

var userFontRepositories sync.Map

func repositoryKey(dir string) string {
	if dir == "" {
		return ""
	}
	return filepath.Clean(dir)
}

// RepositoryForDir returns the shared repository for dir.
// An empty directory selects core fonts only and performs no filesystem access.
func RepositoryForDir(dir string) *Repository {
	dir = repositoryKey(dir)
	repo, _ := userFontRepositories.LoadOrStore(dir, &Repository{dir: dir})
	return repo.(*Repository)
}

func invalidateRepository(dir string) {
	userFontRepositories.Delete(repositoryKey(dir))
}

func (r *Repository) load() error {
	r.once.Do(func() {
		r.metrics, r.err = loadUserFontMetrics(r.dir)
	})
	return r.err
}

func (r *Repository) userFont(fontName string) (TTFLight, bool, error) {
	if err := r.load(); err != nil {
		return TTFLight{}, false, err
	}
	ttf, ok := r.metrics[fontName]
	return ttf, ok, nil
}

// Read reads embedded font bytes from this repository.
func (r *Repository) Read(fontName string) ([]byte, error) {
	if r.dir == "" {
		return nil, fmt.Errorf("font %s: %w", fontName, ErrUnknownFont)
	}
	return readInstalledFont(r.dir, fontName)
}

// Subset creates a subset font file from the font bytes in this repository.
func (r *Repository) Subset(fontName string, usedGIDs map[uint16]bool) ([]byte, error) {
	return subsetWithReader(fontName, usedGIDs, r.Read)
}

// BoundingBox returns the font bounding box for fontName.
func (r *Repository) BoundingBox(fontName string) (*types.Rectangle, error) {
	if IsCoreFont(fontName) {
		return metrics.CoreFontMetrics[fontName].FBox, nil
	}
	ttf, ok, err := r.userFont(fontName)
	if err != nil {
		return nil, fmt.Errorf("font %s: load metrics: %w", fontName, err)
	}
	if !ok {
		return nil, fmt.Errorf("font %s: metrics not found: %w", fontName, ErrUnknownFont)
	}
	return types.NewRectangle(ttf.LLx, ttf.LLy, ttf.URx, ttf.URy), nil
}

func (r *Repository) charWidth(fontName string, ch rune) (int, error) {
	if IsCoreFont(fontName) {
		return metrics.CoreFontCharWidth(fontName, int(ch)), nil
	}
	ttf, ok, err := r.userFont(fontName)
	if err != nil {
		return 0, fmt.Errorf("font %s: load metrics: %w", fontName, err)
	}
	if !ok {
		return 0, fmt.Errorf("font %s: metrics not found: %w", fontName, ErrUnknownFont)
	}
	pos, ok := ttf.Chars[uint32(ch)]
	if !ok {
		pos = 0
	}
	if int(pos) >= len(ttf.GlyphWidths) {
		return 0, fmt.Errorf("font %s: character U+%04X maps to glyph ID %d outside width table", fontName, ch, pos)
	}
	return ttf.GlyphWidths[pos], nil
}

func (r *Repository) glyphSpaceWidth(text, fontName string) (int, error) {
	var width int
	if IsCoreFont(fontName) {
		for i := 0; i < len(text); i++ {
			charWidth, err := r.charWidth(fontName, rune(text[i]))
			if err != nil {
				return 0, err
			}
			width += charWidth
		}
		return width, nil
	}
	for _, ch := range text {
		charWidth, err := r.charWidth(fontName, ch)
		if err != nil {
			return 0, err
		}
		width += charWidth
	}
	return width, nil
}

// TextWidth returns the width of text in user-space units.
func (r *Repository) TextWidth(text, fontName string, fontSize float64) (float64, error) {
	width, err := r.glyphSpaceWidth(text, fontName)
	if err != nil {
		return 0, err
	}
	return UserSpaceUnitsFloat(float64(width), fontSize), nil
}

// TextBoundingBox returns the bounding box for text positioned at the origin.
func (r *Repository) TextBoundingBox(text, fontName string, fontSize float64) (*types.Rectangle, error) {
	width, err := r.TextWidth(text, fontName, fontSize)
	if err != nil {
		return nil, fmt.Errorf("font %s: text width: %w", fontName, err)
	}
	fbb, err := r.BoundingBox(fontName)
	if err != nil {
		return nil, fmt.Errorf("font %s: bounding box: %w", fontName, err)
	}
	height := UserSpaceUnitsFloat(fbb.Height(), fontSize)
	y := -math.Ceil(UserSpaceUnitsFloat(-fbb.LL.Y, fontSize))
	return types.NewRectangle(0, y, width, y+height), nil
}

// Size returns the font size needed to fit text into width.
func (r *Repository) Size(text, fontName string, width float64) (int, error) {
	glyphWidth, err := r.glyphSpaceWidth(text, fontName)
	if err != nil {
		return 0, err
	}
	return fontScalingFactor(float64(glyphWidth), width), nil
}

// IsUserFont reports whether fontName is available as a user font in this repository.
func (r *Repository) IsUserFont(fontName string) (bool, error) {
	if IsCoreFont(fontName) {
		return false, nil
	}
	_, ok, err := r.userFont(fontName)
	return ok, err
}

// UserFont returns detached metrics for fontName from this repository.
func (r *Repository) UserFont(fontName string) (TTFLight, bool, error) {
	ttf, ok, err := r.userFont(fontName)
	if err != nil || !ok {
		return TTFLight{}, ok, err
	}
	return cloneTTFLight(ttf), true, nil
}

// SupportedFont reports whether fontName is a core font or an available user font.
func (r *Repository) SupportedFont(fontName string) (bool, error) {
	if IsCoreFont(fontName) {
		return true, nil
	}
	return r.IsUserFont(fontName)
}
