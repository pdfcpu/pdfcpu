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

package cli

import (
	"errors"
	"strings"
	"testing"

	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/fault"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

func TestDispatchRecoversUnexpectedPanicWithStackMetadata(t *testing.T) {
	mode := model.CommandMode(-1)
	dispatchTable[mode] = func(*Command) ([]string, error) {
		panic("boom")
	}
	defer delete(dispatchTable, mode)

	_, err := Dispatch(&Command{Mode: mode, Conf: model.NewDefaultConfiguration()})
	if err == nil {
		t.Fatal("expected error")
	}

	var p fault.Panic
	if !errors.As(err, &p) {
		t.Fatalf("got %T, want fault.Panic", err)
	}
	if got := err.Error(); !strings.Contains(got, "unexpected panic attack: boom") {
		t.Fatalf("got %q, want unexpected panic message", got)
	}
	if strings.Contains(err.Error(), "goroutine ") {
		t.Fatalf("error string includes stack trace: %q", err.Error())
	}
	if !strings.Contains(string(p.Stack), "TestDispatchRecoversUnexpectedPanicWithStackMetadata") {
		t.Fatalf("stack trace does not include test frame:\n%s", p.Stack)
	}
}

// TestDispatchRejectsAddSignature verifies the unimplemented command mode cannot report false success.
func TestDispatchRejectsAddSignature(t *testing.T) {
	_, err := Dispatch(&Command{Mode: model.ADDSIGNATURE})
	if !errors.Is(err, ErrUnsupportedCommandMode) {
		t.Fatalf("expected unsupported command mode, got %v", err)
	}
	if !strings.Contains(err.Error(), "mode") {
		t.Fatalf("expected command mode context, got %v", err)
	}
}

// TestDispatchUsesOperationOwnedConfiguration verifies successful execution cannot mutate caller-owned command state.
func TestDispatchUsesOperationOwnedConfiguration(t *testing.T) {
	mode := model.CommandMode(-2)
	userPWNew := "new-user"
	ownerPWNew := "new-owner"
	conf := &model.Configuration{
		Cmd:                    model.OPTIMIZE,
		UserPWNew:              &userPWNew,
		OwnerPWNew:             &ownerPWNew,
		AllowedRevocationHosts: []string{"ocsp.example.corp"},
	}
	cmd := &Command{Mode: mode, StringVal: "caller", Conf: conf}

	var executionCommand *Command
	dispatchTable[mode] = func(exec *Command) ([]string, error) {
		executionCommand = exec
		exec.StringVal = "execution"
		*exec.Conf.UserPWNew = "execution-user"
		*exec.Conf.OwnerPWNew = "execution-owner"
		exec.Conf.AllowedRevocationHosts[0] = "execution.example.corp"
		return []string{"ok"}, nil
	}
	defer delete(dispatchTable, mode)

	out, err := Dispatch(cmd)
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0] != "ok" {
		t.Fatalf("output: got %v, want [ok]", out)
	}
	if executionCommand == cmd {
		t.Fatal("dispatch executed the caller's command")
	}
	if executionCommand.Conf == conf {
		t.Fatal("dispatch executed with caller-owned configuration")
	}
	if executionCommand.Conf.Cmd != mode {
		t.Fatalf("execution command mode: got %d, want %d", executionCommand.Conf.Cmd, mode)
	}
	if cmd.StringVal != "caller" {
		t.Fatalf("caller command value: got %q, want caller", cmd.StringVal)
	}
	if cmd.Conf != conf {
		t.Fatal("dispatch replaced caller command configuration")
	}
	if conf.Cmd != model.OPTIMIZE {
		t.Fatalf("caller configuration mode: got %d, want %d", conf.Cmd, model.OPTIMIZE)
	}
	if got, want := *conf.UserPWNew, "new-user"; got != want {
		t.Fatalf("caller new user password: got %q, want %q", got, want)
	}
	if got, want := *conf.OwnerPWNew, "new-owner"; got != want {
		t.Fatalf("caller new owner password: got %q, want %q", got, want)
	}
	if got, want := conf.AllowedRevocationHosts[0], "ocsp.example.corp"; got != want {
		t.Fatalf("caller allowed revocation host: got %q, want %q", got, want)
	}
}

// TestDispatchFailurePreservesCallerState verifies unsupported commands do not mutate caller-owned state.
func TestDispatchFailurePreservesCallerState(t *testing.T) {
	mode := model.CommandMode(-3)
	conf := &model.Configuration{Cmd: model.VALIDATE}
	cmd := &Command{Mode: mode, Conf: conf}

	_, err := Dispatch(cmd)
	if !errors.Is(err, ErrUnsupportedCommandMode) {
		t.Fatalf("expected unsupported command mode, got %v", err)
	}
	if cmd.Conf != conf {
		t.Fatal("dispatch replaced caller command configuration")
	}
	if conf.Cmd != model.VALIDATE {
		t.Fatalf("caller configuration mode: got %d, want %d", conf.Cmd, model.VALIDATE)
	}
}
