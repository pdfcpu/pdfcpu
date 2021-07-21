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

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// DateString returns a string representation of t.
func DateString(t time.Time) string {
	_, tz := t.Zone()
	tzm := tz / 60
	sign := "+"
	if tzm < 0 {
		sign = "-"
		tzm = -tzm
	}

	return fmt.Sprintf("D:%d%02d%02d%02d%02d%02d%s%02d'%02d'",
		t.Year(), t.Month(), t.Day(),
		t.Hour(), t.Minute(), t.Second(),
		sign, tzm/60, tzm%60)
}

func prevalidateDate(s string, relaxed bool) (string, bool) {
	// utf16 conversion if applicable.
	if IsStringUTF16BE(s) {
		utf16s, err := DecodeUTF16String(s)
		if err != nil {
			return "", false
		}
		s = utf16s
	}

	// Remove trailing 0x00
	s = strings.TrimRight(s, "\x00")

	if relaxed {
		// Accept missing "D:" prefix.
		// "YYYY" is mandatory
		s = strings.TrimPrefix(s, "D:")
		return s, len(s) >= 4
	}

	// "D:YYYY" is mandatory
	if len(s) < 6 {
		return "", false
	}

	return s[2:], strings.HasPrefix(s, "D:")
}

func parseTimezoneHours(s string, o byte) (int, bool) {
	tzhours := s[15:17]
	tzh, err := strconv.Atoi(tzhours)
	if err != nil {
		return 0, false
	}

	if tzh > 23 {
		return 0, false
	}

	if o == 'Z' && tzh != 0 {
		return 0, false
	}

	return tzh, true
}

func parseTimezoneMinutes(s string, o byte) (int, bool) {

	if s[17] != '\'' {
		return 0, false
	}

	tzmin := s[18:20]
	tzm, err := strconv.Atoi(tzmin)
	if err != nil {
		return 0, false
	}

	if tzm > 59 {
		return 0, false
	}

	if o == 'Z' && tzm != 0 {
		return 0, false
	}

	// "YYYYMMDDHHmmSSZHH'mm"
	if len(s) == 20 {
		return 0, true
	}

	// Accept a trailing '
	return tzm, s[20] == '\''
}

func validateTimezoneSeparator(c byte) bool {
	return c == '+' || c == '-' || c == 'Z'
}

func parseTimezone(s string, relaxed bool) (h, m int, ok bool) {

	o := s[14]

	if !validateTimezoneSeparator(o) {
		return 0, 0, false
	}

	// local time equal to UT.
	// "YYYYMMDDHHmmSSZ" or
	// "YYYYMMDDHHmmSSZ'" if relaxed
	if o == 'Z' && (len(s) == 15 || (relaxed && len(s) == 16 && s[15] == '\'')) {
		return 0, 0, true
	}

	// if len(s) < 18 {
	// 	return 0, 0, false
	// }

	neg := o == '-'

	tzh, ok := parseTimezoneHours(s, o)
	if !ok {
		return 0, 0, false
	}

	if neg {
		tzh *= -1
	}

	// "YYYYMMDDHHmmSSZHH" or
	// "YYYYMMDDHHmmSSZHH'" if relaxed
	if len(s) == 17 || (relaxed && len(s) == 18 && s[17] == '\'') {
		return tzh, 0, true
	}

	if len(s) != 20 && len(s) != 21 {
		return 0, 0, false
	}

	tzm, ok := parseTimezoneMinutes(s, o)
	if !ok {
		return 0, 0, false
	}

	return tzh, tzm, true
}

func parseYear(s string) (y int, finished, ok bool) {
	year := s[0:4]

	y, err := strconv.Atoi(year)
	if err != nil {
		return 0, false, false
	}

	// "YYYY"
	if len(s) == 4 {
		return y, true, true
	}

	if len(s) == 5 {
		return 0, false, false
	}

	return y, false, true
}

func parseMonth(s string) (m int, finished, ok bool) {
	month := s[4:6]

	var err error
	m, err = strconv.Atoi(month)
	if err != nil {
		return 0, false, false
	}

	if m < 1 || m > 12 {
		return 0, false, false
	}

	// "YYYYMM"
	if len(s) == 6 {
		return m, true, true
	}

	if len(s) == 7 {
		return 0, false, false
	}

	return m, false, true
}

func parseDay(s string, y, m int) (d int, finished, ok bool) {
	day := s[6:8]

	d, err := strconv.Atoi(day)
	if err != nil {
		return 0, false, false
	}

	if d < 1 || d > 31 {
		return 0, false, false
	}

	// check valid Date(year,month,day)
	// The day before the first day of next month:
	t := time.Date(y, time.Month(m+1), 0, 0, 0, 0, 0, time.UTC)
	if d > t.Day() {
		return 0, false, false
	}

	// "YYYYMMDD"
	if len(s) == 8 {
		return d, true, true
	}

	if len(s) == 9 {
		return 0, false, false
	}

	return d, false, true
}

func parseHour(s string) (h int, finished, ok bool) {
	hour := s[8:10]

	h, err := strconv.Atoi(hour)
	if err != nil {
		return 0, false, false
	}

	if h > 23 {
		return 0, false, false
	}

	// "YYYYMMDDHH"
	if len(s) == 10 {
		return h, true, true
	}

	if len(s) == 11 {
		return 0, false, false
	}

	return h, false, true
}

func parseMinute(s string) (min int, finished, ok bool) {
	minute := s[10:12]

	min, err := strconv.Atoi(minute)
	if err != nil {
		return 0, false, false
	}

	if min > 59 {
		return 0, false, false
	}

	// "YYYYMMDDHHmm"
	if len(s) == 12 {
		return min, true, true
	}

	if len(s) == 13 {
		return 0, false, false
	}

	return min, false, true
}

func parseSecond(s string) (sec int, finished, ok bool) {
	second := s[12:14]

	sec, err := strconv.Atoi(second)
	if err != nil {
		return 0, false, false
	}

	if sec > 59 {
		return 0, false, false
	}

	// "YYYYMMDDHHmmSS"
	if len(s) == 14 {
		return sec, true, true
	}

	return sec, false, true
}

func digestPopularOutOfSpecDates(s string) (time.Time, bool) {

	// Mon Jan 2 15:04:05 2006
	// Monday, January 02, 2006 3:04:05 PM
	// 1/2/2006 15:04:05
	// Mon, Jan 2, 2006

	t, err := time.Parse("Mon Jan 2 15:04:05 2006", s)
	if err == nil {
		return t, true
	}

	t, err = time.Parse("Monday, January 02, 2006 3:04:05 PM", s)
	if err == nil {
		return t, true
	}

	t, err = time.Parse("1/2/2006 15:04:05", s)
	if err == nil {
		return t, true
	}

	t, err = time.Parse("Mon, Jan 2, 2006", s)
	if err == nil {
		return t, true
	}

	return t, false
}

// DateTime decodes s into a time.Time.
func DateTime(s string, relaxed bool) (time.Time, bool) {
	// 7.9.4 Dates
	// (D:YYYYMMDDHHmmSSOHH'mm')

	var d time.Time

	var ok bool
	s, ok = prevalidateDate(s, relaxed)
	if !ok {
		return d, false
	}

	y, finished, ok := parseYear(s)
	if !ok {
		// Try workaround
		return digestPopularOutOfSpecDates(s)
	}

	// Construct time for yyyy 01 01 00:00:00
	d = time.Date(y, 1, 1, 0, 0, 0, 0, time.UTC)
	if finished {
		return d, true
	}

	m, finished, ok := parseMonth(s)
	if !ok {
		return d, false
	}

	d = d.AddDate(0, m-1, 0)
	if finished {
		return d, true
	}

	day, finished, ok := parseDay(s, y, m)
	if !ok {
		return d, false
	}

	d = d.AddDate(0, 0, day-1)
	if finished {
		return d, true
	}

	h, finished, ok := parseHour(s)
	if !ok {
		return d, false
	}

	d = d.Add(time.Duration(h) * time.Hour)
	if finished {
		return d, true
	}

	min, finished, ok := parseMinute(s)
	if !ok {
		return d, false
	}

	d = d.Add(time.Duration(min) * time.Minute)
	if finished {
		return d, true
	}

	sec, finished, ok := parseSecond(s)
	if !ok {
		return d, false
	}

	d = d.Add(time.Duration(sec) * time.Second)
	if finished {
		return d, true
	}

	// Process timezone
	tzh, tzm, ok := parseTimezone(s, relaxed)
	if !ok {
		return d, false
	}

	loc := time.FixedZone("", tzh*60*60+tzm*60)
	d = time.Date(y, time.Month(m), day, h, min, sec, 0, loc)

	return d, true
}
