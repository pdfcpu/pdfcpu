/*
	Copyright 2020 The pdfcpu Authors.

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

package api

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/pdfcpu/pdfcpu/internal/contextutil"
	"github.com/pdfcpu/pdfcpu/internal/fileutil"
	"github.com/pdfcpu/pdfcpu/pkg/font"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/color"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/draw"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/fault"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// FontInstallResult reports non-fatal cleanup failures encountered while installing fonts.
type FontInstallResult struct {
	Warnings []error
}

// FontCheatSheetResult identifies cheat-sheet PDFs that were published before the operation returned.
type FontCheatSheetResult struct {
	Paths []string
}

type fontAPIOperations struct {
	userFontDir               string
	reloadUserFonts           func(context.Context) error
	installTrueTypeFont       func(string, string) (font.InstallResult, error)
	installTrueTypeCollection func(string, string) ([]font.InstallResult, error)
	createStagingDir          func(string) (string, error)
	createInputDir            func(string, int) (string, error)
	commitStagedFonts         func(string, string) (fontInstallCommit, error)
	removeAll                 func(string) error
	rename                    func(string, string) error
}

type fontInstallCommit struct {
	rollback func() error
	finalize func() error
}

type committedFontFile struct {
	name        string
	hadOriginal bool
	committed   bool
}

type transactionFileOperations struct {
	mkdirTemp func(string, string) (string, error)
	lstat     func(string) (os.FileInfo, error)
	syncDir   func(string) error
	rename    func(string, string) error
	remove    func(string) error
	removeAll func(string) error
}

type fontInstallFileOperations = transactionFileOperations
type cheatSheetFileOperations = transactionFileOperations

func defaultFontInstallFileOperations() fontInstallFileOperations {
	return fontInstallFileOperations{
		mkdirTemp: os.MkdirTemp,
		lstat:     os.Lstat,
		syncDir:   fileutil.SyncDirectory,
		rename:    fileutil.ReplaceFile,
		remove:    fileutil.RemoveFile,
		removeAll: os.RemoveAll,
	}
}

type fontCheatSheetOperations struct {
	loadUserFonts func(context.Context) error
	userFont      func(context.Context, string) (font.TTFLight, bool, error)
	userFontNames func(context.Context) ([]string, error)
	createXRef    func() (*model.XRefTable, error)
	createPage    func(context.Context, *model.XRefTable, int, int, int, string) (model.Page, error)
	catalog       func(*model.XRefTable) (types.Dict, error)
	addPageTree   func(context.Context, *model.XRefTable, types.Dict, model.Page) error
	createPDFFile func(context.Context, *model.XRefTable, string, *model.Configuration) error
	files         cheatSheetFileOperations
}

type committedCheatSheet struct {
	name        string
	hadOriginal bool
	published   bool
}

func defaultCheatSheetFileOperations() cheatSheetFileOperations {
	return cheatSheetFileOperations{
		mkdirTemp: os.MkdirTemp,
		lstat:     os.Lstat,
		syncDir:   fileutil.SyncDirectory,
		rename:    fileutil.ReplaceFile,
		remove:    fileutil.RemoveFile,
		removeAll: os.RemoveAll,
	}
}

func defaultFontAPIOperations() fontAPIOperations {
	return defaultFontAPIOperationsWithResult(nil)
}

func defaultFontAPIOperationsWithResult(result *FontInstallResult) fontAPIOperations {
	ops := fontAPIOperations{
		userFontDir:     font.UserFontDir,
		reloadUserFonts: font.ReloadUserFonts,
		createStagingDir: func(fontDir string) (string, error) {
			return os.MkdirTemp(fontDir, ".pdfcpu-font-install-")
		},
		createInputDir: func(stagingDir string, input int) (string, error) {
			return os.MkdirTemp(stagingDir, fmt.Sprintf(".input-%d-", input))
		},
		commitStagedFonts: commitStagedFonts,
		removeAll:         os.RemoveAll,
		rename:            fileutil.ReplaceFile,
	}
	ops.installTrueTypeFont = func(fontDir, fileName string) (font.InstallResult, error) {
		report, err := font.InstallTrueTypeFontResult(fontDir, fileName)
		for _, warning := range report.Warnings {
			addFontInstallWarning(result, warning)
		}
		if err != nil {
			return font.InstallResult{}, err
		}
		if len(report.Fonts) != 1 {
			return font.InstallResult{}, fmt.Errorf("font %s: expected one installed font, got %d", fileName, len(report.Fonts))
		}
		return report.Fonts[0], nil
	}
	ops.installTrueTypeCollection = func(fontDir, fileName string) ([]font.InstallResult, error) {
		report, err := font.InstallTrueTypeCollectionResults(fontDir, fileName)
		for _, warning := range report.Warnings {
			addFontInstallWarning(result, warning)
		}
		return report.Fonts, err
	}
	return ops
}

func addFontInstallWarning(result *FontInstallResult, err error) {
	if result != nil && err != nil {
		result.Warnings = append(result.Warnings, err)
	}
}

func defaultFontCheatSheetOperations() fontCheatSheetOperations {
	return fontCheatSheetOperations{
		loadUserFonts: font.LoadUserFonts,
		userFont:      font.UserFont,
		userFontNames: font.UserFontNames,
		createXRef:    createFontXRefTable,
		createPage:    createUserFontCheatSheetPage,
		catalog:       func(xRefTable *model.XRefTable) (types.Dict, error) { return xRefTable.Catalog() },
		addPageTree:   addFontPageTree,
		createPDFFile: CreatePDFFile,
		files:         defaultCheatSheetFileOperations(),
	}
}

func loadCheatSheetUserFonts(c context.Context, ops fontCheatSheetOperations) error {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	if err := ops.loadUserFonts(c); err != nil {
		return err
	}
	return contextutil.Check(c)
}

func cheatSheetUserFont(c context.Context, fontName string, ops fontCheatSheetOperations) (font.TTFLight, bool, error) {
	if err := contextutil.Check(c); err != nil {
		return font.TTFLight{}, false, err
	}
	ttf, ok, err := ops.userFont(c, fontName)
	if err != nil {
		return font.TTFLight{}, false, err
	}
	if err := contextutil.Check(c); err != nil {
		return font.TTFLight{}, false, err
	}
	return ttf, ok, nil
}

func cheatSheetUserFontNames(c context.Context, ops fontCheatSheetOperations) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	names, err := ops.userFontNames(c)
	if err != nil {
		return nil, err
	}
	return names, contextutil.Check(c)
}

func createCheatSheetPage(c context.Context, xRefTable *model.XRefTable, w, h, plane int, fontName string, ops fontCheatSheetOperations) (model.Page, error) {
	if err := contextutil.Check(c); err != nil {
		return model.Page{}, err
	}
	p, err := ops.createPage(c, xRefTable, w, h, plane, fontName)
	if err != nil {
		return model.Page{}, err
	}
	return p, contextutil.Check(c)
}

func createCheatSheetPDF(c context.Context, xRefTable *model.XRefTable, fileName string, ops fontCheatSheetOperations) error {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	if err := ops.createPDFFile(c, xRefTable, fileName, nil); err != nil {
		return err
	}
	return contextutil.Check(c)
}

// ListFonts returns a list of supported fonts and supports cancellation.
func ListFonts(c context.Context) ([]string, error) {
	return listFontsUsing(c, font.UserFontNamesVerbose)
}

func listFontsUsing(c context.Context, userFontNames func(context.Context) ([]string, error)) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	// Get list of PDF core fonts.
	coreFonts := font.CoreFontNames()
	for i, s := range coreFonts {
		if err := contextutil.Check(c); err != nil {
			return nil, err
		}
		coreFonts[i] = "  " + s
	}
	sort.Strings(coreFonts)
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}

	sscf := []string{"Corefonts:"}
	sscf = append(sscf, coreFonts...)

	// Get installed fonts from pdfcpu config dir in users home dir
	userFonts, err := userFontNames(c)
	if err != nil {
		return nil, fmt.Errorf("list fonts: load user fonts: %w", err)
	}
	for i, s := range userFonts {
		if err := contextutil.Check(c); err != nil {
			return nil, err
		}
		userFonts[i] = "  " + s
	}
	sort.Strings(userFonts)
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}

	ssuf := []string{fmt.Sprintf("Userfonts(%s):", font.UserFontDir)}
	ssuf = append(ssuf, userFonts...)

	sscf = append(sscf, "")
	result := append(sscf, ssuf...)
	return result, contextutil.Check(c)
}

func validateFontFiles(fileNames []string) error {
	for i, fn := range fileNames {
		if strings.TrimSpace(fn) == "" {
			return fmt.Errorf("install fonts: input %d: %w", i+1, ErrMissingFontInput)
		}
		switch strings.ToLower(filepath.Ext(fn)) {
		case ".ttf", ".ttc":
		default:
			return fmt.Errorf("install fonts: input %d %s: %w", i+1, fn, ErrUnsupportedFontFile)
		}
	}
	return nil
}

// InstallFontsWithResult transactionally installs true type fonts, supports cancellation and reports non-fatal
// cleanup warnings. Cancellation before publication leaves the installed font directory unchanged.
func InstallFontsWithResult(c context.Context, fileNames []string) (result FontInstallResult, err error) {
	ops := defaultFontAPIOperationsWithResult(&result)
	err = installFontsUsing(c, fileNames, ops, &result)
	return result, err
}

// InstallFonts transactionally installs true type fonts and supports cancellation.
func InstallFonts(c context.Context, fileNames []string) error {
	_, err := InstallFontsWithResult(c, fileNames)
	return err
}

func stagedFontFiles(stagingDir string) ([]string, error) {
	entries, err := os.ReadDir(stagingDir)
	if err != nil {
		return nil, fmt.Errorf("read staging directory: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || strings.ToLower(filepath.Ext(entry.Name())) != ".gob" {
			return nil, fmt.Errorf("unexpected staged font entry %s", entry.Name())
		}
		names = append(names, entry.Name())
	}
	sort.Strings(names)
	return names, nil
}

func syncTransactionDirectories(fs transactionFileOperations, dirs ...string) error {
	seen := map[string]bool{}
	var err error
	for _, dir := range dirs {
		if seen[dir] {
			continue
		}
		seen[dir] = true
		if syncErr := fs.syncDir(dir); syncErr != nil {
			err = errors.Join(err, fmt.Errorf("sync directory %s: %w", dir, syncErr))
		}
	}
	return err
}

func rollbackCommittedFonts(fontDir, backupDir string, files []committedFontFile, fs fontInstallFileOperations) error {
	var err error
	for i := len(files) - 1; i >= 0; i-- {
		file := files[i]
		target := filepath.Join(fontDir, file.name)
		if file.committed {
			if removeErr := fs.remove(target); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
				err = errors.Join(err, fmt.Errorf("remove committed font %s: %w", file.name, removeErr))
			}
		}
		if file.hadOriginal {
			if renameErr := fs.rename(filepath.Join(backupDir, file.name), target); renameErr != nil {
				err = errors.Join(err, fmt.Errorf("restore font %s: %w", file.name, renameErr))
			}
		}
	}
	err = errors.Join(err, syncTransactionDirectories(fs, fontDir, backupDir))
	if err != nil {
		return errors.Join(err, fmt.Errorf("font backup retained at %s", backupDir))
	}
	if removeErr := fs.removeAll(backupDir); removeErr != nil {
		err = errors.Join(err, fmt.Errorf("remove font backup: %w", removeErr))
	} else {
		err = errors.Join(err, syncTransactionDirectories(fs, fontDir))
	}
	return err
}

func commitStagedFonts(fontDir, stagingDir string) (fontInstallCommit, error) {
	return commitStagedFontsWithOperations(fontDir, stagingDir, defaultFontInstallFileOperations())
}

func commitStagedFontsWithOperations(fontDir, stagingDir string, fs fontInstallFileOperations) (fontInstallCommit, error) {
	names, err := stagedFontFiles(stagingDir)
	if err != nil {
		return fontInstallCommit{}, err
	}
	backupDir, err := fs.mkdirTemp(fontDir, ".pdfcpu-font-backup-")
	if err != nil {
		return fontInstallCommit{}, fmt.Errorf("create backup directory: %w", err)
	}
	files := make([]committedFontFile, 0, len(names))
	rollback := func() error { return rollbackCommittedFonts(fontDir, backupDir, files, fs) }
	for _, name := range names {
		files = append(files, committedFontFile{name: name})
		file := &files[len(files)-1]
		target := filepath.Join(fontDir, name)
		if _, err := fs.lstat(target); err == nil {
			if err := fileutil.PreserveGroup(filepath.Join(stagingDir, name), target); err != nil {
				return fontInstallCommit{}, errors.Join(err, rollback())
			}
			if err := fs.rename(target, filepath.Join(backupDir, name)); err != nil {
				return fontInstallCommit{}, errors.Join(fmt.Errorf("backup font %s: %w", name, err), rollback())
			}
			file.hadOriginal = true
			if err := syncTransactionDirectories(fs, fontDir, backupDir); err != nil {
				return fontInstallCommit{}, errors.Join(fmt.Errorf("backup font %s: %w", name, err), rollback())
			}
		} else if !errors.Is(err, os.ErrNotExist) {
			return fontInstallCommit{}, errors.Join(fmt.Errorf("inspect font %s: %w", name, err), rollback())
		}
		if err := fs.rename(filepath.Join(stagingDir, name), target); err != nil {
			return fontInstallCommit{}, errors.Join(fmt.Errorf("commit font %s: %w", name, err), rollback())
		}
		file.committed = true
		if err := syncTransactionDirectories(fs, stagingDir, fontDir); err != nil {
			return fontInstallCommit{}, errors.Join(fmt.Errorf("commit font %s: %w", name, err), rollback())
		}
	}
	return fontInstallCommit{
		rollback: rollback,
		finalize: func() error {
			if err := fs.removeAll(backupDir); err != nil {
				return fmt.Errorf("remove font backup: %w", err)
			}
			if err := syncTransactionDirectories(fs, fontDir); err != nil {
				return fmt.Errorf("sync after removing font backup: %w", err)
			}
			return nil
		},
	}, nil
}

func stagedFontOrigin(input int, fileName string, result font.InstallResult) string {
	origin := fmt.Sprintf("input %d %s", input, fileName)
	if result.Member > 0 {
		origin += fmt.Sprintf(" member %d", result.Member)
	}
	return origin
}

func mergeStagedFont(inputDir, stagingDir string, result font.InstallResult, origin string, owners map[string]string, ops fontAPIOperations) error {
	name := result.PostScriptName
	if strings.TrimSpace(name) == "" {
		return fmt.Errorf("%s: missing PostScript output name", origin)
	}
	if firstOrigin, found := owners[name]; found {
		return fmt.Errorf("%w %s: %s conflicts with %s", font.ErrDuplicatePostScriptName, name, origin, firstOrigin)
	}
	source := filepath.Join(inputDir, name+".gob")
	target := filepath.Join(stagingDir, name+".gob")
	if err := ops.rename(source, target); err != nil {
		return fmt.Errorf("%s: stage PostScript name %s: %w", origin, name, err)
	}
	owners[name] = origin
	return nil
}

func installFontInput(stagingDir, fileName string, input int, ops fontAPIOperations) ([]font.InstallResult, string, error) {
	inputDir, err := ops.createInputDir(stagingDir, input)
	if err != nil {
		return nil, "", fmt.Errorf("create input staging directory: %w", err)
	}
	switch strings.ToLower(filepath.Ext(fileName)) {
	case ".ttf":
		result, err := ops.installTrueTypeFont(inputDir, fileName)
		if err != nil {
			return nil, "", err
		}
		return []font.InstallResult{result}, inputDir, nil
	case ".ttc":
		results, err := ops.installTrueTypeCollection(inputDir, fileName)
		return results, inputDir, err
	}
	return nil, inputDir, nil
}

func installFontInputs(c context.Context, fileNames []string, stagingDir string, ops fontAPIOperations) error {
	owners := map[string]string{}
	for i, fn := range fileNames {
		if err := contextutil.Check(c); err != nil {
			return err
		}
		input := i + 1
		results, inputDir, err := installFontInput(stagingDir, fn, input, ops)
		if err != nil {
			return fmt.Errorf("install fonts: input %d: %w", input, err)
		}
		for _, result := range results {
			if err := contextutil.Check(c); err != nil {
				return err
			}
			origin := stagedFontOrigin(input, fn, result)
			if err := mergeStagedFont(inputDir, stagingDir, result, origin, owners, ops); err != nil {
				return fmt.Errorf("install fonts: %w", err)
			}
		}
		if err := ops.removeAll(inputDir); err != nil {
			return fmt.Errorf("install fonts: input %d: remove input staging directory: %w", input, err)
		}
	}
	return nil
}

func reloadInstalledFonts(c context.Context, ops fontAPIOperations) error {
	if err := ops.reloadUserFonts(c); err != nil {
		return fmt.Errorf("install fonts: reload user fonts: %w", err)
	}
	return contextutil.Check(c)
}

func rollbackFontInstall(commit fontInstallCommit, cause error) error {
	rollbackErr := commit.rollback()
	if rollbackErr != nil {
		rollbackErr = fmt.Errorf("install fonts: rollback batch: %w", rollbackErr)
	}
	return errors.Join(cause, rollbackErr)
}

func installFontsUsing(c context.Context, fileNames []string, ops fontAPIOperations, result *FontInstallResult) (err error) {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	if len(fileNames) == 0 {
		return reloadInstalledFonts(c, ops)
	}
	if err := validateFontFiles(fileNames); err != nil {
		return err
	}
	if ops.userFontDir == "" {
		return fmt.Errorf("install fonts: user font directory: %w", ErrMissingConfiguration)
	}

	stagingDir, err := ops.createStagingDir(ops.userFontDir)
	if err != nil {
		return fmt.Errorf("install fonts: create staging directory: %w", err)
	}
	installed := false
	defer func() {
		if cleanupErr := ops.removeAll(stagingDir); cleanupErr != nil {
			cleanupErr = fmt.Errorf("install fonts: remove staging directory: %w", cleanupErr)
			if installed {
				addFontInstallWarning(result, cleanupErr)
			} else {
				err = errors.Join(err, cleanupErr)
			}
		}
	}()
	if err := installFontInputs(c, fileNames, stagingDir, ops); err != nil {
		return err
	}
	if err := contextutil.Check(c); err != nil {
		return err
	}
	commit, err := ops.commitStagedFonts(ops.userFontDir, stagingDir)
	if err != nil {
		return fmt.Errorf("install fonts: commit batch: %w", err)
	}
	if err := contextutil.Check(c); err != nil {
		return rollbackFontInstall(commit, err)
	}
	if err := reloadInstalledFonts(c, ops); err != nil {
		return rollbackFontInstall(commit, err)
	}
	installed = true
	if err := commit.finalize(); err != nil {
		addFontInstallWarning(result, fmt.Errorf("install fonts: finalize batch: %w", err))
	}
	return nil
}

func rowLabel(c context.Context, xRefTable *model.XRefTable, i int, td model.TextDescriptor, baseFontName, baseFontKey string, buf *bytes.Buffer, mb *types.Rectangle, left bool) error {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	x := 39.
	if !left {
		x = 7750
	}
	s := fmt.Sprintf("#%02X", i)
	td.X, td.Y, td.Text = x, float64(7677-i*30), s
	td.StrokeCol, td.FillCol = color.Black, color.SimpleColor{B: .8}
	td.FontName, td.FontKey, td.FontSize = baseFontName, baseFontKey, 14

	if _, err := model.WriteMultiLine(c, xRefTable, buf, mb, nil, td); err != nil {
		return fmt.Errorf("render row label %d: %w", i, err)
	}
	return nil
}

func columnsLabel(c context.Context, xRefTable *model.XRefTable, td model.TextDescriptor, baseFontName, baseFontKey string, buf *bytes.Buffer, mb *types.Rectangle, top bool) error {
	y := 7700.
	if !top {
		y = 0
	}

	td.FontName, td.FontKey = baseFontName, baseFontKey

	for i := range 256 {
		if err := contextutil.Check(c); err != nil {
			return err
		}
		s := fmt.Sprintf("#%02X", i)
		td.X, td.Y, td.Text, td.FontSize = float64(70+i*30), y, s, 14
		td.StrokeCol, td.FillCol = color.Black, color.SimpleColor{B: .8}
		if _, err := model.WriteMultiLine(c, xRefTable, buf, mb, nil, td); err != nil {
			return fmt.Errorf("render column label %d: %w", i, err)
		}
	}
	return nil
}

func surrogate(r rune) bool {
	return r >= 0xD800 && r <= 0xDFFF
}

func writeUserFontCheatSheetContent(c context.Context, xRefTable *model.XRefTable, p model.Page, fontName string, plane int) error {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	baseFontName := "Helvetica"
	baseFontSize := 24
	baseFontKey := p.Fm.EnsureKey(baseFontName)

	fontKey := p.Fm.EnsureKey(fontName)
	fontSize := 24

	fillCol := color.NewSimpleColor(0xf7e6c7)
	draw.DrawGrid(p.Buf, 16*16, 16*16, types.RectForWidthAndHeight(55, 16, 16*480, 16*480), color.Black, &fillCol)
	if err := contextutil.Check(c); err != nil {
		return err
	}

	td := model.TextDescriptor{
		FontName:       fontName,
		Embed:          true,
		FontKey:        fontKey,
		FontSize:       float64(baseFontSize),
		HAlign:         types.AlignCenter,
		VAlign:         types.AlignBaseline,
		Scale:          1.0,
		ScaleAbs:       true,
		RMode:          draw.RMFill,
		StrokeCol:      color.Black,
		FillCol:        color.NewSimpleColor(0xab6f30),
		ShowBackground: true,
		BackgroundCol:  color.SimpleColor{R: 1., G: .98, B: .77},
	}

	from := plane * 0x10000
	to := (plane+1)*0x10000 - 1
	s := fmt.Sprintf("%s %d points (%04X - %04X)", fontName, fontSize, from, to)

	td.X, td.Y, td.Text = p.MediaBox.Width()/2, 7750, s
	td.FontName, td.FontKey = baseFontName, baseFontKey
	td.StrokeCol, td.FillCol = color.NewSimpleColor(0x77bdbd), color.NewSimpleColor(0xab6f30)
	if _, err := model.WriteMultiLine(c, xRefTable, p.Buf, p.MediaBox, nil, td); err != nil {
		return fmt.Errorf("render font demo heading: %w", err)
	}

	if err := columnsLabel(c, xRefTable, td, baseFontName, baseFontKey, p.Buf, p.MediaBox, true); err != nil {
		return err
	}
	base := rune(plane * 0x10000)
	for j := range 256 {
		if err := contextutil.Check(c); err != nil {
			return err
		}
		if err := rowLabel(c, xRefTable, j, td, baseFontName, baseFontKey, p.Buf, p.MediaBox, true); err != nil {
			return err
		}
		buf := make([]byte, 4)
		td.StrokeCol, td.FillCol = color.Black, color.Black
		td.FontName, td.FontKey, td.FontSize = fontName, fontKey, float64(fontSize-2)
		for i := range 256 {
			if err := contextutil.Check(c); err != nil {
				return err
			}
			r := base + rune(j*256+i)
			s = " "
			if !surrogate(r) {
				n := utf8.EncodeRune(buf, r)
				s = string(buf[:n])
			}
			td.X, td.Y, td.Text = float64(70+i*30), float64(7672-j*30), s
			if _, err := model.WriteMultiLine(c, xRefTable, p.Buf, p.MediaBox, nil, td); err != nil {
				return fmt.Errorf("render font demo glyph U+%04X: %w", r, err)
			}
		}
		if err := rowLabel(c, xRefTable, j, td, baseFontName, baseFontKey, p.Buf, p.MediaBox, false); err != nil {
			return err
		}
	}
	if err := columnsLabel(c, xRefTable, td, baseFontName, baseFontKey, p.Buf, p.MediaBox, false); err != nil {
		return err
	}
	return nil
}

func createUserFontCheatSheetPage(c context.Context, xRefTable *model.XRefTable, w, h, plane int, fontName string) (p model.Page, err error) {
	defer fault.Catch(&err)
	if err := contextutil.Check(c); err != nil {
		return model.Page{}, err
	}
	mediaBox := types.RectForDim(float64(w), float64(h))
	p = model.NewPageWithBg(mediaBox, color.NewSimpleColor(0xbeded9))
	if err := writeUserFontCheatSheetContent(c, xRefTable, p, fontName, plane); err != nil {
		return model.Page{}, fmt.Errorf("render font demo page: %w", err)
	}
	return p, nil
}

func planeString(i int) (string, error) {
	switch i {
	case 0:
		return "BMP", nil // Basic Multilingual Plane
	case 1:
		return "SMP", nil // Supplementary Multilingual Plane
	case 2:
		return "SIP", nil // Supplementary Ideographic Plane
	case 3:
		return "TIP", nil // Tertiary Ideographic Plane
	case 4, 5, 6, 7, 8, 9, 10, 11, 12, 13:
		return fmt.Sprintf("P%02d", i), nil
	case 14:
		return "SSP", nil // Supplementary Special-purpose Plane
	case 15:
		return "SPUA", nil // Supplementary Private Use Area-A
	case 16:
		return "SPUB", nil // Supplementary Private Use Area-B
	}
	return "", fmt.Errorf("%d: %w", i, ErrInvalidUnicodePlane)
}

func coveredUnicodePlanes(m map[int]bool) ([]int, error) {
	planes := make([]int, 0, len(m))
	for plane, covered := range m {
		if !covered {
			continue
		}
		if _, err := planeString(plane); err != nil {
			return nil, err
		}
		planes = append(planes, plane)
	}
	sort.Ints(planes)
	return planes, nil
}

func rollbackCheatSheets(dir, backupDir string, files []committedCheatSheet, fs cheatSheetFileOperations) error {
	var err error
	for i := len(files) - 1; i >= 0; i-- {
		file := files[i]
		target := filepath.Join(dir, file.name)
		if file.published {
			if removeErr := fs.remove(target); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
				err = errors.Join(err, fmt.Errorf("remove published cheat sheet %s: %w", file.name, removeErr))
			}
		}
		if file.hadOriginal {
			if renameErr := fs.rename(filepath.Join(backupDir, file.name), target); renameErr != nil {
				err = errors.Join(err, fmt.Errorf("restore cheat sheet %s: %w", file.name, renameErr))
			}
		}
	}
	err = errors.Join(err, syncTransactionDirectories(fs, dir, backupDir))
	if err != nil {
		return errors.Join(err, fmt.Errorf("cheat-sheet backup retained at %s", backupDir))
	}
	if removeErr := fs.removeAll(backupDir); removeErr != nil {
		return fmt.Errorf("remove cheat-sheet backup: %w", removeErr)
	}
	return syncTransactionDirectories(fs, dir)
}

func publishCheatSheet(c context.Context, dir, stagingDir, backupDir string, file *committedCheatSheet, fs cheatSheetFileOperations) error {
	if err := contextutil.Check(c); err != nil {
		return err
	}
	target := filepath.Join(dir, file.name)
	if _, err := fs.lstat(target); err == nil {
		if err := fileutil.PreserveGroup(filepath.Join(stagingDir, file.name), target); err != nil {
			return err
		}
		if err := fs.rename(target, filepath.Join(backupDir, file.name)); err != nil {
			return fmt.Errorf("backup cheat sheet %s: %w", file.name, err)
		}
		file.hadOriginal = true
		if err := contextutil.Check(c); err != nil {
			return err
		}
		if err := syncTransactionDirectories(fs, dir, backupDir); err != nil {
			return fmt.Errorf("backup cheat sheet %s: %w", file.name, err)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("inspect cheat sheet %s: %w", file.name, err)
	}
	if err := fs.rename(filepath.Join(stagingDir, file.name), target); err != nil {
		return fmt.Errorf("publish cheat sheet %s: %w", file.name, err)
	}
	file.published = true
	if err := contextutil.Check(c); err != nil {
		return err
	}
	if err := syncTransactionDirectories(fs, stagingDir, dir); err != nil {
		return fmt.Errorf("publish cheat sheet %s: %w", file.name, err)
	}
	return nil
}

func publishCheatSheets(c context.Context, dir, stagingDir string, names []string, fs cheatSheetFileOperations) (published bool, err error) {
	if err := contextutil.Check(c); err != nil {
		return false, err
	}
	backupDir, err := fs.mkdirTemp(dir, ".pdfcpu-font-cheatsheet-backup-")
	if err != nil {
		return false, fmt.Errorf("create cheat-sheet backup: %w", err)
	}
	files := make([]committedCheatSheet, 0, len(names))
	rollback := func(publishErr error) error {
		return errors.Join(publishErr, rollbackCheatSheets(dir, backupDir, files, fs))
	}
	for _, name := range names {
		files = append(files, committedCheatSheet{name: name})
		file := &files[len(files)-1]
		if err := publishCheatSheet(c, dir, stagingDir, backupDir, file, fs); err != nil {
			return false, rollback(err)
		}
	}
	if err := contextutil.Check(c); err != nil {
		return false, rollback(err)
	}
	if err := fs.removeAll(backupDir); err != nil {
		return true, fmt.Errorf("published cheat sheets: remove backup: %w", err)
	}
	if err := syncTransactionDirectories(fs, dir); err != nil {
		return true, fmt.Errorf("published cheat sheets: sync after removing backup: %w", err)
	}
	return true, nil
}

func normalizeCheatSheetDir(dir string) string {
	if dir == "" {
		return "."
	}
	return dir
}

// CreateUserFontCheatSheetsWithResult atomically generates and publishes one PDF for each covered Unicode plane,
// supports cancellation and reports published paths.
func CreateUserFontCheatSheetsWithResult(c context.Context, dir, fn string) (result FontCheatSheetResult, err error) {
	defer fault.Catch(&err)
	return createUserFontCheatSheetsUsing(c, dir, fn, defaultFontCheatSheetOperations())
}

// CreateUserFontCheatSheets atomically generates and publishes one PDF for each covered Unicode plane and supports
// cancellation.
func CreateUserFontCheatSheets(c context.Context, dir, fn string) error {
	_, err := CreateUserFontCheatSheetsWithResult(c, dir, fn)
	return err
}

func createUserFontCheatSheetsUsing(c context.Context, dir string, fn string, ops fontCheatSheetOperations) (FontCheatSheetResult, error) {
	if err := contextutil.Check(c); err != nil {
		return FontCheatSheetResult{}, err
	}
	if err := validateNoEmptyStrings([]string{fn}, "font name"); err != nil {
		return FontCheatSheetResult{}, fmt.Errorf("create font cheat sheet: %w", err)
	}
	if err := loadCheatSheetUserFonts(c, ops); err != nil {
		return FontCheatSheetResult{}, fmt.Errorf("create font cheat sheet: load user fonts: %w", err)
	}

	ttf, ok, err := cheatSheetUserFont(c, fn, ops)
	if err != nil {
		return FontCheatSheetResult{}, fmt.Errorf("create font cheat sheet: font %s metrics: %w", fn, err)
	}
	if !ok {
		return FontCheatSheetResult{}, fmt.Errorf("create font cheat sheet: font %s: %w", fn, ErrUserFontNotFound)
	}
	paths, err := createUserFontCheatSheetBatch(
		c,
		dir,
		[]string{fn},
		map[string]font.TTFLight{fn: ttf},
		ops,
	)
	result := FontCheatSheetResult{Paths: paths}
	if err != nil {
		return result, fmt.Errorf("create font cheat sheet: %w", err)
	}
	return result, nil
}

func stageUserFontCheatSheets(c context.Context, dir, fn string, ttf font.TTFLight, ops fontCheatSheetOperations) ([]string, error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	const w, h = 7800, 7800
	planes, err := coveredUnicodePlanes(ttf.Planes)
	if err != nil {
		return nil, fmt.Errorf("font %s plane: %w", fn, err)
	}
	names := make([]string, 0, len(planes))
	// Create a single page PDF for each Unicode plane with existing glyphs.
	for _, i := range planes {
		if err := contextutil.Check(c); err != nil {
			return nil, err
		}
		context := fmt.Sprintf("font %s plane %d", fn, i)
		suffix, err := planeString(i)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", context, err)
		}
		xRefTable, err := ops.createXRef()
		if err != nil {
			return nil, fmt.Errorf("%s: create PDF context: %w", context, err)
		}
		p, err := createCheatSheetPage(c, xRefTable, w, h, i, fn, ops)
		if err != nil {
			return nil, fmt.Errorf("%s: render page: %w", context, err)
		}

		rootDict, err := ops.catalog(xRefTable)
		if err != nil {
			return nil, fmt.Errorf("%s: access catalog: %w", context, err)
		}
		if err = ops.addPageTree(c, xRefTable, rootDict, p); err != nil {
			return nil, fmt.Errorf("%s: build page tree: %w", context, err)
		}
		name := fn + "_" + suffix + ".pdf"
		fileName := filepath.Join(dir, name)
		if err := createCheatSheetPDF(c, xRefTable, fileName, ops); err != nil {
			return nil, fmt.Errorf("%s: %w", context, err)
		}
		names = append(names, name)
	}
	return names, contextutil.Check(c)
}

func createUserFontCheatSheetBatch(c context.Context, dir string, fontNames []string, fonts map[string]font.TTFLight, ops fontCheatSheetOperations) (paths []string, err error) {
	if err := contextutil.Check(c); err != nil {
		return nil, err
	}
	dir = normalizeCheatSheetDir(dir)
	stagingDir, err := ops.files.mkdirTemp(dir, ".pdfcpu-font-cheatsheets-")
	if err != nil {
		return nil, fmt.Errorf("create staging directory: %w", err)
	}
	published := false
	defer func() {
		if cleanupErr := ops.files.removeAll(stagingDir); cleanupErr != nil {
			cleanupErr = fmt.Errorf("remove cheat-sheet staging directory: %w", cleanupErr)
			if published {
				err = errors.Join(err, fmt.Errorf("published cheat sheets: %w", cleanupErr))
			} else {
				err = errors.Join(err, cleanupErr)
			}
		}
	}()
	names := []string{}
	seen := map[string]bool{}
	for _, fn := range fontNames {
		if err := contextutil.Check(c); err != nil {
			return nil, err
		}
		staged, err := stageUserFontCheatSheets(c, stagingDir, fn, fonts[fn], ops)
		if err != nil {
			return nil, err
		}
		for _, name := range staged {
			if err := contextutil.Check(c); err != nil {
				return nil, err
			}
			if seen[name] {
				return nil, fmt.Errorf("duplicate cheat-sheet output %s", name)
			}
			seen[name] = true
			names = append(names, name)
		}
	}
	published, err = publishCheatSheets(c, dir, stagingDir, names, ops.files)
	if published {
		paths = make([]string, len(names))
		for i, name := range names {
			paths[i] = filepath.Join(dir, name)
		}
	}
	return paths, err
}

// CreateCheatSheetsUserFontsWithResult atomically generates a batch of user-font cheat sheets, supports cancellation
// and reports published paths.
func CreateCheatSheetsUserFontsWithResult(c context.Context, fontNames []string) (result FontCheatSheetResult, err error) {
	defer fault.Catch(&err)
	return createCheatSheetsUserFontsUsing(c, fontNames, defaultFontCheatSheetOperations())
}

// CreateCheatSheetsUserFonts atomically generates a batch of user-font cheat sheets and supports cancellation.
func CreateCheatSheetsUserFonts(c context.Context, fontNames []string) error {
	_, err := CreateCheatSheetsUserFontsWithResult(c, fontNames)
	return err
}

func createCheatSheetsUserFontsUsing(c context.Context, fontNames []string, ops fontCheatSheetOperations) (FontCheatSheetResult, error) {
	if err := contextutil.Check(c); err != nil {
		return FontCheatSheetResult{}, err
	}
	if err := validateNoEmptyStrings(fontNames, "font name"); err != nil {
		return FontCheatSheetResult{}, fmt.Errorf("create font cheat sheets: %w", err)
	}
	if err := loadCheatSheetUserFonts(c, ops); err != nil {
		return FontCheatSheetResult{}, fmt.Errorf("create font cheat sheets: load user fonts: %w", err)
	}

	names := slices.Clone(fontNames)
	if err := contextutil.Check(c); err != nil {
		return FontCheatSheetResult{}, err
	}
	if len(names) == 0 {
		var err error
		names, err = cheatSheetUserFontNames(c, ops)
		if err != nil {
			return FontCheatSheetResult{}, fmt.Errorf("create font cheat sheets: list user fonts: %w", err)
		}
	}
	sort.Strings(names)
	if err := contextutil.Check(c); err != nil {
		return FontCheatSheetResult{}, err
	}
	fonts := make(map[string]font.TTFLight, len(names))
	for _, fn := range names {
		if err := contextutil.Check(c); err != nil {
			return FontCheatSheetResult{}, err
		}
		ttf, ok, err := cheatSheetUserFont(c, fn, ops)
		if err != nil {
			return FontCheatSheetResult{}, fmt.Errorf("create font cheat sheets: font %s metrics: %w", fn, err)
		}
		if !ok {
			return FontCheatSheetResult{}, fmt.Errorf("create font cheat sheets: font %s: %w", fn, ErrUserFontNotFound)
		}
		fonts[fn] = ttf
	}
	paths, err := createUserFontCheatSheetBatch(c, ".", names, fonts, ops)
	result := FontCheatSheetResult{Paths: paths}
	if err != nil {
		return result, fmt.Errorf("create font cheat sheets: %w", err)
	}
	return result, nil
}
