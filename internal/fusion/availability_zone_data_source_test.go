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

// Contains correct list of AZ
func TestAccAvailabilityZoneDataSource_basic(t *testing.T) {
	utilities.CheckTestSkip(t)

	dsNameConfig := acctest.RandomWithPrefix("az_ds_test")
	azCount := 3
	availabilityZones := make([]map[string]interface{}, azCount)
	azConfigs := make([]string, azCount)

	regionName := acctest.RandomWithPrefix("region_test")
	regionConfig := testRegionConfig(regionName, regionName, regionName)

	for i := 0; i < azCount; i++ {
		azResourceNameConfig := acctest.RandomWithPrefix("az_test")
		azName := acctest.RandomWithPrefix("az_test")

		availabilityZones[i] = map[string]interface{}{
			"name":         azName,
			"display_name": azName,
		}

		azConfigs[i] = testAvailabilityZoneConfigRef(azResourceNameConfig, azName, regionName)
	}

	allConfigs := strings.Join(azConfigs, "\n") + regionConfig
	partialConfig := azConfigs[0] + azConfigs[1] + regionConfig
	partialAZs := []map[string]interface{}{availabilityZones[0], availabilityZones[1]}
	excludedAZs := []map[string]interface{}{availabilityZones[2]}

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckAvailabilityZoneDestroy,
		Steps: []resource.TestStep{
			// Create n availability zones
			{
				Config: allConfigs,
			},
			// Check if they are contained in the data source
			{
				Config: allConfigs + "\n" + testAvailabilityZoneDataSourceConfig(dsNameConfig, regionName),
				Check:  utilities.TestCheckDataSource("fusion_availability_zone", dsNameConfig, "items", availabilityZones),
			},
			{
				Config: partialConfig,
			},
			// Remove one availability zone. Check if only two of them are contained in the data source
			{
				Config: partialConfig + "\n" + testAvailabilityZoneDataSourceConfig(dsNameConfig, regionName),
				Check: resource.ComposeTestCheckFunc(
					utilities.TestCheckDataSource("fusion_availability_zone", dsNameConfig, "items", partialAZs),
					utilities.TestCheckDataSourceNotHave("fusion_availability_zone", dsNameConfig, "items", excludedAZs),
				),
			},
		},
	})
}

func testAvailabilityZoneDataSourceConfig(dsName string, region string) string {
	return fmt.Sprintf(`data "fusion_availability_zone" "%[1]s" {
		region = fusion_region.%[2]s.name
	}`, dsName, region)
}
