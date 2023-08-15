/*
	Copyright 2021 The pdfcpu Authors.

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

package model

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/pdfcpu/pdfcpu/pkg/filter"
	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
	"github.com/pkg/errors"
)

var ErrNoContent = errors.New("pdfcpu: page without content")

var zero int64 = 0

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
	Object          types.Object
	Compressed      bool
	ObjectStream    *int
	ObjectStreamInd *int
	Valid           bool
}

// NewXRefTableEntryGen0 returns a cross reference table entry for an object with generation 0.
func NewXRefTableEntryGen0(obj types.Object) *XRefTableEntry {
	zero := 0
	return &XRefTableEntry{Generation: &zero, Object: obj}
}

// NewFreeHeadXRefTableEntry returns the xref table entry for object 0
// which is per definition the head of the free list (list of free objects).
func NewFreeHeadXRefTableEntry() *XRefTableEntry {

	freeHeadGeneration := types.FreeHeadGeneration

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

// AnnotMap represents annotations by object number of the corresponding annotation dict.
type AnnotMap map[int]AnnotationRenderer

type Annot struct {
	IndRefs *[]types.IndirectRef
	Map     AnnotMap
}

// PgAnnots represents a map of page annotations by type.
type PgAnnots map[AnnotationType]Annot

// XRefTable represents a PDF cross reference table plus stats for a PDF file.
type XRefTable struct {
	Table               map[int]*XRefTableEntry
	Size                *int               // Object count from PDF trailer dict.
	PageCount           int                // Number of pages.
	Root                *types.IndirectRef // Pointer to catalog (reference to root object).
	RootDict            types.Dict         // Catalog
	Names               map[string]*Node   // Cache for name trees as found in catalog.
	NameRefs            map[string]NameMap // Name refs for merging only
	Encrypt             *types.IndirectRef // Encrypt dict.
	E                   *Enc
	EncKey              []byte // Encrypt key.
	AES4Strings         bool
	AES4Streams         bool
	AES4EmbeddedStreams bool

	// PDF Version
	HeaderVersion *Version // The PDF version the source is claiming to us as per its header.
	RootVersion   *Version // Optional PDF version taking precedence over the header version.

	// Document information section
	ID           types.Array        // from trailer
	Info         *types.IndirectRef // Infodict (reference to info dict object)
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
	LinearizationObjs       types.IntSet

	// Page annotation cache
	PageAnnots map[int]PgAnnots

	// Thumbnail images
	PageThumbs map[int]types.IndirectRef

	// Offspec section
	AdditionalStreams *types.Array // array of IndirectRef - trailer :e.g., Oasis "Open Doc"

	// Statistics
	Stats PDFStats

	Tagged bool // File is using tags. This is important for ???

	// Validation
	CurPage        int                       // current page during validation
	CurObj         int                       // current object during validation, the last dereferenced object
	Conf           *Configuration            // current command being executed
	ValidationMode int                       // see Configuration
	ValidateLinks  bool                      // check for broken links in LinkAnnotations/URIDicts.
	Valid          bool                      // true means successful validated against ISO 32000.
	URIs           map[int]map[string]string // URIs for link checking

	Optimized      bool
	Watermarked    bool
	Form           types.Dict
	Outlines       types.Dict
	SignatureExist bool
	AppendOnly     bool

	// Fonts
	UsedGIDs map[string]map[uint16]bool
}

// NewXRefTable creates a new XRefTable.
func newXRefTable(conf *Configuration) (xRefTable *XRefTable) {
	return &XRefTable{
		Table:             map[int]*XRefTableEntry{},
		Names:             map[string]*Node{},
		NameRefs:          map[string]NameMap{},
		Properties:        map[string]string{},
		LinearizationObjs: types.IntSet{},
		PageAnnots:        map[int]PgAnnots{},
		PageThumbs:        map[int]types.IndirectRef{},
		Stats:             NewPDFStats(),
		ValidationMode:    conf.ValidationMode,
		ValidateLinks:     conf.ValidateLinks,
		URIs:              map[int]map[string]string{},
		UsedGIDs:          map[string]map[uint16]bool{},
		Conf:              conf,
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

func (xRefTable *XRefTable) currentCommand() CommandMode {
	return xRefTable.Conf.Cmd
}

func (xRefTable *XRefTable) IsMerging() bool {
	cmd := xRefTable.currentCommand()
	return cmd == MERGECREATE || cmd == MERGEAPPEND
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
func (xRefTable *XRefTable) FindObject(objNr int) (types.Object, error) {
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
		return nil, nil
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
	log.Trace.Printf("FindTableEntry: obj#:%d gen:%d \n", objNr, genNr)
	return xRefTable.Find(objNr)
}

// FindTableEntryForIndRef returns the XRefTable entry for given indirect reference.
func (xRefTable *XRefTable) FindTableEntryForIndRef(ir *types.IndirectRef) (*XRefTableEntry, bool) {
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
func (xRefTable *XRefTable) InsertObject(obj types.Object) (objNr int, err error) {
	xRefTableEntry := NewXRefTableEntryGen0(obj)
	xRefTableEntry.RefCount = 1
	return xRefTable.InsertNew(*xRefTableEntry), nil
}

// IndRefForNewObject inserts an object into the xRefTable and returns an indirect reference to it.
func (xRefTable *XRefTable) IndRefForNewObject(obj types.Object) (*types.IndirectRef, error) {
	xRefTableEntry := NewXRefTableEntryGen0(obj)
	objNr, err := xRefTable.InsertAndUseRecycled(*xRefTableEntry)
	if err != nil {
		return nil, err
	}

	return types.NewIndirectRef(objNr, *xRefTableEntry.Generation), nil
}

// NewStreamDictForBuf creates a streamDict for buf.
func (xRefTable *XRefTable) NewStreamDictForBuf(buf []byte) (*types.StreamDict, error) {
	sd := types.StreamDict{
		Dict:           types.NewDict(),
		Content:        buf,
		FilterPipeline: []types.PDFFilter{{Name: filter.Flate, DecodeParms: nil}},
	}
	sd.InsertName("Filter", filter.Flate)
	return &sd, nil
}

// NewStreamDictForFile creates a streamDict for filename.
func (xRefTable *XRefTable) NewStreamDictForFile(filename string) (*types.StreamDict, error) {
	buf, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	return xRefTable.NewStreamDictForBuf(buf)
}

// NewEmbeddedStreamDict creates and returns an embeddedStreamDict containing the bytes represented by r.
func (xRefTable *XRefTable) NewEmbeddedStreamDict(r io.Reader, modDate time.Time) (*types.IndirectRef, error) {
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		return nil, err
	}

	bb := buf.Bytes()

	sd, err := xRefTable.NewStreamDictForBuf(bb)
	if err != nil {
		return nil, err
	}

	sd.InsertName("Type", "EmbeddedFile")
	d := types.NewDict()
	d.InsertInt("Size", len(bb))
	d.Insert("ModDate", types.StringLiteral(types.DateString(modDate)))
	sd.Insert("Params", d)
	if err = sd.Encode(); err != nil {
		return nil, err
	}

	return xRefTable.IndRefForNewObject(*sd)
}

func (xRefTable *XRefTable) locateObjForIndRef(ir types.IndirectRef) (types.Object, error) {

	objNr := int(ir.ObjectNumber)

	entry, found := xRefTable.FindTableEntryLight(objNr)
	if !found {
		return nil, errors.Errorf("pdfcpu: locateObjForIndRef: no xref entry found for obj #%d\n", objNr)
	}

	// Check for multiple indRefs.
	if entry.RefCount > 1 {
		entry.RefCount--
		// By returning nil we signal this object is still in use and can't be deleted.
		return nil, nil
	}

	// Since this is the only indRef we can move on and delete the entire object graph.
	return xRefTable.Dereference(ir)
}

// FreeObject marks an objects xref table entry as free and inserts it into the free list right after the head.
func (xRefTable *XRefTable) FreeObject(objNr int) error {

	// see 7.5.4 Cross-Reference Table

	log.Debug.Printf("FreeObject: begin %d\n", objNr)

	freeListHeadEntry, err := xRefTable.Free(0)
	if err != nil {
		return err
	}

	entry, found := xRefTable.FindTableEntryLight(objNr)
	if !found {
		return errors.Errorf("FreeObject: no entry for obj #%d\n", objNr)
	}

	if entry.Free {
		log.Debug.Printf("FreeObject: end %d already free\n", objNr)
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

	log.Debug.Printf("FreeObject: end %d\n", objNr)

	return nil
}

// DeleteObject makes a deep remove of o.
func (xRefTable *XRefTable) DeleteObject(o types.Object) error {

	var err error

	ir, ok := o.(types.IndirectRef)
	if ok {
		o, err = xRefTable.locateObjForIndRef(ir)
		if err != nil || o == nil {
			return err
		}
		if err = xRefTable.FreeObject(ir.ObjectNumber.Value()); err != nil {
			return err
		}
	}

	switch o := o.(type) {

	case types.Dict:
		for _, v := range o {
			err := xRefTable.DeleteObject(v)
			if err != nil {
				return err
			}
		}

	case types.StreamDict:
		for _, v := range o.Dict {
			err := xRefTable.DeleteObject(v)
			if err != nil {
				return err
			}
		}

	case types.Array:
		for _, v := range o {
			err := xRefTable.DeleteObject(v)
			if err != nil {
				return err
			}
		}

	}

	return nil
}

// DeleteObjectGraph deletes all objects reachable by indRef.
func (xRefTable *XRefTable) DeleteObjectGraph(o types.Object) error {

	log.Debug.Println("DeleteObjectGraph: begin")

	ir, ok := o.(types.IndirectRef)
	if !ok {
		return nil
	}

	// Delete ObjectGraph for object indRef.ObjectNumber.Value() via recursion.
	if err := xRefTable.DeleteObject(ir); err != nil {
		return err
	}

	log.Debug.Println("DeleteObjectGraph: end")
	return nil
}

// NewEmbeddedFileStreamDict returns an embeddedFileStreamDict containing the file "filename".
func (xRefTable *XRefTable) NewEmbeddedFileStreamDict(filename string) (*types.IndirectRef, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}

	return xRefTable.NewEmbeddedStreamDict(f, fi.ModTime())
}

// NewSoundStreamDict returns a new sound stream dict.
func (xRefTable *XRefTable) NewSoundStreamDict(filename string, samplingRate int, fileSpecDict types.Dict) (*types.IndirectRef, error) {
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
		sd.Insert("F", types.StringLiteral(path.Base(filename)))
	}

	if err = sd.Encode(); err != nil {
		return nil, err
	}

	return xRefTable.IndRefForNewObject(*sd)
}

// NewFileSpecDict creates and returns a new fileSpec dictionary.
func (xRefTable *XRefTable) NewFileSpecDict(f, uf, desc string, indRefStreamDict types.IndirectRef) (types.Dict, error) {

	d := types.NewDict()
	d.InsertName("Type", "Filespec")
	d.InsertString("F", f)
	d.InsertString("UF", uf)

	efDict := types.NewDict()
	efDict.Insert("F", indRefStreamDict)
	efDict.Insert("UF", indRefStreamDict)
	d.Insert("EF", efDict)

	if desc != "" {
		d.InsertString("Desc", desc)
	}

	// CI, optional, collection item dict, since V1.7
	// a corresponding collection schema dict in a collection.
	ciDict := types.NewDict()
	//add contextual meta info here.
	d.Insert("CI", ciDict)

	return d, nil
}

func (xRefTable *XRefTable) freeObjects() types.IntSet {
	m := types.IntSet{}

	for k, v := range xRefTable.Table {
		if v != nil && v.Free && k > 0 {
			m[k] = true
		}
	}

	return m
}

func anyKey(m types.IntSet) int {
	for k := range m {
		return k
	}
	return -1
}

func (xRefTable *XRefTable) handleDanglingFree(m types.IntSet, head *XRefTableEntry) error {
	for i := range m {

		entry, found := xRefTable.FindTableEntryLight(i)
		if !found {
			return errors.Errorf("pdfcpu: ensureValidFreeList: no xref entry found for obj #%d\n", i)
		}

		if !entry.Free {
			return errors.Errorf("pdfcpu: ensureValidFreeList: xref entry is not free for obj #%d\n", i)
		}

		if *entry.Generation == types.FreeHeadGeneration {
			entry.Offset = &zero
			continue
		}

		entry.Offset = head.Offset
		next := int64(i)
		head.Offset = &next
	}
	return nil
}

// EnsureValidFreeList ensures the integrity of the free list associated with the recorded free objects.
// See 7.5.4 Cross-Reference Table
func (xRefTable *XRefTable) EnsureValidFreeList() error {
	log.Trace.Println("EnsureValidFreeList begin")

	m := xRefTable.freeObjects()

	// Verify free object 0 as free list head.
	head, _ := xRefTable.Find(0)
	if head == nil {
		g0 := types.FreeHeadGeneration
		head = &XRefTableEntry{Free: true, Offset: &zero, Generation: &g0}
		xRefTable.Table[0] = head
	}

	// verify generation of 56535
	if *head.Generation != types.FreeHeadGeneration {
		// Fix generation for obj 0.
		*head.Generation = types.FreeHeadGeneration
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

	e := head
	f := int(*e.Offset)
	var lastValid *XRefTableEntry
	var nextFree int

	// until we have found the last free object which should point to obj 0.
	for f != 0 {

		log.Trace.Printf("EnsureValidFreeList: validating obj #%d %v\n", f, m)
		// verify if obj f is one of the free objects recorded.
		if !m[f] {
			if len(m) > 0 && lastValid == nil {
				lastValid = e
				f = anyKey(m)
				nextFree = f
				continue
			}
			// Repair last entry.
			*e.Offset = 0
			break
		}

		delete(m, f)

		var err error
		e, err = xRefTable.Free(f)
		if err != nil {
			return err
		}
		if e == nil {
			return errors.Errorf("pdfcpu: ensureValidFreeList: no xref entry found for obj #%d\n", f)
		}

		f = int(*e.Offset)
	}

	if lastValid != nil {
		*lastValid.Offset = int64(nextFree)
	}

	if len(m) == 0 {
		log.Trace.Println("EnsureValidFreeList: end, regular linked list")
		return nil
	}

	// insert remaining free objects into verified linked list
	// unless they are forever deleted with generation 65535.
	// In that case they have to point to obj 0.
	err := xRefTable.handleDanglingFree(m, head)

	log.Trace.Println("EnsureValidFreeList: end, linked list plus some dangling free objects.")

	return err
}

func (xRefTable *XRefTable) DeleteDictEntry(d types.Dict, key string) error {
	o, found := d.Find(key)
	if !found {
		return nil
	}
	if err := xRefTable.DeleteObject(o); err != nil {
		return err
	}
	d.Delete(key)
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

// IsValid returns true if the object referenced by ir has already been validated.
func (xRefTable *XRefTable) IsValid(ir types.IndirectRef) (bool, error) {
	entry, found := xRefTable.FindTableEntry(ir.ObjectNumber.Value(), ir.GenerationNumber.Value())
	if !found {
		return false, errors.Errorf("pdfcpu: IsValid: no entry for obj#%d\n", ir.ObjectNumber.Value())
	}
	if entry.Free {
		return false, errors.Errorf("pdfcpu: IsValid: unexpected free entry for obj#%d\n", ir.ObjectNumber.Value())
	}
	return entry.Valid, nil
}

// SetValid marks the xreftable entry of the object referenced by ir as valid.
func (xRefTable *XRefTable) SetValid(ir types.IndirectRef) error {
	entry, found := xRefTable.FindTableEntry(ir.ObjectNumber.Value(), ir.GenerationNumber.Value())
	if !found {
		return errors.Errorf("pdfcpu: SetValid: no entry for obj#%d\n", ir.ObjectNumber.Value())
	}
	if entry.Free {
		return errors.Errorf("pdfcpu: SetValid: unexpected free entry for obj#%d\n", ir.ObjectNumber.Value())
	}
	entry.Valid = true
	return nil
}

// DereferenceStreamDict resolves stream dictionary objects.
func (xRefTable *XRefTable) DereferenceStreamDict(o types.Object) (*types.StreamDict, bool, error) {
	// TODO Check if we still need the bool return value
	ir, ok := o.(types.IndirectRef)
	if !ok {
		sd, ok := o.(types.StreamDict)
		if !ok {
			return nil, false, errors.Errorf("pdfcpu: DereferenceStreamDict: wrong type <%v> %T", o, o)
		}
		return &sd, false, nil
	}

	// 7.3.10
	// An indirect reference to an undefined object shall not be considered an error by a conforming reader;
	// it shall be treated as a reference to the null object.
	entry, found := xRefTable.FindTableEntry(ir.ObjectNumber.Value(), ir.GenerationNumber.Value())
	if !found || entry.Object == nil || entry.Free {
		return nil, false, nil
	}
	ev := entry.Valid
	if !entry.Valid {
		entry.Valid = true
	}
	sd, ok := entry.Object.(types.StreamDict)
	if !ok {
		return nil, false, errors.Errorf("pdfcpu: DereferenceStreamDict: wrong type <%v> %T", o, entry.Object)
	}

	return &sd, ev, nil
}

// Catalog returns a pointer to the root object / catalog.
func (xRefTable *XRefTable) Catalog() (types.Dict, error) {

	if xRefTable.RootDict != nil {
		return xRefTable.RootDict, nil
	}

	if xRefTable.Root == nil {
		return nil, errors.New("pdfcpu: Catalog: missing root dict")
	}

	o, err := xRefTable.indRefToObject(xRefTable.Root)
	if err != nil || o == nil {
		return nil, err
	}

	d, ok := o.(types.Dict)
	if !ok {
		return nil, errors.New("pdfcpu: catalog: corrupt root catalog")
	}

	xRefTable.RootDict = d

	return xRefTable.RootDict, nil
}

// EncryptDict returns a pointer to the root object / catalog.
func (xRefTable *XRefTable) EncryptDict() (types.Dict, error) {

	o, err := xRefTable.indRefToObject(xRefTable.Encrypt)
	if err != nil || o == nil {
		return nil, err
	}

	d, ok := o.(types.Dict)
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
func (xRefTable *XRefTable) Pages() (*types.IndirectRef, error) {
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

func (xRefTable *XRefTable) sortedKeys() []int {
	var keys []int
	for k := range xRefTable.Table {
		keys = append(keys, k)
	}
	sort.Ints(keys)
	return keys
}

func (xRefTable *XRefTable) DumpStream(objNr int, hexOut bool) {

	entry := xRefTable.Table[objNr]
	if entry == nil || entry.Free || entry.Compressed || entry.Object == nil {
		fmt.Println(":(")
		return
	}

	typeStr := fmt.Sprintf("%T", entry.Object)

	d, ok := entry.Object.(types.Dict)

	if ok {
		if d.Type() != nil {
			typeStr += fmt.Sprintf(" type=%s", *d.Type())
		}
		if d.Subtype() != nil {
			typeStr += fmt.Sprintf(" subType=%s", *d.Subtype())
		}
	}

	var str string

	if entry.ObjectStream != nil {
		// was compressed, offset is nil.
		str = fmt.Sprintf("%5d: was compressed %d[%d] generation=%d %s \n%s\n", objNr, *entry.ObjectStream, *entry.ObjectStreamInd, *entry.Generation, typeStr, entry.Object)
	} else {
		// regular in use object with offset.
		if entry.Offset != nil {
			str = fmt.Sprintf("%5d:   offset=%8d generation=%d %s \n%s\n", objNr, *entry.Offset, *entry.Generation, typeStr, entry.Object)
		} else {
			str = fmt.Sprintf("%5d:   offset=nil generation=%d %s \n%s\n", objNr, *entry.Generation, typeStr, entry.Object)
		}
	}

	sd, ok := entry.Object.(types.StreamDict)
	if ok {

		err := sd.Decode()
		if err == filter.ErrUnsupportedFilter {
			str += "stream filter unsupported!"
			fmt.Println(str)
			return
		}
		if err != nil {
			str += "decoding problem encountered!"
			fmt.Println(str)
			return
		}

		s := "decoded stream content (length = %d)\n%s\n"
		if hexOut {
			str += fmt.Sprintf(s, len(sd.Content), hex.Dump(sd.Content))
		} else {
			str += fmt.Sprintf(s, len(sd.Content), sd.Content)
		}
	}

	osd, ok := entry.Object.(types.ObjectStreamDict)
	if ok {
		str += fmt.Sprintf("object stream count:%d size of objectarray:%d\n", osd.ObjCount, len(osd.ObjArray))
	}

	fmt.Println(str)
}

func (xRefTable *XRefTable) list(logStr []string) []string {

	// Print list of XRefTable entries to logString.
	for _, k := range xRefTable.sortedKeys() {

		entry := xRefTable.Table[k]

		var str string

		if entry.Free {
			str = fmt.Sprintf("%5d: f   next=%8d generation=%d\n", k, *entry.Offset, *entry.Generation)
		} else if entry.Compressed {
			str = fmt.Sprintf("%5d: c => obj:%d[%d] generation=%d \n%s\n", k, *entry.ObjectStream, *entry.ObjectStreamInd, *entry.Generation, entry.Object)
		} else {
			if entry.Object != nil {

				typeStr := fmt.Sprintf("%T", entry.Object)

				d, ok := entry.Object.(types.Dict)

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

				sd, ok := entry.Object.(types.StreamDict)
				if ok && log.IsTraceLoggerEnabled() {
					s := "decoded stream content (length = %d)\n%s\n"
					if sd.IsPageContent {
						str += fmt.Sprintf(s, len(sd.Content), sd.Content)
					} else {
						str += fmt.Sprintf(s, len(sd.Content), hex.Dump(sd.Content))
					}
				}

				osd, ok := entry.Object.(types.ObjectStreamDict)
				if ok {
					str += fmt.Sprintf("object stream count:%d size of objectarray:%d\n", osd.ObjCount, len(osd.ObjArray))
				}

			} else {
				if entry.Offset == nil {
					str = fmt.Sprintf("%5d:   offset=    none generation=%d nil\n", k, *entry.Generation)
				} else {
					str = fmt.Sprintf("%5d:   offset=%8d generation=%d nil\n", k, *entry.Offset, *entry.Generation)
				}
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
	logStr = append(logStr, fmt.Sprintf("%5d %5d %5d\n", 0, f, types.FreeHeadGeneration))

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

	var dict types.Dict

	if n.D == nil {
		dict = types.NewDict()
		n.D = dict
	} else {
		if root {
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
		dict.Update("Limits", types.NewHexLiteralArray(n.Kmin, n.Kmax))
	} else {
		dict.Delete("Limits")
	}

	if n.leaf() {
		a := types.Array{}
		for _, e := range n.Names {
			a = append(a, types.NewHexLiteral([]byte(e.k)))
			a = append(a, e.v)
		}
		dict.Update("Names", a)
		log.Debug.Printf("bound nametree node(leaf): %s/n", dict)
		return nil
	}

	kids := types.Array{}
	for _, k := range n.Kids {
		if err := xRefTable.bindNameTreeNode(name, k, false); err != nil {
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
		dict := types.NewDict()

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
		dict := types.NewDict()
		dict.Insert("Names", types.Array{})

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
func (xRefTable *XRefTable) NamesDict() (types.Dict, error) {

	d, err := xRefTable.Catalog()
	if err != nil {
		return nil, err
	}

	o, found := d.Find("Names")
	if !found {
		dict := types.NewDict()
		ir, err := xRefTable.IndRefForNewObject(dict)
		if err != nil {
			return nil, err
		}
		d["Names"] = *ir
		return dict, nil
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
	if err = xRefTable.DeleteDictEntry(namesDict, nameTreeName); err != nil {
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
	if err = xRefTable.DeleteDictEntry(rootDict, "Names"); err != nil {
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
	return xRefTable.DeleteDictEntry(rootDict, "Collection")
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

	dict := types.NewDict()
	dict.Insert("Type", types.Name("Collection"))
	dict.Insert("View", types.Name("D"))

	schemaDict := types.NewDict()
	schemaDict.Insert("Type", types.Name("CollectionSchema"))

	fileNameCFDict := types.NewDict()
	fileNameCFDict.Insert("Type", types.Name("CollectionField"))
	fileNameCFDict.Insert("Subtype", types.Name("F"))
	fileNameCFDict.Insert("N", types.StringLiteral("Filename"))
	fileNameCFDict.Insert("O", types.Integer(1))
	schemaDict.Insert("FileName", fileNameCFDict)

	descCFDict := types.NewDict()
	descCFDict.Insert("Type", types.Name("CollectionField"))
	descCFDict.Insert("Subtype", types.Name("Desc"))
	descCFDict.Insert("N", types.StringLiteral("Description"))
	descCFDict.Insert("O", types.Integer(2))
	schemaDict.Insert("Description", descCFDict)

	sizeCFDict := types.NewDict()
	sizeCFDict.Insert("Type", types.Name("CollectionField"))
	sizeCFDict.Insert("Subtype", types.Name("Size"))
	sizeCFDict.Insert("N", types.StringLiteral("Size"))
	sizeCFDict.Insert("O", types.Integer(3))
	schemaDict.Insert("Size", sizeCFDict)

	modDateCFDict := types.NewDict()
	modDateCFDict.Insert("Type", types.Name("CollectionField"))
	modDateCFDict.Insert("Subtype", types.Name("ModDate"))
	modDateCFDict.Insert("N", types.StringLiteral("Last Modification"))
	modDateCFDict.Insert("O", types.Integer(4))
	schemaDict.Insert("ModDate", modDateCFDict)

	//TODO use xRefTable.InsertAndUseRecycled(xRefTableEntry)

	ir, err := xRefTable.IndRefForNewObject(schemaDict)
	if err != nil {
		return err
	}
	dict.Insert("Schema", *ir)

	sortDict := types.NewDict()
	sortDict.Insert("S", types.Name("ModDate"))
	sortDict.Insert("A", types.Boolean(false))
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

	hl, ok := xRefTable.ID[0].(types.HexLiteral)
	if ok {
		return hl.Bytes()
	}

	sl, ok := xRefTable.ID[0].(types.StringLiteral)
	if !ok {
		return nil, errors.New("pdfcpu: ID must contain hex literals or string literals")
	}

	bb, err := types.Unescape(sl.Value(), false)
	if err != nil {
		return nil, err
	}

	return bb, nil
}

// InheritedPageAttrs represents all inherited page attributes.
type InheritedPageAttrs struct {
	Resources types.Dict
	MediaBox  *types.Rectangle
	CropBox   *types.Rectangle
	Rotate    int
}

func rect(xRefTable *XRefTable, a types.Array) (*types.Rectangle, error) {

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

	return types.NewRectangle(llx, lly, urx, ury), nil
}

func weaveResourceSubDict(d1, d2 types.Dict) {
	for k, v := range d1 {
		if v != nil {
			v = v.Clone()
		}
		d2[k] = v
	}
}

func (xRefTable *XRefTable) consolidateResources(obj types.Object, pAttrs *InheritedPageAttrs) error {
	d, err := xRefTable.DereferenceDict(obj)
	if err != nil {
		return err
	}
	if len(d) == 0 {
		return nil
	}

	if pAttrs.Resources == nil {
		// Create a resource dict that eventually will contain any inherited resources
		// walking down from page root to leaf node representing the page in question.
		pAttrs.Resources = d.Clone().(types.Dict)
		for k, v := range pAttrs.Resources {
			o, err := xRefTable.Dereference(v)
			if err != nil {
				return err
			}
			pAttrs.Resources[k] = o.Clone()
		}
		log.Write.Printf("pA:\n%s\n", pAttrs.Resources)
		return nil
	}

	// Accumulate any resources defined in this page node into the inherited resources.
	for k, v := range d {
		if k == "ProcSet" || v == nil {
			continue
		}
		d1, err := xRefTable.DereferenceDict(v)
		if err != nil {
			return err
		}
		if d1 == nil {
			continue
		}
		// We have identified a subdict that needs to go into the inherited res dict.
		if pAttrs.Resources[k] == nil {
			pAttrs.Resources[k] = d1.Clone()
			continue
		}
		d2, ok := pAttrs.Resources[k].(types.Dict)
		if !ok {
			return errors.Errorf("pdfcpu: checkInheritedPageAttrs: expected Dict d2: %T", pAttrs.Resources[k])
		}
		// Weave sub dict d1 into inherited sub dict d2.
		// Any existing resource names will be overridden.
		weaveResourceSubDict(d1, d2)
	}

	return nil
}

func (xRefTable *XRefTable) checkInheritedPageAttrs(pageDict types.Dict, pAttrs *InheritedPageAttrs, consolidateRes bool) error {
	// Return mediaBox, cropBox and rotate as inherited.
	// if consolidateRes is true
	// then consolidate all inherited resources as required by content stream
	// else return pageDict resources.
	var (
		obj   types.Object
		found bool
	)

	if obj, found = pageDict.Find("MediaBox"); found {
		a, err := xRefTable.DereferenceArray(obj)
		if err != nil {
			return err
		}
		if pAttrs.MediaBox, err = rect(xRefTable, a); err != nil {
			return err
		}
	}

	if obj, found = pageDict.Find("CropBox"); found {
		a, err := xRefTable.DereferenceArray(obj)
		if err != nil {
			return err
		}
		if pAttrs.CropBox, err = rect(xRefTable, a); err != nil {
			return err
		}
	}

	if obj, found = pageDict.Find("Rotate"); found {
		i, err := xRefTable.DereferenceInteger(obj)
		if err != nil {
			return err
		}
		pAttrs.Rotate = i.Value()
	}

	if obj, found = pageDict.Find("Resources"); !found {
		return nil
	}

	if !consolidateRes {
		// Return resourceDict as is.
		d, err := xRefTable.DereferenceDict(obj)
		if err != nil {
			return err
		}
		pAttrs.Resources = d
		return nil
	}

	// Accumulate inherited resources.
	return xRefTable.consolidateResources(obj, pAttrs)
}

// PageContent returns the content in PDF syntax for page dict d.
func (xRefTable *XRefTable) PageContent(d types.Dict) ([]byte, error) {

	o, _ := d.Find("Contents")

	o, err := xRefTable.Dereference(o)
	if err != nil || o == nil {
		return nil, err
	}

	bb := []byte{}

	switch o := o.(type) {

	case types.StreamDict:
		// no further processing.
		err := o.Decode()
		if err == filter.ErrUnsupportedFilter {
			return nil, errors.New("pdfcpu: unsupported filter: unable to decode content")
		}
		if err != nil {
			return nil, err
		}

		bb = append(bb, o.Content...)

	case types.Array:
		// process array of content stream dicts.
		for _, o := range o {
			if o == nil {
				continue
			}
			o, _, err := xRefTable.DereferenceStreamDict(o)
			if err != nil {
				return nil, err
			}
			if o == nil {
				continue
			}
			err = o.Decode()
			if err == filter.ErrUnsupportedFilter {
				return nil, errors.New("pdfcpu: unsupported filter: unable to decode content")
			}
			if err != nil {
				return nil, err
			}
			bb = append(bb, o.Content...)
		}

	default:
		return nil, errors.Errorf("pdfcpu: page content must be stream dict or array")
	}

	if len(bb) == 0 {
		return nil, ErrNoContent
	}

	return bb, nil
}

func consolidateResourceSubDict(d types.Dict, key string, prn PageResourceNames, pageNr int) error {
	o := d[key]
	if o == nil {
		if prn.HasResources(key) {
			return errors.Errorf("pdfcpu: page %d: missing required resource subdict: %s\n%s", pageNr, key, prn)
		}
		return nil
	}
	if !prn.HasResources(key) {
		d.Delete(key)
		return nil
	}
	d1 := o.(types.Dict)
	set := types.StringSet{}
	res := prn.Resources(key)
	// Iterate over inherited resource sub dict and remove any entries not required.
	for k := range d1 {
		ki := types.Name(k).Value()
		if !res[ki] {
			d1.Delete(k)
			continue
		}
		set[ki] = true
	}
	// Check for missing resource sub dict entries.
	for k := range res {
		if !set[k] {
			return errors.Errorf("pdfcpu: page %d: missing required %s: %s", pageNr, key, k)
		}
	}
	d[key] = d1
	return nil
}

func consolidateResourceDict(d types.Dict, prn PageResourceNames, pageNr int) error {
	for k := range resourceTypes {
		if err := consolidateResourceSubDict(d, k, prn, pageNr); err != nil {
			return err
		}
	}
	return nil
}

func consolidateResources(consolidateRes bool, xRefTable *XRefTable, pageDict, resDict types.Dict, page int) error {
	if !consolidateRes {
		return nil
	}

	bb, err := xRefTable.PageContent(pageDict)
	if err != nil {
		if err == ErrNoContent {
			return nil
		}
		return err
	}

	// Calculate resources required by the content stream of this page.
	prn, err := parseContent(string(bb))
	if err != nil {
		return err
	}

	// Compare required resouces (prn) with available resources (pAttrs.resources).
	// Remove any resource that's not required.
	// Return an error for any required resource missing.
	// TODO Calculate and accumulate resources required by content streams of any present form or type 3 fonts.
	return consolidateResourceDict(resDict, prn, page)
}

func (xRefTable *XRefTable) processPageTreeForPageDict(root *types.IndirectRef, pAttrs *InheritedPageAttrs, p *int, page int, consolidateRes bool) (types.Dict, *types.IndirectRef, error) {
	// Walk this page tree all the way down to the leaf node representing page.

	//fmt.Printf("entering processPageTreeForPageDict: p=%d obj#%d\n", *p, root.ObjectNumber.Value())

	d, err := xRefTable.DereferenceDict(*root)
	if err != nil {
		return nil, nil, err
	}

	pageCount := d.IntEntry("Count")
	if pageCount != nil {
		if *p+*pageCount < page {
			// Skip sub pagetree.
			*p += *pageCount
			return nil, nil, nil
		}
	}

	// Return the current state of all page attributes that may be inherited.
	if err = xRefTable.checkInheritedPageAttrs(d, pAttrs, consolidateRes); err != nil {
		return nil, nil, err
	}

	// Iterate over page tree.
	kids := d.ArrayEntry("Kids")
	if kids == nil {
		return d, root, consolidateResources(consolidateRes, xRefTable, d, pAttrs.Resources, page)
	}

	for _, o := range kids {

		if o == nil {
			continue
		}

		// Dereference next page node dict.
		ir, ok := o.(types.IndirectRef)
		if !ok {
			return nil, nil, errors.Errorf("pdfcpu: processPageTreeForPageDict: corrupt page node dict")
		}

		pageNodeDict, err := xRefTable.DereferenceDict(ir)
		if err != nil {
			return nil, nil, err
		}

		switch *pageNodeDict.Type() {

		case "Pages":
			// Recurse over sub pagetree.
			pageDict, pageDictIndRef, err := xRefTable.processPageTreeForPageDict(&ir, pAttrs, p, page, consolidateRes)
			if err != nil {
				return nil, nil, err
			}
			if pageDict != nil {
				return pageDict, pageDictIndRef, nil
			}

		case "Page":
			*p++
			if *p == page {
				return xRefTable.processPageTreeForPageDict(&ir, pAttrs, p, page, consolidateRes)
			}

		}

	}

	return nil, nil, nil
}

// PageDict returns a specific page dict along with the resources, mediaBox and CropBox in effect.
// consolidateRes ensures optimized resources in InheritedPageAttrs.
func (xRefTable *XRefTable) PageDict(pageNr int, consolidateRes bool) (types.Dict, *types.IndirectRef, *InheritedPageAttrs, error) {

	var (
		inhPAttrs InheritedPageAttrs
		pageCount int
	)

	// Get an indirect reference to the page tree root dict.
	pageRootDictIndRef, err := xRefTable.Pages()
	if err != nil {
		return nil, nil, nil, err
	}

	// Calculate and return only resources that are really needed by
	// any content stream of this page and any possible forms or type 3 fonts referenced.
	pageDict, pageDictindRef, err := xRefTable.processPageTreeForPageDict(pageRootDictIndRef, &inhPAttrs, &pageCount, pageNr, consolidateRes)
	if err != nil {
		return nil, nil, nil, err
	}

	return pageDict, pageDictindRef, &inhPAttrs, nil
}

// PageDictIndRef returns the pageDict IndRef for a logical page number.
func (xRefTable *XRefTable) PageDictIndRef(page int) (*types.IndirectRef, error) {

	var (
		inhPAttrs InheritedPageAttrs
		pageCount int
	)

	// Get an indirect reference to the page tree root dict.
	pageRootDictIndRef, err := xRefTable.Pages()
	if err != nil {
		return nil, err
	}

	// Calculate and return only resources that are really needed by
	// any content stream of this page and any possible forms or type 3 fonts referenced.
	consolidateRes := false
	_, ir, err := xRefTable.processPageTreeForPageDict(pageRootDictIndRef, &inhPAttrs, &pageCount, page, consolidateRes)

	return ir, err
}

// Calculate logical page number for page dict object number.
func (xRefTable *XRefTable) processPageTreeForPageNumber(root *types.IndirectRef, pageCount *int, pageObjNr int) (int, error) {

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
		ir, ok := o.(types.IndirectRef)
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
// Important when validation is turned off.
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

func (xRefTable *XRefTable) resolvePageBoundary(d types.Dict, boxName string) (*types.Rectangle, error) {
	obj, found := d.Find(boxName)
	if !found {
		return nil, nil
	}
	a, err := xRefTable.DereferenceArray(obj)
	if err != nil {
		return nil, err
	}
	return rect(xRefTable, a)
}

func (xRefTable *XRefTable) collectPageBoundariesForPage(d types.Dict, pb []PageBoundaries, inhMediaBox, inhCropBox *types.Rectangle, rot, p int) error {
	if inhMediaBox != nil {
		pb[p].Media = &Box{Rect: inhMediaBox, Inherited: true}
	}
	r, err := xRefTable.resolvePageBoundary(d, "MediaBox")
	if err != nil {
		return err
	}
	if r != nil {
		pb[p].Media = &Box{Rect: r, Inherited: false}
	}
	if pb[p].Media == nil {
		return errors.New("pdfcpu: collectMediaBoxesForPageTree: mediaBox is nil")
	}

	//if inhCropBox != nil && inhCropBox.Rectangle != nil {
	if inhCropBox != nil {
		pb[p].Crop = &Box{Rect: inhCropBox, Inherited: true}
	}
	r, err = xRefTable.resolvePageBoundary(d, "CropBox")
	if err != nil {
		return err
	}
	if r != nil {
		pb[p].Crop = &Box{Rect: r, Inherited: false}
	}

	r, err = xRefTable.resolvePageBoundary(d, "TrimBox")
	if err != nil {
		return err
	}
	if r != nil {
		pb[p].Trim = &Box{Rect: r}
	}

	r, err = xRefTable.resolvePageBoundary(d, "BleedBox")
	if err != nil {
		return err
	}
	if r != nil {
		pb[p].Bleed = &Box{Rect: r}
	}

	r, err = xRefTable.resolvePageBoundary(d, "ArtBox")
	if err != nil {
		return err
	}
	if r != nil {
		pb[p].Art = &Box{Rect: r}
	}

	pb[p].Rot = rot

	return nil
}

func (xRefTable *XRefTable) collectMediaBoxAndCropBox(d types.Dict, inhMediaBox, inhCropBox **types.Rectangle) error {
	obj, found := d.Find("MediaBox")
	if found {
		a, err := xRefTable.DereferenceArray(obj)
		if err != nil {
			return err
		}
		if *inhMediaBox, err = rect(xRefTable, a); err != nil {
			return err
		}
		*inhCropBox = nil
	}

	obj, found = d.Find("CropBox")
	if found {
		a, err := xRefTable.DereferenceArray(obj)
		if err != nil {
			return err
		}
		if *inhCropBox, err = rect(xRefTable, a); err != nil {
			return err
		}
	}
	return nil
}

func (xRefTable *XRefTable) collectPageBoundariesForPageTree(
	root *types.IndirectRef,
	inhMediaBox, inhCropBox **types.Rectangle,
	pb []PageBoundaries,
	r int,
	p *int,
	selectedPages types.IntSet) error {

	d, err := xRefTable.DereferenceDict(*root)
	if err != nil {
		return err
	}

	if obj, found := d.Find("Rotate"); found {
		i, err := xRefTable.DereferenceInteger(obj)
		if err != nil {
			return err
		}
		r = i.Value()
	}

	if err := xRefTable.collectMediaBoxAndCropBox(d, inhMediaBox, inhCropBox); err != nil {
		return err
	}

	o, _ := d.Find("Kids")
	o, _ = xRefTable.Dereference(o)
	if o == nil {
		return xRefTable.collectPageBoundariesForPage(d, pb, *inhMediaBox, *inhCropBox, r, *p)
	}

	kids, ok := o.(types.Array)
	if !ok {
		return errors.New("pdfcpu: validatePagesDict: corrupt \"Kids\" entry")
	}

	// Iterate over page tree.
	for _, o := range kids {

		if o == nil {
			continue
		}

		// Dereference next page node dict.
		ir, ok := o.(types.IndirectRef)
		if !ok {
			return errors.Errorf("pdfcpu: collectMediaBoxesForPageTree: corrupt page node dict")
		}

		pageNodeDict, err := xRefTable.DereferenceDict(ir)
		if err != nil {
			return err
		}

		switch *pageNodeDict.Type() {

		case "Pages":
			if err = xRefTable.collectPageBoundariesForPageTree(&ir, inhMediaBox, inhCropBox, pb, r, p, selectedPages); err != nil {
				return err
			}

		case "Page":
			collect := len(selectedPages) == 0
			if !collect {
				_, collect = selectedPages[(*p)+1]
			}
			if collect {
				if err = xRefTable.collectPageBoundariesForPageTree(&ir, inhMediaBox, inhCropBox, pb, r, p, selectedPages); err != nil {
					return err
				}
			}
			*p++
		}

	}

	return nil
}

// PageBoundaries returns a sorted slice with page boundaries
// for all pages sorted ascending by page number.
func (xRefTable *XRefTable) PageBoundaries(selectedPages types.IntSet) ([]PageBoundaries, error) {
	if err := xRefTable.EnsurePageCount(); err != nil {
		return nil, err
	}

	// Get an indirect reference to the page tree root dict.
	root, err := xRefTable.Pages()
	if err != nil {
		return nil, err
	}

	i := 0
	mb := &types.Rectangle{}
	cb := &types.Rectangle{}
	pbs := make([]PageBoundaries, xRefTable.PageCount)
	if err := xRefTable.collectPageBoundariesForPageTree(root, &mb, &cb, pbs, 0, &i, selectedPages); err != nil {
		return nil, err
	}
	return pbs, nil
}

// PageDims returns a sorted slice with effective media box dimensions
// for all pages sorted ascending by page number.
func (xRefTable *XRefTable) PageDims() ([]types.Dim, error) {
	pbs, err := xRefTable.PageBoundaries(nil)
	if err != nil {
		return nil, err
	}

	dims := make([]types.Dim, len(pbs))
	for i, pb := range pbs {
		d := pb.CropBox().Dimensions()
		if pb.Rot%180 != 0 {
			d.Width, d.Height = d.Height, d.Width
		}
		dims[i] = d
	}

	return dims, nil
}

func (xRefTable *XRefTable) emptyPage(parentIndRef *types.IndirectRef, mediaBox *types.Rectangle) (*types.IndirectRef, error) {
	sd, _ := xRefTable.NewStreamDictForBuf(nil)

	if err := sd.Encode(); err != nil {
		return nil, err
	}

	contentsIndRef, err := xRefTable.IndRefForNewObject(*sd)
	if err != nil {
		return nil, err
	}

	pageDict := types.Dict(
		map[string]types.Object{
			"Type":      types.Name("Page"),
			"Parent":    *parentIndRef,
			"Resources": types.NewDict(),
			"MediaBox":  mediaBox.Array(),
			"Contents":  *contentsIndRef,
		},
	)

	return xRefTable.IndRefForNewObject(pageDict)
}

func (xRefTable *XRefTable) pageMediaBox(d *types.Dict) (*types.Rectangle, error) {

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

func (xRefTable *XRefTable) insertEmptyPage(root *types.IndirectRef, pAttrs *InheritedPageAttrs, pageNodeDict types.Dict) (indRef *types.IndirectRef, err error) {
	mediaBox := pAttrs.MediaBox
	if mediaBox == nil {
		mediaBox, err = xRefTable.pageMediaBox(&pageNodeDict)
		if err != nil {
			return nil, err
		}
	}

	return xRefTable.emptyPage(root, mediaBox)
}

func (xRefTable *XRefTable) insertBlankPagesIntoPageTree(root *types.IndirectRef, pAttrs *InheritedPageAttrs, p *int, selectedPages types.IntSet, before bool) (int, error) {

	d, err := xRefTable.DereferenceDict(*root)
	if err != nil {
		return 0, err
	}

	consolidateRes := false
	err = xRefTable.checkInheritedPageAttrs(d, pAttrs, consolidateRes)
	if err != nil {
		return 0, err
	}

	kids := d.ArrayEntry("Kids")
	if kids == nil {
		return 0, nil
	}

	i := 0
	a := types.Array{}

	for _, o := range kids {

		if o == nil {
			continue
		}

		// Dereference next page node dict.
		ir, ok := o.(types.IndirectRef)
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
			j, err := xRefTable.insertBlankPagesIntoPageTree(&ir, pAttrs, p, selectedPages, before)
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
	d.Update("Count", types.Integer(i))

	return i, nil
}

// InsertBlankPages inserts a blank page before or after each selected page.
func (xRefTable *XRefTable) InsertBlankPages(pages types.IntSet, before bool) error {

	root, err := xRefTable.Pages()
	if err != nil {
		return err
	}

	var inhPAttrs InheritedPageAttrs
	p := 0

	_, err = xRefTable.insertBlankPagesIntoPageTree(root, &inhPAttrs, &p, pages, before)

	return err
}

// StreamDictIndRef creates a new stream dict for bb.
func (xRefTable *XRefTable) StreamDictIndRef(bb []byte) (*types.IndirectRef, error) {
	sd, _ := xRefTable.NewStreamDictForBuf(bb)
	if err := sd.Encode(); err != nil {
		return nil, err
	}
	return xRefTable.IndRefForNewObject(*sd)
}

func (xRefTable *XRefTable) insertContent(pageDict types.Dict, bb []byte) error {

	sd, _ := xRefTable.NewStreamDictForBuf(bb)
	if err := sd.Encode(); err != nil {
		return err
	}

	ir, err := xRefTable.IndRefForNewObject(*sd)
	if err != nil {
		return err
	}

	pageDict.Insert("Contents", *ir)
	return nil
}

func appendToContentStream(sd *types.StreamDict, bb []byte) error {
	err := sd.Decode()
	if err == filter.ErrUnsupportedFilter {
		log.Info.Println("unsupported filter: unable to patch content with watermark.")
		return nil
	}
	if err != nil {
		return err
	}

	sd.Content = append(sd.Content, ' ')
	sd.Content = append(sd.Content, bb...)
	return sd.Encode()
}

// AppendContent appends bb to pageDict's content stream.
func (xRefTable *XRefTable) AppendContent(pageDict types.Dict, bb []byte) error {

	obj, found := pageDict.Find("Contents")
	if !found {
		return xRefTable.insertContent(pageDict, bb)
	}

	var entry *XRefTableEntry
	var objNr int

	ir, ok := obj.(types.IndirectRef)
	if ok {
		objNr = ir.ObjectNumber.Value()
		genNr := ir.GenerationNumber.Value()
		entry, _ = xRefTable.FindTableEntry(objNr, genNr)
		obj = entry.Object
	}

	switch o := obj.(type) {

	case types.StreamDict:

		if err := appendToContentStream(&o, bb); err != nil {
			return err
		}
		entry.Object = o

	case types.Array:

		// Get stream dict for last array element.
		o1 := o[len(o)-1]
		ir, _ = o1.(types.IndirectRef)
		objNr = ir.ObjectNumber.Value()
		genNr := ir.GenerationNumber.Value()
		entry, _ = xRefTable.FindTableEntry(objNr, genNr)
		sd, _ := (entry.Object).(types.StreamDict)

		if err := appendToContentStream(&sd, bb); err != nil {
			return err
		}
		entry.Object = o

	default:
		return errors.Errorf("pdfcpu: corrupt page \"Content\"")

	}

	return nil
}

func (xRefTable *XRefTable) HasUsedGIDs(fontName string) bool {
	usedGIDs, ok := xRefTable.UsedGIDs[fontName]
	return ok && len(usedGIDs) > 0
}

func (xRefTable *XRefTable) NameRef(nameType string) NameMap {
	nm, ok := xRefTable.NameRefs[nameType]
	if !ok {
		nm = NameMap{}
		xRefTable.NameRefs[nameType] = nm
		return nm
	}
	return nm
}
