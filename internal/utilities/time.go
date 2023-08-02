/*
Copyright 2023 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package utilities

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

const (
	hourInMinutes = 60
	dayInMinutes  = 24 * hourInMinutes
	weekInMinutes = 7 * dayInMinutes
	yearInMinutes = 365 * dayInMinutes
)

var (
	ErrWrongHumanReadableFormat  = errors.New("wrong format, expected human-readable time period (e.g. 2d, 3w5h, 1Y32D)")
	ErrWrongISO8601MinutesFormat = errors.New("wrong format, expected ISO8601 minutes (e.g. PT10M)")
)

var (
	humanReadableTimePeriodRegex = regexp.MustCompile(`^(\d+Y)?(\d+W)?(\d+D)?(\d+H)?(\d+M)?$`)
	stringISO8601Regex           = regexp.MustCompile(`^PT\d+M$`)
)

func splitTimePeriodString(input string) []string {
	var output []string
	word := ""
	for _, char := range input {
		word = word + string(char)
		if unicode.IsLetter(char) {
			output = append(output, word)
			word = ""
		}
	}
	return output
}

func ParseHumanReadableTimePeriodIntoMinutes(s string) (int, error) {
	if len(s) == 0 {
		return 0, ErrWrongHumanReadableFormat
	}

	s = strings.ToUpper(s)
	// If the input string is a number without units, simply return the input number.
	if value, err := strconv.Atoi(s); err == nil {
		return value, nil
	}
	if humanReadableTimePeriodRegex.MatchString(s) {
		minutes := 0
		for _, field := range splitTimePeriodString(s) {
			value, err := parseSingleUnitString(field)
			if err != nil {
				return 0, err
			}
			minutes = minutes + value
		}
		return minutes, nil
	}
	return 0, ErrWrongHumanReadableFormat
}

func parseSingleUnitString(s string) (int, error) {
	unit := s[len(s)-1:]
	value, _ := strconv.Atoi(s[:len(s)-1])
	return value * timePeriodUnitToMinutes()[unit], nil
}

func timePeriodUnitToMinutes() map[string]int {
	return map[string]int{
		"Y": yearInMinutes,
		"W": weekInMinutes,
		"D": dayInMinutes,
		"H": hourInMinutes,
		"M": 1,
	}
}

func MinutesToStringISO8601(minutes int) string {
	return fmt.Sprintf("PT%dM", minutes)
}

func StringISO8601MinutesToInt(s string) (int, error) {
	if !stringISO8601Regex.MatchString(s) {
		return 0, ErrWrongISO8601MinutesFormat
	}

	s = strings.TrimLeft(s, "PT")
	minutes, _ := strconv.Atoi(s[:len(s)-1])
	return minutes, nil
}
