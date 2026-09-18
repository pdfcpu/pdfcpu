/*
Copyright 2026 The pdfcpu Authors.

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
	"fmt"

	"github.com/pdfcpu/pdfcpu/internal/contextutil"
	"github.com/pdfcpu/pdfcpu/pkg/filter"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// colorSpaceComponents obtains the component count of a previously validated colour space without walking its graph.
// Zero denotes an absent colour space or a name already tolerated by relaxed validation.
func colorSpaceComponents(x *model.XRefTable, o types.Object, ownerObjNr int) (n int, err error) {
	objNr := validationObjectNumber(ownerObjNr, o)
	defer func() { err = model.WithValidationErrorObject(err, objNr) }()
	o, err = x.Dereference(o)
	if err != nil || o == nil {
		return 0, err
	}
	if name, ok := o.(types.Name); ok {
		return namedColorSpaceComponents(name), nil
	}
	a, ok := o.(types.Array)
	if !ok {
		return 0, fmt.Errorf("colour-space components: expected name or array, got %T", o)
	}
	if len(a) == 0 {
		return 0, fmt.Errorf("colour-space components: empty array")
	}
	name, err := colorSpaceArrayName(x, a, objNr)
	if err != nil {
		return 0, err
	}
	switch name {
	case model.ICCBasedCS:
		return iccColorSpaceComponents(x, a, objNr)
	case model.DeviceNCS:
		if len(a) < 2 {
			return 0, fmt.Errorf("DeviceN colour-space components: missing names")
		}
		names, err := x.DereferenceArray(a[1])
		if err != nil {
			return 0, model.WithValidationErrorObject(err, validationObjectNumber(objNr, a[1]))
		}
		if len(names) == 0 {
			return 0, fmt.Errorf("DeviceN colour-space components: empty names")
		}
		return len(names), nil
	}
	return namedColorSpaceComponents(name), nil
}

func namedColorSpaceComponents(name types.Name) int {
	switch name {
	case model.DeviceGrayCS, model.CalGrayCS, model.IndexedCS, model.SeparationCS:
		return 1
	case model.DeviceRGBCS, model.CalRGBCS, model.LabCS:
		return 3
	case model.DeviceCMYKCS:
		return 4
	}
	return 0
}

func iccColorSpaceComponents(x *model.XRefTable, a types.Array, ownerObjNr int) (n int, err error) {
	if len(a) != 2 {
		return 0, fmt.Errorf("ICCBased colour-space components: expected profile")
	}
	objNr := validationObjectNumber(ownerObjNr, a[1])
	defer func() { err = model.WithValidationErrorObject(err, objNr) }()
	sd, err := validateStreamDictForObject(x, a[1], objNr)
	if err != nil {
		return 0, err
	}
	if sd == nil {
		return 0, fmt.Errorf("ICCBased colour-space components: missing profile")
	}
	v, err := validateIntegerEntry(x, sd.Dict, objNr, "ICCBasedColorSpace", "N", REQUIRED, model.V10,
		func(n int) bool { return n == 1 || n == 3 || n == 4 })
	if err != nil {
		return 0, err
	}
	return v.Value(), nil
}

func validateImageArrays(c context.Context, x *model.XRefTable, sd *types.StreamDict, ownerObjNr int, dictName string, imageMask bool, bpc *types.Integer) error {
	components := 1
	if !imageMask {
		var err error
		components, err = colorSpaceComponents(x, sd.Dict["ColorSpace"], ownerObjNr)
		if err != nil {
			return fmt.Errorf("%s.ColorSpace: %w", dictName, err)
		}
		bits := bpc
		if sd.HasSoleFilterNamed(filter.JPX) {
			bits = nil
		}
		if err = validateMaskEntry(c, x, sd.Dict, ownerObjNr, dictName, "Mask", OPTIONAL, model.V13, components, bits); err != nil {
			return err
		}
		// JPX without an explicit colour space ignores Decode; its component count comes from the codestream.
		if components == 0 && sd.HasSoleFilterNamed(filter.JPX) {
			cs, err := x.Dereference(sd.Dict["ColorSpace"])
			if err != nil {
				return err
			}
			if cs == nil {
				return nil
			}
		}
	}
	return validateImageDecode(c, x, sd.Dict, ownerObjNr, dictName, components, imageMask)
}

func validateImageArrayLength(a types.Array, objNr int, dictName, entryName string, components int) error {
	if components == 0 {
		return validateArrayPairs(a, objNr, dictName, entryName, 1)
	}
	if len(a)%2 == 0 && len(a)/2 == components {
		return nil
	}
	expected := fmt.Sprintf("two values per colour component (%d components)", components)
	return arrayCardinalityError(dictName, entryName, objNr, len(a), expected)
}

func validateImageDecode(c context.Context, x *model.XRefTable, d types.Dict, ownerObjNr int, dictName string, components int, imageMask bool) error {
	a, err := validateArrayEntry(x, d, ownerObjNr, dictName, "Decode", OPTIONAL, model.V10, nil)
	if err != nil || a == nil {
		return err
	}
	objNr := validationEntryObjectNumber(ownerObjNr, d, "Decode")
	if err = validateImageArrayLength(a, objNr, dictName, "Decode", components); err != nil {
		return err
	}
	previous := float64(0)
	for i, o := range a {
		if err := contextutil.Check(c); err != nil {
			return err
		}
		n, err := x.DereferenceNumber(o)
		if err != nil {
			err = fmt.Errorf("%s.Decode[%d]: expected number: %w", dictName, i, err)
		} else if imageMask && (n != 0 && n != 1 || i == 1 && n == previous) {
			err = fmt.Errorf("%s.Decode[%d]: image mask requires [0 1] or [1 0]", dictName, i)
		}
		if err != nil {
			return model.WithValidationErrorObject(err, validationObjectNumber(objNr, o))
		}
		previous = n
	}
	return nil
}

// relaxedIndexedMaskComponents accepts base-space mask pairs emitted by older image producers.
// The image colour space has already been validated; Decode keeps the Indexed component count.
func relaxedIndexedMaskComponents(x *model.XRefTable, d types.Dict, a types.Array, ownerObjNr, components int) int {
	if x.ValidationMode != model.ValidationRelaxed || len(a) == 2*components {
		return components
	}
	cs, err := x.DereferenceArray(d["ColorSpace"])
	if err != nil || len(cs) != 4 {
		return components
	}
	name, err := colorSpaceArrayName(x, cs, ownerObjNr)
	if err != nil || name != model.IndexedCS {
		return components
	}
	n, err := colorSpaceComponents(x, cs[1], ownerObjNr)
	if err == nil && n > 1 && len(a) == 2*n {
		return n
	}
	return components
}

func validateColorKeyMask(c context.Context, x *model.XRefTable, a types.Array, objNr int, dictName string, components int, bits *types.Integer) error {
	if err := validateImageArrayLength(a, objNr, dictName, "Mask", components); err != nil {
		return err
	}
	previous := 0
	for i, o := range a {
		if err := contextutil.Check(c); err != nil {
			return err
		}
		n, err := x.DereferenceInteger(o)
		if err != nil {
			err = fmt.Errorf("%s.Mask[%d]: expected integer: %w", dictName, i, err)
		} else if n == nil {
			err = fmt.Errorf("%s.Mask[%d]: expected integer, got null", dictName, i)
		} else {
			err = validateColorKeyMaskValue(n.Value(), previous, bits, i, dictName)
		}
		if err != nil {
			return model.WithValidationErrorObject(err, validationObjectNumber(objNr, o))
		}
		previous = n.Value()
	}
	return nil
}

func validateColorKeyMaskValue(n, previous int, bits *types.Integer, index int, dictName string) error {
	if bits != nil && *bits <= 0 {
		return fmt.Errorf("%s.Mask: invalid BitsPerComponent %d, expected positive integer", dictName, *bits)
	}
	if n < 0 {
		return fmt.Errorf("%s.Mask[%d]: invalid sample %d, expected nonnegative integer", dictName, index, n)
	}
	// Compare the sample's bit length without an overflowing shift, including when bit depth is unavailable.
	if bits != nil && uint(n)>>uint(*bits) != 0 {
		return fmt.Errorf("%s.Mask[%d]: sample %d exceeds BitsPerComponent %d", dictName, index, n, *bits)
	}
	if index%2 == 1 && n < previous {
		return fmt.Errorf("%s.Mask[%d]: maximum %d is less than minimum %d", dictName, index, n, previous)
	}
	return nil
}
