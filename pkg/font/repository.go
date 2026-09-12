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
	"fmt"
	"math"
	"path/filepath"
	"sync"

	"github.com/pdfcpu/pdfcpu/internal/contextutil"
	"github.com/pdfcpu/pdfcpu/internal/corefont/metrics"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// Repository provides immutable, lazily loaded font metrics for one user-font directory.
type Repository struct {
	dir     string
	mu      sync.Mutex
	loaded  bool
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

func (r *Repository) load(c context.Context) error {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if err := c.Err(); err != nil {
		return err
	}
	if r.loaded {
		return r.err
	}
	metrics, err := loadUserFontMetrics(r.dir, c.Err)
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}
	r.metrics = metrics
	r.err = err
	r.loaded = true
	return r.err
}

func (r *Repository) userFont(c context.Context, fontName string) (TTFLight, bool, error) {
	if err := r.load(c); err != nil {
		return TTFLight{}, false, err
	}
	if err := c.Err(); err != nil {
		return TTFLight{}, false, err
	}
	ttf, ok := r.metrics[fontName]
	return ttf, ok, nil
}

// Read reads embedded font bytes from this repository and supports cancellation.
func (r *Repository) Read(c context.Context, fontName string) ([]byte, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if r.dir == "" {
		return nil, fmt.Errorf("font %s: %w", fontName, ErrUnknownFont)
	}
	return readInstalledFont(c, r.dir, fontName)
}

// Subset creates a subset font file from this repository and supports cancellation.
func (r *Repository) Subset(c context.Context, fontName string, usedGIDs map[uint16]bool) ([]byte, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	readFont := func(fontName string) ([]byte, error) {
		return r.Read(c, fontName)
	}
	return subsetWithReader(fontName, usedGIDs, readFont, c.Err)
}

// BoundingBox returns the font bounding box for fontName and supports cancellation.
func (r *Repository) BoundingBox(c context.Context, fontName string) (*types.Rectangle, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	if IsCoreFont(fontName) {
		return metrics.CoreFontMetrics[fontName].FBox, nil
	}
	ttf, ok, err := r.userFont(c, fontName)
	if err != nil {
		return nil, fmt.Errorf("font %s: load metrics: %w", fontName, err)
	}
	if !ok {
		return nil, fmt.Errorf("font %s: metrics not found: %w", fontName, ErrUnknownFont)
	}
	return types.NewRectangle(ttf.LLx, ttf.LLy, ttf.URx, ttf.URy), nil
}

func (r *Repository) charWidth(c context.Context, fontName string, ch rune) (int, error) {
	if err := contextutil.Check(c); err != nil {
		return 0, err
	}
	if IsCoreFont(fontName) {
		return metrics.CoreFontCharWidth(fontName, int(ch)), nil
	}
	ttf, ok, err := r.userFont(c, fontName)
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

func (r *Repository) glyphSpaceWidth(c context.Context, text, fontName string) (int, error) {
	if err := contextutil.Check(c); err != nil {
		return 0, err
	}
	var width int
	if IsCoreFont(fontName) {
		for i := 0; i < len(text); i++ {
			charWidth, err := r.charWidth(c, fontName, rune(text[i]))
			if err != nil {
				return 0, err
			}
			width += charWidth
		}
		return width, nil
	}
	for _, ch := range text {
		charWidth, err := r.charWidth(c, fontName, ch)
		if err != nil {
			return 0, err
		}
		width += charWidth
	}
	return width, nil
}

// TextWidth returns the width of text in user-space units and supports cancellation.
func (r *Repository) TextWidth(c context.Context, text, fontName string, fontSize float64) (float64, error) {
	width, err := r.glyphSpaceWidth(c, text, fontName)
	if err != nil {
		return 0, err
	}
	return UserSpaceUnitsFloat(float64(width), fontSize), nil
}

// TextBoundingBox returns the bounding box for text positioned at the origin and supports cancellation.
func (r *Repository) TextBoundingBox(c context.Context, text, fontName string, fontSize float64) (*types.Rectangle, error) {
	width, err := r.TextWidth(c, text, fontName, fontSize)
	if err != nil {
		return nil, fmt.Errorf("font %s: text width: %w", fontName, err)
	}
	fbb, err := r.BoundingBox(c, fontName)
	if err != nil {
		return nil, fmt.Errorf("font %s: bounding box: %w", fontName, err)
	}
	height := UserSpaceUnitsFloat(fbb.Height(), fontSize)
	y := -math.Ceil(UserSpaceUnitsFloat(-fbb.LL.Y, fontSize))
	return types.NewRectangle(0, y, width, y+height), nil
}

// Size returns the font size needed to fit text into width and supports cancellation.
func (r *Repository) Size(c context.Context, text, fontName string, width float64) (int, error) {
	glyphWidth, err := r.glyphSpaceWidth(c, text, fontName)
	if err != nil {
		return 0, err
	}
	return fontScalingFactor(float64(glyphWidth), width), nil
}

// IsUserFont reports whether fontName is available as a user font and supports cancellation.
func (r *Repository) IsUserFont(c context.Context, fontName string) (bool, error) {
	if err := contextutil.Check(c); err != nil {
		return false, err
	}
	if IsCoreFont(fontName) {
		return false, nil
	}
	_, ok, err := r.userFont(c, fontName)
	return ok, err
}

// UserFont returns detached metrics for fontName from this repository and supports cancellation.
func (r *Repository) UserFont(c context.Context, fontName string) (TTFLight, bool, error) {
	if err := contextutil.Check(c); err != nil {
		return TTFLight{}, false, err
	}
	ttf, ok, err := r.userFont(c, fontName)
	if err != nil || !ok {
		return TTFLight{}, ok, err
	}
	ttf, err = cloneTTFLight(ttf, c.Err)
	if err != nil {
		return TTFLight{}, false, err
	}
	return ttf, true, nil
}

// GlyphIDs returns the glyph IDs for the supported runes in text without exposing repository-owned font metrics.
// It supports cancellation.
func (r *Repository) GlyphIDs(c context.Context, text, fontName string) ([]uint16, bool, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, false, err
	}
	ttf, ok, err := r.userFont(c, fontName)
	if err != nil || !ok {
		return nil, ok, err
	}
	gids := make([]uint16, 0, len(text))
	for _, ch := range text {
		if err := c.Err(); err != nil {
			return nil, false, err
		}
		if gid, ok := ttf.Chars[uint32(ch)]; ok {
			gids = append(gids, gid)
		}
	}
	return gids, true, nil
}

// SupportedFont reports whether fontName is a core or available user font and supports cancellation.
func (r *Repository) SupportedFont(c context.Context, fontName string) (bool, error) {
	if err := contextutil.Check(c); err != nil {
		return false, err
	}
	if IsCoreFont(fontName) {
		return true, nil
	}
	return r.IsUserFont(c, fontName)
}
