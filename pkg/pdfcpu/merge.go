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
	"github.com/pdfcpu/pdfcpu/pkg/log"
)

func patchIndRef(ir *IndirectRef, lookup map[int]int) {
	i := ir.ObjectNumber.Value()
	ir.ObjectNumber = Integer(lookup[i])
}

func patchObject(o Object, lookup map[int]int) Object {

	log.Trace.Printf("patchObject before: %v\n", o)

	var ob Object

	switch obj := o.(type) {

	case IndirectRef:
		patchIndRef(&obj, lookup)
		ob = obj

	case Dict:
		patchDict(obj, lookup)
		ob = obj

	case StreamDict:
		patchDict(obj.Dict, lookup)
		ob = obj

	case ObjectStreamDict:
		patchDict(obj.Dict, lookup)
		ob = obj

	case XRefStreamDict:
		patchDict(obj.Dict, lookup)
		ob = obj

	case Array:
		patchArray(obj, lookup)
		ob = obj

	}

	log.Trace.Printf("patchObject end: %v\n", ob)

	return ob
}

func patchDict(d Dict, lookup map[int]int) {

	log.Trace.Printf("patchDict before: %v\n", d)

	for k, obj := range d {
		o := patchObject(obj, lookup)
		if o != nil {
			d[k] = o
		}
	}

	log.Trace.Printf("patchDict after: %v\n", d)
}

func patchArray(a Array, lookup map[int]int) {

	log.Trace.Printf("patchArray begin: %v\n", a)

	for i, obj := range a {
		o := patchObject(obj, lookup)
		if o != nil {
			a[i] = o
		}
	}

	log.Trace.Printf("patchArray end: %v\n", a)
}

func objNrsIntSet(ctx *Context) IntSet {

	objNrs := IntSet{}

	for k := range ctx.Table {
		if k == 0 {
			// obj#0 is always the head of the freelist.
			continue
		}
		objNrs[k] = true
	}

	return objNrs
}

func lookupTable(keys IntSet, i int) map[int]int {

	m := map[int]int{}

	for k := range keys {
		m[k] = i
		i++
	}

	return m
}

// Patch an IntSet of objNrs using lookup.
func patchObjects(s IntSet, lookup map[int]int) IntSet {

	t := IntSet{}

	for k, v := range s {
		if v {
			t[lookup[k]] = v
		}
	}

	return t
}

func patchSourceObjectNumbers(ctxSource, ctxDest *Context) {

	log.Debug.Printf("patchSourceObjectNumbers: ctxSource: xRefTableSize:%d trailer.Size:%d - %s\n", len(ctxSource.Table), *ctxSource.Size, ctxSource.Read.FileName)
	log.Debug.Printf("patchSourceObjectNumbers:   ctxDest: xRefTableSize:%d trailer.Size:%d - %s\n", len(ctxDest.Table), *ctxDest.Size, ctxDest.Read.FileName)

	// Patch source xref tables obj numbers which are essentially the keys.
	//logInfoMerge.Printf("Source XRefTable before:\n%s\n", ctxSource)

	objNrs := objNrsIntSet(ctxSource)

	// Create lookup table for object numbers.
	// The first number is the successor of the last number in ctxDest.
	lookup := lookupTable(objNrs, *ctxDest.Size)

	// Patch pointer to root object
	patchIndRef(ctxSource.Root, lookup)

	// Patch pointer to info object
	if ctxSource.Info != nil {
		patchIndRef(ctxSource.Info, lookup)
	}

	// Patch free object zero
	entry := ctxSource.Table[0]
	off := int(*entry.Offset)
	if off != 0 {
		i := int64(lookup[off])
		entry.Offset = &i
	}

	// Patch all indRefs for xref table entries.
	for k := range objNrs {

		//logDebugMerge.Printf("patching obj #%d\n", k)

		entry := ctxSource.Table[k]

		if entry.Free {
			log.Debug.Printf("patch free entry: old offset:%d\n", *entry.Offset)
			off := int(*entry.Offset)
			if off == 0 {
				continue
			}
			i := int64(lookup[off])
			entry.Offset = &i
			log.Debug.Printf("patch free entry: new offset:%d\n", *entry.Offset)
			continue
		}

		patchObject(entry.Object, lookup)
	}

	// Patch xref entry object numbers.
	m := make(map[int]*XRefTableEntry, *ctxSource.Size)
	for k, v := range lookup {
		m[v] = ctxSource.Table[k]
		*ctxSource.Size++
	}
	m[0] = ctxSource.Table[0]
	ctxSource.Table = m

	// Patch DuplicateInfo object numbers.
	ctxSource.Optimize.DuplicateInfoObjects = patchObjects(ctxSource.Optimize.DuplicateInfoObjects, lookup)

	// Patch Linearization object numbers.
	ctxSource.LinearizationObjs = patchObjects(ctxSource.LinearizationObjs, lookup)

	// Patch XRefStream objects numbers.
	ctxSource.Read.XRefStreams = patchObjects(ctxSource.Read.XRefStreams, lookup)

	// Patch object stream object numbers.
	ctxSource.Read.ObjectStreams = patchObjects(ctxSource.Read.ObjectStreams, lookup)

	log.Debug.Printf("patchSourceObjectNumbers end")
}

func appendSourcePageTreeToDestPageTree(ctxSource, ctxDest *Context) error {

	log.Debug.Println("appendSourcePageTreeToDestPageTree begin")

	indRefPageTreeRootDictSource, err := ctxSource.Pages()
	if err != nil {
		return err
	}

	pageTreeRootDictSource, _ := ctxSource.XRefTable.DereferenceDict(*indRefPageTreeRootDictSource)
	pageCountSource := pageTreeRootDictSource.IntEntry("Count")

	indRefPageTreeRootDictDest, err := ctxDest.Pages()
	if err != nil {
		return err
	}

	pageTreeRootDictDest, _ := ctxDest.XRefTable.DereferenceDict(*indRefPageTreeRootDictDest)
	pageCountDest := pageTreeRootDictDest.IntEntry("Count")

	a := pageTreeRootDictDest.ArrayEntry("Kids")
	log.Debug.Printf("Kids before: %v\n", a)

	pageTreeRootDictSource.Insert("Parent", *indRefPageTreeRootDictDest)

	// The source page tree gets appended on to the dest page tree.
	a = append(a, *indRefPageTreeRootDictSource)
	log.Debug.Printf("Kids after: %v\n", a)

	pageTreeRootDictDest.Update("Count", Integer(*pageCountDest+*pageCountSource))
	pageTreeRootDictDest.Update("Kids", a)

	ctxDest.PageCount += ctxSource.PageCount

	log.Debug.Println("appendSourcePageTreeToDestPageTree end")

	return nil
}

func appendSourceObjectsToDest(ctxSource, ctxDest *Context) {

	log.Debug.Println("appendSourceObjectsToDest begin")

	for objNr, entry := range ctxSource.Table {

		// Do not copy free list head.
		if objNr == 0 {
			continue
		}

		log.Debug.Printf("adding obj %d from src to dest\n", objNr)

		ctxDest.Table[objNr] = entry

		*ctxDest.Size++

	}

	log.Debug.Println("appendSourceObjectsToDest end")
}

// merge two disjunct IntSets
func mergeIntSets(src, dest IntSet) {
	for k := range src {
		dest[k] = true
	}
}

func mergeDuplicateObjNumberIntSets(ctxSource, ctxDest *Context) {

	log.Debug.Println("mergeDuplicateObjNumberIntSets begin")

	mergeIntSets(ctxSource.Optimize.DuplicateInfoObjects, ctxDest.Optimize.DuplicateInfoObjects)
	mergeIntSets(ctxSource.LinearizationObjs, ctxDest.LinearizationObjs)
	mergeIntSets(ctxSource.Read.XRefStreams, ctxDest.Read.XRefStreams)
	mergeIntSets(ctxSource.Read.ObjectStreams, ctxDest.Read.ObjectStreams)

	log.Debug.Println("mergeDuplicateObjNumberIntSets end")
}

// MergeXRefTables merges Context ctxSource into ctxDest by appending its page tree.
func MergeXRefTables(ctxSource, ctxDest *Context) (err error) {

	// Sweep over ctxSource cross ref table and ensure valid object numbers in ctxDest's space.
	patchSourceObjectNumbers(ctxSource, ctxDest)

	mergeAcroForms(ctxSource, ctxDest)

	// Append ctxSource pageTree to ctxDest pageTree.
	log.Debug.Println("appendSourcePageTreeToDestPageTree")
	err = appendSourcePageTreeToDestPageTree(ctxSource, ctxDest)
	if err != nil {
		return err
	}

	// Append ctxSource objects to ctxDest
	log.Debug.Println("appendSourceObjectsToDest")
	appendSourceObjectsToDest(ctxSource, ctxDest)

	// Mark source's root object as free.
	err = ctxDest.turnEntryToFree(int(ctxSource.Root.ObjectNumber))
	if err != nil {
		return
	}

	// Mark source's info object as free.
	// Note: Any indRefs this info object depends on are missed.
	if ctxSource.Info != nil {
		err = ctxDest.turnEntryToFree(int(ctxSource.Info.ObjectNumber))
		if err != nil {
			return
		}
	}

	// Merge all IntSets containing redundant object numbers.
	log.Debug.Println("mergeDuplicateObjNumberIntSets")
	mergeDuplicateObjNumberIntSets(ctxSource, ctxDest)

	log.Info.Printf("Dest XRefTable after merge:\n%s\n", ctxDest)

	return nil
}
