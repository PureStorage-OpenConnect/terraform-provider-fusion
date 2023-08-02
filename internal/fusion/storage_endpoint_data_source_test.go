/*
Copyright 2023 Pure Storage Inc
SPDX-License-Identifier: Apache-storageEndpointsCount-numberOfEmptyDataSources.0
*/

package fusion

import (
	"fmt"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/utilities"
)

func TestAccStorageEndpointDataSource_basic(t *testing.T) {
	utilities.CheckTestSkip(t)

	storageEndpointsCount := 5

	regionName := acctest.RandomWithPrefix("region")
	regionConfig := testRegionConfigNoDisplayName(regionName, regionName)

	storageEndpoints := make([]map[string]interface{}, storageEndpointsCount)
	seConfigs := make([]string, storageEndpointsCount)

	azNames := make([]string, storageEndpointsCount)
	azConfigs := make([]string, storageEndpointsCount)

	seDataSourceConfigs := make([]string, storageEndpointsCount)
	seDataSourceNames := make([]string, storageEndpointsCount)

	// Create n AZs, SEs and SEDataSources configs
	for i := 0; i < storageEndpointsCount; i++ {
		azNames[i] = acctest.RandomWithPrefix("az")
		azConfigs[i] = testAvailabilityZoneConfigRef(azNames[i], azNames[i], regionName)

		seDataSourceNames[i] = acctest.RandomWithPrefix("storage_endpoint_ds_test")

		storageEndpointResourceNameConfig := acctest.RandomWithPrefix("storage_endpoint_test")
		storageEndpointName := acctest.RandomWithPrefix("test_se")
		storageEndpointDisplayName := acctest.RandomWithPrefix("storage_endpoint-display-name")

		iscsi := []map[string]interface{}{
			{
				"discovery_interfaces": []map[string]interface{}{
					{
						"address": fmt.Sprintf("10.21.200.%d/%d", i, i+8),
					},
				},
			},
		}

		storageEndpoints[i] = map[string]interface{}{
			"name":              storageEndpointName,
			"display_name":      storageEndpointDisplayName,
			"region":            regionName,
			"availability_zone": azNames[i],
			"iscsi":             iscsi,
		}

		seConfigs[i] = testStorageEndpointConfig(storageEndpointResourceNameConfig, storageEndpointName,
			storageEndpointDisplayName, regionName, azNames[i], iscsi)
		seDataSourceConfigs[i] = testStorageEndpointDataSourceConfig(seDataSourceNames[i], regionName, azNames[i])
	}

	allConfigs := regionConfig + "\n" + strings.Join(azConfigs, "\n") + "\n" + strings.Join(seConfigs, "\n")

	numberOfEmptyDataSources := 2
	partialConfig := regionConfig + "\n" + strings.Join(azConfigs, "\n") + "\n" + strings.Join(seConfigs[:storageEndpointsCount-numberOfEmptyDataSources], "\n")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckStorageEndpointDestroy,
		Steps: []resource.TestStep{
			// Create n Storage Endpoints
			{
				Config: allConfigs,
			},
			// Test all DataSources
			{
				Config: allConfigs + "\n" + strings.Join(seDataSourceConfigs, "\n"),
				Check:  getStorageEndpointDataSourcesCheckFunc(seDataSourceNames, storageEndpoints),
			},
			{
				Config: partialConfig,
			},
			{
				Config: partialConfig + "\n" + strings.Join(seDataSourceConfigs[:storageEndpointsCount], "\n"),
				Check: resource.ComposeTestCheckFunc(
					getStorageEndpointDataSourcesCheckFunc(seDataSourceNames[:storageEndpointsCount-numberOfEmptyDataSources], storageEndpoints[:storageEndpointsCount-numberOfEmptyDataSources]),
					getEmptyStorageEndpointDataSourcesCheckFunc(seDataSourceNames[storageEndpointsCount-numberOfEmptyDataSources:]),
				),
			},
		},
	})
}

func getStorageEndpointDataSourcesCheckFunc(dsStorageEndpointNames []string, storageEndpoints []map[string]interface{}) resource.TestCheckFunc {
	checkFuncs := make([]resource.TestCheckFunc, len(dsStorageEndpointNames))
	for i, se := range storageEndpoints {
		checkFuncs[i] = utilities.TestCheckDataSourceExact("fusion_storage_endpoint", dsStorageEndpointNames[i], "items", []map[string]interface{}{se})
	}
	return resource.ComposeTestCheckFunc(checkFuncs...)
}

func getEmptyStorageEndpointDataSourcesCheckFunc(dsStorageEndpointNames []string) resource.TestCheckFunc {
	checkFuncs := make([]resource.TestCheckFunc, len(dsStorageEndpointNames))
	for i, seName := range dsStorageEndpointNames {
		checkFuncs[i] = utilities.TestCheckDataSourceExact("fusion_storage_endpoint", seName, "items", []map[string]interface{}{})
	}
	return resource.ComposeTestCheckFunc(checkFuncs...)
}

func testStorageEndpointDataSourceConfig(dsName, region, availabilityZone string) string {
	return fmt.Sprintf(`data "fusion_storage_endpoint" "%[1]s" {
		region = "%[2]s"
		availability_zone = "%[3]s"
	}`, dsName, region, availabilityZone)
}
