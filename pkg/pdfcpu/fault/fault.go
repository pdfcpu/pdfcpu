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

package fault

import (
	"fmt"
	"runtime/debug"
)

// Panic wraps an error with the stack trace captured at the moment of failure.
type Panic struct {
	Err   error
	Stack []byte
}

// Error implements the error interface.
func (p Panic) Error() string {
	return p.Err.Error()
}

// Unwrap allows standard library errors.Is/As to work.
func (p Panic) Unwrap() error {
	return p.Err
}

// Fail triggers a panic with a formatted message and a fresh stack trace.
func Fail(format string, args ...interface{}) {
	panic(Panic{
		// Use %w if you want to allow wrapping other errors passed in args
		Err:   fmt.Errorf(format, args...),
		Stack: debug.Stack(),
	})
}

// Catch recovers from a fault.Panic and assigns it to the provided error pointer.
// It re-panics if the recovered value is not a fault.Panic.
func Catch(err *error) {
	if r := recover(); r != nil {
		if p, ok := r.(Panic); ok {
			*err = p
			//fmt.Printf("recovering from internal panic:\n%s\n", p.Stack)
		} else {
			panic(r)
		}
	}
}
