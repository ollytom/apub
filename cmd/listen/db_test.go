package main

import "testing"

func TestSelectExpr(t *testing.T) {
	columns := []string{"from", "to", "date", "subject"}
	want := "IN (?, ?, ?, ?)"
	got := sqlInExpr(len(columns))
	if want != got {
		t.Errorf("want %s, got %s", want, got)
	}
}
