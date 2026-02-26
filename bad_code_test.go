package main

import "testing"

func TestReverseString(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "olleh"},
		{"world", "dlrow"},
		{"", ""},
		{"a", "a"},
	}

	for _, tt := range tests {
		result := ReverseString(tt.input)
		if result != tt.expected {
			t.Errorf("ReverseString(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

func TestAdd(t *testing.T) {
	tests := []struct {
		a, b     int
		expected int
	}{
		{1, 2, 3},
		{0, 0, 0},
		{-1, 1, 0},
		{100, 200, 300},
	}

	for _, tt := range tests {
		result := Add(tt.a, tt.b)
		if result != tt.expected {
			t.Errorf("Add(%d, %d) = %d, expected %d", tt.a, tt.b, result, tt.expected)
		}
	}
}
