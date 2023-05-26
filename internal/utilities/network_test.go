/*
Copyright 2023 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/
package utilities

import (
	"testing"
)

func TestIsValidAddress(t *testing.T) {
	tests := []struct {
		name, addr string
		expected   bool
	}{
		{"localhost", "127.0.0.1", true},
		{"all zeros", "0.0.0.0", true},
		{"all 255", "255.255.255.255", true},
		{"empty string", "", false},
		{"random letters", "hello there", false},
		{"random numbers", "1337", false},
		{"incorrect address 1", "127.127.127.127.127", false},
		{"incorrect address 2", "127.127.127.127.", false},
		{"incorrect address 3", "....", false},
		{"incorrect address 4", "01.127.127.127.127", false},
		{"incorrect address 5", "0...0", false},
		{"incorrect address 6", "255.255.255.256", false},
		{"incorrect address 7", "a.b.c.d", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := IsValidAddress(tt.addr)
			if tt.expected != actual {
				t.Errorf("expected: %t actual: %t", tt.expected, actual)
			}
		})
	}
}

func TestIsValidPrefix(t *testing.T) {
	tests := []struct {
		name, addr string
		expected   bool
	}{
		{"localhost", "127.0.0.1/32", true},
		{"all zeros, subnet mask 8", "0.0.0.0/8", true},
		{"all zeros, subnet mask 32", "0.0.0.0/32", true},
		{"empty string", "", false},
		{"random letters", "hello there", false},
		{"random numbers", "1337", false},
		{"incorrect subnet mask 1", "0.0.0.0/", false},
		{"incorrect subnet mask 2", "0.0.0.0/1", false},
		{"incorrect subnet mask 3", "0.0.0.0/7", false},
		{"incorrect subnet mask 4", "0.0.0.0/33", false},
		{"incorrect subnet mask 5", "0.0.0.0/08", false},
		{"incorrect subnet mask 6", "0.0.0.0/a", false},
		{"incorrect subnet mask 7", "a.b.c.d/e", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := IsValidPrefix(tt.addr)
			if tt.expected != actual {
				t.Errorf("expected: %t actual: %t", tt.expected, actual)
			}
		})
	}
}

func TestIsAddressInPrefix(t *testing.T) {
	tests := []struct {
		name, addr, prefix string
		expected           bool
	}{
		{"contains", "127.0.0.1", "127.0.0.1/32", true},
		{"not contains", "127.0.0.2", "127.0.0.1/32", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := IsAddressInPrefix(tt.addr, tt.prefix)
			if tt.expected != actual {
				t.Errorf("expected: %t actual: %t", tt.expected, actual)
			}
		})
	}
}
