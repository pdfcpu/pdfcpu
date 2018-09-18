/*
Copyright 2018 The pdfcpu Authors.

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

import "testing"

func doTestValidateDateOK(s string, t *testing.T) {

	if Date(s) {
		//t.Logf("validateDate(%s) valid => ok\n", s)
	} else {
		t.Errorf("validateDate(%s) invalid => not ok!\n", s)
	}

}

func doTestValidateDateFail(s string, t *testing.T) {

	if Date(s) {
		t.Errorf("validateDate(%s) valid => not ok!\n", s)
	} else {
		//t.Logf("validateDate(%s) invalid => ok\n", s)
	}

}

func TestValidateDateCommand(t *testing.T) {

	s := "D:2017"
	doTestValidateDateOK(s, t)

	//UTF-8 bytes for UTF-16 string "D:2017"
	s = "\xfe\xff\x00\x44\x00\x3A\x00\x32\x00\x30\x00\x31\x00\x37"
	doTestValidateDateOK(s, t)

	s = "D:201703"
	doTestValidateDateOK(s, t)

	s = "D:20170430"
	doTestValidateDateOK(s, t)

	s = "D:2017043015"
	doTestValidateDateOK(s, t)

	s = "D:201704301559"
	doTestValidateDateOK(s, t)

	s = "D:20170430155901"
	doTestValidateDateOK(s, t)

	s = "D:20170430155901Z"
	doTestValidateDateOK(s, t)

	s = "D:20170430155901+06'"
	doTestValidateDateOK(s, t)

	s = "D:20170430155901+06'59'"
	doTestValidateDateOK(s, t)

	s = "D:20170430155901Z00'"
	doTestValidateDateOK(s, t)

	s = "D:20170430155901Z00'00'"
	doTestValidateDateOK(s, t)

	s = "D:20170430155901+06'59"
	doTestValidateDateFail(s, t)

	s = "D:20170430155901+66'A9'"
	doTestValidateDateFail(s, t)
}
