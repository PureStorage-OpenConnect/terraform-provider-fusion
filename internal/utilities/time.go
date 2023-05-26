/*
Copyright 2023 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package utilities

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

const (
	hourInMinutes = 60
	dayInMinutes  = 24 * hourInMinutes
	weekInMinutes = 7 * dayInMinutes
	yearInMinutes = 365 * dayInMinutes
)

var (
	humanReadableTimePeriodRegex = regexp.MustCompile(`^(0|[1-9]\d*)[YWDHM]{0,1}$`)
	stringISO8601Regex           = regexp.MustCompile(`^PT(0|[1-9]\d*)M$`)
)

func ParseHumanReadableTimePeriodIntoMinutes(s string) (int, error) {
	if !humanReadableTimePeriodRegex.MatchString(s) {
		return 0, fmt.Errorf("wrong format, expected human-readable time period (e.g. 2d, 3w)")
	}

	if value, err := strconv.Atoi(s); err == nil {
		return value, nil
	}

	unit := strings.ToUpper(s[len(s)-1:])
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
		return 0, fmt.Errorf("wrong format, expected ISO8601 minutes (e.g. PT10M)")
	}

	s = strings.TrimLeft(s, "PT")
	minutes, _ := strconv.Atoi(s[:len(s)-1])
	return minutes, nil
}
