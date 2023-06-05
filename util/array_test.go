package util

import "testing"

func TestArrayNotInArray(t *testing.T) {
	original := []string{"123", "456", "789"}
	search := []string{"123", "4561"}
	result := ArrayNotInArrayString(original, search)
	CDump(result)
}
