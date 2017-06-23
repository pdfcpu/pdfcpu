// Package validate contains validation code for ISO 32000-1:2008.
//
// There is low level validation and validation against the PDF spec for each of the defined PDF object types.
package validate

import (
	"io/ioutil"
	"log"
	"os"
	"time"

	"strings"

	"strconv"

	"github.com/hhrutter/pdflib/types"
	"github.com/pkg/errors"
)

const (

	// REQUIRED is used for required dict entries.
	REQUIRED = true

	// OPTIONAL is used for optional dict entries.
	OPTIONAL = false
)

var (
	logDebugValidate *log.Logger
	logInfoValidate  *log.Logger
	logErrorValidate *log.Logger
)

func init() {
	logDebugValidate = log.New(ioutil.Discard, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
	logInfoValidate = log.New(ioutil.Discard, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
	logErrorValidate = log.New(os.Stdout, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
}

// Verbose controls logging output.
func Verbose(verbose bool) {
	if verbose {
		logDebugValidate = log.New(os.Stdout, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
		logInfoValidate = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	} else {
		logDebugValidate = log.New(ioutil.Discard, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
		logInfoValidate = log.New(ioutil.Discard, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	}
}

func validateStandardType1Font(s string) bool {

	for _, v := range []string{
		"Times-Roman", "Times-Bold", "Times-Italic", "Times-BoldItalic",
		"Helvetica", "Helvetica-Bold", "Helvetica-Oblique", "Helvetica-BoldOblique",
		"Courier", "Courier-Bold", "Courier-Oblique", "Courier-BoldOblique",
		"Symbol", "ZapfDingbats"} {

		if s == v {
			logDebugValidate.Printf("isStandardType1Font: true %s\n", s)
			return true
		}
	}

	logDebugValidate.Printf("isStandardType1Font: false %s\n", s)

	return false
}

// TODO implement
func validateFileSpecString(s string) bool {

	// see 7.11.2
	// A text string encoded using PDFDocEncoding or UTF-16BE with leading byte order marker.

	// e.g.
	// C:\\Documents and Settings\\martiat\\Local Settings\\Temporary Internet Files\\OLK76\\art5079.html
	// tools.wmflabs.org/commonshelper/.pdf

	// TODO
	// read.PrefixBigEndian([]byte(s))

	return true
}

// TODO implement
func validateURLString(s string) bool {

	// see 7.11.5

	//log.Fatalf("validateURLString: unsupported: %s\n", s)

	return true
}

func validateFileSpecStringOrURLString(s string) bool {

	if validateFileSpecString(s) {
		return true
	}

	return validateURLString(s)
}

func validateFontEncodingName(s string) bool {

	for _, v := range []string{"MacRomanEncoding", "MacExpertEncoding", "WinAnsiEncoding"} {
		if s == v {
			return true
		}
	}

	return false
}

func validateStyleDict(dict types.PDFDict) bool {

	// see 9.8.3.2

	if dict.Len() != 1 {
		return false
	}

	_, found := dict.Find("Panose")

	return found
}

// unused ?
func validateExtGStateDictFont(arr types.PDFArray) bool {

	// see 9.3
	if len(arr) != 2 {
		return false
	}

	// ind ref to font dict
	if _, ok := arr[0].(types.PDFIndirectRef); !ok {
		return false
	}

	// size, number (text space units)i
	//writeNumber(source, dest, arr[1])

	return true
}

func validateBlendMode(s string) bool {

	// see 11.3.5; table 136

	for _, v := range []string{
		"None", "Normal", "Compatible", "Multiply", "Screen", "Overlay", "Darken", "Lighten",
		"ColorDodge", "ColorBurn", "HardLight", "SoftLight", "Difference", "Exclusion",
		"Hue", "Saturation", "Color", "Luminosity"} {
		if s == v {
			return true
		}
	}

	return false
}

func validateSpotFunctionName(s string) bool {

	for _, v := range []string{
		"SimpleDot", "InvertedSimpleDot", "DoubleDot", "InvertedDoubleDot", "CosineDot",
		"Double", "InvertedDouble", "Line", "LineX", "LineY"} {
		if s == v {
			return true
		}
	}

	return false
}

func validateICCBasedColorSpaceEntryN(i int) bool {

	for _, v := range []int{1, 3, 4} {
		if i == v {
			return true
		}
	}

	return false
}

// TODO implement
func validateColorKeyMaskArray(arr types.PDFArray) bool {
	// TODO validate integer array.
	return true
}

func validateRenderingIntent(s string) bool {

	// see 8.6.5.8

	for _, v := range []string{"AbsoluteColorimetric", "RelativeColorimetric", "Saturation", "Perceptual"} {
		if s == v {
			return true
		}
	}

	return false
}

func validateOPIVersion(s string) bool {

	for _, v := range []string{"1.3", "2.0"} {
		if s == v {
			return true
		}
	}

	return false
}

func validateDeviceColorSpaceName(s string) bool {

	for _, v := range []string{"DeviceGray", "DeviceRGB", "DeviceCMYK"} {
		if s == v {
			return true
		}
	}

	return false
}

func validateSpecialColorSpaceName(s string) bool {

	for _, v := range []string{"Pattern"} {
		if s == v {
			return true
		}
	}

	return false
}

func validateBitsPerCoordinate(i int) bool {

	for _, v := range []int{1, 2, 4, 8, 12, 16, 24, 32} {
		if i == v {
			return true
		}
	}

	return false
}

func validateBitsPerComponent(i int) bool {

	for _, v := range []int{1, 2, 4, 8, 12, 16} {
		if i == v {
			return true
		}
	}

	return false
}

// ProcedureSetName validates a procedure set name. Unused?
func validateProcedureSetName(s string) bool {

	for _, v := range []string{"PDF", "Text", "ImageB", "ImageC", "ImageI"} {
		if s == v {
			return true
		}
	}

	return false
}

func validateRotate(i int) bool {

	for _, v := range []int{0, 90, 4, 180, 270} {
		if i == v {
			return true
		}
	}

	return false
}

func validateNameTreeName(s string) bool {

	for _, v := range []string{
		"Dests", "AP", "JavaScript", "Pages", "Templates", "IDS",
		"URLS", "EmbeddedFiles", "AlternatePresentations", "Renditions"} {
		if s == v {
			return true
		}
	}

	return false
}

func validateNamedAction(s string) bool {

	for _, v := range []string{"NextPage", "PrevPage", "FirstPage", "Lastpage"} {
		if s == v {
			return true
		}
	}

	// Some known non standard named actions
	for _, v := range []string{"GoToPage", "GoBack", "GoForward", "Find"} {
		if s == v {
			return true
		}
	}

	return false
}

func validatePageLabelDictEntryS(s string) bool {

	// see 12.4.2 Page Labels

	for _, v := range []string{"D", "R", "r", "A", "a"} {
		if s == v {
			return true
		}
	}

	return false
}

// Date validates an ISO/IEC 8824 compliant date string.
func validateDate(s string) bool {

	// 7.9.4 Dates
	// (D:YYYYMMDDHHmmSSOHH'mm')

	logDebugValidate.Printf("validateDate(%s)\n", s)

	// utf16 conversion if applicable.
	if types.IsStringUTF16BE(s) {
		utf16s, err := types.DecodeUTF16String(s)
		if err != nil {
			return false
		}
		s = utf16s
	}

	// "D:YYYY" is mandatory
	if len(s) < 6 {
		return false
	}

	if !strings.HasPrefix(s, "D:") {
		return false
	}

	year := s[2:6]
	logDebugValidate.Printf("validateDate: year string = <%s>\n", year)
	y, err := strconv.Atoi(year)
	if err != nil {
		return false
	}

	// "D:YYYY"
	if len(s) == 6 {
		return true
	}

	if len(s) == 7 {
		return false
	}

	month := s[6:8]
	logDebugValidate.Printf("validateDate: month string = <%s>\n", month)
	m, err := strconv.Atoi(month)
	if err != nil {
		return false
	}

	if m < 1 || m > 12 {
		return false
	}

	// "D:YYYYMM"
	if len(s) == 8 {
		return true
	}

	if len(s) == 9 {
		return false
	}

	day := s[8:10]
	logDebugValidate.Printf("validateDate: day string = <%s>\n", day)
	d, err := strconv.Atoi(day)
	if err != nil {
		return false
	}

	if d < 1 || d > 31 {
		return false
	}

	// check valid Date(year,month,day)
	t := time.Date(y, time.Month(m+1), 0, 0, 0, 0, 0, time.UTC)
	logDebugValidate.Printf("last day of month is %d\n", t.Day())

	if d > t.Day() {
		return false
	}

	// "D:YYYYMMDD"
	if len(s) == 10 {
		return true
	}

	if len(s) == 11 {
		return false
	}

	hour := s[10:12]
	logDebugValidate.Printf("validateDate: hour string = <%s>\n", hour)
	h, err := strconv.Atoi(hour)
	if err != nil {
		return false
	}

	if h > 23 {
		return false
	}

	// "D:YYYYMMDDHH"
	if len(s) == 12 {
		return true
	}

	if len(s) == 13 {
		return false
	}

	minute := s[12:14]
	logDebugValidate.Printf("validateDate: minute string = <%s>\n", minute)
	min, err := strconv.Atoi(minute)
	if err != nil {
		return false
	}

	if min > 59 {
		return false
	}

	// "D:YYYYMMDDHHmm"
	if len(s) == 14 {
		return true
	}

	if len(s) == 15 {
		return false
	}

	second := s[14:16]
	logDebugValidate.Printf("validateDate: second string = <%s>\n", second)
	sec, err := strconv.Atoi(second)
	if err != nil {
		return false
	}

	if sec > 59 {
		return false
	}

	// "D:YYYYMMDDHHmmSS"
	if len(s) == 16 {
		return true
	}

	o := s[16]
	logDebugValidate.Printf("timezone operator:%s\n", string(o))

	if o != '+' && o != '-' && o != 'Z' {
		return false
	}

	// local time equal to UT.
	// "D:YYYYMMDDHHmmSSZ"
	if o == 'Z' && len(s) == 17 {
		return true
	}

	if len(s) < 20 {
		return false
	}

	tzhours := s[17:19]
	logDebugValidate.Printf("validateDate: tz hour offset string = <%s>\n", tzhours)
	tzh, err := strconv.Atoi(tzhours)
	if err != nil {
		return false
	}

	if tzh > 23 {
		return false
	}

	if o == 'Z' && tzh != 0 {
		return false
	}

	if s[19] != '\'' {
		return false
	}

	// "D:YYYYMMDDHHmmSSZHH'"
	if len(s) == 20 {
		return true
	}

	if len(s) != 22 && len(s) != 23 {
		return false
	}

	tzmin := s[20:22]
	logDebugValidate.Printf("validateDate: tz minutes offset string = <%s>\n", tzmin)
	tzm, err := strconv.Atoi(tzmin)
	if err != nil {
		return false
	}

	if tzm > 59 {
		return false
	}

	if o == 'Z' && tzm != 0 {
		return false
	}

	logDebugValidate.Printf("validateDate: returning %v\n", true)

	// "D:YYYYMMDDHHmmSSZHH'mm"
	if len(s) == 22 {
		return false
	}

	// Accept a trailing '
	return s[22] == '\''
}

func validateViewerPreferencesNonFullScreenPageMode(s string) bool {

	for _, v := range []string{"UseNone", "UseOutlines", "UseThumbs", "UseOC"} {
		if s == v {
			return true
		}
	}

	return false
}

func validateViewerPreferencesDirection(s string) bool {

	for _, v := range []string{"L2R", "R2L"} {
		if s == v {
			return true
		}
	}

	return false
}

// TODO Verify "W" (not in spec!)
func validateTabs(s string) bool {

	// see 12.5

	for _, v := range []string{"R", "C", "S", "W"} {
		if s == v {
			return true
		}
	}

	return false
}

// TODO version based validation
func validateTransitionStyle(s string) bool {

	// see 12.4.4

	for _, v := range []string{"Split", "Blinds", "Box", "Wipe", "Dissolve", "Glitter", "R"} {
		if s == v {
			return true
		}
	}

	// TODO
	// if version >= 1.5 the following values are also valid:
	// Fly, Push, Cover, Uncover, Fade

	return false
}

func validateTransitionDimension(s string) bool {

	// see 12.4.4

	for _, v := range []string{"H", "V"} {
		if s == v {
			return true
		}
	}

	return false
}

func validateTransitionDirectionOfMotion(s string) bool {

	// see 12.4.4

	for _, v := range []string{"I", "O"} {
		if s == v {
			return true
		}
	}

	return false
}

func validateBitsPerSample(i int) bool {

	for _, v := range []int{1, 2, 4, 8, 12, 16, 24, 32} {
		if i == v {
			return true
		}
	}

	return false
}

func validateGuideLineStyle(s string) bool {

	for _, v := range []string{"S", "D"} {
		if s == v {
			return true
		}
	}

	return false
}

func validateBaseState(s string) bool {

	for _, v := range []string{"ON", "OFF", "UNCHANGED"} {
		if s == v {
			return true
		}
	}

	return false
}

func validateListMode(s string) bool {

	for _, v := range []string{"AllPages", "VisiblePages"} {
		if s == v {
			return true
		}
	}

	return false
}

func validateOptContentConfigDictIntent(s string) bool {

	for _, v := range []string{"View", "Design", "All"} {
		if s == v {
			return true
		}
	}

	return false
}

func validateDocInfoDictTrapped(s string) bool {

	for _, v := range []string{"True", "False", "Unknown"} {
		if s == v {
			return true
		}
	}

	return false
}

///////////////////////////////////////////////////////////////////////////////////////////
// Exported stubs for package write.
// This should go away.

func StandardType1Font(s string) bool           { return validateStandardType1Font(s) }
func FileSpecString(s string) bool              { return validateFileSpecString(s) }
func URLString(s string) bool                   { return validateURLString(s) }
func FileSpecStringOrURLString(s string) bool   { return validateFileSpecStringOrURLString(s) }
func FontEncodingName(s string) bool            { return validateFontEncodingName(s) }
func StyleDict(dict types.PDFDict) bool         { return validateStyleDict(dict) }
func ExtGStateDictFont(arr types.PDFArray) bool { return validateExtGStateDictFont(arr) }
func BlendMode(s string) bool                   { return validateBlendMode(s) }
func SpotFunctionName(s string) bool            { return validateSpotFunctionName(s) }
func ICCBasedColorSpaceEntryN(i int) bool       { return validateICCBasedColorSpaceEntryN(i) }
func ColorKeyMaskArray(arr types.PDFArray) bool { return validateColorKeyMaskArray(arr) }
func RenderingIntent(s string) bool             { return validateRenderingIntent(s) }
func OPIVersion(s string) bool                  { return validateOPIVersion(s) }
func DeviceColorSpaceName(s string) bool        { return validateDeviceColorSpaceName(s) }
func SpecialColorSpaceName(s string) bool       { return validateSpecialColorSpaceName(s) }
func BitsPerCoordinate(i int) bool              { return validateBitsPerCoordinate(i) }
func BitsPerComponent(i int) bool               { return validateBitsPerComponent(i) }
func ProcedureSetName(s string) bool            { return validateProcedureSetName(s) }
func Rotate(i int) bool                         { return validateRotate(i) }
func NameTreeName(s string) bool                { return validateNameTreeName(s) }
func NamedAction(s string) bool                 { return validateNamedAction(s) }
func PageLabelDictEntryS(s string) bool         { return validatePageLabelDictEntryS(s) }
func Date(s string) bool                        { return validateDate(s) }
func ViewerPreferencesNonFullScreenPageMode(s string) bool {
	return validateViewerPreferencesNonFullScreenPageMode(s)
}
func ViewerPreferencesDirection(s string) bool  { return validateViewerPreferencesDirection(s) }
func Tabs(s string) bool                        { return validateTabs(s) }
func TransitionStyle(s string) bool             { return validateTransitionStyle(s) }
func TransitionDimension(s string) bool         { return validateTransitionDimension(s) }
func TransitionDirectionOfMotion(s string) bool { return validateTransitionDirectionOfMotion(s) }
func BitsPerSample(i int) bool                  { return validateBitsPerSample(i) }
func GuideLineStyle(s string) bool              { return validateGuideLineStyle(s) }
func BaseState(s string) bool                   { return validateBaseState(s) }
func ListMode(s string) bool                    { return validateListMode(s) }
func OptContentConfigDictIntent(s string) bool  { return validateOptContentConfigDictIntent(s) }
func DocInfoDictTrapped(s string) bool          { return validateDocInfoDictTrapped(s) }

///////////////////////////////////////////////////////////////////////////////////////////

func validateAnyEntry(xRefTable *types.XRefTable, dict *types.PDFDict, entryName string, required bool) (err error) {

	logInfoValidate.Printf("writeAnyEntry begin: entry=%s\n", entryName)

	entry, found := dict.Find(entryName)
	if !found || entry == nil {
		if required {
			err = errors.Errorf("writeAnyEntry: missing required entry: %s", entryName)
			return
		}
		logInfoValidate.Printf("writeAnyEntry end: entry %s not found or nil\n", entryName)
		return
	}

	indRef, ok := entry.(types.PDFIndirectRef)
	if !ok {
		logInfoValidate.Println("writeAnyEntry end")
		return
	}

	objNumber := indRef.ObjectNumber.Value()
	//genNumber := indRef.GenerationNumber.Value()

	obj, err := xRefTable.Dereference(indRef)
	if err != nil {
		return errors.Wrapf(err, "writeAnyEntry: unable to dereference object #%d", objNumber)
	}

	if obj == nil {
		return errors.Errorf("writeAnyEntry end: entry %s is nil", entryName)
	}

	switch obj.(type) {

	case types.PDFDict:
	case types.PDFStreamDict:
	case types.PDFArray:
	case types.PDFInteger:
	case types.PDFFloat:
	case types.PDFStringLiteral:
	case types.PDFHexLiteral:
	case types.PDFBoolean:
	case types.PDFName:

	default:
		err = errors.Errorf("writeAnyEntry: unsupported entry: %s", entryName)

	}

	logInfoValidate.Println("writeAnyEntry end")

	return
}

func validateArray(xRefTable *types.XRefTable, obj interface{}) (arrp *types.PDFArray, err error) {

	logInfoValidate.Println("validateArray begin")

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if obj == nil {
		err = errors.New("validateArray: missing object")
		return
	}

	arr, ok := obj.(types.PDFArray)
	if !ok {
		err = errors.New("validateArray: invalid type")
		return
	}

	arrp = &arr

	logInfoValidate.Println("validateArray end")

	return
}

func validateArrayEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string,
	required bool, sinceVersion types.PDFVersion, validate func(types.PDFArray) bool) (arrp *types.PDFArray, err error) {

	logInfoValidate.Printf("validateArrayEntry begin: entry=%s\n", entryName)

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			err = errors.Errorf("validateArrayEntry: dict=%s required entry=%s missing", dictName, entryName)
			return
		}
		logInfoValidate.Printf("validateArrayEntry end: optional entry %s not found or nil\n", entryName)
		return
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if obj == nil {
		if required {
			err = errors.Errorf("validateArrayEntry: dict=%s required entry=%s is nil", dictName, entryName)
			return
		}
		logInfoValidate.Printf("validateArrayEntry end: optional entry %s is nil\n", entryName)
		return
	}

	arr, ok := obj.(types.PDFArray)
	if !ok {
		err = errors.Errorf("validateArrayEntry: dict=%s entry=%s invalid type", dictName, entryName)
		return
	}

	// Version check
	if xRefTable.Version() < sinceVersion {
		err = errors.Errorf("validateArrayEntry: dict=%s entry=%s unsupported in version %s", dictName, entryName, xRefTable.VersionString())
		return
	}

	// Validation
	if validate != nil && !validate(arr) {
		err = errors.Errorf("validateArrayEntry: dict=%s entry=%s invalid dict entry", dictName, entryName)
	}

	arrp = &arr

	logInfoValidate.Printf("validateArrayEntry end: entry=%s\n", entryName)

	return
}

func validateBooleanEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string,
	required bool, sinceVersion types.PDFVersion, validate func(bool) bool) (boolp *types.PDFBoolean, err error) {

	logInfoValidate.Printf("validateBooleanEntry begin: entry=%s\n", entryName)

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			err = errors.Errorf("validateBooleanEntry: dict=%s required entry=%s missing", dictName, entryName)
			return
		}
		logInfoValidate.Printf("validateBooleanEntry end: entry %s is nil\n", entryName)
		return
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if obj == nil {
		if required {
			err = errors.Errorf("validateBooleanEntry: dict=%s required entry=%s missing", dictName, entryName)
			return
		}
		logInfoValidate.Printf("validateBooleanEntry end: entry %s is nil\n", entryName)
		return
	}

	b, ok := obj.(types.PDFBoolean)
	if !ok {
		err = errors.Errorf("validateBooleanEntry: dict=%s entry=%s invalid type", dictName, entryName)
		return
	}

	// Version check
	if xRefTable.Version() < sinceVersion {
		err = errors.Errorf("validateBooleanEntry: dict=%s entry=%s unsupported in version %s", dictName, entryName, xRefTable.VersionString())
		return
	}

	// Validation
	if validate != nil && !validate(b.Value()) {
		err = errors.Errorf("validateBooleanEntry: dict=%s entry=%s invalid name dict entry", dictName, entryName)
		return
	}

	boolp = &b

	logInfoValidate.Printf("validateBooleanEntry end: entry=%s\n", entryName)

	return
}

func validateBooleanArrayEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string,
	required bool, sinceVersion types.PDFVersion, validate func(types.PDFArray) bool) (arrp *types.PDFArray, err error) {

	logInfoValidate.Printf("validateBooleanArrayEntry begin: entry=%s\n", entryName)

	arrp, err = validateArrayEntry(xRefTable, dict, dictName, entryName, required, sinceVersion, validate)
	if err != nil || arrp == nil {
		return
	}

	for i, obj := range *arrp {

		obj, err = xRefTable.Dereference(obj)
		if err != nil {
			return
		}

		if obj == nil {
			continue
		}

		_, ok := obj.(types.PDFBoolean)
		if !ok {
			err = errors.Errorf("validateBooleanArrayEntry: dict=%s entry=%s invalid type at index %d\n", dictName, entryName, i)
			return
		}

	}

	logInfoValidate.Printf("validateBooleanArrayEntry end: entry=%s\n", entryName)

	return
}

func validateDateObject(xRefTable *types.XRefTable, obj interface{}, sinceVersion types.PDFVersion) (s types.PDFStringLiteral, err error) {
	return xRefTable.DereferenceStringLiteral(obj, sinceVersion, validateDate)
}

func validateDateEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string,
	required bool, sinceVersion types.PDFVersion) (s *types.PDFStringLiteral, err error) {

	logInfoValidate.Printf("validateDateEntry begin: entry=%s\n", entryName)

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			err = errors.Errorf("validateDateEntry: missing required entry: %s", entryName)
			return
		}
		logInfoValidate.Printf("validateDateEntry end: entry %s is nil\n", entryName)
		return
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if obj == nil {
		if required {
			err = errors.Errorf("validateDateEntry: dict=%s required entry=%s is nil", dictName, entryName)
			return
		}
		logInfoValidate.Printf("validateDateEntry end: optional entry %s is nil\n", entryName)
		return
	}

	date, ok := obj.(types.PDFStringLiteral)
	if !ok {
		err = errors.Errorf("validateDateEntry: dict=%s entry=%s invalid type", dictName, entryName)
		return
	}

	// Version check
	if xRefTable.Version() < sinceVersion {
		err = errors.Errorf("validateDateEntry: dict=%s entry=%s unsupported in version %s", dictName, entryName, xRefTable.VersionString())
		return
	}

	// Validation
	if ok := validateDate(date.Value()); !ok {
		err = errors.Errorf("validateDateEntry: dict=%s entry=%s invalid dict entry", dictName, entryName)
		return
	}

	s = &date

	logInfoValidate.Printf("validateDateEntry end: entry=%s\n", entryName)

	return
}

func validateDict(xRefTable *types.XRefTable, obj interface{}) (dictp *types.PDFDict, err error) {

	logInfoValidate.Println("validateDict begin")

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if obj == nil {
		err = errors.New("validateDict: missing object")
		return
	}

	dict, ok := obj.(types.PDFDict)
	if !ok {
		err = errors.New("validateDict: invalid type")
		return
	}

	dictp = &dict

	logInfoValidate.Println("validateDict end")

	return
}

func validateDictEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string,
	required bool, sinceVersion types.PDFVersion, validate func(types.PDFDict) bool) (dictp *types.PDFDict, err error) {

	logInfoValidate.Printf("validateDictEntry begin: entry=%s\n", entryName)

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			err = errors.Errorf("validateDictEntry: dict=%s required entry=%s missing", dictName, entryName)
			return
		}
		logInfoValidate.Printf("validateDictEntry end: optional entry %s not found or nil\n", entryName)
		return
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if obj == nil {
		if required {
			err = errors.Errorf("validateDictEntry: dict=%s required entry=%s is nil", dictName, entryName)
			return
		}
		logInfoValidate.Printf("validateDictEntry end: optional entry %s is nil\n", entryName)
		return
	}

	d, ok := obj.(types.PDFDict)
	if !ok {
		err = errors.Errorf("validateDictEntry: dict=%s entry=%s invalid type", dictName, entryName)
		return
	}

	// Version check
	if xRefTable.Version() < sinceVersion {
		err = errors.Errorf("validateDictEntry: dict=%s entry=%s unsupported in version %s", dictName, entryName, xRefTable.VersionString())
		return
	}

	// Validation
	if validate != nil && !validate(d) {
		err = errors.Errorf("validateDictEntry: dict=%s entry=%s invalid dict entry", dictName, entryName)
	}

	dictp = &d

	logInfoValidate.Printf("validateDictEntry end: entry=%s\n", entryName)

	return
}

func validateFloat(xRefTable *types.XRefTable, obj interface{}, validate func(float64) bool) (fp *types.PDFFloat, err error) {

	logInfoValidate.Println("validateFloat begin")

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if obj == nil {
		err = errors.New("validateFloat: missing object")
		return
	}

	f, ok := obj.(types.PDFFloat)
	if !ok {
		err = errors.New("validateFloat: invalid type")
		return
	}

	// Validation
	if validate != nil && !validate(f.Value()) {
		err = errors.Errorf("validateFloat: invalid float: %s\n", f)
		return
	}

	fp = &f

	logInfoValidate.Println("validateFloat end")

	return
}

func validateFloatEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string,
	required bool, sinceVersion types.PDFVersion, validate func(float64) bool) (fp *types.PDFFloat, err error) {

	logInfoValidate.Printf("validateFloatEntry begin: entry=%s\n", entryName)

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			err = errors.Errorf("validateFloatEntry: dict=%s required entry=%s missing", dictName, entryName)
			return
		}
		logInfoValidate.Printf("validateFloatEntry end: entry %s is nil\n", entryName)
		return
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if obj == nil {
		if required {
			err = errors.Errorf("validateFloatEntry: dict=%s required entry=%s is nil", dictName, entryName)
			return
		}
		logInfoValidate.Printf("validateFloatEntry end: optional entry %s is nil\n", entryName)
		return
	}

	f, ok := obj.(types.PDFFloat)
	if !ok {
		err = errors.Errorf("validateFloatEntry: dict=%s entry=%s invalid type", dictName, entryName)
		return
	}

	// Version check
	if xRefTable.Version() < sinceVersion {
		err = errors.Errorf("validateFloatEntry: dict=%s entry=%s unsupported in version %s", dictName, entryName, xRefTable.VersionString())
		return
	}

	// Validation
	if validate != nil && !validate(f.Value()) {
		err = errors.Errorf("validateFloatEntry: dict=%s entry=%s invalid dict entry", dictName, entryName)
		return
	}

	fp = &f

	logInfoValidate.Printf("validateFloatEntry end: entry=%s\n", entryName)

	return
}

func validateFunctionEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string,
	required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoValidate.Printf("validateFunctionEntry begin: entry=%s\n", entryName)

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			err = errors.Errorf("validateFunctionEntry: dict=%s required entry=%s missing", dictName, entryName)
			return
		}
		logInfoValidate.Printf("validateFunctionEntry end: optional entry %s not found or nil\n", entryName)
		return
	}

	// Version check
	if xRefTable.Version() < sinceVersion {
		err = errors.Errorf("validateFunctionEntry: dict=%s entry=%s unsupported in version %s", dictName, entryName, xRefTable.VersionString())
		return
	}

	err = validateFunction(xRefTable, obj)
	if err != nil {
		return
	}

	logInfoValidate.Printf("validateFunctionEntry end: entry=%s\n", entryName)

	return
}

func validateFunctionArrayEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string,
	required bool, sinceVersion types.PDFVersion, validate func(types.PDFArray) bool) (arrp *types.PDFArray, err error) {

	logInfoValidate.Printf("validateFunctionArrayEntry begin: entry=%s\n", entryName)

	arrp, err = validateArrayEntry(xRefTable, dict, dictName, entryName, required, sinceVersion, validate)
	if err != nil || arrp == nil {
		return
	}

	for _, obj := range *arrp {
		err = validateFunction(xRefTable, obj)
		if err != nil {
			return
		}
	}

	logInfoValidate.Printf("validateFunctionArrayEntry end: entry=%s\n", entryName)

	return
}

func validateIndRefEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string, required bool, sinceVersion types.PDFVersion) (indRefp *types.PDFIndirectRef, err error) {

	logInfoValidate.Printf("validateIndRefEntry begin: entry=%s\n", entryName)

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			err = errors.Errorf("validateIndRefEntry: dict=%s required entry=%s missing", dictName, entryName)
			return
		}
		logInfoValidate.Printf("validateIndRefEntry end: entry %s is nil\n", entryName)
		return
	}

	indRef, ok := obj.(types.PDFIndirectRef)
	if !ok {
		err = errors.Errorf("validateIndRefEntry: dict=%s entry=%s invalid type", dictName, entryName)
		return
	}

	// Version check
	if xRefTable.Version() < sinceVersion {
		err = errors.Errorf("validateIndRefEntry: dict=%s entry=%s unsupported in version %s", dictName, entryName, xRefTable.VersionString())
		return
	}

	indRefp = &indRef

	logInfoValidate.Printf("validateIndRefEntry end: entry=%s\n", entryName)

	return
}

func validateIndRefArrayEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string,
	required bool, sinceVersion types.PDFVersion, validate func(types.PDFArray) bool) (arrp *types.PDFArray, err error) {

	logInfoValidate.Printf("validateIndRefArrayEntry begin: entry=%s\n", entryName)

	arrp, err = validateArrayEntry(xRefTable, dict, dictName, entryName, required, sinceVersion, validate)
	if err != nil || arrp == nil {
		return
	}

	for i, obj := range *arrp {

		_, ok := obj.(types.PDFIndirectRef)
		if !ok {
			err = errors.Errorf("validateIndRefArrayEntry: invalid type at index %d\n", i)
			return
		}

	}

	logInfoValidate.Printf("validateIndRefArrayEntry end: entry=%s \n", entryName)

	return
}

func validateInteger(xRefTable *types.XRefTable, obj interface{}, validate func(int) bool) (ip *types.PDFInteger, err error) {

	logInfoValidate.Println("validateInteger begin")

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if obj == nil {
		err = errors.New("validateInteger: missing object")
		return
	}

	i, ok := obj.(types.PDFInteger)
	if !ok {
		err = errors.New("validateInteger: invalid type")
		return
	}

	// Validation
	if validate != nil && !validate(i.Value()) {
		err = errors.Errorf("validateInteger: invalid integer: %s\n", i)
		return
	}

	ip = &i

	logInfoValidate.Println("validateInteger end")

	return
}

func validateIntegerEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string,
	required bool, sinceVersion types.PDFVersion, validate func(int) bool) (ip *types.PDFInteger, err error) {

	logInfoValidate.Printf("validateIntegerEntry begin: entry=%s\n", entryName)

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			err = errors.Errorf("validateIntegerEntry: dict=%s required entry=%s missing", dictName, entryName)
			return
		}
		logInfoValidate.Printf("validateIntegerEntry end: optional entry %s not found or nil\n", entryName)
		return
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if obj == nil {
		if required {
			err = errors.Errorf("validateIntegerEntry: dict=%s required entry=%s is nil", dictName, entryName)
			return
		}
		logInfoValidate.Printf("validateIntegerEntry end: optional entry %s is nil\n", entryName)
		return
	}

	i, ok := obj.(types.PDFInteger)
	if !ok {
		err = errors.Errorf("validateIntegerEntry: dict=%s entry=%s invalid type", dictName, entryName)
		return
	}

	// Version check
	if xRefTable.Version() < sinceVersion {
		err = errors.Errorf("validateIntegerEntry: dict=%s entry=%s unsupported in version %s", dictName, entryName, xRefTable.VersionString())
		return
	}

	// Validation
	if validate != nil && !validate(i.Value()) {
		err = errors.Errorf("validateIntegerEntry: dict=%s entry=%s invalid dict entry", dictName, entryName)
		return
	}

	ip = &i

	logInfoValidate.Printf("validateIntegerEntry end: entry=%s\n", entryName)

	return
}

func validateIntegerArray(xRefTable *types.XRefTable, arr types.PDFArray) (arrp *types.PDFArray, err error) {

	logInfoValidate.Println("validateIntegerArray begin")

	arrp, err = validateArray(xRefTable, arr)
	if err != nil {
		return
	}

	if arrp == nil {
		return
	}

	for i, obj := range *arrp {

		obj, err = xRefTable.Dereference(obj)
		if err != nil {
			return
		}

		if obj == nil {
			continue
		}

		switch obj.(type) {

		case types.PDFInteger:
			// no further processing.

		default:
			err = errors.Errorf("validateIntegerArray: invalid type at index %d\n", i)
		}

	}

	logInfoValidate.Println("validateIntegerArray end")

	return
}

func validateIntegerArrayEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string,
	required bool, sinceVersion types.PDFVersion, validate func(types.PDFArray) bool) (arrp *types.PDFArray, err error) {

	logInfoValidate.Printf("validateIntegerArrayEntry begin: entry=%s\n", entryName)

	arrp, err = validateArrayEntry(xRefTable, dict, dictName, entryName, required, sinceVersion, validate)
	if err != nil || arrp == nil {
		return
	}

	for i, obj := range *arrp {

		obj, err = xRefTable.Dereference(obj)
		if err != nil {
			return
		}

		if obj == nil {
			continue
		}

		_, ok := obj.(types.PDFInteger)
		if !ok {
			err = errors.Errorf("validateIntegerArrayEntry: dict=%s entry=%s invalid type at index %d\n", dictName, entryName, i)
			return
		}

	}

	logInfoValidate.Printf("validateIntegerArrayEntry end: entry=%s\n", entryName)

	return
}

func validateName(xRefTable *types.XRefTable, obj interface{}, validate func(string) bool) (namep *types.PDFName, err error) {

	// TODO written irrelevant?

	logInfoValidate.Println("validateName begin")

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if obj == nil {
		err = errors.New("validateName: missing object")
		return
	}

	name, ok := obj.(types.PDFName)
	if !ok {
		err = errors.New("validateName: invalid type")
		return
	}

	// Validation
	if validate != nil && !validate(name.String()) {
		err = errors.Errorf("validateName: invalid name: %s\n", name)
		return
	}

	namep = &name

	logInfoValidate.Println("validateName end")

	return
}

func validateNameEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string,
	required bool, sinceVersion types.PDFVersion, validate func(string) bool) (namep *types.PDFName, err error) {

	logInfoValidate.Printf("validateNameEntry begin: entry=%s\n", entryName)

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			err = errors.Errorf("validateNameEntry: dict=%s required entry=%s missing", dictName, entryName)
			return
		}
		logInfoValidate.Printf("validateNameEntry end: optional entry %s not found or nil\n", entryName)
		return
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if obj == nil {
		if required {
			err = errors.Errorf("validateNameEntry: dict=%s required entry=%s is nil", dictName, entryName)
			return
		}
		logInfoValidate.Printf("validateNameEntry end: optional entry %s is nil\n", entryName)
		return
	}

	name, ok := obj.(types.PDFName)
	if !ok {
		err = errors.Errorf("validateNameEntry: dict=%s entry=%s invalid type", dictName, entryName)
		return
	}

	// Version check
	if xRefTable.Version() < sinceVersion {
		err = errors.Errorf("validateNameEntry: dict=%s entry=%s unsupported in version %s", dictName, entryName, xRefTable.VersionString())
		return
	}

	// Validation
	if validate != nil && !validate(name.String()) {
		err = errors.Errorf("validateNameEntry: dict=%s entry=%s invalid dict entry", dictName, entryName)
		return
	}

	namep = &name

	logInfoValidate.Printf("validateNameEntry end: entry=%s\n", entryName)

	return
}

func validateNameArray(xRefTable *types.XRefTable, obj interface{}) (arrp *types.PDFArray, err error) {

	logInfoValidate.Println("validateNameArray begin")

	arrp, err = validateArray(xRefTable, obj)
	if err != nil || arrp == nil {
		return
	}

	for i, obj := range *arrp {

		obj, err = xRefTable.Dereference(obj)
		if err != nil {
			return
		}

		if obj == nil {
			continue
		}

		_, ok := obj.(types.PDFName)
		if !ok {
			err = errors.Errorf("validateNameArray: invalid type at index %d\n", i)
			return
		}

	}

	logInfoValidate.Println("validateNameArray end")

	return
}

func validateNameArrayEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string,
	required bool, sinceVersion types.PDFVersion, validate func(string) bool) (arrp *types.PDFArray, err error) {

	logInfoValidate.Printf("validateNameArrayEntry begin: entry=%s\n", entryName)

	arrp, err = validateArrayEntry(xRefTable, dict, dictName, entryName, required, sinceVersion, nil)
	if err != nil || arrp == nil {
		return
	}

	for i, obj := range *arrp {

		obj, err = xRefTable.Dereference(obj)
		if err != nil {
			return
		}

		if obj == nil {
			continue
		}

		name, ok := obj.(types.PDFName)
		if !ok {
			err = errors.Errorf("validateNameArrayEntry: dict=%s entry=%s invalid type at index %d\n", dictName, entryName, i)
			return
		}

		if validate != nil && !validate(name.String()) {
			err = errors.Errorf("validateNameArrayEntry: dict=%s entry=%s invalid entry at index %d\n", dictName, entryName, i)
			return
		}

	}

	logInfoValidate.Printf("validateNameArrayEntry end: entry=%s\n", entryName)

	return
}

func validateNumber(xRefTable *types.XRefTable, obj interface{}) (n interface{}, err error) {

	logInfoValidate.Println("validateNumber begin")

	n, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if n == nil {
		err = errors.New("validateNumber: missing object")
		return
	}

	switch n.(type) {

	case types.PDFInteger:
		// no further processing.

	case types.PDFFloat:
		// no further processing.

	default:
		err = errors.New("validateNumber: invalid type")

	}

	logInfoValidate.Println("validateNumber end ")

	return
}

func validateNumberEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string,
	required bool, sinceVersion types.PDFVersion, validate func(interface{}) bool) (obj interface{}, err error) {

	logInfoValidate.Printf("validateNumberEntry begin: entry=%s\n", entryName)

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			err = errors.Errorf("validateNumberEntry: dict=%s required entry=%s missing", dictName, entryName)
			return
		}
		logInfoValidate.Printf("validateNumberEntry end: optional entry %s not found or nil\n", entryName)
		return
	}

	// Version check
	if xRefTable.Version() < sinceVersion {
		err = errors.Errorf("validateNumberEntry: dict=%s entry=%s unsupported in version %s", dictName, entryName, xRefTable.VersionString())
		return
	}

	obj, err = validateNumber(xRefTable, obj)
	if err != nil {
		return
	}

	// Validation
	if validate != nil && !validate(obj) {
		err = errors.Errorf("validateFloatEntry: dict=%s entry=%s invalid dict entry", dictName, entryName)
		return
	}

	logInfoValidate.Printf("validateNumberEntry end: entry=%s\n", entryName)

	return
}

func validateNumberArray(xRefTable *types.XRefTable, obj interface{}) (arrp *types.PDFArray, err error) {

	logInfoValidate.Println("validateNumberArray begin")

	arrp, err = validateArray(xRefTable, obj)
	if err != nil || arrp == nil {
		return
	}

	for i, obj := range *arrp {

		obj, err = xRefTable.Dereference(obj)
		if err != nil {
			return
		}

		if obj == nil {
			continue
		}

		switch obj.(type) {

		case types.PDFInteger:
			// no further processing.

		case types.PDFFloat:
			// no further processing.

		default:
			err = errors.Errorf("validateNumberArray: invalid type at index %d\n", i)
			return
		}

	}

	logInfoValidate.Println("validateNumberArray end")

	return
}

func validateNumberArrayEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string,
	required bool, sinceVersion types.PDFVersion, validate func(types.PDFArray) bool) (arrp *types.PDFArray, err error) {

	logInfoValidate.Printf("validateNumberArrayEntry begin: entry=%s\n", entryName)

	arrp, err = validateArrayEntry(xRefTable, dict, dictName, entryName, required, sinceVersion, validate)
	if err != nil || arrp == nil {
		return
	}

	for i, obj := range *arrp {

		obj, err = xRefTable.Dereference(obj)
		if err != nil {
			return
		}

		if obj == nil {
			continue
		}

		switch obj.(type) {

		case types.PDFInteger:
			// no further processing.

		case types.PDFFloat:
			// no further processing.

		default:
			err = errors.Errorf("validateNumberArrayEntry: invalid type at index %d\n", i)
			return
		}

	}

	logInfoValidate.Printf("validateNumberArrayEntry end: entry=%s\n", entryName)

	return
}

func validateRectangleEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string,
	required bool, sinceVersion types.PDFVersion, validate func(types.PDFArray) bool) (arrp *types.PDFArray, err error) {

	logInfoValidate.Printf("validateRectangleEntry begin: entry=%s\n", entryName)

	arrp, err = validateNumberArrayEntry(xRefTable, dict, dictName, entryName, required, sinceVersion, func(arr types.PDFArray) bool { return len(arr) == 4 })
	if err != nil {
		return
	}

	if arrp == nil {
		return
	}

	if validate != nil && !validate(*arrp) {
		err = errors.Errorf("validateRectangleEntry: dict=%s entry=%s invalid rectangle entry", dictName, entryName)
		return
	}

	logInfoValidate.Printf("validateRectangleEntry end: entry=%s\n", entryName)

	return
}

func validateStreamDict(xRefTable *types.XRefTable, obj interface{}) (streamDictp *types.PDFStreamDict, err error) {

	logInfoValidate.Println("validateStreamDict begin")

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if obj == nil {
		err = errors.New("validateStreamDict: missing object")
		return
	}

	streamDict, ok := obj.(types.PDFStreamDict)
	if !ok {
		err = errors.New("validateStreamDict: invalid type")
		return
	}

	streamDictp = &streamDict

	logInfoValidate.Println("validateStreamDict endobj")

	return
}

func validateStreamDictEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string,
	required bool, sinceVersion types.PDFVersion, validate func(types.PDFStreamDict) bool) (sdp *types.PDFStreamDict, err error) {

	logInfoValidate.Printf("validateStreamDictEntry begin: entry=%s\n", entryName)

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			err = errors.Errorf("validateStreamDictEntry: dict=%s required entry=%s missing", dictName, entryName)
			return
		}
		logInfoValidate.Printf("validateStreamDictEntry end: optional entry %s not found or nil\n", entryName)
		return
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if obj == nil {
		if required {
			err = errors.Errorf("validateStreamDictEntry: dict=%s required entry=%s is nil", dictName, entryName)
			return
		}
		logInfoValidate.Printf("validateStreamDictEntry end: optional entry %s is nil\n", entryName)
		return
	}

	sd, ok := obj.(types.PDFStreamDict)
	if !ok {
		err = errors.Errorf("validateStreamDictEntry: dict=%s entry=%s invalid type", dictName, entryName)
		return
	}

	// Version check
	if xRefTable.Version() < sinceVersion {
		err = errors.Errorf("validateStreamDictEntry: dict=%s entry=%s unsupported in version %s", dictName, entryName, xRefTable.VersionString())
		return
	}

	// Validation
	if validate != nil && !validate(sd) {
		err = errors.Errorf("validateStreamDictEntry: dict=%s entry=%s invalid dict entry", dictName, entryName)
	}

	sdp = &sd

	logInfoValidate.Printf("validateStreamDictEntry end: entry=%s\n", entryName)

	return
}

func validateStringEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string,
	required bool, sinceVersion types.PDFVersion, validate func(string) bool) (s *string, err error) {

	logInfoValidate.Printf("validateStringEntry begin: entry=%s\n", entryName)

	obj, found := dict.Find(entryName)
	if !found || obj == nil {
		if required {
			err = errors.Errorf("validateStringEntry: dict=%s required entry=%s missing", dictName, entryName)
			return
		}
		logInfoValidate.Printf("validateStringEntry end: entry %s is nil\n", entryName)
		return
	}

	obj, err = xRefTable.Dereference(obj)
	if err != nil {
		return
	}

	if obj == nil {
		if required {
			err = errors.Errorf("validateStringEntry: dict=%s required entry=%s is nil", dictName, entryName)
			return
		}
		logInfoValidate.Printf("validateStringEntry end: optional entry %s is nil\n", entryName)
		return
	}

	var str string

	switch obj := obj.(type) {

	case types.PDFStringLiteral:
		str = obj.Value()

	case types.PDFHexLiteral:
		str = obj.Value()

	default:
		err = errors.Errorf("validateStringEntry: dict=%s entry=%s invalid type", dictName, entryName)
		return
	}

	// Version check
	if xRefTable.Version() < sinceVersion {
		err = errors.Errorf("validateStringEntry: dict=%s entry=%s unsupported in version %s", dictName, entryName, xRefTable.VersionString())
		return
	}

	// Validation
	if validate != nil && !validate(str) {
		err = errors.Errorf("validateStringEntry: dict=%s entry=%s invalid dict entry", dictName, entryName)
		return
	}

	s = &str

	logInfoValidate.Printf("validateStringEntry end: entry=%s\n", entryName)

	return
}

func validateStringArrayEntry(xRefTable *types.XRefTable, dict *types.PDFDict, dictName string, entryName string,
	required bool, sinceVersion types.PDFVersion, validate func(types.PDFArray) bool) (arrp *types.PDFArray, err error) {

	logInfoValidate.Printf("validateStringArrayEntry begin: entry=%s\n", entryName)

	arrp, err = validateArrayEntry(xRefTable, dict, dictName, entryName, required, sinceVersion, validate)
	if err != nil || arrp == nil {
		return
	}

	for i, obj := range *arrp {

		obj, err = xRefTable.Dereference(obj)
		if err != nil {
			return
		}

		if obj == nil {
			continue
		}

		switch obj.(type) {

		case types.PDFStringLiteral:
			// no further processing.

		case types.PDFHexLiteral:
			// no further processing

		default:
			err = errors.Errorf("validateStringArrayEntry: invalid type at index %d\n", i)
			return
		}

	}

	logInfoValidate.Printf("validateStringArrayEntry end: entry=%s\n", entryName)

	return
}
