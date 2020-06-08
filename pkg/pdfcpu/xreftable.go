/*
Copyright 2018 The pdfcpu Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package pdfcpu

import (
	"encoding/hex"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/filter"
	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pkg/errors"
)

// XRefTableEntry represents an entry in the PDF cross reference table.
//
// This may wrap a free object, a compressed object or any in use PDF object:
//
// Dict, StreamDict, ObjectStreamDict, PDFXRefStreamDict,
// Array, Integer, Float, Name, StringLiteral, HexLiteral, Boolean
type XRefTableEntry struct {
	Free            bool
	Offset          *int64
	Generation      *int
	RefCount        int
	Object          Object
	Compressed      bool
	ObjectStream    *int
	ObjectStreamInd *int
	Valid           bool
}

// NewXRefTableEntryGen0 returns a cross reference table entry for an object with generation 0.
func NewXRefTableEntryGen0(obj Object) *XRefTableEntry {
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
	OE, UE     []byte
	Perms      []byte
	L, P, R, V int
	Emd        bool // encrypt meta data
	ID         []byte
}

// XRefTable represents a PDF cross reference table plus stats for a PDF file.
type XRefTable struct {
	Table               map[int]*XRefTableEntry
	Size                *int             // Object count from PDF trailer dict.
	PageCount           int              // Number of pages.
	Root                *IndirectRef     // Pointer to catalog (reference to root object).
	RootDict            Dict             // Catalog
	Names               map[string]*Node // Cache for name trees as found in catalog.
	Encrypt             *IndirectRef     // Encrypt dict.
	E                   *Enc
	EncKey              []byte // Encrypt key.
	AES4Strings         bool
	AES4Streams         bool
	AES4EmbeddedStreams bool

	// PDF Version
	HeaderVersion *Version // The PDF version the source is claiming to us as per its header.
	RootVersion   *Version // Optional PDF version taking precedence over the header version.

	// Document information section
	ID           Array        // from trailer
	Info         *IndirectRef // Infodict (reference to info dict object)
	Title        string
	Subject      string
	Keywords     string
	Author       string
	Creator      string
	Producer     string
	CreationDate string
	ModDate      string
	Properties   map[string]string

	// Linearization section (not yet supported)
	OffsetPrimaryHintTable  *int64
	OffsetOverflowHintTable *int64
	LinearizationObjs       IntSet

	// Offspec section
	AdditionalStreams *Array // array of IndirectRef - trailer :e.g., Oasis "Open Doc"

	// Statistics
	Stats PDFStats

	Tagged bool // File is using tags. This is important for ???

	// Validation
	Valid          bool // true means successful validated against ISO 32000.
	ValidationMode int  // see Configuration

	Optimized   bool
	Watermarked bool
}

// NewXRefTable creates a new XRefTable.
func newXRefTable(validationMode int) (xRefTable *XRefTable) {
	return &XRefTable{
		Table:             map[int]*XRefTableEntry{},
		Names:             map[string]*Node{},
		Properties:        map[string]string{},
		LinearizationObjs: IntSet{},
		Stats:             NewPDFStats(),
		ValidationMode:    validationMode,
	}
}

// Version returns the PDF version of the PDF writer that created this file.
// Before V1.4 this is the header version.
// Since V1.4 the catalog may contain a Version entry which takes precedence over the header version.
func (xRefTable *XRefTable) Version() Version {

	if xRefTable.RootVersion != nil {
		return *xRefTable.RootVersion
	}

	return *xRefTable.HeaderVersion
}

// VersionString return a string representation for this PDF files PDF version.
func (xRefTable *XRefTable) VersionString() string {
	return xRefTable.Version().String()
}

// ParseRootVersion returns a string representation for an optional Version entry in the root object.
func (xRefTable *XRefTable) ParseRootVersion() (v *string, err error) {

	// Look in the catalog/root for a name entry "Version".
	// This entry overrides the header version.

	rootDict, err := xRefTable.Catalog()
	if err != nil {
		return nil, err
	}

	return rootDict.NameEntry("Version"), nil
}

// ValidateVersion validates against the xRefTable's version.
func (xRefTable *XRefTable) ValidateVersion(element string, sinceVersion Version) error {

	if xRefTable.Version() < sinceVersion {
		return errors.Errorf("%s: unsupported in version %s\nThis file could be PDF/A compliant but pdfcpu only supports versions <= PDF V1.7\n", element, xRefTable.VersionString())
	}

	return nil
}

// EnsureVersionForWriting sets the version to the highest supported PDF Version 1.7.
// This is necessary to allow validation after adding features not supported
// by the original version of a document as during watermarking.
func (xRefTable *XRefTable) EnsureVersionForWriting() {
	v := V17
	xRefTable.RootVersion = &v
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
func (xRefTable *XRefTable) Exists(objNr int) bool {
	_, found := xRefTable.Table[objNr]
	return found
}

// Find returns the XRefTable entry for given object number.
func (xRefTable *XRefTable) Find(objNr int) (*XRefTableEntry, bool) {
	e, found := xRefTable.Table[objNr]
	if !found {
		return nil, false
	}
	return e, true
}

// FindObject returns the object of the XRefTableEntry for a specific object number.
func (xRefTable *XRefTable) FindObject(objNr int) (Object, error) {
	entry, ok := xRefTable.Find(objNr)
	if !ok {
		return nil, errors.Errorf("FindObject: obj#%d not registered in xRefTable", objNr)
	}
	return entry.Object, nil
}

// Free returns the cross ref table entry for given number of a free object.
func (xRefTable *XRefTable) Free(objNr int) (*XRefTableEntry, error) {
	entry, found := xRefTable.Find(objNr)
	if !found {
		return nil, nil //errors.Errorf("Free: object #%d not found.", objNr)
	}
	if !entry.Free {
		return nil, errors.Errorf("Free: object #%d found, but not free.", objNr)
	}
	return entry, nil
}

// NextForFree returns the number of the object the free object with objNumber links to.
// This is the successor of this free object in the free list.
func (xRefTable *XRefTable) NextForFree(objNr int) (int, error) {

	entry, err := xRefTable.Free(objNr)
	if err != nil {
		return 0, err
	}

	return int(*entry.Offset), nil
}

// FindTableEntryLight returns the XRefTable entry for given object number.
func (xRefTable *XRefTable) FindTableEntryLight(objNr int) (*XRefTableEntry, bool) {
	return xRefTable.Find(objNr)
}

// FindTableEntry returns the XRefTable entry for given object and generation numbers.
func (xRefTable *XRefTable) FindTableEntry(objNr int, genNr int) (*XRefTableEntry, bool) {

	//fmt.Printf("FindTableEntry: obj#:%d gen:%d \n", objNr, genNr)
	entry, found := xRefTable.Find(objNr)
	if !found || *entry.Generation != genNr {
		return nil, false
	}
	return entry, found
}

// FindTableEntryForIndRef returns the XRefTable entry for given indirect reference.
func (xRefTable *XRefTable) FindTableEntryForIndRef(ir *IndirectRef) (*XRefTableEntry, bool) {
	if ir == nil {
		return nil, false
	}
	return xRefTable.FindTableEntry(ir.ObjectNumber.Value(), ir.GenerationNumber.Value())
}

// InsertNew adds given xRefTableEntry at next new objNumber into the cross reference table.
// Only to be called once an xRefTable has been generated completely and all trailer dicts have been processed.
// xRefTable.Size is the size entry of the first trailer dict processed.
// Called on creation of new object streams.
// Called by InsertAndUseRecycled.
func (xRefTable *XRefTable) InsertNew(xRefTableEntry XRefTableEntry) (objNr int) {
	objNr = *xRefTable.Size
	xRefTable.Table[objNr] = &xRefTableEntry
	*xRefTable.Size++
	return
}

// InsertAndUseRecycled adds given xRefTableEntry into the cross reference table utilizing the freelist.
func (xRefTable *XRefTable) InsertAndUseRecycled(xRefTableEntry XRefTableEntry) (objNr int, err error) {

	// see 7.5.4 Cross-Reference Table

	// Hacky:
	// Although we increment the obj generation when recycling objects,
	// we always use generation 0 when reusing recycled objects.
	// This is because pdfcpu does not reuse objects
	// in an incremental fashion like laid out in the PDF spec.

	log.Write.Println("InsertAndUseRecycled: begin")

	// Get Next free object from freelist.
	freeListHeadEntry, err := xRefTable.Free(0)
	if err != nil {
		return 0, err
	}

	// If none available, add new object & return.
	if *freeListHeadEntry.Offset == 0 {
		xRefTableEntry.RefCount = 1
		objNr = xRefTable.InsertNew(xRefTableEntry)
		log.Write.Printf("InsertAndUseRecycled: end, new objNr=%d\n", objNr)
		return objNr, nil
	}

	// Recycle free object, update free list & return.
	objNr = int(*freeListHeadEntry.Offset)
	entry, found := xRefTable.FindTableEntryLight(objNr)
	if !found {
		return 0, errors.Errorf("InsertAndRecycle: no entry for obj #%d\n", objNr)
	}

	// The new free list head entry becomes the old head entry's successor.
	freeListHeadEntry.Offset = entry.Offset

	// The old head entry becomes garbage.
	entry.Free = false
	entry.Offset = nil

	// Create a new entry for the recycled object.
	// TODO use entrys generation.
	xRefTableEntry.RefCount = 1
	xRefTable.Table[objNr] = &xRefTableEntry

	log.Write.Printf("InsertAndUseRecycled: end, recycled objNr=%d\n", objNr)

	return objNr, nil
}

// InsertObject inserts an object into the xRefTable.
func (xRefTable *XRefTable) InsertObject(obj Object) (objNr int, err error) {
	xRefTableEntry := NewXRefTableEntryGen0(obj)
	xRefTableEntry.RefCount = 1
	return xRefTable.InsertNew(*xRefTableEntry), nil
}

// IndRefForNewObject inserts an object into the xRefTable and returns an indirect reference to it.
func (xRefTable *XRefTable) IndRefForNewObject(obj Object) (*IndirectRef, error) {

	xRefTableEntry := NewXRefTableEntryGen0(obj)

	objNr, err := xRefTable.InsertAndUseRecycled(*xRefTableEntry)
	if err != nil {
		return nil, err
	}

	return NewIndirectRef(objNr, *xRefTableEntry.Generation), nil
}

// NewStreamDictForBuf creates a streamDict for buf.
func (xRefTable *XRefTable) NewStreamDictForBuf(buf []byte) (*StreamDict, error) {
	sd := StreamDict{
		Dict:           NewDict(),
		Content:        buf,
		FilterPipeline: []PDFFilter{{Name: filter.Flate, DecodeParms: nil}},
	}
	sd.InsertName("Filter", filter.Flate)
	return &sd, nil
}

// NewStreamDictForFile creates a streamDict for filename.
func (xRefTable *XRefTable) NewStreamDictForFile(filename string) (*StreamDict, error) {
	buf, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	return xRefTable.NewStreamDictForBuf(buf)
}

// NewEmbeddedFileStreamDict creates and returns an embeddedFileStreamDict containing the file "filename".
func (xRefTable *XRefTable) NewEmbeddedFileStreamDict(filename string) (*IndirectRef, error) {
	sd, err := xRefTable.NewStreamDictForFile(filename)
	if err != nil {
		return nil, err
	}
	fi, err := os.Stat(filename)
	if err != nil {
		return nil, err
	}
	sd.InsertName("Type", "EmbeddedFile")
	d := NewDict()
	d.InsertInt("Size", int(fi.Size()))
	d.Insert("ModDate", StringLiteral(DateString(fi.ModTime())))
	sd.Insert("Params", d)

	if err = encodeStream(sd); err != nil {
		return nil, err
	}

	return xRefTable.IndRefForNewObject(*sd)
}

// NewSoundStreamDict returns a new sound stream dict.
func (xRefTable *XRefTable) NewSoundStreamDict(filename string, samplingRate int, fileSpecDict Dict) (*IndirectRef, error) {
	sd, err := xRefTable.NewStreamDictForFile(filename)
	if err != nil {
		return nil, err
	}
	sd.InsertName("Type", "Sound")
	sd.InsertInt("R", samplingRate)
	sd.InsertInt("C", 2)
	sd.InsertInt("B", 8)
	sd.InsertName("E", "Signed")
	if fileSpecDict != nil {
		sd.Insert("F", fileSpecDict)
	} else {
		sd.Insert("F", StringLiteral(path.Base(filename)))
	}

	if err = encodeStream(sd); err != nil {
		return nil, err
	}

	return xRefTable.IndRefForNewObject(*sd)
}

// NewFileSpecDict creates and returns a new fileSpec dictionary.
func (xRefTable *XRefTable) NewFileSpecDict(filename, desc string, indRefStreamDict IndirectRef) (Dict, error) {

	d := NewDict()
	d.InsertName("Type", "Filespec")
	d.InsertString("F", filename)
	d.InsertString("UF", filename)
	// TODO d.Insert("UF", utf16.Encode([]rune(filename)))

	efDict := NewDict()
	efDict.Insert("F", indRefStreamDict)
	efDict.Insert("UF", indRefStreamDict)
	d.Insert("EF", efDict)

	d.InsertString("Desc", desc)

	// CI, optional, collection item dict, since V1.7
	// a corresponding collection schema dict in a collection.
	ciDict := NewDict()
	//add contextual meta info here.
	d.Insert("CI", ciDict)

	return d, nil
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

	log.Trace.Println("EnsureValidFreeList begin")

	m := xRefTable.freeObjects()

	// Verify free object 0 as free list head.
	head, err := xRefTable.Free(0)
	if err != nil {
		return err
	}

	if head == nil {
		g0 := FreeHeadGeneration
		z := int64(0)
		head = &XRefTableEntry{Free: true, Offset: &z, Generation: &g0}
		xRefTable.Table[0] = head
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

		log.Trace.Println("EnsureValidFreeList: empty free list.")
		return nil
	}

	f := int(*head.Offset)

	// until we have found the last free object which should point to obj 0.
	for f != 0 {

		log.Trace.Printf("EnsureValidFreeList: validating obj #%d %v\n", f, m)
		// verify if obj f is one of the free objects recorded.
		if !m[f] {
			return errors.New("pdfcpu: ensureValidFreeList: freelist corrupted")
		}

		delete(m, f)

		f, err = xRefTable.NextForFree(f)
		if err != nil {
			return err
		}
	}

	if len(m) == 0 {
		log.Trace.Println("EnsureValidFreeList: end, regular linked list")
		return nil
	}

	// insert remaining free objects into verified linked list
	// unless they are forever deleted with generation 65535.
	// In that case they have to point to obj 0.
	for i := range m {

		entry, found := xRefTable.FindTableEntryLight(i)
		if !found {
			return errors.Errorf("pdfcpu: ensureValidFreeList: no xref entry found for obj #%d\n", i)
		}

		if !entry.Free {
			return errors.Errorf("pdfcpu: ensureValidFreeList: xref entry is not free for obj #%d\n", i)
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

	log.Trace.Println("EnsureValidFreeList: end, linked list plus some dangling free objects.")

	return nil
}

func (xRefTable *XRefTable) deleteDictEntry(d Dict, key string) error {
	o, found := d.Find(key)
	if !found {
		return nil
	}
	if err := xRefTable.deleteObject(o); err != nil {
		return err
	}
	d.Delete(key)
	return nil
}

func (xRefTable *XRefTable) locateObjForIndRef(ir IndirectRef) (Object, error) {

	var err error
	objNr := int(ir.ObjectNumber)

	entry, found := xRefTable.FindTableEntryLight(objNr)
	if !found {
		return nil, errors.Errorf("pdfcpu: locateObjForIndRef: no xref entry found for obj #%d\n", objNr)
	}

	if entry.RefCount > 1 {
		entry.RefCount--
		//fmt.Printf("locateObjForIndRef(%d): new refcount: %d\n", objNr, entry.RefCount)
		return nil, nil
	}

	o, err := xRefTable.Dereference(ir)
	if err != nil || o == nil {
		return o, err
	}

	if err = xRefTable.DeleteObject(objNr); err != nil {
		return nil, err
	}

	return o, nil
}

func (xRefTable *XRefTable) deleteObject(o Object) error {

	var err error

	ir, ok := o.(IndirectRef)
	if ok {
		o, err = xRefTable.locateObjForIndRef(ir)
		if err != nil || o == nil {
			return err
		}
	}

	switch o := o.(type) {

	case Dict:
		for _, v := range o {
			err := xRefTable.deleteObject(v)
			if err != nil {
				return err
			}
		}

	case StreamDict:
		for _, v := range o.Dict {
			err := xRefTable.deleteObject(v)
			if err != nil {
				return err
			}
		}

	case Array:
		for _, v := range o {
			err := xRefTable.deleteObject(v)
			if err != nil {
				return err
			}
		}

	}

	return nil
}

// DeleteObjectGraph deletes all objects reachable by indRef.
func (xRefTable *XRefTable) DeleteObjectGraph(o Object) error {

	log.Debug.Println("DeleteObjectGraph: begin")

	ir, ok := o.(IndirectRef)
	if !ok {
		return nil
	}

	// Delete ObjectGraph for object indRef.ObjectNumber.Value() via recursion.
	if err := xRefTable.deleteObject(ir); err != nil {
		return err
	}

	log.Debug.Println("DeleteObjectGraph: end")
	return nil
}

// DeleteObject marks an object as free and inserts it into the free list right after the head.
func (xRefTable *XRefTable) DeleteObject(objNr int) error {

	// see 7.5.4 Cross-Reference Table

	log.Debug.Printf("DeleteObject: begin %d\n", objNr)

	freeListHeadEntry, err := xRefTable.Free(0)
	if err != nil {
		return err
	}

	entry, found := xRefTable.FindTableEntryLight(objNr)
	if !found {
		return errors.Errorf("pdfcpu: deleteObject: no entry for obj #%d\n", objNr)
	}

	if entry.Free {
		log.Debug.Printf("DeleteObject: end %d already free\n", objNr)
		return nil
	}

	*entry.Generation++
	entry.Free = true
	entry.Compressed = false
	entry.Offset = freeListHeadEntry.Offset
	entry.Object = nil
	entry.RefCount = 0

	next := int64(objNr)
	freeListHeadEntry.Offset = &next

	log.Debug.Printf("DeleteObject: end %d\n", objNr)

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
			if *entry.Generation > 0 {
				*entry.Generation--
			}
			entry.Free = false
			return nil
		}

		f = entry
	}

	log.Debug.Printf("UndeleteObject: end: obj#%d not in free list.\n", objectNumber)

	return nil
}

// indRefToObject dereferences an indirect object from the xRefTable and returns the result.
func (xRefTable *XRefTable) indRefToObject(ir *IndirectRef) (Object, error) {
	if ir == nil {
		return nil, errors.New("pdfcpu: indRefToObject: input argument is nil")
	}

	// 7.3.10
	// An indirect reference to an undefined object shall not be considered an error by a conforming reader;
	// it shall be treated as a reference to the null object.
	entry, found := xRefTable.FindTableEntryForIndRef(ir)
	if !found || entry.Free {
		return nil, nil
	}

	// return dereferenced object
	return entry.Object, nil
}

// Dereference resolves an indirect object and returns the resulting PDF object.
func (xRefTable *XRefTable) Dereference(o Object) (Object, error) {
	ir, ok := o.(IndirectRef)
	if !ok {
		// Nothing do dereference.
		return o, nil
	}

	return xRefTable.indRefToObject(&ir)
}

// DereferenceBoolean resolves and validates a boolean object, which may be an indirect reference.
func (xRefTable *XRefTable) DereferenceBoolean(o Object, sinceVersion Version) (*Boolean, error) {

	o, err := xRefTable.Dereference(o)
	if err != nil || o == nil {
		return nil, err
	}

	b, ok := o.(Boolean)
	if !ok {
		return nil, errors.Errorf("pdfcpu: dereferenceBoolean: wrong type <%v>", o)
	}

	// Version check
	if err = xRefTable.ValidateVersion("DereferenceBoolean", sinceVersion); err != nil {
		return nil, err
	}

	return &b, nil
}

// DereferenceInteger resolves and validates an integer object, which may be an indirect reference.
func (xRefTable *XRefTable) DereferenceInteger(o Object) (*Integer, error) {

	o, err := xRefTable.Dereference(o)
	if err != nil || o == nil {
		return nil, err
	}

	i, ok := o.(Integer)
	if !ok {
		return nil, errors.Errorf("pdfcpu: dereferenceInteger: wrong type <%v>", o)
	}

	return &i, nil
}

// DereferenceNumber resolves a number object, which may be an indirect reference and returns a float64.
func (xRefTable *XRefTable) DereferenceNumber(o Object) (float64, error) {

	var (
		f   float64
		err error
	)

	o, _ = xRefTable.Dereference(o)

	switch o := o.(type) {

	case Integer:
		f = float64(o.Value())

	case Float:
		f = o.Value()

	default:
		err = errors.Errorf("pdfcpu: dereferenceNumber: wrong type <%v>", o)

	}

	return f, err
}

// DereferenceName resolves and validates a name object, which may be an indirect reference.
func (xRefTable *XRefTable) DereferenceName(o Object, sinceVersion Version, validate func(string) bool) (n Name, err error) {

	o, err = xRefTable.Dereference(o)
	if err != nil || o == nil {
		return n, err
	}

	n, ok := o.(Name)
	if !ok {
		return n, errors.Errorf("pdfcpu: dereferenceName: wrong type <%v>", o)
	}

	// Version check
	if err = xRefTable.ValidateVersion("DereferenceName", sinceVersion); err != nil {
		return n, err
	}

	// Validation
	if validate != nil && !validate(n.Value()) {
		return n, errors.Errorf("pdfcpu: dereferenceName: invalid <%s>", n.Value())
	}

	return n, nil
}

// DereferenceStringLiteral resolves and validates a string literal object, which may be an indirect reference.
func (xRefTable *XRefTable) DereferenceStringLiteral(o Object, sinceVersion Version, validate func(string) bool) (s StringLiteral, err error) {

	o, err = xRefTable.Dereference(o)
	if err != nil || o == nil {
		return s, err
	}

	s, ok := o.(StringLiteral)
	if !ok {
		return s, errors.Errorf("pdfcpu: dereferenceStringLiteral: wrong type <%v>", o)
	}

	// Ensure UTF16 correctness.
	s1, err := StringLiteralToString(s.Value())
	if err != nil {
		return s, err
	}

	// Version check
	if err = xRefTable.ValidateVersion("DereferenceStringLiteral", sinceVersion); err != nil {
		return s, err
	}

	// Validation
	if validate != nil && !validate(s1) {
		return s, errors.Errorf("pdfcpu: dereferenceStringLiteral: invalid <%s>", s1)
	}

	return s, nil
}

// DereferenceStringOrHexLiteral resolves and validates a string or hex literal object, which may be an indirect reference.
func (xRefTable *XRefTable) DereferenceStringOrHexLiteral(obj Object, sinceVersion Version, validate func(string) bool) (s string, err error) {

	o, err := xRefTable.Dereference(obj)
	if err != nil || o == nil {
		return "", err
	}

	switch str := o.(type) {

	case StringLiteral:
		// Ensure UTF16 correctness.
		if s, err = StringLiteralToString(str.Value()); err != nil {
			return "", err
		}

	case HexLiteral:
		// Ensure UTF16 correctness.
		if s, err = HexLiteralToString(str.Value()); err != nil {
			return "", err
		}

	default:
		return "", errors.Errorf("pdfcpu: dereferenceStringOrHexLiteral: wrong type <%v>", obj)

	}

	// Version check
	if err = xRefTable.ValidateVersion("DereferenceStringOrHexLiteral", sinceVersion); err != nil {
		return "", err
	}

	// Validation
	if validate != nil && !validate(s) {
		return "", errors.Errorf("pdfcpu: dereferenceStringOrHexLiteral: invalid <%s>", s)
	}

	return s, nil
}

// Text returns a string based representation for String and Hexliterals.
func Text(o Object) (string, error) {
	switch obj := o.(type) {
	case StringLiteral:
		return StringLiteralToString(obj.Value())
	case HexLiteral:
		return HexLiteralToString(obj.Value())
	default:
		return "", errors.Errorf("pdfcpu: text: corrupt -  %v\n", obj)
	}
}

// DereferenceText resolves and validates a string or hex literal object to a string.
func (xRefTable *XRefTable) DereferenceText(o Object) (string, error) {
	o, err := xRefTable.Dereference(o)
	if err != nil {
		return "", err
	}
	return Text(o)
}

// DereferenceCSVSafeText resolves and validates a string or hex literal object to a string.
func (xRefTable *XRefTable) DereferenceCSVSafeText(o Object) (string, error) {
	s, err := xRefTable.DereferenceText(o)
	if err != nil {
		return "", err
	}
	return csvSafeString(s), nil
}

// DereferenceArray resolves and validates an array object, which may be an indirect reference.
func (xRefTable *XRefTable) DereferenceArray(o Object) (Array, error) {

	o, err := xRefTable.Dereference(o)
	if err != nil || o == nil {
		return nil, err
	}

	a, ok := o.(Array)
	if ok {
		return a, nil
	}

	d, ok := o.(Dict)
	if !ok {
		return nil, errors.Errorf("pdfcpu: dereferenceArray: dest of wrong type <%v>", o)
	}

	return d["D"].(Array), nil
}

// DereferenceDict resolves and validates a dictionary object, which may be an indirect reference.
func (xRefTable *XRefTable) DereferenceDict(o Object) (Dict, error) {

	o, err := xRefTable.Dereference(o)
	if err != nil || o == nil {
		return nil, err
	}

	d, ok := o.(Dict)
	if !ok {
		return nil, errors.Errorf("pdfcpu: dereferenceDict: wrong type %T <%v>", o, o)
	}

	return d, nil
}

// DereferenceStreamDict resolves and validates a stream dictionary object, which may be an indirect reference.
func (xRefTable *XRefTable) DereferenceStreamDict(o Object) (*StreamDict, error) {

	o, err := xRefTable.Dereference(o)
	if err != nil || o == nil {
		return nil, err
	}

	sd, ok := o.(StreamDict)
	if !ok {
		return nil, errors.Errorf("pdfcpu: dereferenceStreamDict: wrong type <%v>", o)
	}

	return &sd, nil
}

// DereferenceStreamDictForValidation resolves stream dictionary objects
// and ensures they are visited once only during validation.
func (xRefTable *XRefTable) DereferenceStreamDictForValidation(o Object) (*StreamDict, error) {

	ir, ok := o.(IndirectRef)
	if !ok {
		// Nothing do dereference.
		return nil, nil
	}

	// 7.3.10
	// An indirect reference to an undefined object shall not be considered an error by a conforming reader;
	// it shall be treated as a reference to the null object.
	entry, found := xRefTable.FindTableEntry(ir.ObjectNumber.Value(), ir.GenerationNumber.Value())
	if !found || entry.Free || entry.Valid || entry.Object == nil {
		return nil, nil
	}
	entry.Valid = true

	sd, ok := entry.Object.(StreamDict)
	if !ok {
		return nil, errors.Errorf("pdfcpu: dereferenceStreamDict: wrong type <%v> %T", o, entry.Object)
	}

	return &sd, nil
}

// DereferenceDictEntry returns a dereferenced dict entry.
func (xRefTable *XRefTable) DereferenceDictEntry(d Dict, entryName string) (Object, error) {

	o, found := d.Find(entryName)
	if !found || o == nil {
		return nil, errors.Errorf("pdfcpu: dict=%s entry=%s missing.", d, entryName)
	}

	return xRefTable.Dereference(o)
}

// Catalog returns a pointer to the root object / catalog.
func (xRefTable *XRefTable) Catalog() (Dict, error) {

	if xRefTable.RootDict != nil {
		return xRefTable.RootDict, nil
	}

	if xRefTable.Root == nil {
		return nil, errors.New("pdfcpu: catalog: missing root dict")
	}

	o, err := xRefTable.indRefToObject(xRefTable.Root)
	if err != nil || o == nil {
		return nil, err
	}

	d, ok := o.(Dict)
	if !ok {
		return nil, errors.New("pdfcpu: catalog: corrupt root catalog")
	}

	xRefTable.RootDict = d

	return xRefTable.RootDict, nil
}

// EncryptDict returns a pointer to the root object / catalog.
func (xRefTable *XRefTable) EncryptDict() (Dict, error) {

	o, err := xRefTable.indRefToObject(xRefTable.Encrypt)
	if err != nil || o == nil {
		return nil, err
	}

	d, ok := o.(Dict)
	if !ok {
		return nil, errors.New("pdfcpu: encryptDict: corrupt encrypt dict")
	}

	return d, nil
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
func (xRefTable *XRefTable) Pages() (*IndirectRef, error) {
	rootDict, err := xRefTable.Catalog()
	if err != nil {
		return nil, err
	}
	return rootDict.IndirectRefEntry("Pages"), nil
}

// Outlines returns the Outlines reference contained in the catalog.
func (xRefTable *XRefTable) Outlines() (*IndirectRef, error) {
	rootDict, err := xRefTable.Catalog()
	if err != nil {
		return nil, err
	}
	return rootDict.IndirectRefEntry("Outlines"), nil
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

				d, ok := entry.Object.(Dict)

				if ok {
					if d.Type() != nil {
						typeStr += fmt.Sprintf(" type=%s", *d.Type())
					}
					if d.Subtype() != nil {
						typeStr += fmt.Sprintf(" subType=%s", *d.Subtype())
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

				sd, ok := entry.Object.(StreamDict)
				if ok && log.IsTraceLoggerEnabled() { //&& sd.IsPageContent {
					s := "decoded stream content (length = %d)\n<%s>\n"
					if sd.IsPageContent {
						str += fmt.Sprintf(s, len(sd.Content), sd.Content)
					} else {
						str += fmt.Sprintf(s, len(sd.Content), hex.Dump(sd.Content))
					}
				}

				osd, ok := entry.Object.(ObjectStreamDict)
				if ok {
					str += fmt.Sprintf("object stream count:%d size of objectarray:%d\n", osd.ObjCount, len(osd.ObjArray))
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

	log.Trace.Printf("freeList begin")

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

		log.Trace.Printf("freeList validating free object %d\n", f)

		entry, err := xRefTable.Free(f)
		if err != nil {
			return nil, err
		}

		next := int(*entry.Offset)
		generation := *entry.Generation
		s := fmt.Sprintf("%5d %5d %5d\n", f, next, generation)
		logStr = append(logStr, s)
		log.Trace.Printf("freeList: %s", s)

		f = next
	}

	log.Trace.Printf("freeList end")

	return logStr, nil
}

func (xRefTable *XRefTable) bindNameTreeNode(name string, n *Node, root bool) error {

	var dict Dict

	if n.D == nil {
		dict = NewDict()
		n.D = dict
	} else {
		if root {
			// Update root object after possible tree modification after removal of empty kid.
			namesDict, err := xRefTable.NamesDict()
			if err != nil {
				return err
			}
			if namesDict == nil {
				return errors.New("pdfcpu: root entry \"Names\" corrupt")
			}
			namesDict.Update(name, n.D)
		}
		log.Debug.Printf("bind dict = %v\n", n.D)
		dict = n.D
	}

	if !root {
		dict.Update("Limits", NewStringArray(n.Kmin, n.Kmax))
	} else {
		dict.Delete("Limits")
	}

	if n.leaf() {
		a := Array{}
		for _, e := range n.Names {
			a = append(a, StringLiteral(e.k))
			a = append(a, e.v)
		}
		dict.Update("Names", a)
		log.Debug.Printf("bound nametree node(leaf): %s/n", dict)
		return nil
	}

	kids := Array{}
	for _, k := range n.Kids {
		err := xRefTable.bindNameTreeNode(name, k, false)
		if err != nil {
			return err
		}
		indRef, err := xRefTable.IndRefForNewObject(k.D)
		if err != nil {
			return err
		}
		kids = append(kids, *indRef)
	}

	dict.Update("Kids", kids)
	dict.Delete("Names")

	log.Debug.Printf("bound nametree node(intermediary): %s/n", dict)

	return nil
}

// BindNameTrees syncs up the internal name tree cache with the xreftable.
func (xRefTable *XRefTable) BindNameTrees() error {

	log.Write.Println("BindNameTrees..")

	// Iterate over internal name tree rep.
	for k, v := range xRefTable.Names {
		log.Write.Printf("bindNameTree: %s\n", k)
		if err := xRefTable.bindNameTreeNode(k, v, true); err != nil {
			return err
		}
	}

	return nil
}

// LocateNameTree locates/ensures a specific name tree.
func (xRefTable *XRefTable) LocateNameTree(nameTreeName string, ensure bool) error {

	if xRefTable.Names[nameTreeName] != nil {
		return nil
	}

	d, err := xRefTable.Catalog()
	if err != nil {
		return err
	}

	o, found := d.Find("Names")
	if !found {
		if !ensure {
			return nil
		}
		dict := NewDict()

		ir, err := xRefTable.IndRefForNewObject(dict)
		if err != nil {
			return err
		}
		d.Insert("Names", *ir)

		d = dict
	} else {
		d, err = xRefTable.DereferenceDict(o)
		if err != nil {
			return err
		}
	}

	o, found = d.Find(nameTreeName)
	if !found {
		if !ensure {
			return nil
		}
		dict := NewDict()
		dict.Insert("Names", Array{})

		ir, err := xRefTable.IndRefForNewObject(dict)
		if err != nil {
			return err
		}

		d.Insert(nameTreeName, *ir)

		xRefTable.Names[nameTreeName] = &Node{D: dict}

		return nil
	}

	d1, err := xRefTable.DereferenceDict(o)
	if err != nil {
		return err
	}

	xRefTable.Names[nameTreeName] = &Node{D: d1}

	return nil
}

// NamesDict returns the dict that contains all name trees.
func (xRefTable *XRefTable) NamesDict() (Dict, error) {

	rootDict, err := xRefTable.Catalog()
	if err != nil {
		return nil, err
	}

	o, found := rootDict.Find("Names")
	if !found {
		return nil, errors.New("pdfcpu: NamesDict: root entry \"Names\" missing")
	}

	return xRefTable.DereferenceDict(o)
}

// RemoveNameTree removes a specific name tree.
// Also removes a resulting empty names dict.
func (xRefTable *XRefTable) RemoveNameTree(nameTreeName string) error {

	namesDict, err := xRefTable.NamesDict()
	if err != nil {
		return err
	}

	if namesDict == nil {
		return errors.New("pdfcpu: removeNameTree: root entry \"Names\" corrupt")
	}

	// We have an existing name dict.

	// Delete the name tree.
	if err = xRefTable.deleteDictEntry(namesDict, nameTreeName); err != nil {
		return err
	}
	if namesDict.Len() > 0 {
		return nil
	}

	// Remove empty names dict.
	rootDict, err := xRefTable.Catalog()
	if err != nil {
		return err
	}
	if err = xRefTable.deleteDictEntry(rootDict, "Names"); err != nil {
		return err
	}

	log.Debug.Printf("Deleted Names from root: %s\n", rootDict)

	return nil
}

// RemoveCollection removes an existing Collection entry from the catalog.
func (xRefTable *XRefTable) RemoveCollection() error {
	rootDict, err := xRefTable.Catalog()
	if err != nil {
		return err
	}
	return xRefTable.deleteDictEntry(rootDict, "Collection")
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

	dict := NewDict()
	dict.Insert("Type", Name("Collection"))
	dict.Insert("View", Name("D"))

	schemaDict := NewDict()
	schemaDict.Insert("Type", Name("CollectionSchema"))

	fileNameCFDict := NewDict()
	fileNameCFDict.Insert("Type", Name("CollectionField"))
	fileNameCFDict.Insert("Subtype", Name("F"))
	fileNameCFDict.Insert("N", StringLiteral("Filename"))
	fileNameCFDict.Insert("O", Integer(1))
	schemaDict.Insert("FileName", fileNameCFDict)

	descCFDict := NewDict()
	descCFDict.Insert("Type", Name("CollectionField"))
	descCFDict.Insert("Subtype", Name("Desc"))
	descCFDict.Insert("N", StringLiteral("Description"))
	descCFDict.Insert("O", Integer(2))
	schemaDict.Insert("Description", descCFDict)

	sizeCFDict := NewDict()
	sizeCFDict.Insert("Type", Name("CollectionField"))
	sizeCFDict.Insert("Subtype", Name("Size"))
	sizeCFDict.Insert("N", StringLiteral("Size"))
	sizeCFDict.Insert("O", Integer(3))
	schemaDict.Insert("Size", sizeCFDict)

	modDateCFDict := NewDict()
	modDateCFDict.Insert("Type", Name("CollectionField"))
	modDateCFDict.Insert("Subtype", Name("ModDate"))
	modDateCFDict.Insert("N", StringLiteral("Last Modification"))
	modDateCFDict.Insert("O", Integer(4))
	schemaDict.Insert("ModDate", modDateCFDict)

	//TODO use xRefTable.InsertAndUseRecycled(xRefTableEntry)

	ir, err := xRefTable.IndRefForNewObject(schemaDict)
	if err != nil {
		return err
	}
	dict.Insert("Schema", *ir)

	sortDict := NewDict()
	sortDict.Insert("S", Name("ModDate"))
	sortDict.Insert("A", Boolean(false))
	dict.Insert("Sort", sortDict)

	ir, err = xRefTable.IndRefForNewObject(dict)
	if err != nil {
		return err
	}
	rootDict.Insert("Collection", *ir)

	return nil
}

// RemoveEmbeddedFilesNameTree removes both the embedded files name tree and the Collection dict.
func (xRefTable *XRefTable) RemoveEmbeddedFilesNameTree() error {

	delete(xRefTable.Names, "EmbeddedFiles")

	if err := xRefTable.RemoveNameTree("EmbeddedFiles"); err != nil {
		return err
	}

	return xRefTable.RemoveCollection()
}

// IDFirstElement returns the first element of ID.
func (xRefTable *XRefTable) IDFirstElement() (id []byte, err error) {

	hl, ok := xRefTable.ID[0].(HexLiteral)
	if ok {
		return hl.Bytes()
	}

	sl, ok := xRefTable.ID[0].(StringLiteral)
	if !ok {
		return nil, errors.New("pdfcpu: ID must contain hex literals or string literals")
	}

	return Unescape(sl.Value())
}

// InheritedPageAttrs represents all inherited page attributes.
type InheritedPageAttrs struct {
	resources Dict
	mediaBox  *Rectangle
	cropBox   *Rectangle
	rotate    int
}

func rect(xRefTable *XRefTable, a Array) (*Rectangle, error) {

	llx, err := xRefTable.DereferenceNumber(a[0])
	if err != nil {
		return nil, err
	}

	lly, err := xRefTable.DereferenceNumber(a[1])
	if err != nil {
		return nil, err
	}

	urx, err := xRefTable.DereferenceNumber(a[2])
	if err != nil {
		return nil, err
	}

	ury, err := xRefTable.DereferenceNumber(a[3])
	if err != nil {
		return nil, err
	}

	return Rect(llx, lly, urx, ury), nil
}

func (xRefTable *XRefTable) checkInheritedPageAttrs(pageDict Dict, pAttrs *InheritedPageAttrs) error {

	var err error

	obj, found := pageDict.Find("Resources")
	if found {
		pAttrs.resources, err = xRefTable.DereferenceDict(obj)
		if err != nil {
			return err
		}
	}

	obj, found = pageDict.Find("MediaBox")
	if found {
		a, err := xRefTable.DereferenceArray(obj)
		if err != nil {
			return err
		}
		pAttrs.mediaBox, err = rect(xRefTable, a)
		if err != nil {
			return err
		}
	}

	obj, found = pageDict.Find("CropBox")
	if found {
		a, err := xRefTable.DereferenceArray(obj)
		if err != nil {
			return err
		}
		pAttrs.cropBox, err = rect(xRefTable, a)
		if err != nil {
			return err
		}
	}

	obj, found = pageDict.Find("Rotate")
	if found {
		i, err := xRefTable.DereferenceInteger(obj)
		if err != nil {
			return err
		}
		pAttrs.rotate = i.Value()
	}

	return nil
}

func (xRefTable *XRefTable) processPageTreeForPageDict(root *IndirectRef, pAttrs *InheritedPageAttrs, p *int, page int) (Dict, error) {

	//fmt.Printf("entering processPageTreeForPageDict: p=%d obj#%d\n", *p, root.ObjectNumber.Value())

	d, err := xRefTable.DereferenceDict(*root)
	if err != nil {
		return nil, err
	}

	pageCount := d.IntEntry("Count")
	if pageCount != nil {
		if *p+*pageCount < page {
			// Skip sub pagetree.
			*p += *pageCount
			return nil, nil
		}
	}

	if err = xRefTable.checkInheritedPageAttrs(d, pAttrs); err != nil {
		return nil, err
	}

	// Iterate over page tree.
	kids := d.ArrayEntry("Kids")
	if kids == nil {
		//fmt.Println("returning from leaf node")
		return d, nil
	}

	for _, o := range kids {

		if o == nil {
			continue
		}

		// Dereference next page node dict.
		ir, ok := o.(IndirectRef)
		if !ok {
			return nil, errors.Errorf("pdfcpu: processPageTreeForPageDict: corrupt page node dict")
		}

		pageNodeDict, err := xRefTable.DereferenceDict(ir)
		if err != nil {
			return nil, err
		}

		switch *pageNodeDict.Type() {

		case "Pages":
			// Recurse over sub pagetree.
			pageNodeDict, err = xRefTable.processPageTreeForPageDict(&ir, pAttrs, p, page)
			if err != nil {
				return nil, err
			}
			if pageNodeDict != nil {
				return pageNodeDict, nil
			}

		case "Page":
			*p++
			if *p == page {
				return xRefTable.processPageTreeForPageDict(&ir, pAttrs, p, page)
			}

		}

	}

	return nil, nil
}

// PageDict returns a specific page dict along with the resources, mediaBox and CropBox in effect.
func (xRefTable *XRefTable) PageDict(page int) (Dict, *InheritedPageAttrs, error) {

	// Get an indirect reference to the page tree root dict.
	pageRootDictIndRef, _ := xRefTable.Pages()

	var (
		inhPAttrs InheritedPageAttrs
		pageCount int
	)

	pageDict, err := xRefTable.processPageTreeForPageDict(pageRootDictIndRef, &inhPAttrs, &pageCount, page)
	if err != nil {
		return nil, nil, err
	}

	return pageDict, &inhPAttrs, nil
}

func (xRefTable *XRefTable) processPageTreeForPageNumber(root *IndirectRef, pageCount *int, pageObjNr int) (int, error) {

	//fmt.Printf("entering processPageTreeForPageNumber: p=%d obj#%d\n", *p, root.ObjectNumber.Value())

	d, err := xRefTable.DereferenceDict(*root)
	if err != nil {
		return 0, err
	}

	// Iterate over page tree.
	for _, o := range d.ArrayEntry("Kids") {

		if o == nil {
			continue
		}

		// Dereference next page node dict.
		ir, ok := o.(IndirectRef)
		if !ok {
			return 0, errors.Errorf("pdfcpu: processPageTreeForPageNumber: corrupt page node dict")
		}

		objNr := ir.ObjectNumber.Value()

		pageNodeDict, err := xRefTable.DereferenceDict(ir)
		if err != nil {
			return 0, err
		}

		switch *pageNodeDict.Type() {

		case "Pages":
			// Recurse over sub pagetree.
			pageNr, err := xRefTable.processPageTreeForPageNumber(&ir, pageCount, pageObjNr)
			if err != nil {
				return 0, err
			}
			if pageNr > 0 {
				return pageNr, nil
			}

		case "Page":
			*pageCount++
			if objNr == pageObjNr {
				return *pageCount, nil
			}
		}

	}

	return 0, nil
}

// PageNumber returns the logical page number for a page dict object number.
func (xRefTable *XRefTable) PageNumber(pageObjNr int) (int, error) {
	// Get an indirect reference to the page tree root dict.
	pageRootDict, _ := xRefTable.Pages()
	pageCount := 0
	return xRefTable.processPageTreeForPageNumber(pageRootDict, &pageCount, pageObjNr)
}

// EnsurePageCount evaluates the page count for xRefTable if necessary.
func (xRefTable *XRefTable) EnsurePageCount() error {

	if xRefTable.PageCount > 0 {
		return nil
	}

	pageRoot, err := xRefTable.Pages()
	if err != nil {
		return err
	}

	d, err := xRefTable.DereferenceDict(*pageRoot)
	if err != nil {
		return err
	}

	pageCount := d.IntEntry("Count")
	if pageCount == nil {
		return errors.New("pdfcpu: pageDict: missing \"Count\"")
	}

	xRefTable.PageCount = *pageCount

	return nil
}

func (xRefTable *XRefTable) collectMediaBoxesForPageTree(root *IndirectRef, inhMediaBox *Rectangle, mediaBoxes []*Rectangle, p *int) (*Rectangle, error) {

	d, err := xRefTable.DereferenceDict(*root)
	if err != nil {
		return nil, err
	}

	obj, found := d.Find("MediaBox")
	if found {
		a, err := xRefTable.DereferenceArray(obj)
		if err != nil {
			return nil, err
		}
		if inhMediaBox, err = rect(xRefTable, a); err != nil {
			return nil, err
		}
	}

	// Iterate over page tree.
	kids := d.ArrayEntry("Kids")
	if kids == nil {
		return inhMediaBox, nil
	}

	for _, o := range kids {

		if o == nil {
			continue
		}

		// Dereference next page node dict.
		ir, ok := o.(IndirectRef)
		if !ok {
			return nil, errors.Errorf("pdfcpu: collectMediaBoxesForPageTree: corrupt page node dict")
		}

		pageNodeDict, err := xRefTable.DereferenceDict(ir)
		if err != nil {
			return nil, err
		}

		switch *pageNodeDict.Type() {

		case "Pages":
			// Recurse over sub pagetree.
			if _, err = xRefTable.collectMediaBoxesForPageTree(&ir, inhMediaBox, mediaBoxes, p); err != nil {
				return nil, err
			}

		case "Page":
			mediaBox, err := xRefTable.collectMediaBoxesForPageTree(&ir, inhMediaBox, mediaBoxes, p)
			if err != nil {
				return nil, err
			}
			if mediaBox == nil {
				return nil, errors.New("pdfcpu: collectMediaBoxesForPageTree: mediaBox is nil")
			}
			mediaBoxes[*p] = mediaBox
			*p++
		}

	}

	return nil, nil
}

// PageDims returns a sorted slice with media box dimensions
// for all pages sorted ascending by page number.
func (xRefTable *XRefTable) PageDims() ([]Dim, error) {

	if err := xRefTable.EnsurePageCount(); err != nil {
		return nil, err
	}

	mediaBoxes := make([]*Rectangle, xRefTable.PageCount)

	// Get an indirect reference to the page tree root dict.
	root, err := xRefTable.Pages()
	if err != nil {
		return nil, err
	}

	i := 0
	var mb Rectangle
	if _, err := xRefTable.collectMediaBoxesForPageTree(root, &mb, mediaBoxes, &i); err != nil {
		return nil, err
	}

	ps := make([]Dim, len(mediaBoxes))
	for i, mb := range mediaBoxes {
		ps[i] = Dim{mb.Width(), mb.Height()}
	}

	return ps, nil
}

func (xRefTable *XRefTable) emptyPage(parentIndRef *IndirectRef, mediaBox *Rectangle) (*IndirectRef, error) {
	sd, _ := xRefTable.NewStreamDictForBuf(nil)

	if err := encodeStream(sd); err != nil {
		return nil, err
	}

	contentsIndRef, err := xRefTable.IndRefForNewObject(*sd)
	if err != nil {
		return nil, err
	}

	pageDict := Dict(
		map[string]Object{
			"Type":      Name("Page"),
			"Parent":    *parentIndRef,
			"Resources": NewDict(),
			"MediaBox":  mediaBox.Array(),
			"Contents":  *contentsIndRef,
		},
	)

	return xRefTable.IndRefForNewObject(pageDict)
}

func (xRefTable *XRefTable) pageMediaBox(d *Dict) (*Rectangle, error) {

	o, found := d.Find("MediaBox")
	if !found {
		return nil, errors.Errorf("pdfcpu: pageMediaBox: missing mediaBox")
	}

	a, err := xRefTable.DereferenceArray(o)
	if err != nil {
		return nil, err
	}

	return rect(xRefTable, a)
}

func (xRefTable *XRefTable) insertEmptyPage(root *IndirectRef, pAttrs *InheritedPageAttrs, pageNodeDict Dict) (indRef *IndirectRef, err error) {
	mediaBox := pAttrs.mediaBox
	if mediaBox == nil {
		mediaBox, err = xRefTable.pageMediaBox(&pageNodeDict)
		if err != nil {
			return nil, err
		}
	}

	return xRefTable.emptyPage(root, mediaBox)
}

func (xRefTable *XRefTable) insertIntoPageTree(root *IndirectRef, pAttrs *InheritedPageAttrs, p *int, selectedPages IntSet, before bool) (int, error) {

	d, err := xRefTable.DereferenceDict(*root)
	if err != nil {
		return 0, err
	}

	err = xRefTable.checkInheritedPageAttrs(d, pAttrs)
	if err != nil {
		return 0, err
	}

	kids := d.ArrayEntry("Kids")
	if kids == nil {
		return 0, nil
	}

	i := 0
	a := Array{}

	for _, o := range kids {

		if o == nil {
			continue
		}

		// Dereference next page node dict.
		ir, ok := o.(IndirectRef)
		if !ok {
			return 0, errors.Errorf("pdfcpu: insertIntoPageTree: corrupt page node dict")
		}

		pageNodeDict, err := xRefTable.DereferenceDict(ir)
		if err != nil {
			return 0, err
		}

		switch *pageNodeDict.Type() {

		case "Pages":
			// Recurse over sub pagetree.
			j, err := xRefTable.insertIntoPageTree(&ir, pAttrs, p, selectedPages, before)
			if err != nil {
				return 0, err
			}
			a = append(a, ir)
			i += j

		case "Page":
			*p++
			if !before {
				a = append(a, ir)
				i++
			}
			if selectedPages[*p] {
				// Insert empty page.
				indRef, err := xRefTable.insertEmptyPage(root, pAttrs, pageNodeDict)
				if err != nil {
					return 0, err
				}

				a = append(a, *indRef)
				i++
			}
			if before {
				a = append(a, ir)
				i++
			}

		}

	}

	d.Update("Kids", a)

	return i, d.IncrementBy("Count", i)
}

// InsertPages inserts a blank page before or after each selected page.
func (xRefTable *XRefTable) InsertPages(pages IntSet, before bool) error {

	root, err := xRefTable.Pages()
	if err != nil {
		return err
	}

	var inhPAttrs InheritedPageAttrs
	p := 0

	_, err = xRefTable.insertIntoPageTree(root, &inhPAttrs, &p, pages, before)

	return err
}

func (xRefTable *XRefTable) detectPageTreeWatermarks(root *IndirectRef) error {

	d, err := xRefTable.DereferenceDict(*root)
	if err != nil {
		return err
	}

	kids := d.ArrayEntry("Kids")
	if kids == nil {
		return nil
	}

	for _, o := range kids {

		if xRefTable.Watermarked {
			return nil
		}

		if o == nil {
			continue
		}

		// Dereference next page node dict.
		ir, ok := o.(IndirectRef)
		if !ok {
			return errors.Errorf("pdfcpu: detectPageTreeWatermarks: corrupt page node dict")
		}

		pageNodeDict, err := xRefTable.DereferenceDict(ir)
		if err != nil {
			return err
		}

		switch *pageNodeDict.Type() {

		case "Pages":
			// Recurse over sub pagetree.
			if err := xRefTable.detectPageTreeWatermarks(&ir); err != nil {
				return err
			}

		case "Page":
			found, err := findPageWatermarks(xRefTable, &ir)
			if err != nil {
				return err
			}
			if found {
				xRefTable.Watermarked = true
				return nil
			}

		}
	}

	return nil
}

// DetectPageTreeWatermarks checks xRefTable's page tree for watermarks
// and records the result to xRefTable.Watermarked.
func (xRefTable *XRefTable) DetectPageTreeWatermarks() error {
	root, err := xRefTable.Pages()
	if err != nil {
		return err
	}
	return xRefTable.detectPageTreeWatermarks(root)
}
