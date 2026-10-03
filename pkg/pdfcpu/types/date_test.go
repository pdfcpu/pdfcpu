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

package types

import (
	"testing"
	"time"
)

func doParseDateTimeRelaxedOK(s string, t *testing.T) {
	t.Helper()
	if time, ok := DateTime(s, true); ok {
		_ = time
		//t.Logf("DateTime(%s) valid => %s\n", s, time)
	} else {
		t.Errorf("DateTime(%s) invalid => not ok!\n", s)
	}

}

func doParseDateTimeOK(s string, t *testing.T) {
	t.Helper()
	if time, ok := DateTime(s, false); ok {
		_ = time
		t.Logf("DateTime(%s) valid => %s\n", s, time)
	} else {
		t.Errorf("DateTime(%s) invalid => not ok!\n", s)
	}

}

func doParseDateTimeFail(s string, t *testing.T) {
	t.Helper()
	if time, ok := DateTime(s, false); ok {
		t.Errorf("DateTime(%s) valid => not ok! %s\n", s, time)
	} // else {
	//t.Logf("DateTime(%s) invalid => ok\n", s)
	//}

}

// TestParseDateTime verifies parse date time.
func TestParseDateTime(t *testing.T) {
	// (D:YYYYMMDDHHmmSSOHH'mm)
	// O = -,+,Z

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

	s = "D:20170430155901Z"
	doParseDateTimeOK(s, t)

	s = "D:20170430155901"
	doParseDateTimeOK(s, t)

	s = "D:20170430155901+06'59"
	doParseDateTimeOK(s, t)

	s = "D:20170430155901Z00"
	doParseDateTimeOK(s, t)

	s = "D:20170430155901Z00'00"
	doParseDateTimeOK(s, t)

	s = "D:20210602180254-06"
	doParseDateTimeOK(s, t)

	s = "D:20170430155901+06'"
	doParseDateTimeOK(s, t)

	s = "D:20170430155901+06'59"
	doParseDateTimeOK(s, t)

	s = "D:20210515103719-02'00"
	doParseDateTimeOK(s, t)

	s = "D:2025022513222+01'00'"
	doParseDateTimeOK(s, t)

	s = "D:20170430155901+66'A9"
	doParseDateTimeFail(s, t)

	s = "D:20201222164228Z'"
	doParseDateTimeRelaxedOK(s, t)

	s = "D:20230912144809Z'0"
	doParseDateTimeRelaxedOK(s, t)

	s = "20250718155751+"
	doParseDateTimeRelaxedOK(s, t)

	s = "20141117162446Z00'00'"
	doParseDateTimeRelaxedOK(s, t)

	s = "D: 20210827124448+00'00'"
	doParseDateTimeRelaxedOK(s, t)

	s = "D: 20191003062617-07'00'"
	doParseDateTimeRelaxedOK(s, t)

	s = "D:20150521.124925823"
	doParseDateTimeRelaxedOK(s, t)

	s = "D:20210517043452}"
	doParseDateTimeRelaxedOK(s, t)

	s = "D:20210608122455Z00\\'00"
	doParseDateTimeRelaxedOK(s, t)

	s = "D:20020301230221- 5' 0'"
	doParseDateTimeRelaxedOK(s, t)

	s = "D:20061102145045-05'"
	doParseDateTimeRelaxedOK(s, t)

	s = "D:20150312082530-5'00'"
	doParseDateTimeRelaxedOK(s, t)

	s = "D:20191009100417-05'00''"
	doParseDateTimeRelaxedOK(s, t)

	s = "D:20200429084309+ 0' 0'"
	doParseDateTimeRelaxedOK(s, t)

	s = "D:20211028112621--04'00"
	doParseDateTimeRelaxedOK(s, t)

	s = "D:20210419150333-04'00'Z"
	doParseDateTimeRelaxedOK(s, t)

	s = "\357\273\277D:20160404061414+65'53'"
	doParseDateTimeRelaxedOK(s, t)
}

// TestDateTimeTimezone verifies strict timezone validation and relaxed compatibility.
func TestDateTimeTimezone(t *testing.T) {
	for _, tt := range []struct {
		name    string
		suffix  string
		strict  bool
		relaxed bool
		offset  int
	}{
		{"terminal Z", "Z", true, true, 0},
		{"Z hours", "Z00", true, true, 0},
		{"Z hours and minutes", "Z00'00", true, true, 0},
		{"Z full offset", "Z00'00'", true, true, 0},
		{"no timezone", "", true, true, 0},
		{"positive offset", "+02'00'", true, true, 2 * 60 * 60},
		{"negative offset", "-02'00'", true, true, -2 * 60 * 60},
		{"terminal plus", "+", false, true, 0},
		{"terminal minus", "-", false, true, 0},
		{"Z apostrophe", "Z'", false, true, 0},
		{"Z apostrophe and zero", "Z'0", false, true, 0},
		{"Z nonzero hours", "Z01", false, false, 0},
		{"Z nonzero minutes", "Z00'01'", false, false, 0},
	} {
		t.Run(tt.name, func(t *testing.T) {
			for _, mode := range []struct {
				name    string
				relaxed bool
				wantOK  bool
			}{
				{"strict", false, tt.strict},
				{"relaxed", true, tt.relaxed},
			} {
				t.Run(mode.name, func(t *testing.T) {
					got, ok := DateTime("D:20260713095358"+tt.suffix, mode.relaxed)
					if ok != mode.wantOK {
						t.Fatalf("DateTime success = %t, want %t", ok, mode.wantOK)
					}
					if !ok {
						return
					}
					want := time.Date(2026, time.July, 13, 9, 53, 58, 0, time.FixedZone("", tt.offset))
					_, offset := got.Zone()
					if !got.Equal(want) || offset != tt.offset {
						t.Errorf("DateTime = %v (offset %d), want %v (offset %d)", got, offset, want, tt.offset)
					}
				})
			}
		})
	}
}

// TestWriteDateTime verifies write date time.
func TestWriteDateTime(t *testing.T) {

	now := DateString(time.Now())
	doParseDateTimeOK(now, t)

	loc, _ := time.LoadLocation("Europe/Vienna")
	now = DateString(time.Now().In(loc))
	doParseDateTimeOK(now, t)

	loc, _ = time.LoadLocation("Pacific/Honolulu")
	now = DateString(time.Now().In(loc))
	doParseDateTimeOK(now, t)

	loc, _ = time.LoadLocation("Australia/Sydney")
	now = DateString(time.Now().In(loc))
	doParseDateTimeOK(now, t)
}
