package api

import (
	"fmt"
	"github.com/pdfcpu/pdfcpu/pkg/log"
	pdf "github.com/pdfcpu/pdfcpu/pkg/pdfcpu"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// WritableFS defines an interface for a writable file system or directory-like storage.
type WritableFS interface {
	// Base returns the base path or identifier of the file system (e.g., directory path or "memory").
	Base() string

	// WriteFile writes data to the named file, creating it if necessary.
	// If the file exists, it may overwrite it (implementation-dependent).
	// The perm parameter specifies the file permissions (e.g., 0644).
	WriteFile(name string, data []byte, perm fs.FileMode) error

	// Mkdir creates a directory named path, along with any necessary parents.
	// The perm parameter specifies the directory permissions (e.g., 0755).
	Mkdir(path string, perm fs.FileMode) error
}

// OsFS is a WritableFS implementation for the local file system.
type OsFS struct {
	BasePath string // BasePath is the root directory for file operations.
}

// NewOsFS creates a new OsFS with the specified base path.
func NewOsFS(basePath string) *OsFS {
	return &OsFS{BasePath: basePath}
}

// Base returns the base path of the OsFS.
func (o *OsFS) Base() string {
	return o.BasePath
}

// WriteFile writes data to a file in the local file system, creating parent directories as needed.
func (o *OsFS) WriteFile(name string, data []byte, perm fs.FileMode) error {
	fullPath := filepath.Join(o.BasePath, name)
	// Ensure parent directories exist before writing the file.
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return fmt.Errorf("cannot create directory %s: %w", filepath.Dir(fullPath), err)
	}
	// Write the file with the specified permissions.
	return os.WriteFile(fullPath, data, perm)
}

// Mkdir creates a directory and its parents in the local file system.
func (o *OsFS) Mkdir(path string, perm fs.FileMode) error {
	fullPath := filepath.Join(o.BasePath, path)
	// Create the directory and any necessary parents with the specified permissions.
	return os.MkdirAll(fullPath, perm)
}

// MemFS is an in-memory implementation of WritableFS.
type MemFS struct {
	files map[string]*MemFile // files stores the in-memory file system structure.
}

// MemFile represents an in-memory file or directory.
type MemFile struct {
	Data    []byte      // Data holds the file content (empty for directories).
	Mode    fs.FileMode // Mode specifies the file or directory permissions.
	ModTime time.Time   // ModTime tracks the last modification time.
	IsDir   bool        // IsDir indicates whether the entry is a directory.
}

// NewMemFS creates a new in-memory file system.
func NewMemFS() *MemFS {
	return &MemFS{files: make(map[string]*MemFile)}
}

// Base returns the identifier for the in-memory file system.
func (m *MemFS) Base() string {
	return "memory"
}

// WriteFile writes data to an in-memory file, creating parent directories as needed.
func (m *MemFS) WriteFile(name string, data []byte, perm fs.FileMode) error {
	name = filepath.Clean(name) // Normalize the file path.
	dir := filepath.Dir(name)
	// Create parent directories if they don't exist.
	if dir != "." && dir != "/" {
		if err := m.Mkdir(dir, 0755); err != nil {
			return err
		}
	}
	// Store the file in memory with the specified data and permissions.
	m.files[name] = &MemFile{
		Data:    data,
		Mode:    perm,
		ModTime: time.Now(),
		IsDir:   false,
	}
	return nil
}

// Mkdir creates an in-memory directory and its parents if they don't exist.
func (m *MemFS) Mkdir(path string, perm fs.FileMode) error {
	path = filepath.Clean(path) // Normalize the directory path.
	// Check if the path already exists as a non-directory.
	if file, exists := m.files[path]; exists && !file.IsDir {
		return fmt.Errorf("%s: not a directory", path)
	}
	// Recursively create parent directories if needed.
	parent := filepath.Dir(path)
	if parent != "." && parent != "/" {
		if err := m.Mkdir(parent, perm); err != nil {
			return err
		}
	}
	// Store the directory in memory with the specified permissions.
	m.files[path] = &MemFile{
		Mode:    perm | fs.ModeDir,
		ModTime: time.Now(),
		IsDir:   true,
	}
	return nil
}

// doExtractContentFS extracts content streams from selected PDF pages and writes them to the provided WritableFS.
func doExtractContentFS(ctx *pdf.Context, wf WritableFS, selectedPages pdf.IntSet) error {
	visited := pdf.IntSet{} // Tracks visited object numbers to avoid duplicates.

	// Iterate over selected pages.
	for p, v := range selectedPages {
		if v {
			log.Info.Printf("writing content for page %d\n", p)

			// Retrieve object numbers for the page's content streams.
			objNrs, err := contentObjNrs(ctx, p)
			if err != nil {
				return err
			}

			if objNrs == nil {
				continue // Skip pages with no content streams.
			}

			// Process each object number.
			for _, objNr := range objNrs {
				if visited[objNr] {
					continue // Skip already processed objects.
				}

				visited[objNr] = true

				// Extract the stream data for the object.
				b, err := pdf.ExtractStreamData(ctx, objNr)
				if err != nil {
					return err
				}

				if b == nil {
					continue // Skip objects with no stream data.
				}

				// Construct the output file name (e.g., "<page>_<object>.txt").
				fileName := filepath.Join(wf.Base(), fmt.Sprintf("%d_%d.txt", p, objNr))
				// Ensure the parent directory exists.
				if err := wf.Mkdir(filepath.Dir(fileName), 0755); err != nil {
					return fmt.Errorf("failed to create directory for %s: %w", fileName, err)
				}

				// Write the stream data to the file system.
				if err := wf.WriteFile(fileName, b, 0644); err != nil {
					return fmt.Errorf("failed to write file %s for page %d, obj %d: %w", fileName, p, objNr, err)
				}
			}
		}
	}

	return nil
}

// ExtractContentFS extracts PDF content streams from the input ReadSeeker and writes them to the provided WritableFS.
func ExtractContentFS(rs io.ReadSeeker, wf WritableFS, selectedPages []string, conf *pdf.Configuration) error {
	// Use default configuration if none provided.
	if conf == nil {
		conf = pdf.NewDefaultConfiguration()
	}

	// Record the start time for performance tracking.
	fromStart := time.Now()
	// Read, validate, and optimize the PDF context.
	ctx, durRead, durVal, durOpt, err := readValidateAndOptimize(rs, conf, fromStart)
	if err != nil {
		return err
	}

	// Ensure the PDF has a valid page count.
	if err := ctx.EnsurePageCount(); err != nil {
		return err
	}

	// Record the start time for the write operation.
	fromWrite := time.Now()
	// Parse the selected pages into a page set.
	pages, err := PagesForPageSelection(ctx.PageCount, selectedPages, true)
	if err != nil {
		return err
	}

	// Set the output directory name in the PDF context.
	ctx.Write.DirName = filepath.Base(wf.Base())
	// Extract and write content streams for the selected pages.
	if err = doExtractContentFS(ctx, wf, pages); err != nil {
		return err
	}

	// Calculate performance metrics.
	durWrite := time.Since(fromWrite).Seconds()
	durTotal := time.Since(fromStart).Seconds()
	// Log the cross-reference table and timing statistics.
	log.Stats.Printf("XRefTable:\n%s\n", ctx)
	pdf.TimingStats("write content", durRead, durVal, durOpt, durWrite, durTotal)

	return nil
}
