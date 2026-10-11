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

	"github.com/pdfcpu/pdfcpu/internal/contextutil"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

type mediaValidationKind uint8

const (
	threeDStreamValidation mediaValidationKind = iota
	richMediaContentValidation
	richMediaAssetsValidation
)

type mediaValidationObject struct {
	kind mediaValidationKind
	ref  types.IndirectRef
}

type mediaValidationKey struct{}

// withMediaValidation keeps successful structural checks separate by role and local to one root validation pass.
func withMediaValidation(c context.Context) context.Context {
	return context.WithValue(c, mediaValidationKey{}, map[mediaValidationObject]bool{})
}

// validateMediaObject reports a cache hit so callers can still perform owner-specific work.
// Direct objects have no stable identity and are always validated.
func validateMediaObject(c context.Context, raw types.Object, kind mediaValidationKind, validate func() error) (bool, error) {
	if err := contextutil.Check(c); err != nil {
		return false, err
	}
	cache, _ := c.Value(mediaValidationKey{}).(map[mediaValidationObject]bool)
	ref, indirect := raw.(types.IndirectRef)
	key := mediaValidationObject{kind: kind, ref: ref}
	if indirect && cache[key] {
		return true, nil
	}
	if err := validate(); err != nil {
		return false, err
	}
	if err := contextutil.Check(c); err != nil {
		return false, err
	}
	if indirect && cache != nil {
		cache[key] = true
	}
	return false, nil
}
