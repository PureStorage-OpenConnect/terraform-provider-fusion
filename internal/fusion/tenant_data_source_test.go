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

// Contains correct list of Tenants
func TestAccTenantDataSource_basic(t *testing.T) {
	utilities.CheckTestSkip(t)

	dsNameConfig := acctest.RandomWithPrefix("tenant_ds_test")
	count := 3
	tenants := make([]map[string]interface{}, count)
	configs := make([]string, count)

	for i := 0; i < count; i++ {
		configName := acctest.RandomWithPrefix("tenant_test")
		tenantName := acctest.RandomWithPrefix("test_tenant")
		displayName := acctest.RandomWithPrefix("tenant-display-name")

		tenants[i] = map[string]interface{}{
			"name":         tenantName,
			"display_name": displayName,
		}

		configs[i] = testTenantConfig(configName, tenantName, displayName)
	}

	allConfigs := strings.Join(configs, "\n")
	partialConfig := configs[0] + configs[1]

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckTenantDestroy,
		Steps: []resource.TestStep{
			// Create n tenants
			{
				Config: allConfigs,
			},
			// Check if they are contained in the data source
			{
				Config: allConfigs + "\n" + testTenantDataSourceConfig(dsNameConfig),
				Check:  utilities.TestCheckDataSource("fusion_tenant", dsNameConfig, "items", tenants),
			},
			{
				Config: partialConfig,
			},
			// Remove one tenant. Check if only two of them are contained in the data source
			{
				Config: partialConfig + "\n" + testTenantDataSourceConfig(dsNameConfig),
				Check: resource.ComposeTestCheckFunc(
					utilities.TestCheckDataSource(
						"fusion_tenant", dsNameConfig, "items", []map[string]interface{}{tenants[0], tenants[1]},
					),
					utilities.TestCheckDataSourceNotHave(
						"fusion_tenant", dsNameConfig, "items", []map[string]interface{}{tenants[2]},
					),
				),
			},
		},
	})
}

func testTenantDataSourceConfig(dsName string) string {
	return fmt.Sprintf(`data "fusion_tenant" "%[1]s" {}`, dsName)
}
