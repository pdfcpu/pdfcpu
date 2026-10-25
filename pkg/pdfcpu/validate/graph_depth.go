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

import "context"

// graphDepths holds the resource and pattern graph depths of a validation step.
// Resource dictionaries, patterns and property lists recurse through each other, and each graph has its own depth limit.
type graphDepths struct {
	resource int
	pattern  int
}

func currentGraphDepths(c context.Context) graphDepths {
	d := graphDepths{resource: -1}
	if c == nil {
		return d
	}
	if t, ok := c.Value(resourceTraversalContextKey{}).(*resourceTraversal); ok {
		d.resource = t.depth
	}
	if t, ok := c.Value(patternTraversalContextKey{}).(*patternTraversal); ok {
		d.pattern = t.depth
	}
	return d
}

// validatedGraph records objects whose validation succeeded within one traversal.
// A shared object is validated once per path depth instead of once per path.
type validatedGraph map[int]graphDepths

// alreadyValidated reports whether objNr succeeded at depths at least as deep as d.
// A deeper path needs revalidation to preserve descendant depth checks.
func (v validatedGraph) alreadyValidated(objNr int, d graphDepths) bool {
	if objNr <= 0 {
		return false
	}
	done, ok := v[objNr]
	return ok && d.resource <= done.resource && d.pattern <= done.pattern
}

// markValidated records a successful validation of objNr at depths d.
// The graph limits are independent, so successes at different depths combine per graph.
func (v validatedGraph) markValidated(objNr int, d graphDepths) {
	if objNr <= 0 {
		return
	}
	if done, ok := v[objNr]; ok {
		d.resource = max(d.resource, done.resource)
		d.pattern = max(d.pattern, done.pattern)
	}
	v[objNr] = d
}
