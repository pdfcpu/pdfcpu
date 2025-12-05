/*
Copyright 2025 The pdfcpu Authors.

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

package model

import "testing"

func TestWordWrap(t *testing.T) {
	testcases := []struct {
		FontName       string
		FontSize       float64
		MaxWidthPoints float64
		Text           string
		Want           []string
	}{
		{"Helvetica", 12, 10,
			"",
			[]string{""},
		},

		{"Helvetica", 12, 10,
			"   ",
			[]string{""},
		},

		{"Helvetica", 12, 10,
			"           ",
			[]string{""},
		},

		{"Helvetica", 12, 10,
			"      Indent line",
			[]string{"      ", "Indent", "line"},
		},

		{"Helvetica", 12, 60,
			"\tTab Indent line",
			[]string{"\tTab", "Indent line"},
		},

		{"Helvetica", 12, 200,
			"\tLong tab Indent line",
			[]string{"\tLong tab Indent line"},
		},

		{"Helvetica", 12, 20,
			"Lorem ipsum dolor sit amet, consectetur adipiscing elit.",
			[]string{"Lorem", "ipsum", "dolor", "sit", "amet,", "consectetur", "adipiscing", "elit."},
		},

		{"Helvetica", 12, 50,
			"Lorem ipsum dolor sit amet, consectetur adipiscing elit.",
			[]string{"Lorem", "ipsum", "dolor sit", "amet,", "consectetur", "adipiscing", "elit."},
		},

		{"Helvetica", 12, 70,
			"Lorem ipsum dolor sit amet, consectetur adipiscing elit.",
			[]string{"Lorem ipsum", "dolor sit", "amet,", "consectetur", "adipiscing", "elit."},
		},

		{"Courier", 24, 70,
			"Lorem ipsum dolor sit amet, consectetur adipiscing elit.",
			[]string{"Lorem", "ipsum", "dolor", "sit", "amet,", "consectetur", "adipiscing", "elit."},
		},

		{"Helvetica", 12, 100,
			"Lorem ipsum dolor sit amet, consectetur adipiscing elit.",
			[]string{"Lorem ipsum dolor", "sit amet,", "consectetur", "adipiscing elit."},
		},

		{"Helvetica", 12, 200,
			"Lorem ipsum dolor sit amet, consectetur adipiscing elit.",
			[]string{"Lorem ipsum dolor sit amet,", "consectetur adipiscing elit."},
		},

		{"Helvetica", 12, 500,
			"Lorem ipsum dolor sit amet, consectetur adipiscing elit.",
			[]string{"Lorem ipsum dolor sit amet, consectetur adipiscing elit."},
		},

		{"Helvetica", 12, 100,
			"Lorem ipsum\ndolor sit amet,\n consectetur adipiscing\nelit.",
			[]string{"Lorem ipsum", "dolor sit amet,", " consectetur", "adipiscing", "elit."},
		},

		{"Helvetica", 12, 100,
			"Lorem ipsum dolor sit amet,\t\t\t consectetur\nadipiscing\telit.",
			[]string{"Lorem ipsum dolor", "sit amet,", "consectetur", "adipiscing\telit."},
		},

		{"Helvetica", 12, 100,
			"Lorem ipsum dolor sit amet,\n\tconsectetur adipiscing elit.",
			[]string{"Lorem ipsum dolor", "sit amet,", "\tconsectetur", "adipiscing elit."},
		},

		{"Helvetica", 12, 100,
			"Lorem ipsum dolor sit amet, consectetur\nadipiscing elit.",
			[]string{"Lorem ipsum dolor", "sit amet,", "consectetur", "adipiscing elit."},
		},
	}

	for _, tc := range testcases {
		gotLines := WordWrap(tc.Text, tc.FontName, int(tc.FontSize), tc.MaxWidthPoints)
		if len(gotLines) != len(tc.Want) {
			t.Errorf("expected %d lines when wrapping %s, got %d", len(tc.Want), tc.Text, len(gotLines))
			continue
		}
		for i, s := range gotLines {
			if s != tc.Want[i] {
				t.Errorf("expected %s when wrapping %s, got %s", tc.Want[i], tc.Text, s)
			}
		}
	}
}
