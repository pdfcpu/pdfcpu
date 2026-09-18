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
	"fmt"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

func validateRemoteDestinationArray(xRefTable *model.XRefTable, a types.Array, ownerObjNr int) (err error) {
	defer func() {
		err = model.WithValidationErrorObject(err, ownerObjNr)
	}()
	if !validateDestinationArrayLength(a) {
		err = model.WithValidationErrorObject(fmt.Errorf("remote destination array: invalid length %d", len(a)), ownerObjNr)
		if xRefTable.ValidationMode == model.ValidationRelaxed {
			xRefTable.AddValidationNotice(model.NewValidationNotice(
				model.NoticePhaseValidate,
				model.NoticeDigested,
				err.Error(),
				err,
			))
			return nil
		}
		return err
	}
	// See 12.6.4.3 and 12.6.4.4: remote and embedded destinations use a zero-based page number.
	o, err := xRefTable.Dereference(a[0])
	if err != nil {
		return model.WithValidationErrorObject(err, validationObjectNumber(ownerObjNr, a[0]))
	}
	page, ok := o.(types.Integer)
	if !ok || page < 0 {
		strictFailure := fmt.Errorf("remote destination array[0]: expected non-negative page number, got %v (%T)", o, o)
		strictFailure = model.WithValidationErrorObject(strictFailure, validationObjectNumber(ownerObjNr, a[0]))
		if xRefTable.ValidationMode == model.ValidationStrict {
			return strictFailure
		}
		if err := validateDestinationArrayMode(xRefTable, a, ownerObjNr); err != nil {
			return err
		}
		xRefTable.AddValidationNotice(model.NewValidationNotice(
			model.NoticePhaseValidate,
			model.NoticeDigested,
			strictFailure.Error(),
			strictFailure,
		))
		return nil
	}
	return validateDestinationArrayMode(xRefTable, a, ownerObjNr)
}

func validateRemoteActionDestinationEntry(xRefTable *model.XRefTable, d types.Dict, ownerObjNr int, dictName, entryName string) error {
	rawEntry := d[entryName]
	destinationObjNr := validationObjectNumber(ownerObjNr, rawEntry)
	o, err := validateEntry(xRefTable, d, ownerObjNr, dictName, entryName, REQUIRED, model.V10)
	if err == nil && o != nil {
		if a, ok := o.(types.Array); ok {
			err = validateRemoteDestinationArray(xRefTable, a, destinationObjNr)
		} else {
			_, err = validateDestination(xRefTable, rawEntry, ownerObjNr, true)
		}
	}
	if err != nil {
		return fmt.Errorf("%s: %w", dictEntryContext(dictName, entryName, rawEntry), err)
	}
	return nil
}
