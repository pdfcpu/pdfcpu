package read

import (
	"io/ioutil"
	"log"
	"os"
	"strconv"
	"strings"
	"unicode"

	"github.com/hhrutter/pdfcpu/types"
	"github.com/pkg/errors"
)

var (
	logDebugParse *log.Logger
	logInfoParse  *log.Logger

	errArrayCorrupt            = errors.New("parse: corrupt array")
	errArrayNotTerminated      = errors.New("parse: unterminated array")
	errDictionaryCorrupt       = errors.New("parse: corrupt dictionary")
	errDictionaryDuplicateKey  = errors.New("parse: duplicate key")
	errDictionaryNotTerminated = errors.New("parse: unterminated dictionary")
	errHexLiteralCorrupt       = errors.New("parse: corrupt hex literal")
	errHexLiteralNotTerminated = errors.New("parse: hex literal not terminated")
	errNameObjectCorrupt       = errors.New("parse: corrupt name object")
	errNoArray                 = errors.New("parse: no array")
	errNoDictionary            = errors.New("parse: no dictionary")
	errStringLiteralCorrupt    = errors.New("parse: corrupt string literal, possibly unbalanced parenthesis")
	errBufNotAvailable         = errors.New("parse: no buffer available")
	errXrefStreamMissingW      = errors.New("parse: xref stream dict missing entry W")
	errXrefStreamCorruptW      = errors.New("parse: xref stream dict corrupt entry W: expecting array of 3 int")
	errXrefStreamCorruptIndex  = errors.New("parse: xref stream dict corrupt entry Index")
	errObjStreamMissingN       = errors.New("parse: obj stream dict missing entry W")
	errObjStreamMissingFirst   = errors.New("parse: obj stream dict missing entry First")
)

func init() {

	logDebugParse = log.New(ioutil.Discard, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
	//logDebugParse = log.New(os.Stdout, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)

	logInfoParse = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
}

func positionToNextWhitespace(s string) (int, string) {

	for i, c := range s {
		if unicode.IsSpace(c) {
			return i, s[i:]
		}
	}
	return 0, s
}

// PositionToNextWhitespaceOrChar trims a string to next whitespace or one of given chars.
func positionToNextWhitespaceOrChar(s, chars string) (int, string) {

	if len(chars) == 0 {
		return positionToNextWhitespace(s)
	}

	if len(chars) > 0 {
		for i, c := range s {
			for _, m := range chars {
				if c == m || unicode.IsSpace(c) {
					return i, s[i:]
				}
			}
		}
	}
	return 0, s
}

func positionToNextEOL(s string) string {

	chars := "\x0A\x0D"

	for i, c := range s {
		for _, m := range chars {
			if c == m {
				return s[i:]
			}
		}
	}
	return ""
}

// trimLeftSpace trims leading whitespace and trailing comment.
func trimLeftSpace(s string) (outstr string, trimmedSpaces int) {

	logDebugParse.Printf("TrimLeftSpace: begin %s\n", s)

	whitespace := func(c rune) bool { return unicode.IsSpace(c) }

	outstr = s

	for {
		// trim leading whitespace
		outstr = strings.TrimLeftFunc(outstr, whitespace)
		logDebugParse.Printf("1 outstr: <%s>\n", outstr)
		if len(outstr) <= 1 || outstr[0] != '%' {
			break
		}
		// trim PDF comment (= '%' up to eol)
		outstr = positionToNextEOL(outstr)
		logDebugParse.Printf("2 outstr: <%s>\n", outstr)

	}

	trimmedSpaces = len(s) - len(outstr)

	logDebugParse.Printf("TrimLeftSpace: end %s %d\n", outstr, trimmedSpaces)

	return outstr, trimmedSpaces
}

// HexString validates and formats a hex string to be of even length.
func hexString(s string) (*string, bool) {

	logDebugParse.Printf("HexString(%s)\n", s)

	if len(s) == 0 {
		s1 := ""
		return &s1, true
	}

	uc := strings.ToUpper(s)

	for _, c := range uc {
		logDebugParse.Printf("checking <%c>\n", c)
		isHexChar := false
		for _, hexch := range "ABCDEF1234567890" {
			logDebugParse.Printf("checking against <%c>\n", hexch)
			if c == hexch {
				isHexChar = true
				break
			}
		}
		if !isHexChar {
			logDebugParse.Println("isHexStr returning false")
			return nil, false
		}
	}

	logDebugParse.Println("isHexStr returning true")

	// If the final digit of a hexadecimal string is missing -
	// that is, if there is an odd number of digits - the final digit shall be assumed to be 0.
	if len(uc)%2 == 1 {
		uc = uc + "0"
	}

	return &uc, true
}

// balancedParenthesesPrefix returns the index of the end position of the balanced parentheses prefix of s
// or -1 if unbalanced. s has to start with '('
func balancedParenthesesPrefix(s string) int {

	var j int
	escaped := false

	for i := 0; i < len(s); i++ {

		c := s[i]

		if !escaped && c == '\\' {
			escaped = true
			continue
		}

		if escaped {
			escaped = false
			continue
		}

		if c == '(' {
			j++
		}

		if c == ')' {
			j--
		}

		if j == 0 {
			return i
		}

	}

	return -1
}

func forwardParseBuf(buf string, pos int) string {
	if pos < len(buf) {
		return buf[pos:]
	}

	return ""
}

func delimiter(b byte) bool {

	s := "<>[]()/"

	for i := 0; i < len(s); i++ {
		if b == s[i] {
			return true
		}
	}

	return false
}

// parseObjectAttributes parses object number and generation of the next object for given string buffer.
func parseObjectAttributes(line *string) (objectNumber *int, generationNumber *int, err error) {

	logDebugParse.Printf("ParseObjectAttributes: buf=<%s>\n", *line)

	if line == nil || len(*line) == 0 {
		return nil, nil, errors.New("ParseObjectAttributes: buf not available")
	}

	l := *line
	var remainder string

	i := strings.Index(l, "obj")
	if i < 0 {
		return nil, nil, errors.New("ParseObjectAttributes: can't find \"obj\"")
	}

	remainder = l[i+len("obj"):]
	l = l[:i]

	// object number

	l, _ = trimLeftSpace(l)
	if len(l) == 0 {
		return nil, nil, errors.New("ParseObjectAttributes: can't find object number")
	}

	i, _ = positionToNextWhitespaceOrChar(l, "%")
	if i == 0 {
		return nil, nil, errors.New("ParseObjectAttributes: can't find end of object number")
	}

	objNr, err := strconv.Atoi(l[:i])
	if err != nil {
		return nil, nil, err
	}

	// generation number

	l = l[i:]
	l, _ = trimLeftSpace(l)
	if len(l) == 0 {
		return nil, nil, errors.New("ParseObjectAttributes: can't find generation number")
	}

	i, _ = positionToNextWhitespaceOrChar(l, "%")
	if i == 0 {
		return nil, nil, errors.New("ParseObjectAttributes: can't find end of generation number")
	}

	genNr, err := strconv.Atoi(l[:i])
	if err != nil {
		return nil, nil, err
	}

	objectNumber = &objNr
	generationNumber = &genNr

	*line = remainder

	return objectNumber, generationNumber, nil
}

func parseArray(line *string) (*types.PDFArray, error) {

	if line == nil || len(*line) == 0 {
		return nil, errNoArray
	}

	l := *line

	logDebugParse.Printf("ParseArray: %s\n", l)
	//logInfoParse.Println("ParseArray begin")

	if !strings.HasPrefix(l, "[") {
		return nil, errArrayCorrupt
	}

	if len(l) == 1 {
		return nil, errArrayNotTerminated
	}

	// position behind '['
	l = forwardParseBuf(l, 1)

	// position to first non whitespace char after '['
	l, _ = trimLeftSpace(l)

	if len(l) == 0 {
		// only whitespace after '['
		return nil, errArrayNotTerminated
	}

	arr := types.PDFArray{}

	for !strings.HasPrefix(l, "]") {

		obj, err := parseObject(&l)
		if err != nil {
			return nil, err
		}
		logDebugParse.Printf("ParseArray: new array obj=%v\n", obj)
		arr = append(arr, obj)

		// we are positioned on the char behind the last parsed array entry.
		if len(l) == 0 {
			return nil, errArrayNotTerminated
		}

		// position to next non whitespace char.
		l, _ = trimLeftSpace(l)
		if len(l) == 0 {
			return nil, errArrayNotTerminated
		}
	}

	// position behind ']'
	l = forwardParseBuf(l, 1)

	*line = l

	//logInfoParse.Printf("ParseArray end: returning array (len=%d)\n", len(arr))
	logDebugParse.Printf("ParseArray: returning array (len=%d): %v\n", len(arr), arr)

	return &arr, nil
}

func parseStringLiteral(line *string) (types.PDFObject, error) {

	// Balanced pairs of parenthesis are allowed.
	// Empty literals are allowed.
	// \ needs special treatment.
	// Allowed escape sequences:
	// \n	x0A
	// \r	x0D
	// \t	x09
	// \b	x08
	// \f	xFF
	// \(	x28
	// \)	x29
	// \\	x5C
	// \ddd octal code sequence, d=0..7

	// Ignore '\' for undefined escape sequences.

	// Unescaped 0x0A,0x0D or combination gets parsed as 0x0A.

	// Join split lines by '\' eol.

	if line == nil || len(*line) == 0 {
		return nil, errBufNotAvailable
	}

	l := *line

	logDebugParse.Printf("parseStringLiteral: begin <%s>\n", l)

	if len(l) < 2 || !strings.HasPrefix(l, "(") {
		return nil, errStringLiteralCorrupt
	}

	// Calculate prefix with balanced parentheses,
	// return index of enclosing ')'.
	i := balancedParenthesesPrefix(l)
	if i < 0 {
		// No balanced parentheses.
		return nil, errStringLiteralCorrupt
	}

	// remove enclosing '(', ')'
	balParStr := l[1:i]

	// Parse string literal, see 7.3.4.2
	//str := stringLiteral(balParStr)

	// position behind ')'
	*line = forwardParseBuf(l[i:], 1)

	stringLiteral := types.PDFStringLiteral(balParStr)
	logDebugParse.Printf("parseStringLiteral: end <%s>\n", stringLiteral)

	return stringLiteral, nil
}

func parseHexLiteral(line *string) (types.PDFObject, error) {

	// hexliterals have no whitespace and can't be empty.

	if line == nil || len(*line) == 0 {
		return nil, errBufNotAvailable
	}

	l := *line

	logDebugParse.Printf("parseHexLiteral: %s\n", l)

	if len(l) < 3 || !strings.HasPrefix(l, "<") {
		return nil, errHexLiteralCorrupt
	}

	// position behind '<'
	l = forwardParseBuf(l, 1)

	eov := strings.Index(l, ">") // end of hex literal.
	if eov < 0 {
		return nil, errHexLiteralNotTerminated
	}

	hexStr, ok := hexString(l[:eov])
	if !ok {
		return nil, errHexLiteralCorrupt
	}

	// position behind '>'
	*line = forwardParseBuf(l[eov:], 1)

	return types.PDFHexLiteral(*hexStr), nil
}

func parseName(line *string) (*types.PDFName, error) {

	// see 7.3.5

	if line == nil || len(*line) == 0 {
		return nil, errBufNotAvailable
	}

	l := *line

	logDebugParse.Printf("parseNameObject: %s\n", l)

	if len(l) < 2 || !strings.HasPrefix(l, "/") {
		return nil, errNameObjectCorrupt
	}

	// position behind '/'
	l = forwardParseBuf(l, 1)

	// cut off on whitespace or delimiter
	eok, _ := positionToNextWhitespaceOrChar(l, "/<>()[]")

	if eok > 0 || unicode.IsSpace(rune(l[0])) {
		logDebugParse.Printf("parseNameObject: wants to cut off at %d\n", eok)
		*line = l[eok:]
		l = l[:eok]
	} else {
		logDebugParse.Println("parseNameObject: nothing to cut off")
		*line = ""
	}

	nameObj := types.PDFName(l)

	return &nameObj, nil
}

func parseDict(line *string) (*types.PDFDict, error) {

	if line == nil || len(*line) == 0 {
		return nil, errNoDictionary
	}

	l := *line

	logDebugParse.Printf("ParseDict: %s\n", l)

	if len(l) < 4 || !strings.HasPrefix(l, "<<") {
		return nil, errDictionaryCorrupt
	}

	// position behind '<<'
	l = forwardParseBuf(l, 2)

	// position to first non whitespace char after '<<'
	l, _ = trimLeftSpace(l)

	if len(l) == 0 {
		// only whitespace after '['
		return nil, errDictionaryNotTerminated
	}

	dict := types.NewPDFDict()

	for !strings.HasPrefix(l, ">>") {

		key, err := parseName(&l)
		if err != nil {
			return nil, err
		}
		logDebugParse.Printf("ParseDict: key = %s\n", key)

		// position to first non whitespace after key
		l, _ = trimLeftSpace(l)

		if len(l) == 0 {
			logDebugParse.Println("ParseDict: only whitespace after key")
			// only whitespace after key
			return nil, errDictionaryNotTerminated
		}

		obj, err := parseObject(&l)
		if err != nil {
			return nil, err
		}

		logDebugParse.Printf("ParseDict: dict[%s]=%v\n", key, obj)
		if ok := dict.Insert(string(*key), obj); !ok {
			return nil, errDictionaryDuplicateKey
		}

		// we are positioned on the char behind the last parsed dict value.
		if len(l) == 0 {
			return nil, errDictionaryNotTerminated
		}

		// position to next non whitespace char.
		l, _ = trimLeftSpace(l)
		if len(l) == 0 {
			return nil, errDictionaryNotTerminated
		}

	}

	// position behind '>>'
	l = forwardParseBuf(l, 2)

	*line = l

	logDebugParse.Printf("ParseDict: returning dict at: %v\n", dict)

	return &dict, nil
}

func noBuf(l *string) bool {
	return l == nil || len(*l) == 0
}

func parseNumericOrIndRef(line *string) (types.PDFObject, error) {

	if noBuf(line) {
		return nil, errBufNotAvailable
	}

	l := *line

	// if this object is an integer we need to check for an indirect reference eg. 1 0 R
	// otherwise it has to be a float
	// we have to check first for integer

	i1, _ := positionToNextWhitespaceOrChar(l, "/<([]>")
	var l1 string
	if i1 > 0 {
		l1 = l[i1:]
	} else {
		l1 = l[len(l):]
	}

	str := l
	if i1 > 0 {
		str = l[:i1]
	}

	// Try int
	i, err := strconv.Atoi(str)
	if err != nil {

		// Try float
		f, err := strconv.ParseFloat(str, 64)
		if err != nil {
			return nil, err
		}

		// We have a Float!
		logDebugParse.Printf("parseNumericOrIndRef: value is numeric float: %f\n", f)
		*line = l1
		return types.PDFFloat(f), nil
	}

	// We have an Int!

	// if not followed by whitespace return sole integer value.
	if i1 == 0 || delimiter(l[i1]) {
		logDebugParse.Printf("parseNumericOrIndRef: value is numeric int: %d\n", i)
		*line = l1
		return types.PDFInteger(i), nil
	}

	// Must be indirect reference. (123 0 R)
	// Missing is the 2nd int and "R".

	iref1 := i

	l = l[i1:]
	l, _ = trimLeftSpace(l)
	if len(l) == 0 {
		// only whitespace
		*line = l1
		return types.PDFInteger(i), nil
	}

	i2, _ := positionToNextWhitespaceOrChar(l, "/<([]>")

	// if only 2 token, can't be indirect reference.
	// if not followed by whitespace return sole integer value.
	if i2 == 0 || delimiter(l[i2]) {
		logDebugParse.Printf("parseNumericOrIndRef: 2 objects => value is numeric int: %d\n", i)
		*line = l1
		return types.PDFInteger(i), nil
	}

	str = l
	if i2 > 0 {
		str = l[:i2]
	}

	iref2, err := strconv.Atoi(str)
	if err != nil {
		// 2nd int(generation number) not available.
		// Can't be an indirect reference.
		logDebugParse.Printf("parseNumericOrIndRef: 3 objects, 2nd no int, value is no indirect ref but numeric int: %d\n", i)
		*line = l1
		return types.PDFInteger(i), nil
	}

	// We have the 2nd int(generation number).
	// Look for "R"

	l = l[i2:]
	l, _ = trimLeftSpace(l)

	if len(l) == 0 {
		// only whitespace
		l = l1
		return types.PDFInteger(i), nil
	}

	if l[0] == 'R' {
		// We have all 3 components to create an indirect reference.
		*line = forwardParseBuf(l, 1)
		return *types.NewPDFIndirectRef(iref1, iref2), nil
	}

	// 'R' not available.
	// Can't be an indirect reference.
	logDebugParse.Printf("parseNumericOrIndRef: value is no indirect ref(no 'R') but numeric int: %d\n", i)
	*line = l1

	return types.PDFInteger(i), nil
}

func parseHexLiteralOrDict(l *string) (val types.PDFObject, err error) {

	if len(*l) < 2 {
		return nil, errBufNotAvailable
	}

	// if next char = '<' parseDict.
	if (*l)[1] == '<' {
		logDebugParse.Println("parseHexLiteralOrDict: value = Dictionary")
		pdfDict, err := parseDict(l)
		if err != nil {
			return nil, err
		}
		val = *pdfDict
	} else {
		// hex literals
		logDebugParse.Println("parseHexLiteralOrDict: value = Hex Literal")
		if val, err = parseHexLiteral(l); err != nil {
			return nil, err
		}
	}

	return val, nil
}

func parseBoolean(l string) (val types.PDFObject, s string, ok bool) {

	// null, absent object
	if strings.HasPrefix(l, "null") {
		logDebugParse.Println("parseBoolean: value = null")
		return nil, "null", true
	}

	// boolean true
	if strings.HasPrefix(l, "true") {
		logDebugParse.Println("parseBoolean: value = true")
		return types.PDFBoolean(true), "true", true
	}

	// boolean false
	if strings.HasPrefix(l, "false") {
		logDebugParse.Println("parseBoolean: value = false")
		return types.PDFBoolean(false), "false", true
	}

	return nil, "", false
}

// parseObject parses next PDFObject from string buffer.
func parseObject(line *string) (types.PDFObject, error) {

	if noBuf(line) {
		return nil, errBufNotAvailable
	}

	l := *line

	logDebugParse.Printf("ParseObject: buf=<%s>\n", l)

	// position to first non whitespace char
	l, _ = trimLeftSpace(l)
	if len(l) == 0 {
		// only whitespace
		return nil, errBufNotAvailable
	}

	var value types.PDFObject
	var err error

	switch l[0] {

	case '[': // array
		logDebugParse.Println("ParseObject: value = Array")
		pdfArray, err := parseArray(&l)
		if err != nil {
			return nil, err
		}
		value = *pdfArray

	case '/': // name
		logDebugParse.Println("ParseObject: value = Name Object")
		nameObj, err := parseName(&l)
		if err != nil {
			return nil, err
		}
		value = *nameObj

	case '<': // hex literal or dict
		value, err = parseHexLiteralOrDict(&l)
		if err != nil {
			return nil, err
		}

	case '(': // string literal
		logDebugParse.Printf("ParseObject: value = String Literal: <%s>\n", l)
		if value, err = parseStringLiteral(&l); err != nil {
			return nil, err
		}

	default:
		var valStr string
		var ok bool
		value, valStr, ok = parseBoolean(l)
		if ok {
			l = forwardParseBuf(l, len(valStr))
			break
		}
		// Must be numeric or indirect reference:
		// int 0 r
		// int
		// float
		if value, err = parseNumericOrIndRef(&l); err != nil {
			return nil, err
		}

	}

	logDebugParse.Printf("ParseObject returning %v\n", value)

	*line = l

	return value, nil
}

// parseXRefStreamDict creates a PDFXRefStreamDict out of a PDFStreamDict.
func parseXRefStreamDict(pdfStreamDict types.PDFStreamDict) (*types.PDFXRefStreamDict, error) {

	logDebugParse.Println("ParseXRefStreamDict: begin")

	if pdfStreamDict.Size() == nil {
		return nil, errors.New("ParseXRefStreamDict: \"Size\" not available")
	}

	objs := []int{}

	//	Read optional parameter Index
	pIndArr := pdfStreamDict.Index()
	if pIndArr != nil {
		logDebugParse.Println("ParseXRefStreamDict: using index dict")

		indArr := *pIndArr
		if len(indArr)%2 > 1 {
			return nil, errXrefStreamCorruptIndex
		}

		for i := 0; i < len(indArr)/2; i++ {

			startObj, ok := indArr[i*2].(types.PDFInteger)
			if !ok {
				return nil, errXrefStreamCorruptIndex
			}

			count, ok := indArr[i*2+1].(types.PDFInteger)
			if !ok {
				return nil, errXrefStreamCorruptIndex
			}

			for j := 0; j < count.Value(); j++ {
				objs = append(objs, startObj.Value()+j)
			}
		}

	} else {
		logDebugParse.Println("ParseXRefStreamDict: no index dict")
		for i := 0; i < *pdfStreamDict.Size(); i++ {
			objs = append(objs, i)

		}
	}

	// Read parameter W in order to decode the xref table.
	// array of integers representing the size of the fields in a single cross-reference entry.

	var wIntArr [3]int

	w := pdfStreamDict.W()
	if w == nil {
		return nil, errXrefStreamMissingW
	}

	arr := *w
	// validate array with 3 positive integers
	if len(arr) != 3 {
		return nil, errXrefStreamCorruptW
	}

	f := func(ok bool, i int) bool {
		return !ok || i < 0
	}

	i1, ok := arr[0].(types.PDFInteger)
	if f(ok, i1.Value()) {
		return nil, errXrefStreamCorruptW
	}
	wIntArr[0] = int(i1)

	i2, ok := arr[1].(types.PDFInteger)
	if f(ok, i2.Value()) {
		return nil, errXrefStreamCorruptW
	}
	wIntArr[1] = int(i2)

	i3, ok := arr[2].(types.PDFInteger)
	if f(ok, i3.Value()) {
		return nil, errXrefStreamCorruptW
	}
	wIntArr[2] = int(i3)

	xsd := &types.PDFXRefStreamDict{
		PDFStreamDict:  pdfStreamDict,
		Size:           *pdfStreamDict.Size(),
		Objects:        objs,
		W:              wIntArr,
		PreviousOffset: pdfStreamDict.Prev()}

	logDebugParse.Println("ParseXRefStreamDict: end")

	return xsd, nil
}

// objectStreamDict creates a PDFObjectStreamDict out of a PDFStreamDict.
func objectStreamDict(pdfStreamDict types.PDFStreamDict) (*types.PDFObjectStreamDict, error) {

	if pdfStreamDict.First() == nil {
		return nil, errObjStreamMissingFirst
	}

	if pdfStreamDict.N() == nil {
		return nil, errObjStreamMissingN
	}

	osd := &types.PDFObjectStreamDict{
		PDFStreamDict:  pdfStreamDict,
		ObjCount:       *pdfStreamDict.N(),
		FirstObjOffset: *pdfStreamDict.First(),
		ObjArray:       nil}

	return osd, nil
}
