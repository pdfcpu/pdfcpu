package types

import (
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/pkg/errors"

	PDFCPULog "github.com/hhrutter/pdfcpu/log"
)

// PDFDict represents a PDF dict object.
type PDFDict struct {
	Dict map[string]PDFObject
}

// NewPDFDict returns a new PDFDict object.
func NewPDFDict() PDFDict {
	return PDFDict{Dict: map[string]PDFObject{}}
}

// Len returns the length of this PDFDict.
func (d *PDFDict) Len() int {
	return len(d.Dict)
}

// Insert adds a new entry to this PDFDict.
func (d *PDFDict) Insert(key string, value PDFObject) (ok bool) {
	if _, found := d.Find(key); found {
		return false
	}
	d.Dict[key] = value
	return true
}

// InsertInt adds a new int entry to this PDFDict.
func (d *PDFDict) InsertInt(key string, value int) {
	d.Insert(key, PDFInteger(value))
}

// InsertFloat adds a new float entry to this PDFDict.
func (d *PDFDict) InsertFloat(key string, value float32) {
	d.Insert(key, PDFFloat(value))
}

// InsertString adds a new string entry to this PDFDict.
func (d *PDFDict) InsertString(key, value string) {
	d.Insert(key, PDFStringLiteral(value))
}

// InsertName adds a new name entry to this PDFDict.
func (d *PDFDict) InsertName(key, value string) {
	d.Insert(key, PDFName(value))
}

// Update modifies an existing entry of this PDFDict.
func (d *PDFDict) Update(key string, value PDFObject) {
	if value != nil {
		d.Dict[key] = value
	}
}

// Find returns the PDFObject for given key and PDFDict.
func (d PDFDict) Find(key string) (value PDFObject, found bool) {
	value, found = d.Dict[key]
	return
}

// Delete deletes the PDFObject for given key.
func (d *PDFDict) Delete(key string) (value PDFObject) {

	value, found := d.Find(key)
	if !found {
		return nil
	}

	delete(d.Dict, key)

	return
}

// Entry returns the value for given key.
func (d *PDFDict) Entry(dictName, key string, required bool) (PDFObject, error) {
	obj, found := d.Find(key)
	if !found || obj == nil {
		if required {
			return nil, errors.Errorf("dict=%s required entry=%s missing", dictName, key)
		}
		PDFCPULog.Debug.Printf("dict=%s entry %s is nil\n", dictName, key)
		return nil, nil
	}
	return obj, nil
}

// BooleanEntry expects and returns a BooleanEntry for given key.
func (d PDFDict) BooleanEntry(key string) *bool {

	value, found := d.Find(key)
	if !found {
		return nil
	}

	bb, ok := value.(PDFBoolean)
	if ok {
		b := bb.Value()
		return &b
	}

	return nil
}

// StringEntry expects and returns a PDFStringLiteral entry for given key.
// Unused.
func (d PDFDict) StringEntry(key string) *string {

	value, found := d.Find(key)
	if !found {
		return nil
	}

	pdfStr, ok := value.(PDFStringLiteral)
	if ok {
		s := string(pdfStr)
		return &s
	}

	return nil
}

// NameEntry expects and returns a PDFName entry for given key.
func (d PDFDict) NameEntry(key string) *string {

	value, found := d.Find(key)
	if !found {
		return nil
	}

	pdfName, ok := value.(PDFName)
	if ok {
		s := string(pdfName)
		return &s
	}

	return nil
}

// IntEntry expects and returns a PDFInteger entry for given key.
func (d PDFDict) IntEntry(key string) *int {

	value, found := d.Find(key)
	if !found {
		return nil
	}

	pdfInt, ok := value.(PDFInteger)
	if ok {
		i := int(pdfInt)
		return &i
	}

	return nil
}

// Int64Entry expects and returns a PDFInteger entry representing an int64 value for given key.
func (d PDFDict) Int64Entry(key string) *int64 {

	value, found := d.Find(key)
	if !found {
		return nil
	}

	pdfInt, ok := value.(PDFInteger)
	if ok {
		i := int64(pdfInt)
		return &i
	}

	return nil
}

// IndirectRefEntry returns an indirectRefEntry for given key for this dictionary.
func (d PDFDict) IndirectRefEntry(key string) *PDFIndirectRef {

	value, found := d.Find(key)
	if !found {
		return nil
	}

	pdfIndRef, ok := value.(PDFIndirectRef)
	if ok {
		return &pdfIndRef
	}

	// return err?
	return nil
}

// PDFDictEntry expects and returns a PDFDict entry for given key.
func (d PDFDict) PDFDictEntry(key string) *PDFDict {

	value, found := d.Find(key)
	if !found {
		return nil
	}

	dict, ok := value.(PDFDict)
	if ok {
		return &dict
	}

	return nil
}

// PDFStreamDictEntry expects and returns a PDFStreamDict entry for given key.
// unused.
func (d PDFDict) PDFStreamDictEntry(key string) *PDFStreamDict {

	value, found := d.Find(key)
	if !found {
		return nil
	}

	dict, ok := value.(PDFStreamDict)
	if ok {
		return &dict
	}

	return nil
}

// PDFArrayEntry expects and returns a PDFArray entry for given key.
func (d PDFDict) PDFArrayEntry(key string) *PDFArray {

	value, found := d.Find(key)
	if !found {
		return nil
	}

	array, ok := value.(PDFArray)
	if ok {
		return &array
	}

	return nil
}

// PDFStringLiteralEntry returns a PDFStringLiteral object for given key.
func (d PDFDict) PDFStringLiteralEntry(key string) *PDFStringLiteral {

	value, found := d.Find(key)
	if !found {
		return nil
	}

	s, ok := value.(PDFStringLiteral)
	if ok {
		return &s
	}

	return nil
}

// PDFHexLiteralEntry returns a PDFHexLiteral object for given key.
func (d PDFDict) PDFHexLiteralEntry(key string) *PDFHexLiteral {

	value, found := d.Find(key)
	if !found {
		return nil
	}

	s, ok := value.(PDFHexLiteral)
	if ok {
		return &s
	}

	return nil
}

// PDFNameEntry returns a PDFName object for given key.
func (d PDFDict) PDFNameEntry(key string) *PDFName {

	value, found := d.Find(key)
	if !found {
		return nil
	}

	s, ok := value.(PDFName)
	if ok {
		return &s
	}

	return nil
}

// Length returns a *int64 for entry with key "Length".
// Stream length may be referring to an indirect object.
func (d PDFDict) Length() (*int64, *int) {

	val := d.Int64Entry("Length")
	if val != nil {
		return val, nil
	}

	indirectRef := d.IndirectRefEntry("Length")
	if indirectRef == nil {
		return nil, nil
	}

	intVal := indirectRef.ObjectNumber.Value()

	return nil, &intVal
}

// Type returns the value of the name entry for key "Type".
func (d PDFDict) Type() *string {
	return d.NameEntry("Type")
}

// Subtype returns the value of the name entry for key "Subtype".
func (d PDFDict) Subtype() *string {
	return d.NameEntry("Subtype")
}

// Size returns the value of the int entry for key "Size"
func (d PDFDict) Size() *int {
	return d.IntEntry("Size")
}

// IsObjStm returns true if given PDFDict is an object stream.
func (d PDFDict) IsObjStm() bool {
	return d.Type() != nil && *d.Type() == "ObjStm"
}

// W returns a *PDFArray for key "W".
func (d PDFDict) W() *PDFArray {
	return d.PDFArrayEntry("W")
}

// Prev returns the previous offset.
func (d PDFDict) Prev() *int64 {
	return d.Int64Entry("Prev")
}

// Index returns a *PDFArray for key "Index".
func (d PDFDict) Index() *PDFArray {
	return d.PDFArrayEntry("Index")
}

// N returns a *int for key "N".
func (d PDFDict) N() *int {
	return d.IntEntry("N")
}

// First returns a *int for key "First".
func (d PDFDict) First() *int {
	return d.IntEntry("First")
}

// IsLinearizationParmDict returns true if this dict has an int entry for key "Linearized".
func (d PDFDict) IsLinearizationParmDict() bool {
	return d.IntEntry("Linearized") != nil
}

func (d PDFDict) indentedString(level int) string {

	logstr := []string{"<<\n"}
	tabstr := strings.Repeat("\t", level)

	var keys []string
	for k := range d.Dict {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {

		v := d.Dict[k]

		if subdict, ok := v.(PDFDict); ok {
			dictStr := subdict.indentedString(level + 1)
			logstr = append(logstr, fmt.Sprintf("%s<%s, %s>\n", tabstr, k, dictStr))
			continue
		}

		if array, ok := v.(PDFArray); ok {
			arrStr := array.indentedString(level + 1)
			logstr = append(logstr, fmt.Sprintf("%s<%s, %s>\n", tabstr, k, arrStr))
			continue
		}

		logstr = append(logstr, fmt.Sprintf("%s<%s, %v>\n", tabstr, k, v))

	}

	logstr = append(logstr, fmt.Sprintf("%s%s", strings.Repeat("\t", level-1), ">>"))

	return strings.Join(logstr, "")
}

// PDFString returns a string representation as found in and written to a PDF file.
func (d PDFDict) PDFString() string {

	logstr := []string{} //make([]string, 20)
	logstr = append(logstr, "<<")

	var keys []string
	for k := range d.Dict {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {

		v := d.Dict[k]

		if v == nil {
			logstr = append(logstr, fmt.Sprintf("/%s null", k))
			continue
		}

		subdict, ok := v.(PDFDict)
		if ok {
			dictStr := subdict.PDFString()
			logstr = append(logstr, fmt.Sprintf("/%s%s", k, dictStr))
			continue
		}

		array, ok := v.(PDFArray)
		if ok {
			arrStr := array.PDFString()
			logstr = append(logstr, fmt.Sprintf("/%s%s", k, arrStr))
			continue
		}

		indRef, ok := v.(PDFIndirectRef)
		if ok {
			indRefstr := indRef.PDFString()
			logstr = append(logstr, fmt.Sprintf("/%s %s", k, indRefstr))
			continue
		}

		name, ok := v.(PDFName)
		if ok {
			namestr := name.PDFString()
			logstr = append(logstr, fmt.Sprintf("/%s%s", k, namestr))
			continue
		}

		i, ok := v.(PDFInteger)
		if ok {
			logstr = append(logstr, fmt.Sprintf("/%s %s", k, i))
			continue
		}

		f, ok := v.(PDFFloat)
		if ok {
			logstr = append(logstr, fmt.Sprintf("/%s %s", k, f))
			continue
		}

		b, ok := v.(PDFBoolean)
		if ok {
			logstr = append(logstr, fmt.Sprintf("/%s %s", k, b))
			continue
		}

		sl, ok := v.(PDFStringLiteral)
		if ok {
			logstr = append(logstr, fmt.Sprintf("/%s%s", k, sl))
			continue
		}

		hl, ok := v.(PDFHexLiteral)
		if ok {
			logstr = append(logstr, fmt.Sprintf("/%s%s", k, hl))
			continue
		}

		log.Fatalf("PDFDict.PDFString(): entry of unknown object type: %T %[1]v\n", v)
	}

	logstr = append(logstr, ">>")
	return strings.Join(logstr, "")
}

func (d PDFDict) String() string {
	return d.indentedString(1)
}

// StringEntryBytes returns the byte slice representing the string value for key.
func (d PDFDict) StringEntryBytes(key string) ([]byte, error) {

	s := d.PDFStringLiteralEntry(key)
	if s != nil {
		return Unescape(s.Value())
	}

	h := d.PDFHexLiteralEntry(key)
	if h != nil {
		return h.Bytes()
	}

	return nil, nil
}
