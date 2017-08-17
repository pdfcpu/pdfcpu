package extract

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"strconv"

	"github.com/EndFirstCorp/pdflib/filter"
	"github.com/EndFirstCorp/pdflib/types"
	"github.com/pkg/errors"
)

func writeContent(ctx *types.PDFContext, streamDict *types.PDFStreamDict, pageNumber, section int) (err error) {

	var fileName string

	if section >= 0 {
		fileName = fmt.Sprintf("%s/content_p%d_%d.txt", ctx.Write.DirName, pageNumber, section)
	} else {
		fileName = fmt.Sprintf("%s/content_p%d.txt", ctx.Write.DirName, pageNumber)
	}

	// Decode streamDict if used filter is supported only.
	err = filter.DecodeStream(streamDict)
	if err == filter.ErrUnsupportedFilter {
		err = nil
		return
	}
	if err != nil {
		return
	}

	// Dump decoded chunk to file.
	err = ioutil.WriteFile(fileName, streamDict.Content, os.ModePerm)

	return
}

// Process the content of a page which is a stream dict or an array of stream dicts.
func processPageDict(ctx *types.PDFContext, objNumber, genNumber int, dict *types.PDFDict, pageNumber int) (err error) {

	logDebugExtract.Printf("processPageDict begin: page=%d\n", pageNumber)

	obj, found := dict.Find("Contents")
	if !found {
		return
	}

	if obj == nil {
		return
	}

	obj, err = ctx.Dereference(obj)
	if err != nil {
		return
	}

	switch obj := obj.(type) {

	case types.PDFStreamDict:
		err = writeContent(ctx, &obj, pageNumber, -1)
		if err != nil {
			return
		}

	case types.PDFArray:

		// process array of content stream dicts.

		for i, obj := range obj {

			streamDict, err := ctx.DereferenceStreamDict(obj)
			if err != nil {
				return err
			}

			err = writeContent(ctx, streamDict, pageNumber, i)
			if err != nil {
				return err
			}

		}

	default:
		err = errors.Errorf("writePageContents: page content must be stream dict or array")
		return
	}

	return
}

func needsPage(selectedPages types.IntSet, pageCount int) bool {

	return selectedPages == nil || len(selectedPages) == 0 || selectedPages[pageCount]
}

func processPagesDict(ctx *types.PDFContext, indRef *types.PDFIndirectRef, pageCount *int, selectedPages types.IntSet) (err error) {

	logDebugExtract.Printf("processPagesDict begin: pageCount=%d\n", *pageCount)

	dict, err := ctx.DereferenceDict(*indRef)
	if err != nil {
		return
	}

	// Iterate over page tree.
	kidsArray := dict.PDFArrayEntry("Kids")
	if kidsArray == nil {
		return errors.New("writePagesDict: corrupt \"Kids\" entry")
	}

	for _, obj := range *kidsArray {

		if obj == nil {
			continue
		}

		// Dereference next page node dict.
		indRef, ok := obj.(types.PDFIndirectRef)
		if !ok {
			return errors.New("writePagesDict: missing indirect reference for kid")
		}

		objNumber := int(indRef.ObjectNumber)
		genNumber := int(indRef.GenerationNumber)

		pageNodeDict, err := ctx.DereferenceDict(indRef)
		if err != nil {
			return errors.New("writePagesDict: cannot dereference pageNodeDict")
		}

		if pageNodeDict == nil {
			return errors.New("validatePagesDict: pageNodeDict is null")
		}

		dictType := pageNodeDict.Type()
		if dictType == nil {
			return errors.New("writePagesDict: missing pageNodeDict type")
		}

		switch *dictType {

		case "Pages":
			// Recurse over pagetree
			err = processPagesDict(ctx, &indRef, pageCount, selectedPages)
			if err != nil {
				return err
			}

		case "Page":
			*pageCount++
			// extractContent of a page if no pages selected or if page is selected.
			if needsPage(selectedPages, *pageCount) {
				err = processPageDict(ctx, objNumber, genNumber, pageNodeDict, *pageCount)
				if err != nil {
					return err
				}
			}

		default:
			return errors.Errorf("writePagesDict: Unexpected dict type: %s", *dictType)

		}

	}

	logDebugExtract.Printf("processPagesDict end: pageCount=%d\n", *pageCount)

	return
}

// Content writes content streams for selected pages to dirOut.
// Each content stream results in a separate text file.
func Content(ctx *types.PDFContext, selectedPages types.IntSet) (err error) {

	logDebugExtract.Printf("Content begin: dirOut=%s\n", ctx.Write.DirName)

	// Get an indirect reference to the root page dict.
	indRefRootPageDict, err := ctx.Pages()
	if err != nil {
		return err
	}

	pageCount := 0
	err = processPagesDict(ctx, indRefRootPageDict, &pageCount, selectedPages)
	if err != nil {
		return err
	}

	logDebugExtract.Println("Content end")

	return
}

// Text returns io.Reader with text contained in the PDF
func Text(ctx *types.PDFContext, charmap map[string]map[string]string) (io.Reader, error) {

	// Get an indirect reference to the root page dict.
	indRefRootPageDict, err := ctx.Pages()
	if err != nil {
		return nil, err
	}

	pageCount := 0
	r, err := getPagesText(ctx, indRefRootPageDict, &pageCount, charmap)
	if err != nil {
		return nil, err
	}

	return r, nil
}

func getPagesText(ctx *types.PDFContext, indRef *types.PDFIndirectRef, pageCount *int, charmap map[string]map[string]string) (io.Reader, error) {
	buf := &bytes.Buffer{}
	dict, err := ctx.DereferenceDict(*indRef)
	if err != nil {
		return nil, err
	}

	// Iterate over page tree.
	kidsArray := dict.PDFArrayEntry("Kids")
	if kidsArray == nil {
		return nil, errors.New("writePagesDict: corrupt \"Kids\" entry")
	}

	for _, obj := range *kidsArray {

		if obj == nil {
			continue
		}

		// Dereference next page node dict.
		indRef, ok := obj.(types.PDFIndirectRef)
		if !ok {
			return nil, errors.New("writePagesDict: missing indirect reference for kid")
		}

		objNumber := int(indRef.ObjectNumber)
		genNumber := int(indRef.GenerationNumber)

		pageNodeDict, err := ctx.DereferenceDict(indRef)
		if err != nil {
			return nil, errors.New("writePagesDict: cannot dereference pageNodeDict")
		}

		if pageNodeDict == nil {
			return nil, errors.New("validatePagesDict: pageNodeDict is null")
		}

		dictType := pageNodeDict.Type()
		if dictType == nil {
			return nil, errors.New("writePagesDict: missing pageNodeDict type")
		}

		switch *dictType {

		case "Pages":
			// Recurse over pagetree
			b, err := getPagesText(ctx, &indRef, pageCount, charmap)
			if err != nil {
				return nil, err
			}
			buf.ReadFrom(b)

		case "Page":
			*pageCount++
			b, err := getPageText(ctx, objNumber, genNumber, pageNodeDict, *pageCount, charmap)
			if err != nil {
				return nil, err
			}
			buf.ReadFrom(b)

		default:
			return nil, errors.Errorf("writePagesDict: Unexpected dict type: %s", *dictType)

		}
	}
	return buf, nil
}

func getPageText(ctx *types.PDFContext, objNumber, genNumber int, dict *types.PDFDict, pageNumber int, charmap map[string]map[string]string) (io.Reader, error) {
	obj, found := dict.Find("Contents")
	if !found || obj == nil {
		return nil, errors.New("content not found")
	}

	obj, err := ctx.Dereference(obj)
	if err != nil {
		return nil, errors.New("unable to get reference")
	}

	switch obj := obj.(type) {

	case types.PDFStreamDict:
		b, err := getText(obj.Content, charmap)
		return bytes.NewBuffer(b), err

	case types.PDFArray:
		var buf bytes.Buffer
		for _, obj := range obj {

			streamDict, err := ctx.DereferenceStreamDict(obj)
			if err != nil {
				return nil, err
			}

			s, err := getText(streamDict.Content, charmap)
			if err != nil {
				return nil, err
			}
			buf.Write(s)
		}
		return &buf, nil

	default:
		return nil, errors.Errorf("writePageContents: page content must be stream dict or array")
	}
}

type cmd struct {
	font string
	op   string
	text []byte
	data []byte
	err  error
}

func getText(content []byte, charmap map[string]map[string]string) ([]byte, error) {
	var buf bytes.Buffer
	var font string
	var line bytes.Buffer
	for _, b := range content {
		var op string
		var data []byte
		switch b {
		case '\n', '\r': // organize into lines
			if line.Len() == 0 { // skip \n on \r\n
				continue
			}
			data = line.Bytes()
			line.Reset()
			op = string(data[bytes.LastIndexAny(data, " \t")+1:])
		default:
			line.WriteByte(b)
			continue
		}

		// completed line
		switch op {
		case "Tf":
			end := bytes.Index(data, []byte{' '})
			start := bytes.Index(data, []byte{'/'})
			if start < end && start != -1 && end != -1 {
				font = string(data[start+1 : end])
			}
		case "TJ", "Tj":
			var isText bool
			var isHex bool
			var hexBuf bytes.Buffer
			for _, c := range data {
				switch c {
				case '<':
					isHex = true
					continue
				case '>':
					isHex = false
					continue
				case '(':
					isText = true
					continue
				case ')':
					isText = false
					continue
				default:
					if isText {
						buf.WriteByte(c)
					} else if isHex {
						hexBuf.WriteByte(c)
					}
				}
			}
			if hexBuf.Len() == 0 {
				continue
			}

			if hexBuf.Len()%2 == 1 { // if it's an odd number of characters add 0 to the end
				hexBuf.WriteByte('0')
			}
			if charmap != nil {
				newHex := mapHex(charmap, font, hexBuf.String())
				buf.WriteString(newHex)
			} else {
				buf.Write(hexBuf.Bytes())
			}
			hexBuf.Reset()
		}
	}
	return buf.Bytes(), nil
}

func mapHex(charmap map[string]map[string]string, font string, hexChars string) string {
	if charmap == nil {
		return hexChars
	}
	fontMap, ok := charmap[font]
	if !ok {
		return hexChars
	}

	var hexRepl bytes.Buffer
	for i := 0; i < len(hexChars)/4; i++ {
		start := i * 4
		end := (i + 1) * 4
		if end > len(hexChars) {
			fmt.Println("unexpected length of hex string", hexChars)
			return hexChars
		}
		orig := hexChars[start:end]
		if repl, ok := fontMap[orig]; ok {
			hexRepl.WriteString(repl)
		} else {
			v, _ := strconv.ParseInt(orig, 16, 16)
			hexRepl.WriteString(fmt.Sprintf("%c", v))
		}
	}
	return hexRepl.String()
}

func getString(charmap map[string]map[string]string, font string, hexBuf *bytes.Buffer) string {
	if charmap == nil {
		return hexBuf.String()
	}
	fontMap, ok := charmap[font]
	if !ok {
		return hexBuf.String()
	}

	hexString := hexBuf.String()
	var hexRepl bytes.Buffer
	for i := 0; i < hexBuf.Len()/4; i++ {
		start := i * 4
		end := (i+1)*4 + 1
		orig := hexString[start:end]
		if repl, ok := fontMap[orig]; ok {
			hexRepl.WriteString(repl)
		} else {
			hexRepl.WriteString(orig)
		}
	}
	return hexRepl.String()
}
