/*
Copyright 2023 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package utilities

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseHumanReadableTimePeriodIntoMinutes_success(t *testing.T) {
	tests := []struct {
		name, timePeriod string
		expected         int
	}{
		{"only numbers 1", "123", 123},
		{"only numbers 2", "56789", 56789},
		{"only numbers 3", "110011", 110011},
		{"only numbers 4", "0", 0},
		{"555 minutes", "555M", 555},
		{"2 days", "2D", 2880},
		{"400 days", "400D", 576000},
		{"3 weeks", "3W", 30240},
		{"35 weeks", "35W", 352800},
		{"1 year", "1Y", 525600},
		{"5 years", "5Y", 525600 * 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := ParseHumanReadableTimePeriodIntoMinutes(tt.timePeriod)
			assert.Nil(t, err)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestParseHumanReadableTimePeriodIntoMinutes_failure(t *testing.T) {
	tests := []struct {
		name, timePeriod string
	}{
		{"empty string", ""},
		{"only letters", "Y"},
		{"wrong numbers 1", "0112"},
		{"wrong numbers 2", "00001000"},
		{"wrong numbers 3", "0.1"},
		{"wrong numbers 4", "0.100"},
		{"wrong numbers 5", "200.0"},
		{"mixed letters and numbers 1", "123M567"},
		{"mixed letters and numbers 2", "M123"},
		{"mixed letters and numbers 3", "0001M"},
		{"mixed letters and numbers 4", "0001000M"},
		{"multiple letters 1", "123MY"},
		{"multiple letters 2", "123YM"},
		{"multiple letters 3", "123YMHD"},
		{"invalid chars 1", "1 M"},
		{"invalid chars 2", "1-M"},
		{"invalid chars 3", "1_M"},
		{"invalid chars 4", "1M "},
		{"invalid chars 5", "1M_"},
		{"invalid chars 6", "1Q"},
		{"invalid chars 7", "2C"},
		{"invalid chars 8", "3F"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := ParseHumanReadableTimePeriodIntoMinutes(tt.timePeriod)
			assert.EqualError(t, err, "wrong format, expected human-readable time period (e.g. 2d, 3w)")
			assert.Equal(t, 0, actual)
		})
	}
}

func TestMinutesToStringISO8601(t *testing.T) {
	expected := "PT100M"
	actual := MinutesToStringISO8601(100)
	assert.Equal(t, expected, actual)
}

func TestStringISO8601MinutesToInt(t *testing.T) {
	tests := []struct {
		name, input string
		expected    int
		shouldFail  bool
	}{
		{"10101 minutes", "PT10101M", 10101, false},
		{"no numbers", "PTM", 0, true},
		{"only numbers", "10101", 0, true},
		{"no minutes", "PR10101", 0, true},
		{"no PT", "10101M", 0, true},
		{"wrong letters 1", "PT10MY", 0, true},
		{"wrong letters 2", "PT10Y", 0, true},
		{"wrong numbers 1", "PT010101M", 0, true},
		{"wrong numbers 2", "0PT10M", 0, true},
		{"wrong numbers 3", "10PT10M", 0, true},
		{"wrong numbers 4", "PT10M0", 0, true},
		{"wrong numbers 5", "PT10M10", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := StringISO8601MinutesToInt(tt.input)
			if tt.shouldFail {
				assert.EqualError(t, err, "wrong format, expected ISO8601 minutes (e.g. PT10M)")
				assert.Equal(t, 0, actual)
				return
			}

			assert.Nil(t, err)
			assert.Equal(t, tt.expected, actual)
		})
	}
}
