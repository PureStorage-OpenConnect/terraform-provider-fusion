/*
Copyright 2023 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package fusion

const (
	HwTypeArrayX       = "flash-array-x"
	HwTypeArrayC       = "flash-array-c"
	HwTypeArrayXOptane = "flash-array-x-optane"
	HwTypeArrayXl      = "flash-array-xl"
)

var HwTypes = []string{
	HwTypeArrayX,
	HwTypeArrayC,
	HwTypeArrayXOptane,
	HwTypeArrayXl,
}
