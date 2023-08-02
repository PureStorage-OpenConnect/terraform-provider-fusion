/*
Copyright 2023 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package utilities

import (
	"fmt"
	"math"
	"strconv"
	"strings"

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

// Generic function for parsing self links.
// We assume that this helper function is called with ordered list of resource group names
func ParseSelfLink(selfLink string, orderedRequiredGroupNamesInPath []string) (map[string]string, error) {
	err := fmt.Errorf("self link has incorrect format")
	parts := strings.Split(selfLink, "/")
	if len(parts)%2 == 0 || (len(parts)-1)/2 != len(orderedRequiredGroupNamesInPath) || parts[0] != "" {
		return nil, err
	}
	fieldsWithValues := make(map[string]string)
	for orderOfGroup, groupName := range orderedRequiredGroupNamesInPath {
		if groupName != parts[orderOfGroup*2+1] || parts[orderOfGroup*2+2] == "" {
			return nil, err
		}
		fieldsWithValues[groupName] = parts[orderOfGroup*2+2]
	}
	return fieldsWithValues, nil
}
