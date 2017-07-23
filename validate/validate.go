// Package validate contains validation code for ISO 32000-1:2008.
//
// There is low level validation and validation against the PDF spec for each of the defined PDF object types.
package validate

import (
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hhrutter/pdflib/types"
)

var logDebugValidate, logInfoValidate, logErrorValidate *log.Logger

func init() {

	logDebugValidate = log.New(ioutil.Discard, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
	logInfoValidate = log.New(ioutil.Discard, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
	logErrorValidate = log.New(os.Stdout, "ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
}

// Verbose controls logging output.
func Verbose(verbose bool) {

	if verbose {
		//logDebugValidate = log.New(os.Stdout, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
		logInfoValidate = log.New(os.Stdout, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	} else {
		//logDebugValidate = log.New(ioutil.Discard, "DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
		logInfoValidate = log.New(ioutil.Discard, "INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	}
}

func memberOf(s string, list []string) bool {

	for _, v := range list {
		if s == v {
			return true
		}
	}
	return false
}

func intMemberOf(i int, list []int) bool {
	for _, v := range list {
		if i == v {
			return true
		}
	}
	return false
}

func validateStandardType1Font(s string) bool {

	return memberOf(s, []string{"Times-Roman", "Times-Bold", "Times-Italic", "Times-BoldItalic",
		"Helvetica", "Helvetica-Bold", "Helvetica-Oblique", "Helvetica-BoldOblique",
		"Courier", "Courier-Bold", "Courier-Oblique", "Courier-BoldOblique",
		"Symbol", "ZapfDingbats"})
}

func validateFileSpecString(s string) bool {

	// see 7.11.2
	// The standard format for representing a simple file specification in string form divides the string into component substrings
	// separated by the SOLIDUS character (2Fh) (/). The SOLIDUS is a generic component separator that shall be mapped to the appropriate
	// platform-specific separator when generating a platform-dependent file name. Any of the components may be empty.
	// If a component contains one or more literal SOLIDI, each shall be preceded by a REVERSE SOLIDUS (5Ch) (\), which in turn shall be
	// preceded by another REVERSE SOLIDUS to indicate that it is part of the string and not an escape character.
	//
	// EXAMPLE ( in\\/out )
	// represents the file name in/out

	// I have not seen an instance of a single file spec string that actually complies with this definition and uses
	// the double reverse solidi in front of the solidus, because of that we simply
	return true
}

func validateURLString(s string) bool {

	// RFC1738 compliant URL, see 7.11.5

	_, err := url.ParseRequestURI(s)

	return err == nil
}

func validateFontEncodingName(s string) bool {

	return memberOf(s, []string{"MacRomanEncoding", "MacExpertEncoding", "WinAnsiEncoding"})
}

func validateBlendMode(s string) bool {

	// see 11.3.5; table 136

	return memberOf(s, []string{"None", "Normal", "Compatible", "Multiply", "Screen", "Overlay", "Darken", "Lighten",
		"ColorDodge", "ColorBurn", "HardLight", "SoftLight", "Difference", "Exclusion",
		"Hue", "Saturation", "Color", "Luminosity"})
}

func validateSpotFunctionName(s string) bool {

	return memberOf(s, []string{
		"SimpleDot", "InvertedSimpleDot", "DoubleDot", "InvertedDoubleDot", "CosineDot",
		"Double", "InvertedDouble", "Line", "LineX", "LineY"})
}

func validateICCBasedColorSpaceEntryN(i int) bool {

	return intMemberOf(i, []int{1, 3, 4})
}

func validateRenderingIntent(s string) bool {

	// see 8.6.5.8

	return memberOf(s, []string{"AbsoluteColorimetric", "RelativeColorimetric", "Saturation", "Perceptual"})
}

func validateOPIVersion(s string) bool {

	return memberOf(s, []string{"1.3", "2.0"})
}

func validateDeviceColorSpaceName(s string) bool {

	return memberOf(s, []string{"DeviceGray", "DeviceRGB", "DeviceCMYK"})
}

func validateSpecialColorSpaceName(s string) bool {

	return memberOf(s, []string{"Pattern"})
}

func validateBitsPerCoordinate(i int) bool {

	return intMemberOf(i, []int{1, 2, 4, 8, 12, 16, 24, 32})
}

func validateBitsPerComponent(i int) bool {

	return intMemberOf(i, []int{1, 2, 4, 8, 12, 16})
}

func validateRotate(i int) bool {

	return intMemberOf(i, []int{0, 90, 4, 180, 270})
}

func validateNameTreeName(s string) bool {

	return memberOf(s, []string{"Dests", "AP", "JavaScript", "Pages", "Templates", "IDS",
		"URLS", "EmbeddedFiles", "AlternatePresentations", "Renditions"})
}

func validateNamedAction(s string) bool {

	if memberOf(s, []string{"NextPage", "PrevPage", "FirstPage", "Lastpage"}) {
		return true
	}

	// Some known non standard named actions
	if memberOf(s, []string{"GoToPage", "GoBack", "GoForward", "Find"}) {
		return true
	}

	return false
}

func validatePageLabelDictEntryS(s string) bool {

	// see 12.4.2 Page Labels

	return memberOf(s, []string{"D", "R", "r", "A", "a"})
}

func validateViewerPreferencesNonFullScreenPageMode(s string) bool {

	return memberOf(s, []string{"UseNone", "UseOutlines", "UseThumbs", "UseOC"})
}

func validateViewerPreferencesDirection(s string) bool {

	return memberOf(s, []string{"L2R", "R2L"})
}

func validateTransitionStyle(s string) bool {

	// see 12.4.4

	return memberOf(s, []string{"Split", "Blinds", "Box", "Wipe", "Dissolve", "Glitter", "R"})
}

func validateTransitionStyleV15(s string) bool {

	if validateTransitionStyle(s) {
		return true
	}

	return memberOf(s, []string{"Fly", "Push", "Cover", "Uncover", "Fade"})
}

func validateTransitionDimension(s string) bool {

	// see 12.4.4

	return memberOf(s, []string{"H", "V"})
}

func validateTransitionDirectionOfMotion(s string) bool {

	// see 12.4.4

	return memberOf(s, []string{"I", "O"})
}

func validateBitsPerSample(i int) bool {

	return intMemberOf(i, []int{1, 2, 4, 8, 12, 16, 24, 32})
}

func validateGuideLineStyle(s string) bool {

	return memberOf(s, []string{"S", "D"})
}

func validateBaseState(s string) bool {

	return memberOf(s, []string{"ON", "OFF", "UNCHANGED"})
}

func validateListMode(s string) bool {

	return memberOf(s, []string{"AllPages", "VisiblePages"})
}

func validateOptContentConfigDictIntent(s string) bool {

	return memberOf(s, []string{"View", "Design", "All"})
}

func validateDocInfoDictTrapped(s string) bool {

	return memberOf(s, []string{"True", "False", "Unknown"})
}

func validateAdditionalAction(s, source string) bool {

	switch source {

	case "root":
		if memberOf(s, []string{"WC", "WS", "DS", "WP", "DP"}) {
			return true
		}

	case "page":
		if memberOf(s, []string{"O", "C"}) {
			return true
		}

	case "fieldOrAnnot":
		// A terminal acro field may be merged with a widget annotation.
		fieldOptions := []string{"K", "F", "V", "C"}
		annotOptions := []string{"E", "X", "D", "U", "Fo", "Bl", "PO", "PC", "PV", "Pl"}
		options := append(fieldOptions, annotOptions...)
		if memberOf(s, options) {
			return true
		}

	}

	return false
}

func validateAnnotationHighlightingMode(s string) bool {

	return memberOf(s, []string{"N", "I", "O", "P", "T", "A"})
}

func validateAnnotationState(s string) bool {

	return memberOf(s, []string{"None", "Unmarked"})
}

func validateAnnotationStateModel(s string) bool {

	return memberOf(s, []string{"Marked", "Review"})
}

func validateBorderStyle(s string) bool {

	return memberOf(s, []string{"S", "D", "B", "I", "U", "A"})
}

func validateIconFitDict(s string) bool {

	return memberOf(s, []string{"A", "B", "S", "N"})
}

func validateIntentOfFreeTextAnnotation(s string) bool {

	return memberOf(s, []string{"FreeText", "FreeTextCallout", "FreeTextTypeWriter", "FreeTextTypewriter"})
}

func validateVisibilityPolicy(s string) bool {

	return memberOf(s, []string{"AllOn", "AnyOn", "AnyOff", "AllOff"})
}

func validateAcroFieldType(s string) bool {

	return memberOf(s, []string{"Btn", "Tx", "Ch", "Sig"})
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

func Date(s string) bool { return validateDate(s) }
