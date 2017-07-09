package validate

import (
	"github.com/hhrutter/pdflib/types"
	"github.com/pkg/errors"
)

func validatePieceDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	logInfoValidate.Println("*** validatePieceDict begin ***")

	for _, obj := range dict.Dict {

		dict, err = xRefTable.DereferenceDict(obj)
		if err != nil {
			return
		}

		if dict == nil {
			logInfoValidate.Println("validatePieceDict: object is nil.")
			continue
		}

		_, err = validateDateEntry(xRefTable, dict, "PagePieceDataDict", "LastModified", REQUIRED, types.V10)
		if err != nil {
			return err
		}

		err = validateAnyEntry(xRefTable, dict, "Private", OPTIONAL)
		if err != nil {
			return err
		}

	}

	logInfoValidate.Println("*** validatePieceDict end ***")

	return
}

func validatePieceInfo(xRefTable *types.XRefTable, dict *types.PDFDict, required bool, sinceVersion types.PDFVersion) (hasPieceInfo bool, err error) {

	// 14.5 Page-Piece Dictionaries

	logInfoValidate.Println("*** validatePieceInfo begin ***")

	pieceDict, err := validateDictEntry(xRefTable, dict, "dict", "PieceInfo", required, sinceVersion, nil)
	if err != nil {
		return
	}

	if pieceDict == nil {
		logInfoValidate.Println("validatePieceInfo end: pieceDict is nil.")
		return
	}

	// Version check
	if xRefTable.Version() < sinceVersion {
		err = errors.Errorf("validatePieceInfo: unsupported in version %s.\n", xRefTable.VersionString())
		return
	}

	err = validatePieceDict(xRefTable, pieceDict)
	if err != nil {
		return
	}

	hasPieceInfo = true

	logInfoValidate.Println("*** validatePieceInfo end ***")

	return
}

func validateMetadata(xRefTable *types.XRefTable, dict *types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoValidate.Println("*** validateMetadata begin ***")

	// => 14.3 Metadata
	// In general, any PDF stream or dictionary may have metadata attached to it
	// as long as the stream or dictionary represents an actual information resource,
	// as opposed to serving as an implementation artifact.
	// Some PDF constructs are considered implementational, and hence may not have associated metadata.

	streamDict, err := validateStreamDictEntry(xRefTable, dict, "dict", "Metadata", required, sinceVersion, nil)
	if err != nil {
		return
	}

	if streamDict == nil {
		logInfoValidate.Printf("validateMetadata end: streamDict is nil\n")
		return
	}

	if t := (*streamDict).Type(); t != nil && *t != "Metadata" {
		err = errors.New("validateMetadata: corrupt metadata stream dict")
	}

	if subt := (*streamDict).Subtype(); subt != nil && *subt != "XML" {
		err = errors.New("validateMetadata: corrupt metadata stream dict")
	}

	logInfoValidate.Println("*** validateMetadata end ***")

	return
}

func validatePageLabels(xRefTable *types.XRefTable, rootDict *types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	// optional since PDF 1.3
	// => 7.9.7 Number Trees, 12.4.2 Page Labels

	// PDFDict or indirect ref to PDFDict
	// <Nums, [0 (170 0 R)]> or indirect ref

	logInfoValidate.Println("*** validatePageLabels begin ***")

	indRef := rootDict.IndirectRefEntry("PageLabels")
	if indRef == nil {
		if required {
			err = errors.Errorf("validatePageLabels: required entry \"PageLabels\" missing")
			return
		}
		logInfoValidate.Println("validatePageLabels end: indRef is nil.")
		return
	}

	// Version check
	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validatePageLabels: unsupported in version %s.\n", xRefTable.VersionString())
	}

	err = validateNumberTree(xRefTable, "PageLabel", *indRef, true)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validatePageLabels end ***")

	return
}

////////////////////////////////////////////////////////////////////////////////////////////////

// TODO implement
func validateExtensions(xRefTable *types.XRefTable, rootDict *types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	// => 7.12 Extensions Dictionary

	logInfoValidate.Println("*** validateExtensions begin ***")

	dict, err := validateDictEntry(xRefTable, rootDict, "rootDict", "Extensions", required, sinceVersion, nil)
	if err != nil {
		return
	}

	if dict == nil {
		logInfoValidate.Println("validateExtensions end: dict is nil.")
		return
	}

	// Version check
	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateExtensions: unsupported in version %s.\n", xRefTable.VersionString())
	}

	err = errors.New("*** validateExtensions: not supported ***")

	//source.XRefTable.Stats.AddRootAttr(types.RootExtensions)

	logInfoValidate.Println("*** validateExtensions end ***")

	return
}

func validateNames(xRefTable *types.XRefTable, rootDict *types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	// => 7.7.4 Name Dictionary

	// all values are name trees or indirect refs.

	/*
		<Kids, [(86 0 R)]>

		86:
		<Limits, [(F1) (P.9)]>
		<Names, [(F1) (87 0 R) (F2) ...

		87: named destination dict
		<D, [(158 0 R) XYZ]>
	*/

	logInfoValidate.Println("*** validateNames begin ***")

	dict, err := validateDictEntry(xRefTable, rootDict, "rootDict", "Names", required, sinceVersion, nil)
	if err != nil {
		return
	}

	if dict == nil {
		logInfoValidate.Println("validateNames end: dict is nil.")
		return
	}

	// Version check
	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateNames: unsupported in version %s.\n", xRefTable.VersionString())
	}

	for treeName, value := range dict.Dict {

		if ok := validateNameTreeName(treeName); !ok {
			return errors.Errorf("validateNames: unknown name tree name: %s\n", treeName)
		}

		indRef, ok := value.(types.PDFIndirectRef)
		if !ok {
			// TODO: All values are name trees or indirect refs.
			return errors.New("validateNames: name tree must be indirect ref")
		}

		logInfoValidate.Printf("validating Nametree: %s\n", treeName)
		err = validateNameTree(xRefTable, treeName, indRef, true)
		if err != nil {
			return
		}

	}

	logInfoValidate.Println("*** validateNames end ***")

	return
}

func validateNamedDestinations(xRefTable *types.XRefTable, rootDict *types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	// => 12.3.2.3 Named Destinations

	// indRef or dict with destination array values.

	logInfoValidate.Println("*** validateNamedDestinations begin ***")

	dict, err := validateDictEntry(xRefTable, rootDict, "rootDict", "Dests", required, sinceVersion, nil)
	if err != nil {
		return
	}

	if dict == nil {
		logInfoValidate.Println("validateNamedDestinations end: dict is nil.")
		return
	}

	// Version check
	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateNamedDestinations: unsupported in version %s.\n", xRefTable.VersionString())
	}

	for _, value := range dict.Dict {
		err = validateDestination(xRefTable, value)
		if err != nil {
			return
		}
	}

	logInfoValidate.Println("*** validateNamedDestinations end ***")

	return
}

func validateViewerPreferences(xRefTable *types.XRefTable, rootDict *types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	// => 12.2 Viewer Preferences

	logInfoValidate.Println("*** validateViewerPreferences begin ***")

	dict, err := validateDictEntry(xRefTable, rootDict, "rootDict", "ViewerPreferences", required, sinceVersion, nil)
	if err != nil {
		return
	}

	if dict == nil {
		logInfoValidate.Println("validateViewerPreferences end: dict is nil.")
		return
	}

	_, err = validateBooleanEntry(xRefTable, dict, "ViewerPreferences", "HideToolbar", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	_, err = validateBooleanEntry(xRefTable, dict, "ViewerPreferences", "HideMenubar", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	_, err = validateBooleanEntry(xRefTable, dict, "ViewerPreferences", "HideWindowUI", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	_, err = validateBooleanEntry(xRefTable, dict, "ViewerPreferences", "FitWindow", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	_, err = validateBooleanEntry(xRefTable, dict, "ViewerPreferences", "CenterWindow", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	sinceVersion = types.V14
	if xRefTable.ValidationMode == types.ValidationRelaxed {
		sinceVersion = types.V10
	}
	_, err = validateBooleanEntry(xRefTable, dict, "ViewerPreferences", "DisplayDocTitle", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return
	}

	_, err = validateNameEntry(xRefTable, dict, "ViewerPreferences", "NonFullScreenPageMode", OPTIONAL, types.V10, validateViewerPreferencesNonFullScreenPageMode)
	if err != nil {
		return
	}

	_, err = validateNameEntry(xRefTable, dict, "ViewerPreferences", "Direction", OPTIONAL, types.V13, validateViewerPreferencesDirection)
	if err != nil {
		return
	}

	_, err = validateNameEntry(xRefTable, dict, "ViewerPreferences", "ViewArea", OPTIONAL, types.V14, nil)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateViewerPreferences end ***")

	return
}

func validatePageLayout(xRefTable *types.XRefTable, rootDict *types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoValidate.Println("*** validatePageLayout begin ***")

	_, err = validateNameEntry(xRefTable, rootDict, "rootDict", "PageLayout", required, sinceVersion, nil)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validatePageLayout end ***")

	return
}

func validatePageMode(xRefTable *types.XRefTable, rootDict *types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoValidate.Println("*** validatePageMode begin ***")

	_, err = validateNameEntry(xRefTable, rootDict, "rootDict", "PageMode", required, sinceVersion, nil)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validatePageMode end ***")

	return
}

func validateBeadDict(xRefTable *types.XRefTable, indRefBeadDict, indRefThreadDict, indRefPreviousBead, indRefLastBead types.PDFIndirectRef) (err error) {

	objNumber := indRefBeadDict.ObjectNumber.Value()

	logInfoValidate.Printf("*** validateBeadDict begin: objectNumber=%d ***", objNumber)

	dictName := "beadDict"
	sinceVersion := types.V10

	dict, err := xRefTable.DereferenceDict(indRefBeadDict)
	if err != nil {
		return
	}

	if dict == nil {
		err = errors.Errorf("validateBeadDict: obj#%d missing dict", objNumber)
		return
	}

	// Validate optional entry Type, must be "Bead".
	_, err = validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "Bead" })
	if err != nil {
		return
	}

	// Validate entry T, must refer to threadDict.
	indRefT, err := validateIndRefEntry(xRefTable, dict, dictName, "T", REQUIRED, sinceVersion)
	if err != nil {
		return
	}

	if !indRefT.Equals(indRefThreadDict) {
		err = errors.Errorf("validateBeadDict: obj#%d invalid entry T (backpointer to ThreadDict)", objNumber)
		return
	}

	// Validate required entry R, must be rectangle.
	_, err = validateRectangleEntry(xRefTable, dict, dictName, "R", REQUIRED, sinceVersion, nil)
	if err != nil {
		return
	}

	// Validate required entry P, must be indRef to pageDict.
	pageDict, err := validateDictEntry(xRefTable, dict, dictName, "P", REQUIRED, sinceVersion, nil)
	if err != nil || pageDict == nil || pageDict.Type() == nil || *pageDict.Type() != "Page" {
		return errors.Errorf("validateBeadDict: obj#%d invalid entry P, no page dict", objNumber)
	}

	// Validate required entry V, must refer to previous bead.
	previousBeadIndRef, err := validateIndRefEntry(xRefTable, dict, dictName, "V", REQUIRED, sinceVersion)
	if err != nil {
		return
	}

	if !previousBeadIndRef.Equals(indRefPreviousBead) {
		err = errors.Errorf("validateBeadDict: obj#%d invalid entry V, corrupt previous Bead indirect reference", objNumber)
		return
	}

	// Validate required entry N, must refer to last bead.
	nextBeadIndRef, err := validateIndRefEntry(xRefTable, dict, dictName, "N", REQUIRED, sinceVersion)
	if err != nil {
		return
	}

	// Recurse until next bead equals last bead.
	if !nextBeadIndRef.Equals(indRefLastBead) {
		err = validateBeadDict(xRefTable, indRefBeadDict, indRefThreadDict, indRefBeadDict, indRefLastBead)
		if err != nil {
			return err
		}
	}

	logInfoValidate.Printf("*** validateBeadDict end: objectNumber=%d ***", objNumber)

	return
}

func validateFirstBeadDict(xRefTable *types.XRefTable, indRefBeadDict, indRefThreadDict types.PDFIndirectRef) (err error) {

	logInfoValidate.Printf("*** validateFirstBeadDict begin beadDictObj#%d threadDictObj#%d ***",
		indRefBeadDict.ObjectNumber.Value(), indRefThreadDict.ObjectNumber.Value())

	dictName := "firstBeadDict"
	sinceVersion := types.V10

	dict, err := xRefTable.DereferenceDict(indRefBeadDict)
	if err != nil {
		return
	}

	if dict == nil {
		err = errors.New("validateFirstBeadDict: missing dict")
		return
	}

	_, err = validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "Bead" })
	if err != nil {
		return
	}

	indRefT, err := validateIndRefEntry(xRefTable, dict, dictName, "T", REQUIRED, sinceVersion)
	if err != nil {
		return
	}

	if !indRefT.Equals(indRefThreadDict) {
		err = errors.New("validateFirstBeadDict: invalid entry T (backpointer to ThreadDict)")
		return
	}

	_, err = validateRectangleEntry(xRefTable, dict, dictName, "R", REQUIRED, sinceVersion, nil)
	if err != nil {
		return
	}

	pageDict, err := validateDictEntry(xRefTable, dict, dictName, "P", REQUIRED, sinceVersion, nil)
	if err != nil || pageDict == nil || pageDict.Type() == nil || *pageDict.Type() != "Page" {
		return errors.New("validateFirstBeadDict: invalid page dict")
	}

	previousBeadIndRef, err := validateIndRefEntry(xRefTable, dict, dictName, "V", REQUIRED, sinceVersion)
	if err != nil {
		return
	}

	nextBeadIndRef, err := validateIndRefEntry(xRefTable, dict, dictName, "N", REQUIRED, sinceVersion)
	if err != nil {
		return
	}

	// if N and V reference same bead dict, must be the first and only one.
	if previousBeadIndRef.Equals(*nextBeadIndRef) {
		if !indRefBeadDict.Equals(*previousBeadIndRef) {
			err = errors.New("validateFirstBeadDict: corrupt chain of beads")
			return
		}
		logInfoValidate.Println("*** validateFirstBeadDict end single bead ***")
		return
	}

	err = validateBeadDict(xRefTable, *nextBeadIndRef, indRefThreadDict, indRefBeadDict, *previousBeadIndRef)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateFirstBeadDict end ***")

	return
}

func validateThreadDict(xRefTable *types.XRefTable, obj interface{}, sinceVersion types.PDFVersion) (err error) {

	logInfoValidate.Println("*** validateThreadDict begin ***")

	dictName := "threadDict"

	indRefThreadDict, ok := obj.(types.PDFIndirectRef)
	if !ok {
		err = errors.New("validateThreadDict: not an indirect ref")
		return
	}

	objNumber := indRefThreadDict.ObjectNumber.Value()

	dict, err := xRefTable.DereferenceDict(indRefThreadDict)
	if err != nil {
		return
	}

	_, err = validateNameEntry(xRefTable, dict, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "Thread" })
	if err != nil {
		return
	}

	// Validate optional thread information dict entry.
	obj, found := dict.Find("I")
	if found && obj != nil {
		_, err = validateDocumentInfoDict(xRefTable, obj)
		if err != nil {
			return
		}
	}

	firstBeadDict := dict.IndirectRefEntry("F")
	if firstBeadDict == nil {
		err = errors.Errorf("validateThreadDict: obj#%d required indirect entry \"F\" missing", objNumber)
		return
	}

	// Validate the list of beads starting with the first bead dict.
	err = validateFirstBeadDict(xRefTable, *firstBeadDict, indRefThreadDict)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateThreadDict end ***")

	return
}

func validateThreads(xRefTable *types.XRefTable, rootDict *types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	// => 12.4.3 Articles

	logInfoValidate.Println("*** validateThreads begin ***")

	indRef := rootDict.IndirectRefEntry("Threads")
	if indRef == nil {
		if required {
			err = errors.New("validateThreads: required entry \"Threads\" missing")
			return
		}
		logInfoValidate.Println("validateThreads end: object is nil.")
		return
	}

	arr, err := xRefTable.DereferenceArray(*indRef)
	if err != nil {
		return
	}

	if arr == nil {
		logInfoValidate.Println("validateThreads end: object is nil.")
		return
	}

	// Version check
	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateThreads: unsupported in version %s", xRefTable.VersionString())
	}

	for _, obj := range *arr {

		if obj == nil {
			continue
		}

		err = validateThreadDict(xRefTable, obj, sinceVersion)
		if err != nil {
			return
		}

	}

	logInfoValidate.Println("*** validateThreads end ***")

	return
}

func validateURI(xRefTable *types.XRefTable, rootDict *types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	// => 12.6.4.7 URI Actions

	//	<URI
	//		<Base, ()>
	//	>

	// Must be dict with one entry: Base.file

	logInfoValidate.Println("*** validateURI begin ***")

	dict, err := validateDictEntry(xRefTable, rootDict, "rootDict", "URI", required, sinceVersion, nil)
	if err != nil {
		return
	}

	if dict == nil {
		logInfoValidate.Println("validateURI end: dict is nil.")
		return
	}

	// Version check
	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateURI: unsupported in version %s.\n", xRefTable.VersionString())
	}

	// TODO optional entry?
	if dict.PDFStringLiteralEntry("Base") == nil {
		return errors.New("validateURI: corrupt URI dict: missing entry \"Base\"")
	}

	logInfoValidate.Println("*** validateURI end ***")

	return
}

func validateMarkInfo(xRefTable *types.XRefTable, rootDict *types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	// => 14.7 Logical Structure

	logInfoValidate.Println("*** validateMarkInfo begin ***")

	dict, err := validateDictEntry(xRefTable, rootDict, "rootDict", "MarkInfo", required, sinceVersion, nil)
	if err != nil {
		return
	}

	if dict == nil {
		logInfoValidate.Println("validateMarkInfo end: dict is nil.")
		return
	}

	// Version check
	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateMarkInfo: unsupported in version %s.\n", xRefTable.VersionString())
	}

	var isTaggedPDF bool

	dictName := "markInfoDict"

	// Marked, optional, boolean
	marked, err := validateBooleanEntry(xRefTable, dict, dictName, "Marked", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}
	if marked != nil {
		isTaggedPDF = marked.Value()
	}

	// Suspects: optional, since V1.6, boolean
	suspects, err := validateBooleanEntry(xRefTable, dict, dictName, "Suspects", OPTIONAL, types.V16, nil)
	if err != nil {
		return
	}

	if suspects != nil && suspects.Value() {
		isTaggedPDF = false
	}

	xRefTable.Tagged = isTaggedPDF

	// UserProperties: optional, since V1.6, boolean
	_, err = validateBooleanEntry(xRefTable, dict, dictName, "UserProperties", OPTIONAL, types.V16, nil)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateMarkInfo end ***")

	return
}

func validateLang(xRefTable *types.XRefTable, rootDict *types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoValidate.Println("*** validateLang begin ***")

	_, err = validateStringEntry(xRefTable, rootDict, "rootDict", "Lang", required, sinceVersion, nil)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateLang end ***")

	return
}

// TODO implement
func validateSpiderInfo(xRefTable *types.XRefTable, rootDict *types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	// 14.10.2 Web Capture Information Dictionary

	logInfoValidate.Println("*** validateSpiderInfo begin ***")

	dict, err := validateDictEntry(xRefTable, rootDict, "rootDict", "SpiderInfo", required, sinceVersion, nil)
	if err != nil {
		return
	}

	if dict == nil {
		logInfoValidate.Println("validateSpiderInfo end: dict is nil.")
		return
	}

	// Version check
	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateSpiderInfo: unsupported in version %s.\n", xRefTable.VersionString())
	}

	err = errors.New("*** validateSpiderInfo: not supported ***")

	logInfoValidate.Println("*** validateSpiderInfo begin ***")

	return
}

func validateOutputIntentDict(xRefTable *types.XRefTable, dict *types.PDFDict) (err error) {

	logInfoValidate.Println("*** validateOutputIntentDict begin ***")

	if t := dict.Type(); t != nil && *t != "OutputIntent" {
		return errors.New("validateOutputIntentDict: outputIntents corrupted Type")
	}

	dictName := "outputIntentDict"

	// S: required, name
	_, err = validateNameEntry(xRefTable, dict, dictName, "S", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	// OutputCondition, optional, text string
	_, err = validateStringEntry(xRefTable, dict, dictName, "OutputCondition", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// OutputConditionIdentifier, required, text string
	_, err = validateStringEntry(xRefTable, dict, dictName, "OutputConditionIdentifier", REQUIRED, types.V10, nil)
	if err != nil {
		return
	}

	// RegistryName, optional, text string
	_, err = validateStringEntry(xRefTable, dict, dictName, "RegistryName", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// Info, text string
	// TODO Required if OutputConditionIdentifier does not specify a standard production condition; optional otherwise
	_, err = validateStringEntry(xRefTable, dict, dictName, "Info", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	// DestOutputProfile, streamDict
	// TODO Required if OutputConditionIdentifier does not specify a standard production condition; optional otherwise
	_, err = validateStreamDictEntry(xRefTable, dict, dictName, "DestOutputProfile", OPTIONAL, types.V10, nil)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateOutputIntentDict end ***")

	return
}

func validateOutputIntents(xRefTable *types.XRefTable, rootDict *types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	// => 14.11.5 Output Intents

	logInfoValidate.Println("*** validateOutputIntents begin ***")

	arr, err := validateArrayEntry(xRefTable, rootDict, "rootDict", "OutputIntents", required, sinceVersion, nil)
	if err != nil {
		return
	}

	if arr == nil {
		logInfoValidate.Println("validateOutputIntents end: array is nil.")
		return
	}

	// Version check
	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateOutputIntents: unsupported in version %s.\n", xRefTable.VersionString())
	}

	for _, v := range *arr {

		dict, err := xRefTable.DereferenceDict(v)
		if err != nil {
			return err
		}

		if dict == nil {
			continue
		}

		err = validateOutputIntentDict(xRefTable, dict)
		if err != nil {
			return err
		}
	}

	logInfoValidate.Println("*** validateOutputIntents end ***")

	return
}

// TODO implement
func validatePermissions(xRefTable *types.XRefTable, rootDict *types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	// => 12.8.4 Permissions

	logInfoValidate.Println("*** validatePermissions begin ***")

	dict, err := validateDictEntry(xRefTable, rootDict, "rootDict", "Permissions", required, sinceVersion, nil)
	if err != nil {
		return
	}

	if dict == nil {
		logInfoValidate.Println("validatePermissions end: dict is nil.")
		return
	}

	// Version check
	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validatePermissions: unsupported in version %s.\n", xRefTable.VersionString())
	}

	err = errors.New("*** validatePermissions: not supported ***")

	logInfoValidate.Println("*** validatePermissions end ***")

	return
}

// TODO implement
func validateLegal(xRefTable *types.XRefTable, rootDict *types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	// => 12.8.5 Legal Content Attestations

	logInfoValidate.Println("*** validateLegal begin ***")

	dict, err := validateDictEntry(xRefTable, rootDict, "rootDict", "Legal", required, sinceVersion, nil)
	if err != nil {
		return
	}

	if dict == nil {
		logInfoValidate.Println("validateLegal end: dict is nil.")
		return
	}

	// Version check
	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateLegal: unsupported in version %s.\n", xRefTable.VersionString())
	}

	err = errors.New("*** validateLegal: not supported ***")

	logInfoValidate.Println("*** validateLegal end ***")

	return
}

// TODO implement
func validateRequirements(xRefTable *types.XRefTable, rootDict *types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	// => 12.10 Document Requirements

	logInfoValidate.Println("*** validateRequirements begin ***")

	arr, err := validateArrayEntry(xRefTable, rootDict, "rootDict", "Requirements", required, sinceVersion, nil)
	if err != nil {
		return
	}

	if arr == nil {
		logInfoValidate.Printf("validateRequirements end: array is nil.")
		return
	}

	// Version check
	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateRequirements: unsupported in version %s.\n", xRefTable.VersionString())
	}

	err = errors.New("*** validateRequirements: not supported ***")

	logInfoValidate.Println("*** validateRequirements begin ***")

	return
}

// TODO implement
func validateCollection(xRefTable *types.XRefTable, rootDict *types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	// => 12.3.5 Collections

	logInfoValidate.Println("*** validateCollection begin ***")

	dict, err := validateDictEntry(xRefTable, rootDict, "rootDict", "Collection", required, sinceVersion, nil)
	if err != nil {
		return
	}

	if dict == nil {
		logInfoValidate.Println("validateCollection end: dict is nil.")
		return
	}

	// Version check
	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("validateCollection: unsupported in version %s.\n", xRefTable.VersionString())
	}

	err = errors.New("*** validateCollection: not supported ***")

	logInfoValidate.Println("*** validateCollection begin ***")

	return
}

func validateNeedsRendering(xRefTable *types.XRefTable, rootDict *types.PDFDict, required bool, sinceVersion types.PDFVersion) (err error) {

	logInfoValidate.Println("*** validateNeedsRendering begin ***")

	_, err = validateBooleanEntry(xRefTable, rootDict, "rootDict", "NeedsRendering", required, sinceVersion, nil)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateNeedsRendering end ***")

	return
}

func validateRootObject(xRefTable *types.XRefTable) (err error) {

	logInfoValidate.Println("*** validateRootObject begin ***")

	// => 7.7.2 Document Catalog

	// Entry	   	       opt	since		type			info
	//------------------------------------------------------------------------------------
	//*Type			        n				string			"Catalog"
	//*Version		        y	1.4			name			overrules header version if later
	// Extensions	        y	ISO 32000	dict			=> 7.12 Extensions Dictionary
	//*Pages		        n	-			(dict)			=> 7.7.3 Page Tree
	//*PageLabels	        y	1.3			number tree		=> 7.9.7 Number Trees, 12.4.2 Page Labels
	//*Names		        y	1.2			dict			=> 7.7.4 Name Dictionary
	// Dests	    	    y	only 1.1	(dict)			=> 12.3.2.3 Named Destinations
	//*ViewerPreferences    y	1.2			dict			=> 12.2 Viewer Preferences
	// PageLayout	        y	-			name			/SinglePage, /OneColumn etc.
	// PageMode		        y	-			name			/UseNone, /FullScreen etc.
	//*Outlines		        y	-			(dict)			=> 12.3.3 Document Outline
	// Threads		        y	1.1			(array)			=> 12.4.3 Articles
	//*OpenAction	        y	1.1			array or dict	=> 12.3.2 Destinations, 12.6 Actions
	// AA			        y	1.4			dict			=> 12.6.3 Trigger Events
	//*URI			        y	1.1			dict			=> 12.6.4.7 URI Actions
	//*AcroForm		        y	1.2			dict			=> 12.7.2 Interactive Form Dictionary
	//*Metadata		        y	1.4			(stream)		=> 14.3.2 Metadata Streams
	//*StructTreeRoot 	    y 	1.3			dict			=> 14.7.2 Structure Hierarchy
	//*Markinfo		        y	1.4			dict			=> 14.7 Logical Structure
	// Lang			        y	1.4			string
	// SpiderInfo	        y	1.3			dict			=> 14.10.2 Web Capture Information Dictionary
	//*OutputIntents 	    y	1.4			array			=> 14.11.5 Output Intents
	// PieceInfo	        y	1.4			dict			=> 14.5 Page-Piece Dictionaries
	// OCProperties	        y	1.5			dict			=> 8.11.4 Configuring Optional Content
	// Perms		        y	1.5			dict			=> 12.8.4 Permissions
	// Legal		        y	1.5			dict			=> 12.8.5 Legal Content Attestations
	// Requirements	        y	1.7			array			=> 12.10 Document Requirements
	// Collection	        y	1.7			dict			=> 12.3.5 Collections
	// NeedsRendering 	    y	1.7			boolean			=> XML Forms Architecture (XFA) Spec.

	if xRefTable.Root == nil {
		return errors.New("validateRootObject: missing root dict")
	}

	rootDict, err := xRefTable.DereferenceDict(*xRefTable.Root)
	if err != nil {
		return
	}

	if rootDict == nil {
		return errors.New("validateRootObject: missing root dict")
	}

	// Type
	if t := rootDict.Type(); t == nil || *t != "Catalog" {
		return errors.New("validateRootObject: corrupt root dict type")
	}

	// Version
	// Validation of optional entry Version already happened in read.go/identifyRootVersion
	// _, err = xRefTable.ValidateNameEntry(rootDict, "Version", OPTIONAL, types.V14, nil)
	// if err != nil {
	// 	return
	// }

	// TODO implement, since ISO 32000
	err = validateExtensions(xRefTable, rootDict, OPTIONAL, types.V10)
	if err != nil {
		return
	}

	// Pages
	rootPageNodeDict, err := validatePages(xRefTable, rootDict)
	if err != nil {
		return
	}

	// PageLabels
	err = validatePageLabels(xRefTable, rootDict, OPTIONAL, types.V13)
	if err != nil {
		return
	}

	// Names
	err = validateNames(xRefTable, rootDict, OPTIONAL, types.V12)
	if err != nil {
		return
	}

	// Dests
	err = validateNamedDestinations(xRefTable, rootDict, OPTIONAL, types.V11)
	if err != nil {
		return
	}

	// ViewerPreferences
	err = validateViewerPreferences(xRefTable, rootDict, OPTIONAL, types.V12)
	if err != nil {
		return
	}

	// PageLayout
	err = validatePageLayout(xRefTable, rootDict, OPTIONAL, types.V10)
	if err != nil {
		return
	}

	// PageMode
	err = validatePageMode(xRefTable, rootDict, OPTIONAL, types.V10)
	if err != nil {
		return
	}

	// Outlines
	err = validateOutlines(xRefTable, rootDict, OPTIONAL, types.V10)
	if err != nil {
		return
	}

	// Threads
	err = validateThreads(xRefTable, rootDict, OPTIONAL, types.V11)
	if err != nil {
		return
	}

	// OpenAction
	err = validateOpenAction(xRefTable, rootDict, OPTIONAL, types.V11)
	if err != nil {
		return
	}

	// AA
	err = validateAdditionalActions(xRefTable, rootDict, "rootDict", "AA", OPTIONAL, types.V14, "root")
	if err != nil {
		return
	}

	// URI
	err = validateURI(xRefTable, rootDict, OPTIONAL, types.V11)
	if err != nil {
		return err
	}

	// AcroForm
	// Write widget annotations as part of given form field hierarchies.
	err = validateAcroForm(xRefTable, rootDict, OPTIONAL, types.V12)
	if err != nil {
		return err
	}

	// Write remainder of annotations after AcroForm processing only.
	err = validatePagesAnnotations(xRefTable, rootPageNodeDict)
	if err != nil {
		return
	}

	// Metadata
	sinceVersion := types.V14
	if xRefTable.ValidationMode == types.ValidationRelaxed {
		sinceVersion = types.V13
	}
	err = validateMetadata(xRefTable, rootDict, OPTIONAL, sinceVersion)
	if err != nil {
		return err
	}

	// StructTreeRoot
	err = validateStructTree(xRefTable, rootDict, OPTIONAL, types.V13)
	if err != nil {
		return
	}

	// Markinfo
	err = validateMarkInfo(xRefTable, rootDict, OPTIONAL, types.V14)
	if err != nil {
		return
	}

	// Lang
	err = validateLang(xRefTable, rootDict, OPTIONAL, types.V10)
	if err != nil {
		return
	}

	// SpiderInfo
	err = validateSpiderInfo(xRefTable, rootDict, OPTIONAL, types.V13)
	if err != nil {
		return err
	}

	// OutputIntents
	err = validateOutputIntents(xRefTable, rootDict, OPTIONAL, types.V14)
	if err != nil {
		return err
	}

	// PieceInfo
	_, err = validatePieceInfo(xRefTable, rootDict, OPTIONAL, types.V14)
	if err != nil {
		return err
	}

	// OCProperties
	// TODO Required if document contains optional content.
	sinceVersion = types.V15
	if xRefTable.ValidationMode == types.ValidationRelaxed {
		sinceVersion = types.V14
	}
	err = validateOCProperties(xRefTable, rootDict, OPTIONAL, sinceVersion)
	if err != nil {
		return
	}

	// Perms
	err = validatePermissions(xRefTable, rootDict, OPTIONAL, types.V15)
	if err != nil {
		return
	}

	// Legal
	err = validateLegal(xRefTable, rootDict, OPTIONAL, types.V17)
	if err != nil {
		return
	}

	// Requirements
	err = validateRequirements(xRefTable, rootDict, OPTIONAL, types.V17)
	if err != nil {
		return
	}

	// Collection
	err = validateCollection(xRefTable, rootDict, OPTIONAL, types.V17)
	if err != nil {
		return
	}

	// NeedsRendering
	err = validateNeedsRendering(xRefTable, rootDict, OPTIONAL, types.V17)
	if err != nil {
		return
	}

	logInfoValidate.Println("*** validateRootObject end ***")

	return nil
}

func validateAdditionalStreams(xRefTable *types.XRefTable) error {

	return nil
}

// XRefTable validates a PDF cross reference table obeying the validation mode.
func XRefTable(xRefTable *types.XRefTable) (err error) {

	logInfoValidate.Println("*** validateXRefTable begin ***")

	// Validate root object(aka the document catalog) and page tree.
	err = validateRootObject(xRefTable)
	if err != nil {
		return
	}

	// Validate document information dictionary.
	err = validateDocumentInfoObject(xRefTable)
	if err != nil {
		return
	}

	// Validate offspec additional streams as declared in pdf trailer.
	err = validateAdditionalStreams(xRefTable)
	if err != nil {
		return
	}

	xRefTable.Valid = true

	logInfoValidate.Println("*** validateXRefTable end ***")

	return

}
