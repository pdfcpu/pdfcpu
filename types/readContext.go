package types

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/hhrutter/pdfcpu/log"
)

// ReadContext represents the context for reading a PDF file.
type ReadContext struct {

	// The PDF-File which gets processed.
	FileName string
	File     *os.File
	FileSize int64

	BinaryTotalSize     int64 // total stream data
	BinaryImageSize     int64 // total image stream data
	BinaryFontSize      int64 // total font stream data (fontfiles)
	BinaryImageDuplSize int64 // total obsolet image stream data after optimization
	BinaryFontDuplSize  int64 // total obsolet font stream data after optimization

	Linearized bool // File is linearized.
	Hybrid     bool // File is a hybrid PDF file.

	UsingObjectStreams bool   // File is using object streams.
	ObjectStreams      IntSet // All object numbers of any object streams found which need to be decoded.

	UsingXRefStreams bool   // File is using xref streams.
	XRefStreams      IntSet // All object numbers of any xref streams found.
}

func newReadContext(fileName string, file *os.File, fileSize int64) *ReadContext {
	return &ReadContext{
		FileName:      fileName,
		File:          file,
		FileSize:      fileSize,
		ObjectStreams: IntSet{},
		XRefStreams:   IntSet{},
	}
}

// IsObjectStreamObject returns true if object i is a an object stream.
// All compressed objects are object streams.
func (rc *ReadContext) IsObjectStreamObject(i int) bool {
	return rc.ObjectStreams[i]
}

// ObjectStreamsString returns a formatted string and the number of object stream objects.
func (rc *ReadContext) ObjectStreamsString() (int, string) {

	var objs []int
	for k := range rc.ObjectStreams {
		if rc.ObjectStreams[k] {
			objs = append(objs, k)
		}
	}
	sort.Ints(objs)

	var objStreams []string
	for _, i := range objs {
		objStreams = append(objStreams, fmt.Sprintf("%d", i))
	}

	return len(objStreams), strings.Join(objStreams, ",")
}

// IsXRefStreamObject returns true if object #i is a an xref stream.
func (rc *ReadContext) IsXRefStreamObject(i int) bool {
	return rc.XRefStreams[i]
}

// XRefStreamsString returns a formatted string and the number of xref stream objects.
func (rc *ReadContext) XRefStreamsString() (int, string) {

	var objs []int
	for k := range rc.XRefStreams {
		if rc.XRefStreams[k] {
			objs = append(objs, k)
		}
	}
	sort.Ints(objs)

	var xrefStreams []string
	for _, i := range objs {
		xrefStreams = append(xrefStreams, fmt.Sprintf("%d", i))
	}

	return len(xrefStreams), strings.Join(xrefStreams, ",")
}

// LogStats logs stats for read file.
func (rc *ReadContext) LogStats(optimized bool) {

	log := log.Stats

	textSize := rc.FileSize - rc.BinaryTotalSize // = non binary content = non stream data

	log.Println("Original:")
	log.Printf("File Size            : %s (%d bytes)\n", ByteSize(rc.FileSize), rc.FileSize)
	log.Printf("Total Binary Data    : %s (%d bytes) %4.1f%%\n", ByteSize(rc.BinaryTotalSize), rc.BinaryTotalSize, float32(rc.BinaryTotalSize)/float32(rc.FileSize)*100)
	log.Printf("Total Text   Data    : %s (%d bytes) %4.1f%%\n\n", ByteSize(textSize), textSize, float32(textSize)/float32(rc.FileSize)*100)

	// Only when optimizing we get details about resource data usage.
	if optimized {

		// Image stream data of original file.
		binaryImageSize := rc.BinaryImageSize + rc.BinaryImageDuplSize

		// Font stream data of original file. (just font files)
		binaryFontSize := rc.BinaryFontSize + rc.BinaryFontDuplSize

		// Content stream data, other font related stream data.
		binaryOtherSize := rc.BinaryTotalSize - binaryImageSize - binaryFontSize

		log.Println("Breakup of binary data:")
		log.Printf("images               : %s (%d bytes) %4.1f%%\n", ByteSize(binaryImageSize), binaryImageSize, float32(binaryImageSize)/float32(rc.BinaryTotalSize)*100)
		log.Printf("fonts                : %s (%d bytes) %4.1f%%\n", ByteSize(binaryFontSize), binaryFontSize, float32(binaryFontSize)/float32(rc.BinaryTotalSize)*100)
		log.Printf("other                : %s (%d bytes) %4.1f%%\n\n", ByteSize(binaryOtherSize), binaryOtherSize, float32(binaryOtherSize)/float32(rc.BinaryTotalSize)*100)
	}
}
