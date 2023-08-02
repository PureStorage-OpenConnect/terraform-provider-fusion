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
		{"only numbers 5", "0112", 112},
		{"only numbers 6", "00001000", 1000},
		{"555 minutes", "555M", 555},
		{"2 days", "2D", 2880},
		{"400 days", "400D", 576000},
		{"3 weeks", "3W", 30240},
		{"35 weeks", "35W", 352800},
		{"1 year", "1Y", 525600},
		{"5 years", "5Y", 525600 * 5},
		{"400 days", "400d", 576000},
		{"3 weeks", "3w", 30240},
		{"35 weeks", "35w", 352800},
		{"1 year", "1y", 525600},
		{"5 years", "5y", 525600 * 5},
		{"1 day and 12 hours", "1d12h", 2160},
		{"1 day, 12 hours and 3 minutes", "1D12H3M", 2163},
		{"leading zeros 1", "0001M", 1},
		{"leading zeros 2", "0001000M", 1000},
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
		{"wrong numbers 3", "0.1"},
		{"wrong numbers 4", "0.100"},
		{"wrong numbers 5", "200.0"},
		{"mixed letters and numbers 1", "123M567"},
		{"mixed letters and numbers 2", "M123"},
		{"multiple letters 1", "123MY"},
		{"multiple letters 2", "123YM"},
		{"multiple letters 4", "123YMHD"},
		{"multiple letters 1", "123My"},
		{"multiple letters 2", "123yM"},
		{"multiple letters 4", "123ymhd"},
		{"invalid chars 1", "1 M"},
		{"invalid chars 2", "1-M"},
		{"invalid chars 3", "1_M"},
		{"invalid chars 4", "1M "},
		{"invalid chars 5", "1M_"},
		{"invalid chars 6", "1Q"},
		{"invalid chars 7", "2C"},
		{"invalid chars 8", "3F"},
		{"invalid order of units 1", "10M10H"},
		{"invalid order of units 2", "10H10M1D"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := ParseHumanReadableTimePeriodIntoMinutes(tt.timePeriod)
			assert.EqualError(t, err, ErrWrongHumanReadableFormat.Error())
			assert.Equal(t, 0, actual)
		})
	}
}

func TestSplitTimePeriodString(t *testing.T) {
	tests := []struct {
		name, timeString string
		expected         []string
	}{
		{"empty string", "", nil},
		{"single unit 1", "1H", []string{"1H"}},
		{"single unit 2", "2w", []string{"2w"}},
		{"multiple units 1", "1D1H", []string{"1D", "1H"}},
		{"multiple units 2", "1D1H1M", []string{"1D", "1H", "1M"}},
		{"multiple units 3", "123D22H042M", []string{"123D", "22H", "042M"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := splitTimePeriodString(tt.timeString)
			assert.Equal(t, tt.expected, actual)
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
		{"10101 minutes with leading zero", "PT010101M", 10101, false},
		{"zero minutes", "PT0M", 0, false},
		{"no numbers", "PTM", 0, true},
		{"only numbers", "10101", 0, true},
		{"no minutes", "PR10101", 0, true},
		{"no PT", "10101M", 0, true},
		{"wrong letters 1", "PT10MY", 0, true},
		{"wrong letters 2", "PT10Y", 0, true},
		{"wrong numbers 2", "0PT10M", 0, true},
		{"wrong numbers 3", "10PT10M", 0, true},
		{"wrong numbers 4", "PT10M0", 0, true},
		{"wrong numbers 5", "PT10M10", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := StringISO8601MinutesToInt(tt.input)
			if tt.shouldFail {
				assert.EqualError(t, err, ErrWrongISO8601MinutesFormat.Error())
				assert.Equal(t, 0, actual)
				return
			}

			assert.Nil(t, err)
			assert.Equal(t, tt.expected, actual)
		})
	}
}
