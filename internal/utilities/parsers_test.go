/*
Copyright 2023 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/
package utilities

import (
	"strconv"
	"testing"
)

func TestConvertDataUnitsToInt64_validInputs(t *testing.T) {
	tests := []struct {
		number         string
		factor         int
		expectedNumber int64
	}{
		{"0", 1024, 0},
		{"1", 1024, 1},
		{"1K", 1024, 1024},
		{"124M", 1024, 124 * 1024 * 1024},
		{"10G", 1024, 10 * 1024 * 1024 * 1024},
		{"12T", 1024, 12 * 1024 * 1024 * 1024 * 1024},
		{"1P", 1024, 1 * 1024 * 1024 * 1024 * 1024 * 1024},
		{"4P", 1024, 4 * 1024 * 1024 * 1024 * 1024 * 1024},
		{"10P", 1024, 10 * 1024 * 1024 * 1024 * 1024 * 1024},
		{"0", 1000, 0},
		{"1", 1000, 1},
		{"1K", 1000, 1000},
		{"124M", 1000, 124 * 1000 * 1000},
		{"10G", 1000, 10 * 1000 * 1000 * 1000},
		{"12T", 1000, 12 * 1000 * 1000 * 1000 * 1000},
		{"1P", 1000, 1 * 1000 * 1000 * 1000 * 1000 * 1000},
		{"4P", 1000, 4 * 1000 * 1000 * 1000 * 1000 * 1000},
		{"11P", 1000, 11 * 1000 * 1000 * 1000 * 1000 * 1000},
	}

	for i, tt := range tests {
		t.Run("Test_"+strconv.Itoa(i), func(t *testing.T) {
			actual, err := ConvertDataUnitsToInt64(tt.number, tt.factor)
			if tt.expectedNumber != actual || err != nil {
				t.Errorf("expected: %d actual: %d. expected error nil actual: %s",
					tt.expectedNumber, actual, err)
			}
		})
	}

}

func TestConvertDataUnitsToInt64_invalidInputs(t *testing.T) {
	tests := []struct {
		number string
		factor int
	}{
		{"", 1024},
		{"00", 1024},
		{"01", 1024},
		{"1 K", 1024},
		{"M124", 1024},
		{"10G1", 1024},
		{"12TA", 1024},
		{"1PA", 1024},
		{"4PP", 1024},
		{"10PG", 1024},
		{"010G", 1000},
		{"111-1", 10001},
		{"-1K", 1000},
		{"K", 1000},
		{"0000KK", 1000},
		{"999999G9", 1000},
	}

	for i, tt := range tests {
		t.Run("Test_"+strconv.Itoa(i), func(t *testing.T) {
			_, err := ConvertDataUnitsToInt64(tt.number, tt.factor)
			if err != errInvalidDataUnitFormat {
				t.Errorf(" expected error %s actual: %s", errInvalidDataUnitFormat, err)
			}
		})
	}

}
