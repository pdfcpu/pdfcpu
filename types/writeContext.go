package types

import (
	"bufio"

	"github.com/hhrutter/pdfcpu/log"
)

// WriteContext represents the context for writing a PDF file.
type WriteContext struct {

	// The PDF-File which gets generated.
	DirName  string
	FileName string
	FileSize int64
	*bufio.Writer

	Command       string // command in effect.
	ExtractPageNr int    // page to be generated for rendering a single-page/PDF.
	ExtractPages  IntSet // pages to be generated for a trimmed PDF.

	BinaryTotalSize int64 // total stream data, counts 100% all stream data written.
	BinaryImageSize int64 // total image stream data written = Read.BinaryImageSize.
	BinaryFontSize  int64 // total font stream data (fontfiles) = copy of Read.BinaryFontSize.

	Table  map[int]int64 // object write offsets
	Offset int64         // current write offset

	WriteToObjectStream bool // if true start to embed objects into object streams and obey ObjectStreamMaxObjects.
	CurrentObjStream    *int // if not nil, any new non-stream-object gets added to the object stream with this object number.

	Eol string // end of line char sequence
}

// NewWriteContext returns a new WriteContext.
func NewWriteContext(eol string) *WriteContext {
	return &WriteContext{Table: map[int]int64{}, Eol: eol}
}

// SetWriteOffset saves the current write offset to the PDFDestination.
func (wc *WriteContext) SetWriteOffset(objNumber int) {
	wc.Table[objNumber] = wc.Offset
}

// HasWriteOffset returns true if an object has already been written to PDFDestination.
func (wc *WriteContext) HasWriteOffset(objNumber int) bool {
	_, found := wc.Table[objNumber]
	return found
}

// ReducedFeatureSet returns true for Split,Trim,Merge,ExtractPages.
// Don't confuse with pdfcpu commands, these are internal triggers.
func (wc *WriteContext) ReducedFeatureSet() bool {
	switch wc.Command {
	case "Split", "Trim", "Merge":
		return true
	}
	return false
}

// ExtractPage returns true if page i needs to be generated.
func (wc *WriteContext) ExtractPage(i int) bool {

	if wc.ExtractPages == nil {
		return false
	}

	return wc.ExtractPages[i]

}

// LogStats logs stats for written file.
func (wc *WriteContext) LogStats() {

	fileSize := wc.FileSize
	binaryTotalSize := wc.BinaryTotalSize  // stream data
	textSize := fileSize - binaryTotalSize // non stream data

	binaryImageSize := wc.BinaryImageSize
	binaryFontSize := wc.BinaryFontSize
	binaryOtherSize := binaryTotalSize - binaryImageSize - binaryFontSize // content streams

	log.Stats.Println("Optimized:")
	log.Stats.Printf("File Size            : %s (%d bytes)\n", ByteSize(fileSize), fileSize)
	log.Stats.Printf("Total Binary Data    : %s (%d bytes) %4.1f%%\n", ByteSize(binaryTotalSize), binaryTotalSize, float32(binaryTotalSize)/float32(fileSize)*100)
	log.Stats.Printf("Total Text   Data    : %s (%d bytes) %4.1f%%\n\n", ByteSize(textSize), textSize, float32(textSize)/float32(fileSize)*100)

	log.Stats.Println("Breakup of binary data:")
	log.Stats.Printf("images               : %s (%d bytes) %4.1f%%\n", ByteSize(binaryImageSize), binaryImageSize, float32(binaryImageSize)/float32(binaryTotalSize)*100)
	log.Stats.Printf("fonts                : %s (%d bytes) %4.1f%%\n", ByteSize(binaryFontSize), binaryFontSize, float32(binaryFontSize)/float32(binaryTotalSize)*100)
	log.Stats.Printf("other                : %s (%d bytes) %4.1f%%\n\n", ByteSize(binaryOtherSize), binaryOtherSize, float32(binaryOtherSize)/float32(binaryTotalSize)*100)
}

// WriteEol writes an end of line sequence.
func (wc *WriteContext) WriteEol() error {

	_, err := wc.WriteString(wc.Eol)

	return err
}
