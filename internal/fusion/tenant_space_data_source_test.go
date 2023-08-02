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

// Contains correct list of Tenant Spaces
func TestAccTenantSpaceDataSource_basic(t *testing.T) {
	utilities.CheckTestSkip(t)

	dsNameConfig := acctest.RandomWithPrefix("tenant_space_ds_test")
	count := 3
	tenantSpaces := make([]map[string]interface{}, count)
	configs := make([]string, count)

	tenantName := acctest.RandomWithPrefix("tenant-test")
	tenatConfig := testTenantConfig(tenantName, tenantName, tenantName)

	for i := 0; i < count; i++ {
		configName := acctest.RandomWithPrefix("tenant_space_test")
		tenantSpaceName := acctest.RandomWithPrefix("test_tenant_space")
		displayName := acctest.RandomWithPrefix("tenant-space-display-name")

		tenantSpaces[i] = map[string]interface{}{
			"name":         tenantSpaceName,
			"display_name": displayName,
			"tenant":       tenantName,
		}

		configs[i] = testTenantSpaceConfigWithRefs(configName, displayName, tenantSpaceName, tenantName)
	}

	allConfigs := tenatConfig + strings.Join(configs, "\n")
	partialConfig := tenatConfig + configs[0] + configs[1]

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckTenantSpaceDestroy,
		Steps: []resource.TestStep{
			// Create n TenantSpaces
			{
				Config: allConfigs,
			},
			// Check if they are contained in the data source
			{
				Config: allConfigs + "\n" + testTenantSpaceDataSourceConfig(dsNameConfig, tenantName),
				Check:  utilities.TestCheckDataSourceExact("fusion_tenant_space", dsNameConfig, "items", tenantSpaces),
			},
			{
				Config: partialConfig,
			},
			// Remove one tenant space. Check if only two of them are contained in the data source
			{
				Config: partialConfig + "\n" + testTenantSpaceDataSourceConfig(dsNameConfig, tenantName),
				Check: utilities.TestCheckDataSourceExact(
					"fusion_tenant_space", dsNameConfig, "items", []map[string]interface{}{tenantSpaces[0], tenantSpaces[1]},
				),
			},
		},
	})
}

func testTenantSpaceDataSourceConfig(dsName, tenantName string) string {
	return fmt.Sprintf(`data "fusion_tenant_space" "%[1]s" {
		tenant = fusion_tenant.%[2]s.name
	}`, dsName, tenantName)
}
