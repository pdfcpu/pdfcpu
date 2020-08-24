/*
Copyright 2020 The pdfcpu Authors.

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

import "testing"

func doParseDateTimeOK(s string, t *testing.T) {
	t.Helper()
	if time, ok := DateTime(s); ok {
		t.Logf("DateTime(%s) valid => %s\n", s, time)
	} else {
		t.Errorf("DateTime(%s) invalid => not ok!\n", s)
	}

}

func doParseDateTimeFail(s string, t *testing.T) {
	t.Helper()
	if time, ok := DateTime(s); ok {
		t.Errorf("DateTime(%s) valid => not ok! %s\n", s, time)
	} else {
		t.Logf("DateTime(%s) invalid => ok\n", s)
	}

}

func TestDateTime(t *testing.T) {

	s := "D:2017"
	doParseDateTimeOK(s, t)

	//UTF-8 bytes for UTF-16 string "D:2017"
	s = "\xfe\xff\x00\x44\x00\x3A\x00\x32\x00\x30\x00\x31\x00\x37"
	doParseDateTimeOK(s, t)

	s = "D:201703"
	doParseDateTimeOK(s, t)

	s = "D:20170430"
	doParseDateTimeOK(s, t)

	s = "D:2017043015"
	doParseDateTimeOK(s, t)

	s = "D:201704301559"
	doParseDateTimeOK(s, t)

	s = "D:20170430155901"
	doParseDateTimeOK(s, t)

	s = "D:20170430155901Z"
	doParseDateTimeOK(s, t)

	s = "D:20170430155901+06'"
	doParseDateTimeOK(s, t)

	s = "D:20170430155901+06'59'"
	doParseDateTimeOK(s, t)

	s = "D:20170430155901Z00'"
	doParseDateTimeOK(s, t)

	s = "D:20170430155901Z00'00'"
	doParseDateTimeOK(s, t)

	s = "D:20170430155901+06'59"
	doParseDateTimeFail(s, t)

	s = "D:20170430155901+66'A9'"
	doParseDateTimeFail(s, t)
}
