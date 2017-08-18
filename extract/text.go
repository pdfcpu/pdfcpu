package extract

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/hhrutter/pdflib/types"
	"github.com/pkg/errors"
)

// Text returns io.Reader with text contained in the PDF
func Text(ctx *types.PDFContext) (io.Reader, error) {

	// Get an indirect reference to the root page dict.
	indRefRootPageDict, err := ctx.Pages()
	if err != nil {
		return nil, err
	}

	charmap := generateCharmap(ctx)

	pageCount := 0
	r, err := getPagesText(ctx, indRefRootPageDict, &pageCount, charmap)
	if err != nil {
		return nil, err
	}

	return r, nil
}

func generateCharmap(ctx *types.PDFContext) map[string]map[string]string {
	charmap := make(map[string]map[string]string)
	fonts := ctx.Optimize.FontObjects
	for i := range ctx.Optimize.FontObjects {
		font := fonts[i]
		var fontMap map[string]string
		ref := font.FontDict.IndirectRefEntry("ToUnicode")
		if ref == nil {
			continue
		}
		if u, found := ctx.Find(int(ref.ObjectNumber)); found {
			sd := u.Object.(types.PDFStreamDict)
			fontMap = parseMap(sd.Content)
		}
		for _, name := range font.ResourceNames {
			charmap[name] = fontMap
		}
	}
	return charmap
}

func parseMap(content []byte) map[string]string {
	fontMap := make(map[string]string)
	var line bytes.Buffer
	var isMap bool
	var isRange bool
	for _, b := range content {
		var op string
		var data []byte
		switch b {
		case '\n', '\r': // organize into lines
			data = line.Bytes()
			cmdStart := bytes.LastIndexAny(data, " \t")
			op = string(data[cmdStart+1:])
			line.Reset()
		default:
			line.WriteByte(b)
			continue
		}

		switch op {
		case "beginbfchar":
			isMap = true
		case "endbfchar":
			isMap = false
		case "beginbfrange":
			isRange = true
		case "endbfrange":
			isRange = false
		default:
			if isMap {
				v := bytes.Split(data, []byte{'<'})
				if len(v) < 2 {
					fmt.Println("unexpected length for data line", string(data))
					continue
				}
				orig := string(v[1][:bytes.IndexByte(v[1], '>')])
				repl := string(v[2][:bytes.IndexByte(v[2], '>')])
				num, _ := strconv.ParseInt(repl, 16, 16)
				repl = fmt.Sprintf("%c", num)

				fontMap[orig] = repl
			} else if isRange {
				v := bytes.Split(data, []byte{'<'})
				from := string(v[1][:bytes.IndexByte(v[1], '>')])
				to := string(v[2][:bytes.IndexByte(v[2], '>')])
				repl := string(v[3][:bytes.IndexByte(v[3], '>')])
				start, _ := strconv.ParseInt(from, 16, 16)
				end, _ := strconv.ParseInt(to, 16, 16)
				replInt, _ := strconv.ParseInt(repl, 16, 16)
				var count int64
				for i := start; i <= end; i++ {
					fontMap[strings.ToUpper(fmt.Sprintf("%04x", i))] = fmt.Sprintf("%c", replInt+count) // 4-digit hex value
					count++
				}
			}
		}
	}
	return fontMap
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
				buf.WriteString(mapHex(charmap, font, hexBuf.String()))
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
