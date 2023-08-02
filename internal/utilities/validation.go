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

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var isAlphaStringRegex = regexp.MustCompile(`^[a-zA-Z]+$`)
var dataUnitRegex = regexp.MustCompile(`^(0|[1-9]\d*)[KMGTP]{0,1}$`)
var errInvalidDataUnitFormat = errors.New("invalid data unit format")

func AllValid(functions ...schema.SchemaValidateDiagFunc) schema.SchemaValidateDiagFunc {
	return func(i interface{}, p cty.Path) diag.Diagnostics {
		var diags diag.Diagnostics

		for _, fun := range functions {
			funDiags := fun(i, p)

			if funDiags != nil {
				diags = append(diags, funDiags...)
			}
		}

		return diags
	}
}

func DataUnitsBeetween(min, max int64, factor int) schema.SchemaValidateDiagFunc {
	return func(value any, p cty.Path) diag.Diagnostics {
		valueString, ok := value.(string)
		if !ok {
			return diag.Diagnostics{
				diag.Diagnostic{
					Severity: diag.Error,
					Summary:  "invalid value type",
					Detail:   fmt.Sprintf("%v expected to be string", value),
				},
			}
		}

		convertedValue, err := ConvertDataUnitsToInt64(valueString, factor)
		if err != nil {
			return diag.Diagnostics{
				diag.Diagnostic{
					Severity: diag.Error,
					Summary:  errInvalidDataUnitFormat.Error(),
					Detail:   fmt.Sprintf("%s unexpected data unit format", valueString),
				},
			}
		}

		if convertedValue < min || convertedValue > max {
			return diag.Diagnostics{
				diag.Diagnostic{
					Severity: diag.Error,
					Summary:  "out of range",
					Detail:   fmt.Sprintf("expected to be in the range (%d - %d), got %d", min, max, convertedValue),
				},
			}
		}

		return diag.Diagnostics{}
	}
}

func StringIsInt64(value any, p cty.Path) diag.Diagnostics {
	valueString, ok := value.(string)
	if !ok {
		return diag.Diagnostics{
			diag.Diagnostic{
				Severity: diag.Error,
				Summary:  "invalid value type",
				Detail:   fmt.Sprintf("%v expected to be string", value),
			},
		}
	}

	if _, err := strconv.ParseInt(valueString, 10, 64); err != nil {
		return diag.Diagnostics{
			diag.Diagnostic{
				Severity: diag.Error,
				Summary:  errInvalidDataUnitFormat.Error(),
				Detail:   fmt.Sprintf("%s unexpected data unit format", valueString),
			},
		}
	}

	return diag.Diagnostics{}
}

func AllowedDataUnitSuffix(suffixes ...byte) schema.SchemaValidateDiagFunc {
	if !isAlphaStringRegex.Match(suffixes) {
		panic("AllowedDataUnitSuffix accepts only letters")
	}
	specificDataUnitRegex := regexp.MustCompile(fmt.Sprintf(`^(0|[1-9]\d*)[%s]{0,1}$`, string(suffixes)))

	return func(value any, p cty.Path) diag.Diagnostics {
		valueString, ok := value.(string)
		if !ok {
			return diag.Diagnostics{
				diag.Diagnostic{
					Severity: diag.Error,
					Summary:  "invalid type",
					Detail:   fmt.Sprintf("%v expected to be string", value),
				},
			}
		}

		if !specificDataUnitRegex.MatchString(valueString) {
			return diag.Diagnostics{
				diag.Diagnostic{
					Severity: diag.Error,
					Summary:  errInvalidDataUnitFormat.Error(),
					Detail:   fmt.Sprintf("%s unexpected data unit format", valueString),
				},
			}
		}

		return diag.Diagnostics{}
	}
}
