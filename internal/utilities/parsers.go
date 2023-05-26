/*
Copyright 2023 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package utilities

import (
	"math"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

func GetDiffSuppressForDataUnits(factor int) schema.SchemaDiffSuppressFunc {
	return func(key, old, new string, d *schema.ResourceData) bool {
		oldValue, err := ConvertDataUnitsToInt64(old, factor)
		if err != nil {
			return false
		}

		newValue, err := ConvertDataUnitsToInt64(new, factor)
		if err != nil {
			return false
		}

		return newValue == oldValue
	}
}

func getDataUnitsSuffixes() map[byte]int {
	return map[byte]int{
		'K': 1,
		'M': 2,
		'G': 3,
		'T': 4,
		'P': 5,
	}
}

func ConvertDataUnitsToInt64(number string, factor int) (int64, error) {
	if !dataUnitRegex.MatchString(number) {
		return -1, errInvalidDataUnitFormat
	}

	if convertedNumber, err := strconv.ParseInt(number, 10, 64); err == nil {
		return convertedNumber, nil
	}

	suffixes := getDataUnitsSuffixes()
	suffix := number[len(number)-1]

	convertedNumber, _ := strconv.ParseInt(number[:len(number)-1], 10, 64)

	return convertedNumber * int64(math.Pow(float64(factor), float64(suffixes[suffix]))), nil
}
