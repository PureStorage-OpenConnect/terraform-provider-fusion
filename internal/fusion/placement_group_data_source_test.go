/*
Copyright 2023 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
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

// Contains correct list of Placement Groups
func TestAccPlacementGroupDataSource_basic(t *testing.T) {
	utilities.CheckTestSkip(t)

	dsNameConfig := acctest.RandomWithPrefix("pg_ds_test")
	pgCount := 2
	placementGroups := make([]map[string]interface{}, pgCount)
	configs := make([]string, pgCount)

	tenant := acctest.RandomWithPrefix("pg-test-tenant")
	tenantSpace := acctest.RandomWithPrefix("pg-test-ts")
	commonConfig := testTenantConfig(tenant, tenant, tenant) +
		testTenantSpaceConfigWithRefs(tenantSpace, tenantSpace, tenantSpace, tenant)

	arrays := getArraysInPreexistingRegionAndAZ(t)
	hwTypes := getHWTypesFromArrays(arrays)

	for i := 0; i < pgCount; i++ {
		name := acctest.RandomWithPrefix("pg-ds-test-name")
		ssName := acctest.RandomWithPrefix("pg-test-ss")
		ssConfig := testStorageServiceConfig(ssName, ssName, ssName, []string{hwTypes[i]})

		placementGroups[i] = map[string]interface{}{
			"name":                        name,
			"display_name":                name,
			"tenant":                      tenant,
			"tenant_space":                tenantSpace,
			"region":                      preexistingRegion,
			"availability_zone":           preexistingAvailabilityZone,
			"storage_service":             ssName,
			"destroy_snapshots_on_delete": false,
		}

		configs[i] = ssConfig + testPlacementGroupConfigWithRefsNoArray(name, name, name, tenant, tenantSpace,
			preexistingRegion, preexistingAvailabilityZone, ssName, false)
	}

	allConfigs := strings.Join(configs, "\n")
	partialConfig := configs[0]

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckPlacementGroupDestroy,
		Steps: []resource.TestStep{
			// Create n placement groups
			{
				Config: commonConfig + allConfigs,
			},
			// Check if they are contained in the data source
			{
				Config: commonConfig + allConfigs + "\n" + testPlacementGroupDataSourceConfig(dsNameConfig, tenant, tenantSpace),
				Check:  utilities.TestCheckDataSourceExact("fusion_placement_group", dsNameConfig, "items", placementGroups),
			},
			// Remove one placement group
			{
				Config: commonConfig + partialConfig,
			},
			// Check if only one placement group is contained in the data source
			{
				Config: commonConfig + partialConfig + "\n" + testPlacementGroupDataSourceConfig(dsNameConfig, tenant, tenantSpace),
				Check: utilities.TestCheckDataSourceExact(
					"fusion_placement_group", dsNameConfig, "items", []map[string]interface{}{placementGroups[0]},
				),
			},
		},
	})
}

func testPlacementGroupDataSourceConfig(dsName, tenant, tenantSpace string) string {
	return fmt.Sprintf(`data "fusion_placement_group" "%[1]s" {
		tenant       = "%[2]s"
		tenant_space = "%[3]s"
	}
	`, dsName, tenant, tenantSpace)
}
