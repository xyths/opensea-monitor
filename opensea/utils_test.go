package opensea

import "testing"

func TestBeijingTime(t *testing.T) {
	tests := []string{
		"2021-08-15T05:34:52.669499",
		"2021-08-15T08:00:37.309132",
		"2021-08-28T09:44:43.664713",
	}
	for _, tt := range tests {
		t.Logf("%s => %s", tt, toBeijingTime(tt))
	}
}
