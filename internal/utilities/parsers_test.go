/*
Copyright 2023 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/
package utilities

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/assert"
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

func TestParseAndValidateSelfLink_success(t *testing.T) {
	groupNames := []string{
		"tenants",
		"tenant-spaces",
		"volumes",
	}

	expectedTenant, expectedTenantSpace, expectedVolume := "a", "b", "c"
	selfLink := "/tenants/a/tenant-spaces/b/volumes/c"

	parsedSelfLink, err := ParseSelfLink(selfLink, groupNames)
	assert.Nil(t, err)
	assert.Equal(t, expectedTenant, parsedSelfLink["tenants"])
	assert.Equal(t, expectedTenantSpace, parsedSelfLink["tenant-spaces"])
	assert.Equal(t, expectedVolume, parsedSelfLink["volumes"])
}

func TestParseAndValidateSelfLink_failure(t *testing.T) {
	expectedErr := "self link has incorrect format"

	groupNames := []string{
		"tenants",
		"tenant-spaces",
		"volumes",
	}

	tests := []struct {
		name string
		id   string
	}{
		{"empty string", ""},
		{"slash only", "/"},
		{"only tenants", "/tenants/"},
		{"only tenants with value", "/tenants/abc"},
		{"no tenant space value", "/tenants/a/tenant-spaces//volumes/c"},
		{"no volume value", "/tenants/a/tenant-spaces/b/volumes/"},
		{"no tenant value", "/tenants//tenant-spaces/b/volumes/c"},
		{"switched tenant space and tenants", "/tenant-spaces/a/tenants/b/volumes/c"},
		{"no leading slash", "tenant-spaces/a/tenants/b/volumes/c"},
		{"trailing slash", "/tenant-spaces/a/tenants/b/volumes/c/"},
		{"redudant string on the start of path", "a/tenant-spaces/a/tenants/b/volumes/c"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			parsedSelfLink, err := ParseSelfLink(tt.id, groupNames)
			assert.Nil(t, parsedSelfLink)
			assert.EqualError(t, err, expectedErr)
		})
	}
}
