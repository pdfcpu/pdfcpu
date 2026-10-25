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

package pdfcpu

import (
	"fmt"
	"slices"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/types"
)

// maxErrorPathEntries limits the entries an error path names; a longer path keeps its first and last entries.
const maxErrorPathEntries = 32

// objectFrame iterates the entries of one dictionary or array in an objectStack.
// A dictionary frame iterates the keys stored at keyStart..end in objectStack.keys; an array frame iterates 0..end.
type objectFrame struct {
	d        types.Dict
	a        types.Array
	keyStart int
	todo     int // next entry to return
	end      int
	pos      int // entry last returned by next
	owner    int // number of the indirect object this frame iterates, 0 for a direct object
}

// objectStack holds the frames of an explicit-stack object graph traversal and the pending keys of their dictionaries.
// Traversals that follow indirect references keep their frames on the heap, so a long reference chain cannot exhaust
// the goroutine stack. Entries of descendants are visited before the following entry of a frame.
type objectStack struct {
	frames []objectFrame
	keys   []string
}

func (s *objectStack) empty() bool {
	return len(s.frames) == 0
}

// pushDict schedules the entries of d whose keys skip does not reject.
func (s *objectStack) pushDict(d types.Dict, skip func(key string) bool) {
	start := len(s.keys)
	for key := range d {
		if skip == nil || !skip(key) {
			s.keys = append(s.keys, key)
		}
	}
	s.frames = append(s.frames, objectFrame{d: d, keyStart: start, todo: start, end: len(s.keys)})
}

// sortTop orders the keys of the dictionary frame on top of the stack.
func (s *objectStack) sortTop() {
	f := s.frames[len(s.frames)-1]
	slices.Sort(s.keys[f.keyStart:f.end])
}

func (s *objectStack) pushArray(a types.Array) {
	s.frames = append(s.frames, objectFrame{a: a, keyStart: len(s.keys), end: len(a)})
}

// setOwner records the indirect object the frame on top of the stack iterates.
func (s *objectStack) setOwner(objNr int) {
	s.frames[len(s.frames)-1].owner = objNr
}

// owner returns the indirect object frame i iterates, or 0 for a direct object.
func (s *objectStack) owner(i int) int {
	return s.frames[i].owner
}

// next returns the next entry of the top frame and the index of that frame.
// If the top frame is done, next pops it and reports false.
func (s *objectStack) next() (int, types.Object, bool) {
	i := len(s.frames) - 1
	f := &s.frames[i]
	if f.todo >= f.end {
		s.keys = s.keys[:f.keyStart]
		s.frames = s.frames[:i]
		return 0, nil, false
	}
	f.pos = f.todo
	f.todo++
	if f.d != nil {
		return i, f.d[s.keys[f.pos]], true
	}
	return i, f.a[f.pos], true
}

// replace replaces the entry last returned by next for frame i.
func (s *objectStack) replace(i int, o types.Object) {
	f := &s.frames[i]
	if f.d != nil {
		f.d[s.keys[f.pos]] = o
		return
	}
	f.a[f.pos] = o
}

// position reports the entry last returned by next for frame i: its key for a dictionary, otherwise its index.
func (s *objectStack) position(i int) (key string, index int, isDict bool) {
	f := &s.frames[i]
	if f.d != nil {
		return s.keys[f.pos], 0, true
	}
	return "", f.pos, false
}

// label describes the entry last returned by next for frame i.
func (s *objectStack) label(i int) string {
	key, index, isDict := s.position(i)
	if isDict {
		return "entry " + key
	}
	return fmt.Sprintf("array[%d]", index)
}

// errorPath joins label(0) .. label(n-1), outermost first, with ": ". It is built once and keeps only the first and
// last entries of a long path, so a failure at the end of a long chain costs linear time and yields a bounded message.
func errorPath(n int, label func(j int) string) string {
	var b strings.Builder
	write := func(j int) {
		if b.Len() > 0 {
			b.WriteString(": ")
		}
		b.WriteString(label(j))
	}
	if n <= maxErrorPathEntries {
		for j := range n {
			write(j)
		}
		return b.String()
	}
	half := maxErrorPathEntries / 2
	for j := range half {
		write(j)
	}
	fmt.Fprintf(&b, ": ... %d more entries ...", n-maxErrorPathEntries)
	for j := n - half; j < n; j++ {
		write(j)
	}
	return b.String()
}
