package model

import "testing"

var dates = []string{"2024-09-18T15:51:13+02:00", "2024-09-18T15:51:13-02:00", "2024-09-18T15:51:13Z", "2024-09-18T15:51+02:00", "2024-09-18T15:51-02:00", "2024-09-18T15:51Z"}

func TestUserDate_UnmarshalXMLDate(t *testing.T) {
	for _, date := range dates {
		_, err := ParseUserDate(date)
		if err != nil {
			t.Errorf("failed to parse user date %s: %s", date, err)
		}
	}
}
