package model

import "testing"

func TestWordWrap(t *testing.T) {
	testcases := []struct {
		Wrap     string
		FontName string
		FontSize float64
		Width    float64
		Expected []string
	}{
		{"", "Helvetica", 12, 10, []string{""}},

		{"   ", "Helvetica", 12, 10, []string{""}},

		{"           ", "Helvetica", 12, 10, []string{""}},

		{"      Indent line", "Helvetica", 12, 10,
			[]string{"      ", "Indent", "line"}},

		{"\tTab Indent line", "Helvetica", 12, 60,
			[]string{"\tTab", "Indent line"}},

		{"\tLong tab Indent line", "Helvetica", 12, 200,
			[]string{"\tLong tab Indent line"}},

		{"Lorem ipsum dolor sit amet, consectetur adipiscing elit.", "Helvetica", 12, 20,
			[]string{"Lorem", "ipsum", "dolor", "sit", "amet,", "consectetur", "adipiscing", "elit."}},

		{"Lorem ipsum dolor sit amet, consectetur adipiscing elit.", "Helvetica", 12, 50,
			[]string{"Lorem", "ipsum", "dolor sit", "amet,", "consectetur", "adipiscing", "elit."}},

		{"Lorem ipsum dolor sit amet, consectetur adipiscing elit.", "Helvetica", 12, 70,
			[]string{"Lorem ipsum", "dolor sit", "amet,", "consectetur", "adipiscing", "elit."}},

		{"Lorem ipsum dolor sit amet, consectetur adipiscing elit.", "Courier", 24, 70,
			[]string{"Lorem", "ipsum", "dolor", "sit", "amet,", "consectetur", "adipiscing", "elit."}},

		{"Lorem ipsum dolor sit amet, consectetur adipiscing elit.", "Helvetica", 12, 100,
			[]string{"Lorem ipsum dolor", "sit amet,", "consectetur", "adipiscing elit."}},

		{"Lorem ipsum dolor sit amet, consectetur adipiscing elit.", "Helvetica", 12, 200,
			[]string{"Lorem ipsum dolor sit amet,", "consectetur adipiscing elit."}},

		{"Lorem ipsum dolor sit amet, consectetur adipiscing elit.", "Helvetica", 12, 500,
			[]string{"Lorem ipsum dolor sit amet, consectetur adipiscing elit."}},

		{"Lorem ipsum\ndolor sit amet,\n consectetur adipiscing\nelit.", "Helvetica", 12, 100,
			[]string{"Lorem ipsum", "dolor sit amet,", " consectetur", "adipiscing", "elit."}},

		{"Lorem ipsum dolor sit amet,\t\t\t consectetur\nadipiscing\telit.", "Helvetica", 12, 100,
			[]string{"Lorem ipsum dolor", "sit amet,", "consectetur", "adipiscing\telit."}},

		{"Lorem ipsum dolor sit amet,\n\tconsectetur adipiscing elit.", "Helvetica", 12, 100,
			[]string{"Lorem ipsum dolor", "sit amet,", "\tconsectetur", "adipiscing elit."}},

		{"Lorem ipsum dolor sit amet, consectetur\nadipiscing elit.", "Helvetica", 12, 100,
			[]string{"Lorem ipsum dolor", "sit amet,", "consectetur", "adipiscing elit."}},
	}
	for _, tc := range testcases {
		wrapped := WordWrap(tc.Wrap, tc.FontName, int(tc.FontSize), tc.Width)
		if len(wrapped) != len(tc.Expected) {
			t.Errorf("expected %d lines when wrapping %s, got %d", len(tc.Expected), tc.Wrap, len(wrapped))
			continue
		}
		for i, line := range wrapped {
			if line != tc.Expected[i] {
				t.Errorf("expected %s when wrapping %s, got %s", tc.Expected[i], tc.Wrap, line)
			}
		}
	}
}
