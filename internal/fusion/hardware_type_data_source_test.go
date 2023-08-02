/*
Copyright 2023 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package fusion

import (
	"fmt"
	"testing"

	"github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/utilities"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccHardwareTypeDataSource_basic(t *testing.T) {
	utilities.CheckTestSkip(t)

	dsNameConfig := acctest.RandomWithPrefix("hw_ds_test")

	expectedHWTypes := []map[string]interface{}{
		{
			"name":         "flash-array-x",
			"display_name": "FlashArray//X",
			"array_type":   "FA//X",
			"media_type":   "TLC",
		},
		{
			"name":         "flash-array-c",
			"display_name": "FlashArray//C",
			"array_type":   "FA//C",
			"media_type":   "QLC",
		},
		{
			"name":         "flash-array-x-optane",
			"display_name": "FlashArray//X-Optane",
			"array_type":   "FA//X",
			"media_type":   "Optane",
		},
		{
			"name":         "flash-array-xl",
			"display_name": "FlashArray//XL",
			"array_type":   "FA//XL",
			"media_type":   "TLC",
		},
		{
			"name":         "flash-array-cbs-azure",
			"display_name": "FlashArray//CBS on Azure",
			"array_type":   "FA//CBS",
			"media_type":   "Cloud",
		},
	}

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		Steps: []resource.TestStep{
			// Get all hw types
			{
				Config: testHardwareTypeDataSourceBasicConfig(dsNameConfig),
				Check:  utilities.TestCheckDataSource("fusion_hardware_type", dsNameConfig, "items", expectedHWTypes),
			},
		},
	})
}

func TestAccHardwareTypeDataSource_filter(t *testing.T) {
	utilities.CheckTestSkip(t)

	dsNameConfig := acctest.RandomWithPrefix("hw_ds_test")

	tlcHWTypes := []map[string]interface{}{
		{
			"name":         "flash-array-x",
			"display_name": "FlashArray//X",
			"array_type":   "FA//X",
			"media_type":   "TLC",
		},
		{
			"name":         "flash-array-xl",
			"display_name": "FlashArray//XL",
			"array_type":   "FA//XL",
			"media_type":   "TLC",
		},
	}

	xHWTypes := []map[string]interface{}{
		{
			"name":         "flash-array-x",
			"display_name": "FlashArray//X",
			"array_type":   "FA//X",
			"media_type":   "TLC",
		},
		{
			"name":         "flash-array-x-optane",
			"display_name": "FlashArray//X-Optane",
			"array_type":   "FA//X",
			"media_type":   "Optane",
		},
	}

	tlcXHWType := []map[string]interface{}{
		{
			"name":         "flash-array-x",
			"display_name": "FlashArray//X",
			"array_type":   "FA//X",
			"media_type":   "TLC",
		},
	}

	filterMedia := "media_type = \"TLC\""
	filterArray := "array_type = \"FA//X\""
	filterBoth := filterMedia + "\n" + filterArray

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		Steps: []resource.TestStep{
			// Get hw types with TLC media type
			{
				Config: testHardwareTypeDataSourceFilterConfig(dsNameConfig, filterMedia),
				Check:  utilities.TestCheckDataSourceExact("fusion_hardware_type", dsNameConfig, "items", tlcHWTypes),
			},
			// Get hw types with X array type
			{
				Config: testHardwareTypeDataSourceFilterConfig(dsNameConfig, filterArray),
				Check:  utilities.TestCheckDataSourceExact("fusion_hardware_type", dsNameConfig, "items", xHWTypes),
			},
			// Get hw types with TLC media type and X array type
			{
				Config: testHardwareTypeDataSourceFilterConfig(dsNameConfig, filterBoth),
				Check:  utilities.TestCheckDataSourceExact("fusion_hardware_type", dsNameConfig, "items", tlcXHWType),
			},
		},
	})
}

func testHardwareTypeDataSourceBasicConfig(dsName string) string {
	return fmt.Sprintf(`data "fusion_hardware_type" "%[1]s" {}`, dsName)
}

func testHardwareTypeDataSourceFilterConfig(dsName string, filter string) string {
	return fmt.Sprintf(`data "fusion_hardware_type" "%[1]s" {
		%[2]s
	}`, dsName, filter)
}
