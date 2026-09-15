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
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func validateTrapNetStateValues(c context.Context, x *model.XRefTable, d types.Dict, dictName string) error {
	a, err := validateArrayEntry(x, d, 0, dictName, "AnnotStates", OPTIONAL, model.V10, nil)
	if err != nil || a == nil {
		return err
	}
	objNr := validationEntryObjectNumber(0, d, "AnnotStates")
	for i, raw := range a {
		if err := contextutil.Check(c); err != nil {
			return err
		}
		o, err := x.Dereference(raw)
		if err != nil {
			err = fmt.Errorf("%s.AnnotStates[%d]: %w", dictName, i, err)
			return model.WithValidationErrorObject(err, validationObjectNumber(objNr, raw))
		}
		if o == nil {
			continue
		}
		if _, ok := o.(types.Name); !ok {
			err := fmt.Errorf("%s.AnnotStates[%d]: expected name or null, got %T", dictName, i, o)
			return model.WithValidationErrorObject(err, validationObjectNumber(objNr, raw))
		}
	}
	return nil
}

func validatePageAnnotationDict(c context.Context, x *model.XRefTable, d types.Dict, precedingCount int) (bool, error) {
	trapNet, err := validateAnnotationDict(c, x, d)
	if err != nil || !trapNet {
		return trapNet, err
	}
	// The page walk requires TrapNet to be last.
	// Count retained annotations, excluding this annotation and removed entries.
	a, err := x.DereferenceArray(d["AnnotStates"])
	if err != nil || a == nil {
		return true, err
	}
	objNr := validationEntryObjectNumber(0, d, "AnnotStates")
	if len(a) != precedingCount {
		expected := fmt.Sprintf("%d, one state per page annotation excluding TrapNet", precedingCount)
		return true, arrayCardinalityError("annotDict", "AnnotStates", objNr, len(a), expected)
	}
	return true, nil
}
