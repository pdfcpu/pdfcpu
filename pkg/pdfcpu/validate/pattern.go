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

package validate

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"slices"

	"github.com/pdfcpu/pdfcpu/internal/contextutil"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func validateTilingPatternDict(c context.Context, xRefTable *model.XRefTable, sd *types.StreamDict, sinceVersion model.Version) error {
	dictName := "tilingPatternDict"

	if err := xRefTable.ValidateVersion(dictName, sinceVersion); err != nil {
		return fmt.Errorf("%s: %w", dictName, err)
	}

	_, err := validateNameEntry(xRefTable, sd.Dict, 0, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "Pattern" })
	if err != nil {
		return fmt.Errorf("%s.Type: %w", dictName, err)
	}

	_, err = validateIntegerEntry(xRefTable, sd.Dict, 0, dictName, "PatternType", REQUIRED, sinceVersion, func(i int) bool { return i == 1 })
	if err != nil {
		return fmt.Errorf("%s.PatternType: %w", dictName, err)
	}

	_, err = validateIntegerEntry(xRefTable, sd.Dict, 0, dictName, "PaintType", REQUIRED, sinceVersion, nil)
	if err != nil {
		return fmt.Errorf("%s.PaintType: %w", dictName, err)
	}

	_, err = validateIntegerEntry(xRefTable, sd.Dict, 0, dictName, "TilingType", REQUIRED, sinceVersion, nil)
	if err != nil {
		return fmt.Errorf("%s.TilingType: %w", dictName, err)
	}

	_, err = validateRectangleEntry(xRefTable, sd.Dict, 0, dictName, "BBox", REQUIRED, sinceVersion, nil)
	if err != nil {
		return fmt.Errorf("%s.BBox: %w", dictName, err)
	}

	_, err = validateNumberEntry(xRefTable, sd.Dict, 0, dictName, "XStep", REQUIRED, sinceVersion, func(f float64) bool { return f != 0 })
	if err != nil {
		return fmt.Errorf("%s.XStep: %w", dictName, err)
	}

	_, err = validateNumberEntry(xRefTable, sd.Dict, 0, dictName, "YStep", REQUIRED, sinceVersion, func(f float64) bool { return f != 0 })
	if err != nil {
		return fmt.Errorf("%s.YStep: %w", dictName, err)
	}

	_, err = validateNumberArrayEntry(xRefTable, sd.Dict, 0, dictName, "Matrix", OPTIONAL, sinceVersion, func(a types.Array) bool { return len(a) == 6 })
	if err != nil {
		return fmt.Errorf("%s.Matrix: %w", dictName, err)
	}

	o, ok := sd.Find("Resources")
	if !ok {
		return fmt.Errorf("%s.Resources: missing required entry", dictName)
	}

	_, err = validateResourceDict(c, xRefTable, o)
	if err != nil {
		return fmt.Errorf("%s.Resources: %w", dictName, err)
	}
	return nil
}

func validateShadingPatternDict(c context.Context, xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, sinceVersion model.Version) error {
	dictName := "shadingPatternDict"

	if err := xRefTable.ValidateVersion(dictName, sinceVersion); err != nil {
		return fmt.Errorf("%s: %w", dictName, err)
	}

	_, err := validateNameEntry(xRefTable, d, ownerObjNr, dictName, "Type", OPTIONAL, sinceVersion, func(s string) bool { return s == "Pattern" })
	if err != nil {
		return fmt.Errorf("%s.Type: %w", dictName, err)
	}

	_, err = validateIntegerEntry(xRefTable, d, ownerObjNr, dictName, "PatternType", REQUIRED, sinceVersion, func(i int) bool { return i == 2 })
	if err != nil {
		return fmt.Errorf("%s.PatternType: %w", dictName, err)
	}

	_, err = validateNumberArrayEntry(xRefTable, d, ownerObjNr, dictName, "Matrix", OPTIONAL, sinceVersion, func(a types.Array) bool { return len(a) == 6 })
	if err != nil {
		return fmt.Errorf("%s.Matrix: %w", dictName, err)
	}

	rawExtGState := d["ExtGState"]
	d1, err := validateDictEntry(xRefTable, d, ownerObjNr, dictName, "ExtGState", OPTIONAL, sinceVersion, nil)
	if err != nil {
		return fmt.Errorf("%s.ExtGState: %w", dictName, err)
	}

	if d1 != nil {
		err = validateExtGStateDict(c, xRefTable, rawExtGState)
		if err != nil {
			return fmt.Errorf("%s.ExtGState: %w", dictName, err)
		}
	}

	// Shading: required, dict or stream dict.
	o, ok := d.Find("Shading")
	if !ok {
		return fmt.Errorf("%s.Shading: missing required entry", dictName)
	}

	if err := validateShading(c, xRefTable, o, ownerObjNr); err != nil {
		err = fmt.Errorf("%s.Shading: %w", dictName, err)
		return model.WithValidationErrorObject(err, validationObjectNumber(ownerObjNr, o))
	}
	return nil
}

type patternTraversalContextKey struct{}

type patternTraversal struct {
	xRefTable  *model.XRefTable
	depth      int
	ancestors  map[int]bool
	ownerObjNr int
}

func patternTraversalFromContext(c context.Context, xRefTable *model.XRefTable) (context.Context, *patternTraversal) {
	if c != nil {
		if t, ok := c.Value(patternTraversalContextKey{}).(*patternTraversal); ok {
			return c, t
		}
	}
	t := &patternTraversal{xRefTable: xRefTable, ancestors: map[int]bool{}}
	if c == nil {
		return nil, t
	}
	return context.WithValue(c, patternTraversalContextKey{}, t), t
}

func patternObjectIdentity(o types.Object) int {
	ir, ok := o.(types.IndirectRef)
	if !ok {
		return 0
	}
	return ir.ObjectNumber.Value()
}

func (t *patternTraversal) enter(objNr int) error {
	if objNr <= 0 {
		return nil
	}
	if t.ancestors[objNr] {
		return model.ErrPatternCycle
	}
	t.ancestors[objNr] = true
	return nil
}

func (t *patternTraversal) leave(objNr int) {
	if objNr > 0 {
		delete(t.ancestors, objNr)
	}
	t.depth--
}

func validatePattern(c context.Context, xRefTable *model.XRefTable, o types.Object) error {
	c, traversal := patternTraversalFromContext(c, xRefTable)
	return traversal.validate(c, o)
}

func (t *patternTraversal) validate(c context.Context, o types.Object) (err error) {
	objNr := validationObjectNumber(t.ownerObjNr, o)
	defer func() {
		err = model.WithValidationErrorObject(err, objNr)
	}()
	previousOwnerObjNr := t.ownerObjNr
	t.ownerObjNr = objNr
	defer func() {
		t.ownerObjNr = previousOwnerObjNr
	}()
	if err := contextutil.Check(c); err != nil {
		return err
	}

	depth := t.depth + 1
	if err := t.xRefTable.CheckRecursionDepth("Pattern graph", depth); err != nil {
		return err
	}
	patternObjNr := patternObjectIdentity(o)
	if err := t.enter(patternObjNr); err != nil {
		return err
	}
	t.depth = depth
	defer t.leave(patternObjNr)

	o, err = t.xRefTable.Dereference(o)
	if err != nil || o == nil {
		if err != nil {
			return fmt.Errorf("pattern: dereference: %w", err)
		}
		return nil
	}

	switch o := o.(type) {

	case types.StreamDict:
		if err = validateTilingPatternDict(c, t.xRefTable, &o, model.V10); err != nil {
			return fmt.Errorf("tiling pattern: %w", err)
		}

	case types.Dict:
		if err = validateShadingPatternDict(c, t.xRefTable, o, objNr, model.V13); err != nil {
			return fmt.Errorf("shading pattern: %w", err)
		}

	default:
		err = errors.New("corrupt obj type, must be dict or stream dict")

	}

	return err
}

func validatePatternResourceDict(c context.Context, xRefTable *model.XRefTable, o types.Object, sinceVersion model.Version) (err error) {
	objNr := validationObjectNumber(0, o)
	defer func() {
		err = model.WithValidationErrorObject(err, objNr)
	}()

	// see 8.7 Patterns

	// Version check
	if err := xRefTable.ValidateVersion("PatternResourceDict", sinceVersion); err != nil {
		return fmt.Errorf("patternResourceDict: %w", err)
	}

	d, err := xRefTable.DereferenceDict(o)
	if err != nil || d == nil {
		if err != nil {
			return fmt.Errorf("patternResourceDict: dereference: %w", err)
		}
		return nil
	}
	c, _ = patternTraversalFromContext(c, xRefTable)

	// Iterate over pattern resource dictionary
	for _, name := range slices.Sorted(maps.Keys(d)) {
		o := d[name]
		// Process pattern
		if err = validatePattern(c, xRefTable, o); err != nil {
			return fmt.Errorf("%s: %w", objectContext(fmt.Sprintf("patternResourceDict.%s", name), o), err)
		}

	}

	return nil
}
