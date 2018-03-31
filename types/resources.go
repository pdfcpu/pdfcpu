package types

import (
	"fmt"
	"strings"
)

// FontObject represents a font used in a PDF file.
type FontObject struct {
	ResourceNames []string
	Prefix        string
	FontName      string
	FontDict      *PDFDict
	Data          []byte
	Extension     string
}

// AddResourceName adds a resourceName referring to this font.
func (fo *FontObject) AddResourceName(resourceName string) {
	for _, resName := range fo.ResourceNames {
		if resName == resourceName {
			return
		}
	}
	fo.ResourceNames = append(fo.ResourceNames, resourceName)
}

// ResourceNamesString returns a string representation of all the resource names of this font.
func (fo FontObject) ResourceNamesString() string {
	var resNames []string
	for _, resName := range fo.ResourceNames {
		resNames = append(resNames, resName)
	}
	return strings.Join(resNames, ",")
}

// Data returns the raw data belonging to this image object.
// func (fo FontObject) Data() []byte {
// 	return nil
// }

// SubType returns the SubType of this font.
func (fo FontObject) SubType() string {
	var subType string
	if fo.FontDict.Subtype() != nil {
		subType = *fo.FontDict.Subtype()
	}
	return subType
}

// Encoding returns the Encoding of this font.
func (fo FontObject) Encoding() string {
	encoding := "Built-in"
	pdfObject, found := fo.FontDict.Find("Encoding")
	if found {
		switch enc := pdfObject.(type) {
		case PDFName:
			encoding = enc.String()
		default:
			encoding = "Custom"
		}
	}
	return encoding
}

// Embedded returns true if the font is embedded into this PDF file.
func (fo FontObject) Embedded() (embedded bool) {

	_, embedded = fo.FontDict.Find("FontDescriptor")

	if !embedded {
		_, embedded = fo.FontDict.Find("DescendantFonts")
	}

	return
}

func (fo FontObject) String() string {
	return fmt.Sprintf("%-10s %-30s %-10s %-20s %-8v %s\n",
		fo.Prefix, fo.FontName,
		fo.SubType(), fo.Encoding(),
		fo.Embedded(), fo.ResourceNamesString())
}

// ImageObject represents an image used in a PDF file.
type ImageObject struct {
	ResourceNames []string
	ImageDict     *PDFStreamDict
	Extension     string
}

// AddResourceName adds a resourceName to this imageObject's ResourceNames dict.
func (io *ImageObject) AddResourceName(resourceName string) {
	for _, resName := range io.ResourceNames {
		if resName == resourceName {
			return
		}
	}
	io.ResourceNames = append(io.ResourceNames, resourceName)
}

// ResourceNamesString returns a string representation of the ResourceNames for this image.
func (io ImageObject) ResourceNamesString() string {
	var resNames []string
	for _, resName := range io.ResourceNames {
		resNames = append(resNames, resName)
	}
	return strings.Join(resNames, ",")
}

// Data returns the raw data belonging to this image object.
func (io ImageObject) Data() []byte {
	return io.ImageDict.Raw
}
