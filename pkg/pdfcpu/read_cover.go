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
	"context"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/log"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// FastCover enables fast cover generation by only loading necessary objects

func coverRefObj(c context.Context, ctx *model.Context, objNr int, ids map[int]struct{}, stream types.IntSet) {

	if _, ok := ids[objNr]; ok {
		return
	}
	ids[objNr] = struct{}{}

	xRefTable := ctx.XRefTable

	if err := dereferenceObject(c, ctx, objNr); err != nil {
		errstr := err.Error()
		if strings.Contains(errstr, "decompressXRefTableEntry: problem dereferencing object stream") && strings.Contains(errstr, "no object stream") {
			entry := ctx.Table[objNr]
			iidd := *entry.ObjectStream
			stream[iidd] = true
			// getDir(iidd, ids, stream)
			if err := decodeObjectStream(c, ctx, iidd); err != nil {
				return
			}
			ids[iidd] = struct{}{}
			if err := dereferenceObject(c, ctx, objNr); err != nil {
				return
			}
		} else {
			return
		}
	}

	entry := ctx.Table[objNr]
	obj := entry.Object

	switch obj.(type) {
	case types.LazyObjectStreamObject:
		dereferencedObject(c, ctx, objNr)
	}
	switch entry.Object.(type) {
	case types.ObjectStreamDict:
		decompressXRefTableEntry(xRefTable, objNr, entry)
	}

	var action func(types.Object)
	action = func(obj types.Object) {
		switch v := obj.(type) {
		case types.Array:
			for _, aa := range v {
				action(aa)
			}
		case types.IndirectRef:
			coverRefObj(c, ctx, v.ObjectNumber.Value(), ids, stream)
		case types.Dict:
			// Recursively process dictionary entries
			for _, val := range v {
				action(val)
			}
		}
	}

	switch obj := entry.Object.(type) {
	case types.StringLiteral:
	case types.Array:
		for _, o := range obj {
			if inf, ok := o.(types.IndirectRef); ok {
				nnnn := inf.ObjectNumber.Value()
				coverRefObj(c, ctx, nnnn, ids, stream)

				if true {
					//check first page
					ent := ctx.Table[nnnn]
					if oooo, ok := ent.Object.(types.Dict); ok {
						tt := oooo["Type"]
						tname, _ := tt.(types.Name)
						if tname == "Page" {
							return
						}
					}
				}
			}
		}
	case types.StreamDict:
		// Process StreamDict's dictionary part for any indirect references
		for _, val := range obj.Dict {
			action(val)
		}
	case types.Dict:
		rootDict, err := ctx.XRefTable.DereferenceDict(obj)
		if err != nil {
			return
		}

		tt := rootDict["Type"]
		tname, _ := tt.(types.Name)
		switch tname {
		case "Catalog":
			for k, v := range rootDict {
				switch k {
				case "PageLabels":
					action(v)
				case "Pages":
					action(v)
					// default:
					// 	action(v)
				}
			}
		case "Pages":
			Kids := rootDict["Kids"]
			Count := rootDict["Count"]
			// Parent := rootDict["Parent"]
			log.Info.Printf("Count: %v", Count)
			// logx.Printf("Parent: %v", Parent)
			Kid, isArray := Kids.(types.Array)
			if isArray {
				// Don't modify the original rootDict here, we'll create a new one later
				// rootDict["Kids"] = types.Array{Kid[0]}
				// rootDict["Count"] = types.Integer(1)
				//get first kid, page or pages for cover page
				inf := Kid[0].(types.IndirectRef)
				coverRefObj(c, ctx, inf.ObjectNumber.Value(), ids, stream)
			} else {
				action(Kids)
			}
		case "Page":
			delete(rootDict, "PieceInfo")
			fallthrough
		default:
			for k, v := range rootDict {
				if k == "Type" || k == "Parent" {
					continue
				}
				action(v)
			}
		}
	default:
		log.Info.Printf("default 2")
	}

}

func coverShrinkTable(c context.Context, ctx *model.Context) {
	log.Info.Printf("FastCover: coverTable started")
	xRefTable := ctx.XRefTable

	ids := make(map[int]struct{})
	streams := make(types.IntSet)

	if xRefTable.Encrypt != nil {
		if err := checkForEncryption(c, ctx); err != nil {
			return
		}
		objNr := xRefTable.Encrypt.ObjectNumber.Value()
		coverRefObj(c, ctx, objNr, ids, streams)
	}

	objNr := int(xRefTable.Root.ObjectNumber)
	coverRefObj(c, ctx, objNr, ids, streams)

	Table := make(map[int]*model.XRefTableEntry)
	for k, _ := range ids {
		entry := ctx.Table[k]

		// For Pages objects, create a modified copy that only includes the first page
		if entry != nil && !entry.Free {
			if dict, ok := entry.Object.(types.Dict); ok {
				if pageType, exists := dict["Type"]; exists {
					if typeName, ok := pageType.(types.Name); ok && typeName == "Pages" {
						// Create a new dict with only the first kid
						if kids, exists := dict["Kids"]; exists {
							var kkk = kids

							if ref, ok := kkk.(types.IndirectRef); ok {
								kk := ref.ObjectNumber.Value()
								next := ctx.Table[kk]
								kkk = next.Object
							}

							if kidsArray, ok := kkk.(types.Array); ok && len(kidsArray) > 0 {
								// Create a copy of the dict
								newDict := make(types.Dict)
								for key, val := range dict {
									newDict[key] = val
								}
								// Modify the copy to only include first page
								newDict["Kids"] = types.Array{kidsArray[0]}
								newDict["Count"] = types.Integer(1)

								// Create a new entry with the modified dict
								newEntry := &model.XRefTableEntry{
									Free:         entry.Free,
									Offset:       entry.Offset,
									Generation:   entry.Generation,
									Object:       newDict,
									Compressed:   entry.Compressed,
									ObjectStream: entry.ObjectStream,
								}
								Table[k] = newEntry
								continue
							}
						}
					}
				}
			}
		}

		// For all other objects, use the original entry
		Table[k] = entry
	}

	// 记录优化效果
	originalCount := len(ctx.Table)
	optimizedCount := len(Table)
	log.Info.Printf("FastCover optimization: %d -> %d objects (%.1f%% reduction)",
		originalCount, optimizedCount,
		float64(originalCount-optimizedCount)/float64(originalCount)*100)

	ctx.Table = Table
	ctx.PageCount = 1
	ctx.Read.ObjectStreams = streams

}
