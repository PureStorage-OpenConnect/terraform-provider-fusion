/*
Copyright 2023 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package fusion

import (
	"regexp"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/utilities"
)

const (
	localRpoMin       = 10
	localRetentionMin = 10
)

var iqnValidRegex *regexp.Regexp = regexp.MustCompile(`^iqn\.[^\ _]*\.[^\ _]*`)

func IsValidIQN(v interface{}, path cty.Path) diag.Diagnostics {
	var diags diag.Diagnostics

	value := v.(string)
	if !iqnValidRegex.MatchString(value) {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Invalid IQN",
			Detail:   "Must be a valid IQN",
		})
	}

	return diags
}

func IsValidAddress(v interface{}, path cty.Path) diag.Diagnostics {
	var diags diag.Diagnostics

	value := v.(string)
	if !utilities.IsValidAddress(value) {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Bad address",
			Detail:   "Address should be IPv4",
		})
	}

	return diags
}

func IsValidPrefix(v interface{}, path cty.Path) diag.Diagnostics {
	var diags diag.Diagnostics

	value := v.(string)
	if !utilities.IsValidPrefix(value) {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Bad prefix",
			Detail:   "Prefix should be between 8 and 32",
		})
	}

	return diags
}

func DiffSuppressForHumanReadableTimePeriod(k, old, new string, d *schema.ResourceData) bool {
	oldValue, err := utilities.ParseHumanReadableTimePeriodIntoMinutes(old)
	if err != nil {
		return false
	}

	newValue, err := utilities.ParseHumanReadableTimePeriodIntoMinutes(new)
	if err != nil {
		return false
	}
	return oldValue == newValue
}

func IsValidLocalRetention(v interface{}, path cty.Path) diag.Diagnostics {
	var diags diag.Diagnostics

	value := v.(string)
	minutes, err := utilities.ParseHumanReadableTimePeriodIntoMinutes(value)
	if err != nil {
		return diag.Diagnostics{diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Bad local retention",
			Detail:   err.Error(),
		}}
	}

	if minutes < localRetentionMin {
		diags = append(diags, diag.Diagnostic{
			Severity: diag.Error,
			Summary:  "Bad local retention",
			Detail:   "Local retention must be a minimum of 1 minutes",
		})
	}

	return diags
}
