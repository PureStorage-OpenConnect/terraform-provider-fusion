/*
Copyright 2023 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package fusion

import (
	"fmt"
	"strings"
	"testing"

	"github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/utilities"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// Contains correct list of Regions
func TestAccRegionDataSource_basic(t *testing.T) {
	dsNameConfig := acctest.RandomWithPrefix("region_ds_test")
	regionsCount := 3
	regions := make([]map[string]interface{}, regionsCount)
	regionConfigs := make([]string, regionsCount)

	for i := 0; i < regionsCount; i++ {
		regionResourceNameConfig := acctest.RandomWithPrefix("region_test")
		regionName := acctest.RandomWithPrefix("test_ss")
		regionDisplayName := acctest.RandomWithPrefix("region-display-name")

		regions[i] = map[string]interface{}{
			"name":         regionName,
			"display_name": regionDisplayName,
		}

		regionConfigs[i] = testRegionConfig(regionResourceNameConfig, regionName, regionDisplayName)
	}

	allConfigs := strings.Join(regionConfigs, "\n")
	partialConfig := regionConfigs[0] + regionConfigs[1]

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckRegionDestroy,
		Steps: []resource.TestStep{
			// Create n regions
			{
				Config: allConfigs,
			},
			// Check if they are contained in the data source
			{
				Config: allConfigs + "\n" + testRegionDataSourceConfig(dsNameConfig),
				Check:  utilities.TestCheckDataSource("fusion_region", dsNameConfig, "items", regions),
			},
			// Remove one region. Check if only two of them are contained in the data source
			{
				Config: partialConfig + "\n" + testRegionDataSourceConfig(dsNameConfig),
				Check: utilities.TestCheckDataSource(
					"fusion_region", dsNameConfig, "items", []map[string]interface{}{regions[0], regions[1]},
				),
			},
		},
	})
}

func testRegionDataSourceConfig(dsName string) string {
	return fmt.Sprintf(`data "fusion_region" "%[1]s" {}`, dsName)
}
