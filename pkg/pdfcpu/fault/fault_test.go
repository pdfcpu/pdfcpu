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
	"errors"
	"strings"
	"testing"
)

func TestPanicErrorOmitsStackTrace(t *testing.T) {
	baseErr := errors.New("boom")
	err := Panic{
		Err:   baseErr,
		Stack: []byte("goroutine 1 [running]:\nstack frame"),
	}

	if got := err.Error(); got != baseErr.Error() {
		t.Fatalf("got %q, want %q", got, baseErr.Error())
	}
	if strings.Contains(err.Error(), "Stack Trace") {
		t.Fatalf("error string includes stack trace: %q", err.Error())
	}
	if !errors.Is(err, baseErr) {
		t.Fatalf("errors.Is failed for wrapped error")
	}
}
