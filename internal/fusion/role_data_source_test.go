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

// Contains correct list of Role
func TestAccRoleDataSource_basic(t *testing.T) {
	utilities.CheckTestSkip(t)

	dsNameConfig := acctest.RandomWithPrefix("role_ds_test")

	roles := []map[string]interface{}{
		{
			"name":         "az-admin",
			"display_name": "AZ Admin",
		},
		{
			"name":         "tenant-admin",
			"display_name": "Tenant Admin",
		},
		{
			"name":         "tenant-space-admin",
			"display_name": "Tenant Space Admin",
		},
	}

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		Steps: []resource.TestStep{
			{
				Config: testRoleDataSourceConfig(dsNameConfig, ""),
				Check:  utilities.TestCheckDataSource("fusion_role", dsNameConfig, "items", roles),
			},
			{
				Config: testRoleDataSourceConfig(dsNameConfig, "TenantSpace"),
				Check: resource.ComposeTestCheckFunc(
					utilities.TestCheckDataSource("fusion_role", dsNameConfig, "items", []map[string]interface{}{roles[2]}),
					utilities.TestCheckDataSourceNotHave("fusion_role", dsNameConfig, "items", []map[string]interface{}{roles[0], roles[1]}),
				),
			},
			{
				Config: testRoleDataSourceConfig(dsNameConfig, "Tenant"),
				Check: resource.ComposeTestCheckFunc(
					utilities.TestCheckDataSource("fusion_role", dsNameConfig, "items", []map[string]interface{}{roles[1], roles[2]}),
					utilities.TestCheckDataSourceNotHave("fusion_role", dsNameConfig, "items", []map[string]interface{}{roles[0]}),
				),
			},
			{
				Config: testRoleDataSourceConfig(dsNameConfig, "Organization"),
				Check:  utilities.TestCheckDataSource("fusion_role", dsNameConfig, "items", roles),
			},
		},
	})
}

func testRoleDataSourceConfig(dsName string, assignableScope string) string {
	if assignableScope == "" {
		return fmt.Sprintf(`data "fusion_role" "%s" {
		}`, dsName)
	}

	return fmt.Sprintf(`data "fusion_role" "%[1]s" {
		assignable_scope = "%[2]s"
	}`, dsName, assignableScope)
}
