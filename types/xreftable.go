package types

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/hhrutter/pdfcpu/log"
	"github.com/pkg/errors"
)

// XRefTableEntry represents an entry in the PDF cross reference table.
//
// This may wrap a free object, a compressed object or any in use PDF object:
//
// PDFDict, PDFStreamDict, PDFObjectStreamDict, PDFXRefStreamDict,
// PDFArray, PDFInteger, PDFFloat, PDFName, PDFStringLiteral, PDFHexLiteral, PDFBoolean
type XRefTableEntry struct {
	Free            bool
	Offset          *int64
	Generation      *int
	Object          PDFObject
	Compressed      bool
	ObjectStream    *int
	ObjectStreamInd *int
}

// NewXRefTableEntryGen0 returns a cross reference table entry for an object with generation 0.
func NewXRefTableEntryGen0(obj PDFObject) *XRefTableEntry {
	zero := 0
	return &XRefTableEntry{Generation: &zero, Object: obj}
}

// NewFreeHeadXRefTableEntry returns the xref table entry for object 0
// which is per definition the head of the free list (list of free objects).
func NewFreeHeadXRefTableEntry() *XRefTableEntry {

	freeHeadGeneration := FreeHeadGeneration
	zero := int64(0)

	return &XRefTableEntry{
		Free:       true,
		Generation: &freeHeadGeneration,
		Offset:     &zero,
	}
}

// Enc wraps around all defined encryption attributes.
type Enc struct {
	O, U       []byte
	L, P, R, V int
	Emd        bool // encrypt meta data
	ID         []byte
}

// XRefTable represents a PDF cross reference table plus stats for a PDF file.
type XRefTable struct {
	Table               map[int]*XRefTableEntry
	Size                *int             // Object count from PDF trailer dict.
	PageCount           int              // Number of pages, set during validation.
	Root                *PDFIndirectRef  // Pointer to catalog (reference to root object).
	RootDict            *PDFDict         // Catalog
	Names               map[string]*Node // Cache for name trees as found in catalog.
	Encrypt             *PDFIndirectRef  // Encrypt dict.
	E                   *Enc
	EncKey              []byte // Encrypt key.
	AES4Strings         bool
	AES4Streams         bool
	AES4EmbeddedStreams bool

	// PDF Version
	HeaderVersion *PDFVersion // The PDF version the source is claiming to us as per its header.
	RootVersion   *PDFVersion // Optional PDF version taking precedence over the header version.

	// Document information section
	Info     *PDFIndirectRef // Infodict (reference to info dict object)
	ID       *PDFArray       // from info dict (or trailer?)
	Author   string
	Creator  string
	Producer string

	// Linearization section (not yet supported)
	OffsetPrimaryHintTable  *int64
	OffsetOverflowHintTable *int64
	LinearizationObjs       IntSet

	// Offspec section
	AdditionalStreams *PDFArray // array of PDFIndirectRef - trailer :e.g., Oasis "Open Doc"

	// Statistics
	Stats PDFStats

	Tagged bool // File is using tags. This is important for ???

	// Validation
	Valid          bool // true means successful validated against ISO 32000.
	ValidationMode int  // see Configuration

	Optimized bool
}

// NewXRefTable creates a new XRefTable.
func newXRefTable(validationMode int) (xRefTable *XRefTable) {
	return &XRefTable{
		Table:             map[int]*XRefTableEntry{},
		Names:             map[string]*Node{},
		LinearizationObjs: IntSet{},
		Stats:             NewPDFStats(),
		ValidationMode:    validationMode,
	}
}

// Version returns the PDF version of the PDF writer that created this file.
// Before V1.4 this is the header version.
// Since V1.4 the catalog may contain a Version entry which takes precedence over the header version.
func (xRefTable *XRefTable) Version() PDFVersion {

	if xRefTable.RootVersion != nil {
		return *xRefTable.RootVersion
	}

	return *xRefTable.HeaderVersion
}

// VersionString return a string representation for this PDF files PDF version.
func (xRefTable *XRefTable) VersionString() string {
	return VersionString(xRefTable.Version())
}

// ParseRootVersion returns a string representation for an optional Version entry in the root object.
func (xRefTable *XRefTable) ParseRootVersion() (v *string, err error) {

	// Look in the catalog/root for a name entry "Version".
	// This entry overrides the header version.

	rootDict, err := xRefTable.Catalog()
	if err != nil {
		return nil, err
	}

	if n := rootDict.PDFNameEntry("Version"); n != nil {
		s := n.String()
		v = &s
	}

	return v, nil
}

// ValidateVersion validates against the xRefTable's version.
func (xRefTable *XRefTable) ValidateVersion(element string, sinceVersion PDFVersion) error {

	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("%s: unsupported in version %s\n", element, xRefTable.VersionString())
	}

	return nil
}

// IsLinearizationObject returns true if object #i is a a linearization object.
func (xRefTable *XRefTable) IsLinearizationObject(i int) bool {
	return xRefTable.LinearizationObjs[i]
}

// LinearizationObjsString returns a formatted string and the number of objs.
func (xRefTable *XRefTable) LinearizationObjsString() (int, string) {

	var objs []int
	for k := range xRefTable.LinearizationObjs {
		if xRefTable.LinearizationObjs[k] {
			objs = append(objs, k)
		}
	}
	sort.Ints(objs)

	var linObj []string
	for _, i := range objs {
		linObj = append(linObj, fmt.Sprintf("%d", i))
	}

	return len(linObj), strings.Join(linObj, ",")
}

// Exists returns true if xRefTable contains an entry for objNumber.
func (xRefTable *XRefTable) Exists(objNumber int) bool {
	_, found := xRefTable.Table[objNumber]
	return found
}

// Find returns the XRefTable entry for given object number.
func (xRefTable *XRefTable) Find(objNumber int) (*XRefTableEntry, bool) {
	e, found := xRefTable.Table[objNumber]
	if !found {
		return nil, false
	}
	return e, true
}

// FindObject returns the object of the XRefTableEntry for a specific object number.
func (xRefTable *XRefTable) FindObject(objNumber int) (PDFObject, error) {

	entry, ok := xRefTable.Find(objNumber)
	if !ok {
		return nil, errors.Errorf("FindObject: obj#%d not registered in xRefTable", objNumber)
	}

	return entry.Object, nil
}

// Free returns the cross ref table entry for given number of a free object.
func (xRefTable *XRefTable) Free(objNumber int) (*XRefTableEntry, error) {

	entry, found := xRefTable.Find(objNumber)

	if !found {
		return nil, errors.Errorf("Free: object #%d not found.", objNumber)
	}

	if !entry.Free {
		return nil, errors.Errorf("Free: object #%d found, but not free.", objNumber)
	}

	return entry, nil
}

// NextForFree returns the number of the object the free object with objNumber links to.
// This is the successor of this free object in the free list.
func (xRefTable *XRefTable) NextForFree(objNumber int) (int, error) {

	entry, err := xRefTable.Free(objNumber)
	if err != nil {
		return 0, err
	}

	return int(*entry.Offset), nil
}

// FindTableEntryLight returns the XRefTable entry for given object number.
func (xRefTable *XRefTable) FindTableEntryLight(objNumber int) (*XRefTableEntry, bool) {
	return xRefTable.Find(objNumber)
}

// FindTableEntry returns the XRefTable entry for given object and generation numbers.
func (xRefTable *XRefTable) FindTableEntry(objNumber int, generationNumber int) (*XRefTableEntry, bool) {
	entry, found := xRefTable.Find(objNumber)
	if found && *entry.Generation == generationNumber {
		return entry, found
	}
	return nil, false
}

// FindTableEntryForIndRef returns the XRefTable entry for given indirect reference.
func (xRefTable *XRefTable) FindTableEntryForIndRef(indRef *PDFIndirectRef) (*XRefTableEntry, bool) {
	if indRef == nil {
		//logErrorTypes.Println("FindTableEntryForIndRef: returning false on absent indRef")
		return nil, false
	}
	return xRefTable.FindTableEntry(indRef.ObjectNumber.Value(), indRef.GenerationNumber.Value())
}

// InsertNew adds given xRefTableEntry at next new objNumber into the cross reference table.
// Only to be called once an xRefTable has been generated completely and all trailer dicts have been processed.
// xRefTable.Size is the size entry of the first trailer dict processed.
// Called on creation of new object streams.
// Called by InsertAndUseRecycled.
func (xRefTable *XRefTable) InsertNew(xRefTableEntry XRefTableEntry) (objNumber int) {
	objNumber = *xRefTable.Size
	xRefTable.Table[objNumber] = &xRefTableEntry
	*xRefTable.Size++
	return
}

// InsertAndUseRecycled adds given xRefTableEntry into the cross reference table utilizing the freelist.
func (xRefTable *XRefTable) InsertAndUseRecycled(xRefTableEntry XRefTableEntry) (objNumber int, err error) {

	// see 7.5.4 Cross-Reference Table

	log.Debug.Println("InsertAndUseRecycled: begin")

	// Get Next free object from freelist.
	freeListHeadEntry, err := xRefTable.Free(0)
	if err != nil {
		return 0, err
	}

	// If none available, add new object & return.
	if *freeListHeadEntry.Offset == 0 {
		objNumber = xRefTable.InsertNew(xRefTableEntry)
		log.Debug.Printf("InsertAndUseRecycled: end, new objNr=%d\n", objNumber)
		return objNumber, nil
	}

	// Recycle free object, update free list & return.
	objNumber = int(*freeListHeadEntry.Offset)
	entry, found := xRefTable.FindTableEntryLight(objNumber)
	if !found {
		return 0, errors.Errorf("InsertAndRecycle: no entry for obj #%d\n", objNumber)
	}

	// The new free list head entry becomes the old head entry's successor.
	freeListHeadEntry.Offset = entry.Offset

	// The old head entry becomes garbage.
	entry.Free = false
	entry.Offset = nil

	// Create a new entry for the recycled object.
	// TODO use entrys generation.
	xRefTable.Table[objNumber] = &xRefTableEntry

	log.Debug.Printf("InsertAndUseRecycled: end, recycled objNr=%d\n", objNumber)

	return objNumber, nil
}

// InsertObject inserts an object into the xRefTable.
func (xRefTable *XRefTable) InsertObject(obj PDFObject) (objNumber int, err error) {
	xRefTableEntry := NewXRefTableEntryGen0(obj)
	return xRefTable.InsertNew(*xRefTableEntry), nil
}

// IndRefForNewObject inserts an object into the xRefTable and returns an indirect reference to it.
func (xRefTable *XRefTable) IndRefForNewObject(obj PDFObject) (*PDFIndirectRef, error) {

	objNr, err := xRefTable.InsertObject(obj)
	if err != nil {
		return nil, err
	}

	return NewPDFIndirectRef(objNr, 0), nil
}

// NewPDFStreamDict creates a streamDict for buf.
func (xRefTable *XRefTable) NewPDFStreamDict(filename string) (*PDFStreamDict, error) {

	buf, err := ioutil.ReadFile(filename)
	if err != nil {
		//logErrorTypes.Printf("%s: %s\n", filename, err)
		return nil, err
	}

	sd :=
		&PDFStreamDict{
			PDFDict:        NewPDFDict(),
			Content:        buf,
			FilterPipeline: []PDFFilter{{Name: "FlateDecode", DecodeParms: nil}}}

	sd.InsertName("Filter", "FlateDecode")

	return sd, nil
}

// NewEmbeddedFileStreamDict creates and returns an embeddedFileStreamDict containing the file "filename".
func (xRefTable *XRefTable) NewEmbeddedFileStreamDict(filename string) (*PDFStreamDict, error) {

	sd, err := xRefTable.NewPDFStreamDict(filename)
	if err != nil {
		return nil, err
	}

	fi, err := os.Stat(filename)
	if err != nil {
		return nil, err
	}

	sd.InsertName("Type", "EmbeddedFile")

	d := NewPDFDict()
	d.InsertInt("Size", int(fi.Size()))
	d.Insert("ModDate", DateStringLiteral(fi.ModTime()))
	sd.Insert("Params", d)

	return sd, nil
}

// NewSoundStreamDict returns a new sound stream dict.
func (xRefTable *XRefTable) NewSoundStreamDict(filename string, samplingRate int, fileSpecDict *PDFDict) (*PDFStreamDict, error) {

	sd, err := xRefTable.NewPDFStreamDict(filename)
	if err != nil {
		return nil, err
	}

	sd.InsertName("Type", "Sound")
	sd.InsertInt("R", samplingRate)
	sd.InsertInt("C", 2)
	sd.InsertInt("B", 8)
	sd.InsertName("E", "Signed")

	if fileSpecDict != nil {
		sd.Insert("F", *fileSpecDict)
	} else {
		sd.Insert("F", PDFStringLiteral(path.Base(filename)))
	}

	return sd, nil
}

// NewFileSpecDict creates and returns a new fileSpec dictionary.
func (xRefTable *XRefTable) NewFileSpecDict(filename string, indRefStreamDict PDFIndirectRef) (*PDFDict, error) {

	d := NewPDFDict()
	d.InsertName("Type", "Filespec")
	d.InsertString("F", filename)
	d.InsertString("UF", filename)
	// TODO d.Insert("UF", utf16.Encode([]rune(filename)))

	efDict := NewPDFDict()
	efDict.Insert("F", indRefStreamDict)
	efDict.Insert("UF", indRefStreamDict)
	d.Insert("EF", efDict)

	d.InsertString("Desc", "attached by "+PDFCPULongVersion)

	// CI, optional, collection item dict, since V1.7
	// a corresponding collection schema dict in a collection.
	ciDict := NewPDFDict()
	//add contextual meta info here.
	d.Insert("CI", ciDict)

	return &d, nil
}

func (xRefTable *XRefTable) freeObjects() IntSet {

	m := IntSet{}

	for k, v := range xRefTable.Table {
		if v.Free && k > 0 {
			m[k] = true
		}
	}

	return m
}

// EnsureValidFreeList ensures the integrity of the free list associated with the recorded free objects.
// See 7.5.4 Cross-Reference Table
func (xRefTable *XRefTable) EnsureValidFreeList() error {

	log.Debug.Println("EnsureValidFreeList begin")

	m := xRefTable.freeObjects()

	// Verify free object 0 as free list head.
	head, err := xRefTable.Free(0)
	if err != nil {
		return err
	}

	// verify generation of 56535
	if *head.Generation != FreeHeadGeneration {
		// Fix generation for obj 0.
		*head.Generation = FreeHeadGeneration
	}

	if len(m) == 0 {

		// no free object other than 0.

		// repair if necessary
		if *head.Offset != 0 {
			*head.Offset = 0
		}

		log.Debug.Println("EnsureValidFreeList: empty free list.")
		return nil
	}

	f := int(*head.Offset)

	// until we have found the last free object which should point to obj 0.
	for f != 0 {

		log.Debug.Printf("EnsureValidFreeList: validating obj #%d %v\n", f, m)
		// verify if obj f is one of the free objects recorded.
		if !m[f] {
			return errors.New("EnsureValidFreeList: freelist corrupted")
		}

		delete(m, f)

		f, err = xRefTable.NextForFree(f)
		if err != nil {
			return err
		}
	}

	if len(m) == 0 {
		log.Debug.Println("EnsureValidFreeList: end, regular linked list")
		return nil
	}

	// insert remaining free objects into verified linked list
	// unless they are forever deleted with generation 65535.
	// In that case they have to point to obj 0.
	for i := range m {

		entry, found := xRefTable.FindTableEntryLight(i)
		if !found {
			return errors.Errorf("EnsureValidFreeList: no xref entry found for obj #%d\n", i)
		}

		if !entry.Free {
			return errors.Errorf("EnsureValidFreeList: xref entry is not free for obj #%d\n", i)
		}

		if *entry.Generation == FreeHeadGeneration {
			zero := int64(0)
			entry.Offset = &zero
			continue
		}

		entry.Offset = head.Offset
		next := int64(i)
		head.Offset = &next
	}

	log.Debug.Println("EnsureValidFreeList: end, linked list plus some dangling free objects.")

	return nil
}

func (xRefTable *XRefTable) deleteObject(obj PDFObject) error {

	indRef, ok := obj.(PDFIndirectRef)
	if ok {

		var err error

		objNumber := int(indRef.ObjectNumber)
		obj, err = xRefTable.Dereference(indRef)
		if err != nil {
			return err
		}

		err = xRefTable.DeleteObject(objNumber)
		if err != nil {
			return err
		}

		if obj == nil {
			log.Debug.Println("deleteObject: end, obj == nil")
			return err
		}
	}

	switch obj := obj.(type) {

	case PDFDict:
		for _, v := range obj.Dict {
			err := xRefTable.deleteObject(v)
			if err != nil {
				return err
			}
		}

	case PDFStreamDict:
		for _, v := range obj.Dict {
			err := xRefTable.deleteObject(v)
			if err != nil {
				return err
			}
		}

	case PDFArray:
		for _, v := range obj {
			err := xRefTable.deleteObject(v)
			if err != nil {
				return err
			}
		}

	}

	return nil
}

// DeleteObjectGraph deletes all objects reachable by indRef.
func (xRefTable *XRefTable) DeleteObjectGraph(obj PDFObject) error {

	log.Debug.Println("DeleteObjectGraph: begin")

	indRef, ok := obj.(PDFIndirectRef)
	if !ok {
		return nil
	}

	// Delete ObjectGraph for object indRef.ObjectNumber.Value() via recursion.
	err := xRefTable.deleteObject(indRef)
	if err != nil {
		return err
	}

	log.Debug.Println("DeleteObjectGraph: end")
	return nil
}

// DeleteObject marks an object as free and inserts it into the free list right after the head.
func (xRefTable *XRefTable) DeleteObject(objectNumber int) error {

	// see 7.5.4 Cross-Reference Table

	log.Debug.Printf("DeleteObject: begin %d\n", objectNumber)

	freeListHeadEntry, err := xRefTable.Free(0)
	if err != nil {
		return err
	}

	entry, found := xRefTable.FindTableEntryLight(objectNumber)
	if !found {
		return errors.Errorf("DeleteObject: no entry for obj #%d\n", objectNumber)
	}

	if entry.Free {
		log.Debug.Printf("DeleteObject: end %d already free\n", objectNumber)
		return nil
	}

	*entry.Generation++
	entry.Free = true
	entry.Compressed = false
	entry.Offset = freeListHeadEntry.Offset
	entry.Object = nil

	next := int64(objectNumber)
	freeListHeadEntry.Offset = &next

	log.Debug.Printf("DeleteObject: end %d\n", objectNumber)

	return nil
}

// UndeleteObject ensures an object is not recorded in the free list.
// e.g. sometimes caused by indirect references to free objects in the original PDF file.
func (xRefTable *XRefTable) UndeleteObject(objectNumber int) error {

	log.Debug.Printf("UndeleteObject: begin %d\n", objectNumber)

	f, err := xRefTable.Free(0)
	if err != nil {
		return err
	}

	// until we have found the last free object which should point to obj 0.
	for *f.Offset != 0 {

		objNr := int(*f.Offset)

		entry, err := xRefTable.Free(objNr)
		if err != nil {
			return err
		}

		if objNr == objectNumber {
			log.Debug.Printf("UndeleteObject end: undeleting obj#%d\n", objectNumber)
			*f.Offset = *entry.Offset
			entry.Offset = nil
			entry.Free = false
			return nil
		}

		f = entry

	}

	log.Debug.Printf("UndeleteObject: end: obj#%d not in free list.\n", objectNumber)

	return nil
}

// indRefToObject dereferences an indirect object from the xRefTable and returns the result.
func (xRefTable *XRefTable) indRefToObject(indObjRef *PDFIndirectRef) (PDFObject, error) {

	if indObjRef == nil {
		return nil, errors.New("indRefToObject: input argument is nil")
	}

	objectNumber := indObjRef.ObjectNumber.Value()

	generationNumber := indObjRef.GenerationNumber.Value()

	entry, found := xRefTable.FindTableEntry(objectNumber, generationNumber)
	if !found {
		return nil, nil
	}

	if entry.Free {
		// TODO return err
		return nil, nil
	}

	if entry.Object == nil {
		return nil, nil
	}

	// return dereferenced object
	return entry.Object, nil
}

// Dereference resolves an indirect object and returns the resulting PDF object.
func (xRefTable *XRefTable) Dereference(obj PDFObject) (PDFObject, error) {

	indRef, ok := obj.(PDFIndirectRef)
	if !ok {
		// Nothing do dereference.
		return obj, nil
	}

	return xRefTable.indRefToObject(&indRef)
}

// DereferenceInteger resolves and validates an integer object, which may be an indirect reference.
func (xRefTable *XRefTable) DereferenceInteger(obj PDFObject) (*PDFInteger, error) {

	obj, err := xRefTable.Dereference(obj)
	if err != nil || obj == nil {
		return nil, err
	}

	i, ok := obj.(PDFInteger)
	if !ok {
		return nil, errors.Errorf("ValidateInteger: wrong type <%v>", obj)
	}

	return &i, nil
}

// DereferenceName resolves and validates a name object, which may be an indirect reference.
func (xRefTable *XRefTable) DereferenceName(obj PDFObject, sinceVersion PDFVersion, validate func(string) bool) (n PDFName, err error) {

	obj, err = xRefTable.Dereference(obj)
	if err != nil || obj == nil {
		return n, err
	}

	n, ok := obj.(PDFName)
	if !ok {
		return n, errors.Errorf("DereferenceName: wrong type <%v>", obj)
	}

	// Version check
	err = xRefTable.ValidateVersion("DereferenceName", sinceVersion)
	if err != nil {
		return n, err
	}

	// Validation
	if validate != nil && !validate(n.Value()) {
		return n, errors.Errorf("DereferenceName: invalid <%s>", n.Value())
	}

	return n, nil
}

// DereferenceStringLiteral resolves and validates a string literal object, which may be an indirect reference.
func (xRefTable *XRefTable) DereferenceStringLiteral(obj PDFObject, sinceVersion PDFVersion, validate func(string) bool) (s PDFStringLiteral, err error) {

	obj, err = xRefTable.Dereference(obj)
	if err != nil || obj == nil {
		return s, err
	}

	s, ok := obj.(PDFStringLiteral)
	if !ok {
		return s, errors.Errorf("DereferenceStringLiteral: wrong type <%v>", obj)
	}

	// Ensure UTF16 correctness.
	s1, err := StringLiteralToString(s.Value())
	if err != nil {
		return s, err
	}

	// Version check
	err = xRefTable.ValidateVersion("DereferenceStringLiteral", sinceVersion)
	if err != nil {
		return s, err
	}

	// Validation
	if validate != nil && !validate(s1) {
		return s, errors.Errorf("DereferenceStringLiteral: invalid <%s>", s1)
	}

	return s, nil
}

// DereferenceStringOrHexLiteral resolves and validates a string or hex literal object, which may be an indirect reference.
func (xRefTable *XRefTable) DereferenceStringOrHexLiteral(obj PDFObject, sinceVersion PDFVersion, validate func(string) bool) (o PDFObject, err error) {

	o, err = xRefTable.Dereference(obj)
	if err != nil || o == nil {
		return nil, err
	}

	var s string

	switch str := o.(type) {

	case PDFStringLiteral:
		// Ensure UTF16 correctness.
		s, err = StringLiteralToString(str.Value())
		if err != nil {
			return nil, err
		}

	case PDFHexLiteral:
		// Ensure UTF16 correctness.
		s, err = HexLiteralToString(str.Value())
		if err != nil {
			return nil, err
		}

	default:
		return nil, errors.Errorf("DereferenceStringOrHexLiteral: wrong type <%v>", obj)

	}

	// Version check
	err = xRefTable.ValidateVersion("DereferenceStringOrHexLiteral", sinceVersion)
	if err != nil {
		return nil, err
	}

	// Validation
	if validate != nil && !validate(s) {
		return nil, errors.Errorf("DereferenceStringOrHexLiteral: invalid <%s>", s)
	}

	return o, nil
}

// DereferenceArray resolves and validates an array object, which may be an indirect reference.
func (xRefTable *XRefTable) DereferenceArray(obj PDFObject) (*PDFArray, error) {

	obj, err := xRefTable.Dereference(obj)
	if err != nil || obj == nil {
		return nil, err
	}

	arr, ok := obj.(PDFArray)
	if !ok {
		return nil, errors.Errorf("DereferenceArray: wrong type <%v>", obj)
	}

	return &arr, nil
}

// DereferenceDict resolves and validates a dictionary object, which may be an indirect reference.
func (xRefTable *XRefTable) DereferenceDict(obj PDFObject) (*PDFDict, error) {

	obj, err := xRefTable.Dereference(obj)
	if err != nil || obj == nil {
		return nil, err
	}

	dict, ok := obj.(PDFDict)
	if !ok {
		return nil, errors.Errorf("DereferenceDict: wrong type %T <%v>", obj, obj)
	}

	return &dict, nil
}

// DereferenceStreamDict resolves and validates a stream dictionary object, which may be an indirect reference.
func (xRefTable *XRefTable) DereferenceStreamDict(obj PDFObject) (*PDFStreamDict, error) {

	obj, err := xRefTable.Dereference(obj)
	if err != nil || obj == nil {
		return nil, err
	}

	streamDict, ok := obj.(PDFStreamDict)
	if !ok {
		return nil, errors.Errorf("DereferenceStreamDict: wrong type <%v>", obj)
	}

	return &streamDict, nil
}

// Catalog returns a pointer to the root object / catalog.
func (xRefTable *XRefTable) Catalog() (*PDFDict, error) {

	if xRefTable.RootDict != nil {
		return xRefTable.RootDict, nil
	}

	if xRefTable.Root == nil {
		return nil, errors.New("Catalog: missing root dict")
	}

	o, err := xRefTable.indRefToObject(xRefTable.Root)
	if err != nil || o == nil {
		return nil, err
	}

	dict, ok := o.(PDFDict)
	if !ok {
		return nil, errors.New("Catalog: corrupt root catalog")
	}

	xRefTable.RootDict = &dict

	return xRefTable.RootDict, nil
}

// EncryptDict returns a pointer to the root object / catalog.
func (xRefTable *XRefTable) EncryptDict() (*PDFDict, error) {

	pdfObject, err := xRefTable.indRefToObject(xRefTable.Encrypt)
	if err != nil || pdfObject == nil {
		return nil, err
	}

	pdfDict, ok := pdfObject.(PDFDict)
	if !ok {
		return nil, errors.New("EncryptDict: corrupt encrypt dict")
	}

	return &pdfDict, nil
}

// CatalogHasPieceInfo returns true if the root has an entry for \"PieceInfo\".
func (xRefTable *XRefTable) CatalogHasPieceInfo() (bool, error) {

	rootDict, err := xRefTable.Catalog()
	if err != nil {
		return false, err
	}

	obj, hasPieceInfo := rootDict.Find("PieceInfo")

	return hasPieceInfo && obj != nil, nil
}

// Pages returns the Pages reference contained in the catalog.
func (xRefTable *XRefTable) Pages() (*PDFIndirectRef, error) {

	rootDict, err := xRefTable.Catalog()
	if err != nil {
		return nil, err
	}

	return rootDict.IndirectRefEntry("Pages"), nil
}

// MissingObjects returns the number of objects that were not written
// plus the corresponding comma separated string representation.
func (xRefTable *XRefTable) MissingObjects() (int, *string) {

	var missing []string

	for i := 0; i < *xRefTable.Size; i++ {
		if !xRefTable.Exists(i) {
			missing = append(missing, fmt.Sprintf("%d", i))
		}
	}

	var s *string

	if len(missing) > 0 {
		joined := strings.Join(missing, ",")
		s = &joined
	}

	return len(missing), s
}

func (xRefTable *XRefTable) list(logStr []string) []string {

	var keys []int
	for k := range xRefTable.Table {
		keys = append(keys, k)
	}
	sort.Ints(keys)

	// Print list of XRefTable entries to logString.
	for _, k := range keys {

		entry := xRefTable.Table[k]

		var str string

		if entry.Free {
			str = fmt.Sprintf("%5d: f   next=%8d generation=%d\n", k, *entry.Offset, *entry.Generation)
		} else if entry.Compressed {
			str = fmt.Sprintf("%5d: c => obj:%d[%d] generation=%d \n%s\n", k, *entry.ObjectStream, *entry.ObjectStreamInd, *entry.Generation, entry.Object)
		} else {
			if entry.Object != nil {

				typeStr := fmt.Sprintf("%T", entry.Object)

				pdfDict, ok := entry.Object.(PDFDict)

				if ok {
					if pdfDict.Type() != nil {
						typeStr += fmt.Sprintf(" type=%s", *pdfDict.Type())
					}
					if pdfDict.Subtype() != nil {
						typeStr += fmt.Sprintf(" subType=%s", *pdfDict.Subtype())
					}
				}

				if entry.ObjectStream != nil {
					// was compressed, offset is nil.
					str = fmt.Sprintf("%5d: was compressed %d[%d] generation=%d %s \n%s\n",
						k, *entry.ObjectStream, *entry.ObjectStreamInd, *entry.Generation, typeStr, entry.Object)
				} else {
					// regular in use object with offset.
					if entry.Offset != nil {
						str = fmt.Sprintf("%5d:   offset=%8d generation=%d %s \n%s\n",
							k, *entry.Offset, *entry.Generation, typeStr, entry.Object)
					} else {
						str = fmt.Sprintf("%5d:   offset=nil generation=%d %s \n%s\n",
							k, *entry.Generation, typeStr, entry.Object)
					}

				}

				if typeStr == "types.PDFStreamDict" {
					pdfStreamDict, _ := entry.Object.(PDFStreamDict)
					str += fmt.Sprintf("stream content length = %d\n", len(pdfStreamDict.Content))
					if pdfStreamDict.IsPageContent {
						str += fmt.Sprintf("content: <%s>\n", pdfStreamDict.Content)
					}
				}

				if typeStr == "types.PDFObjectStreamDict" {
					pdfObjectStreamDict, _ := entry.Object.(PDFObjectStreamDict)
					str += fmt.Sprintf("object stream count:%d size of objectarray:%d\n", pdfObjectStreamDict.ObjCount, len(pdfObjectStreamDict.ObjArray))
				}

			} else {

				str = fmt.Sprintf("%5d:   offset=%8d generation=%d nil\n", k, *entry.Offset, *entry.Generation)
			}
		}

		logStr = append(logStr, str)
	}

	return logStr
}

// Dump the free list to logStr.
// At this point the free list is assumed to be a linked list with its last node linked to the beginning.
func (xRefTable *XRefTable) freeList(logStr []string) ([]string, error) {

	log.Debug.Printf("freeList begin")

	head, err := xRefTable.Free(0)
	if err != nil {
		return nil, err
	}

	if *head.Offset == 0 {
		return append(logStr, "\nEmpty free list.\n"), nil
	}

	f := int(*head.Offset)

	logStr = append(logStr, "\nfree list:\n  obj  next  generation\n")
	logStr = append(logStr, fmt.Sprintf("%5d %5d %5d\n", 0, f, FreeHeadGeneration))

	for f != 0 {

		log.Debug.Printf("freeList validating free object %d\n", f)

		entry, err := xRefTable.Free(f)
		if err != nil {
			return nil, err
		}

		next := int(*entry.Offset)
		generation := *entry.Generation
		s := fmt.Sprintf("%5d %5d %5d\n", f, next, generation)
		logStr = append(logStr, s)
		log.Debug.Printf("freeList: %s", s)

		f = next
	}

	return logStr, nil
}

func (xRefTable *XRefTable) bindNameTreeNode(name string, n *Node, root bool) error {

	var dict PDFDict

	if n.IndRef == nil {
		d := NewPDFDict()
		indRef, err := xRefTable.IndRefForNewObject(d)
		if err != nil {
			return err
		}
		n.IndRef = indRef
		dict = d
	} else {
		if root {
			// Update root object after possible tree modification after removal of empty kid.
			namesDict, err := xRefTable.NamesDict()
			if err != nil {
				return err
			}
			if namesDict == nil {
				return errors.New("Root entry \"Names\" corrupt")
			}
			namesDict.Update(name, *n.IndRef)
		}
		log.Debug.Printf("bind IndRef = %v\n", n.IndRef)
		d, err := xRefTable.DereferenceDict(*n.IndRef)
		if err != nil {
			return err
		}
		if d == nil {
			return errors.Errorf("name tree node dict is nil for node: %v\n", n)
		}
		dict = *d
	}

	if !root {
		dict.Update("Limits", NewStringArray(n.Kmin, n.Kmax))
	} else {
		dict.Delete("Limits")
	}

	if n.leaf() {
		a := PDFArray{}
		for _, e := range n.Names {
			a = append(a, PDFStringLiteral(e.k))
			a = append(a, e.v)
		}
		dict.Update("Names", a)
		log.Debug.Printf("bound nametree node(leaf): %s/n", dict)
		return nil
	}

	kids := PDFArray{}
	for _, k := range n.Kids {
		err := xRefTable.bindNameTreeNode(name, k, false)
		if err != nil {
			return err
		}
		kids = append(kids, *k.IndRef)
	}

	dict.Update("Kids", kids)
	dict.Delete("Names")

	log.Debug.Printf("bound nametree node(intermediary): %s/n", dict)

	return nil
}

// BindNameTrees syncs up the internal name tree cache with the xreftable.
func (xRefTable *XRefTable) BindNameTrees() error {

	log.Debug.Println("BindNameTrees..")
	for k, v := range xRefTable.Names {
		log.Debug.Printf("bindNameTree: %s\n", k)
		err := xRefTable.bindNameTreeNode(k, v, true)
		if err != nil {
			return err
		}
	}

	return nil
}

// LocateNameTree locates/ensures a specific name tree.
func (xRefTable *XRefTable) LocateNameTree(nameTreeName string, ensure bool) error {

	d, err := xRefTable.Catalog()
	if err != nil {
		return err
	}

	obj, found := d.Find("Names")
	if !found {
		if !ensure {
			return nil
		}
		dict := NewPDFDict()

		indRef, err := xRefTable.IndRefForNewObject(dict)
		if err != nil {
			return err
		}
		d.Insert("Names", *indRef)

		d = &dict
	} else {
		d, err = xRefTable.DereferenceDict(obj)
		if err != nil {
			return err
		}
	}

	obj, found = d.Find(nameTreeName)
	if !found {
		if !ensure {
			return nil
		}
		dict := NewPDFDict()
		dict.Insert("Names", PDFArray{})

		indRef, err := xRefTable.IndRefForNewObject(dict)
		if err != nil {
			return err
		}

		d.Insert(nameTreeName, *indRef)

		xRefTable.Names[nameTreeName] = &Node{IndRef: indRef}

		return nil
	}

	indRef, ok := obj.(PDFIndirectRef)
	if !ok {
		return errors.New("LocateNameTree: name tree must be indirect ref")
	}

	xRefTable.Names[nameTreeName] = &Node{IndRef: &indRef}

	return nil
}

// NamesDict returns the dict that contains all name trees.
func (xRefTable *XRefTable) NamesDict() (*PDFDict, error) {

	rootDict, err := xRefTable.Catalog()
	if err != nil {
		return nil, err
	}

	obj, found := rootDict.Find("Names")
	if !found {
		return nil, errors.New("NamesDict: root entry \"Names\" missing")
	}

	return xRefTable.DereferenceDict(obj)
}

// RemoveNameTree removes a specific name tree.
// Also removes a resulting empty names dict.
func (xRefTable *XRefTable) RemoveNameTree(nameTreeName string) error {

	namesDict, err := xRefTable.NamesDict()
	if err != nil {
		return err
	}

	if namesDict == nil {
		return errors.New("RemoveNameTree: root entry \"Names\" corrupt")
	}

	// We have an existing name dict.

	// Delete the name tree.
	if indRef := namesDict.IndirectRefEntry(nameTreeName); indRef != nil {
		err = xRefTable.DeleteObjectGraph(*indRef)
		if err != nil {
			return err
		}
	}

	// Delete the name tree entry.
	namesDict.Delete(nameTreeName)
	if namesDict.Len() > 0 {
		return nil
	}

	// Remove empty names dict.

	rootDict, err := xRefTable.Catalog()
	if err != nil {
		return err
	}

	if indRef := rootDict.IndirectRefEntry("Names"); indRef != nil {
		err = xRefTable.DeleteObject(indRef.ObjectNumber.Value())
		if err != nil {
			return err
		}
	}

	rootDict.Delete("Names")

	log.Debug.Printf("Deleted Names from root: %s\n", rootDict)

	return nil
}

// RemoveCollection removes an existing Collection entry from the catalog.
func (xRefTable *XRefTable) RemoveCollection() error {

	rootDict, err := xRefTable.Catalog()
	if err != nil {
		return err
	}

	if indRef := rootDict.IndirectRefEntry("Collection"); indRef != nil {
		err = xRefTable.DeleteObjectGraph(*indRef)
		if err != nil {
			return err
		}
	}

	rootDict.Delete("Collection")
	log.Debug.Printf("deleted collection from root: %s\n", rootDict)

	return nil
}

// EnsureCollection makes sure there is a Collection entry in the catalog.
// Needed for portfolio / portable collections eg. for file attachments.
func (xRefTable *XRefTable) EnsureCollection() error {

	rootDict, err := xRefTable.Catalog()
	if err != nil {
		return err
	}

	_, found := rootDict.Find("Collection")
	if found {
		return nil
	}

	dict := NewPDFDict()
	dict.Insert("Type", PDFName("Collection"))
	dict.Insert("View", PDFName("D"))

	schemaDict := NewPDFDict()
	schemaDict.Insert("Type", PDFName("CollectionSchema"))

	fileNameCFDict := NewPDFDict()
	fileNameCFDict.Insert("Type", PDFName("CollectionField"))
	fileNameCFDict.Insert("Subtype", PDFName("F"))
	fileNameCFDict.Insert("N", PDFStringLiteral("Filename"))
	fileNameCFDict.Insert("O", PDFInteger(1))
	schemaDict.Insert("FileName", fileNameCFDict)

	descCFDict := NewPDFDict()
	descCFDict.Insert("Type", PDFName("CollectionField"))
	descCFDict.Insert("Subtype", PDFName("Desc"))
	descCFDict.Insert("N", PDFStringLiteral("Description"))
	descCFDict.Insert("O", PDFInteger(2))
	schemaDict.Insert("Description", descCFDict)

	sizeCFDict := NewPDFDict()
	sizeCFDict.Insert("Type", PDFName("CollectionField"))
	sizeCFDict.Insert("Subtype", PDFName("Size"))
	sizeCFDict.Insert("N", PDFStringLiteral("Size"))
	sizeCFDict.Insert("O", PDFInteger(3))
	schemaDict.Insert("Size", sizeCFDict)

	modDateCFDict := NewPDFDict()
	modDateCFDict.Insert("Type", PDFName("CollectionField"))
	modDateCFDict.Insert("Subtype", PDFName("ModDate"))
	modDateCFDict.Insert("N", PDFStringLiteral("Last Modification"))
	modDateCFDict.Insert("O", PDFInteger(4))
	schemaDict.Insert("ModDate", modDateCFDict)

	//TODO use xRefTable.InsertAndUseRecycled(xRefTableEntry)

	indRef, err := xRefTable.IndRefForNewObject(schemaDict)
	if err != nil {
		return err
	}
	dict.Insert("Schema", *indRef)

	sortDict := NewPDFDict()
	sortDict.Insert("S", PDFName("ModDate"))
	sortDict.Insert("A", PDFBoolean(false))
	dict.Insert("Sort", sortDict)

	indRef, err = xRefTable.IndRefForNewObject(dict)
	if err != nil {
		return err
	}
	rootDict.Insert("Collection", *indRef)

	return nil
}

// RemoveEmbeddedFilesNameTree removes both the embedded files name tree and the Collection dict.
func (xRefTable *XRefTable) RemoveEmbeddedFilesNameTree() error {

	delete(xRefTable.Names, "EmbeddedFiles")

	err := xRefTable.RemoveNameTree("EmbeddedFiles")
	if err != nil {
		return err
	}

	return xRefTable.RemoveCollection()
}

// IDFirstElement returns the first element of ID.
func (xRefTable *XRefTable) IDFirstElement() (id []byte, err error) {

	hl, ok := ((*xRefTable.ID)[0]).(PDFHexLiteral)
	if ok {
		id, err = hl.Bytes()
	} else {
		sl, ok := ((*xRefTable.ID)[0]).(PDFStringLiteral)
		if !ok {
			return nil, errors.New("ID must contain PDFHexLiterals or PDFStringLiterals")
		}
		id, err = Unescape(sl.Value())
	}

	return id, nil
}

func (xRefTable *XRefTable) processPageTree(root *PDFIndirectRef, p *int, page int) (*PDFDict, error) {

	dict, err := xRefTable.DereferenceDict(*root)
	if err != nil {
		return nil, err
	}

	// Iterate over page tree.
	kids := dict.PDFArrayEntry("Kids")

	for _, obj := range *kids {

		if obj == nil {
			continue
		}

		// Dereference next page node dict.
		indRef, ok := obj.(PDFIndirectRef)
		if !ok {
			return nil, errors.Errorf("processPageTree: corrupt page node dict")
		}

		pageNodeDict, err := xRefTable.DereferenceDict(indRef)
		if err != nil {
			return nil, err
		}

		if pageNodeDict == nil {
			return nil, errors.New("processPagesDict: pageNodeDict is null")
		}

		switch *pageNodeDict.Type() {

		case "Pages":
			// Recurse over sub pagetree.
			d, err := xRefTable.processPageTree(&indRef, p, page)
			if err != nil {
				return nil, err
			}

			if d != nil {
				return d, nil
			}

		case "Page":
			*p++
			// page found.
			if *p == page {
				return pageNodeDict, nil
			}

		}

	}

	return nil, nil
}

// PageDict returns a specific page dict.
func (xRefTable *XRefTable) PageDict(page int) (*PDFDict, error) {

	// Get an indirect reference to the root page dict.
	root, err := xRefTable.Pages()
	if err != nil {
		return nil, err
	}

	pageCount := 0

	return xRefTable.processPageTree(root, &pageCount, page)
}
