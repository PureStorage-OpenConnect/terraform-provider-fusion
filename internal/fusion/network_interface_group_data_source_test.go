/*
Copyright 2023 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package fusion

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"

	"github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/utilities"
)

// Contains correct list of Network Interface Groups
func TestAccNetworkInterfaceGroupDataSource_basic(t *testing.T) {
	utilities.CheckTestSkip(t)

	dsNameConfig := acctest.RandomWithPrefix("network_interface_group_ds_test")
	count := 3
	nigs := make([]map[string]interface{}, count)
	configs := make([]string, count)

	for i := 0; i < count; i++ {
		configName := acctest.RandomWithPrefix("network_interface_group_test")
		nigName := acctest.RandomWithPrefix("test_nig")
		displayName := acctest.RandomWithPrefix("nig-display-name")
		mtu := strconv.Itoa(acctest.RandIntRange(1280, 9216))

		nigs[i] = map[string]interface{}{
			"name":              nigName,
			"display_name":      displayName,
			"availability_zone": preexistingAvailabilityZone,
			"region":            preexistingRegion,
			"group_type":        groupType,
			"eth": []map[string]interface{}{{
				"gateway": gateway,
				"prefix":  prefix,
				"mtu":     mtu,
			}},
		}

		configs[i] = testNetworkInterfaceGroupConfig(configName, nigName, displayName, preexistingAvailabilityZone, preexistingRegion, groupType, gateway, prefix, mtu)
	}

	allConfigs := strings.Join(configs, "\n")
	partialConfig := configs[0] + configs[1]

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckNetworkInterfaceGroupDestroy,
		Steps: []resource.TestStep{
			// Create n nigs
			{
				Config: allConfigs,
			},
			// Check if they are contained in the data source
			{
				Config: allConfigs + "\n" + testNetworkInterfaceGroupDataSourceConfig(dsNameConfig, preexistingAvailabilityZone, preexistingRegion),
				Check:  utilities.TestCheckDataSource("fusion_network_interface_group", dsNameConfig, "items", nigs),
			},
			{
				Config: partialConfig,
			},
			// Remove one nig. Check if only two of them are contained in the data source
			{
				Config: partialConfig + "\n" + testNetworkInterfaceGroupDataSourceConfig(dsNameConfig, preexistingAvailabilityZone, preexistingRegion),
				Check: resource.ComposeTestCheckFunc(
					utilities.TestCheckDataSource(
						"fusion_network_interface_group", dsNameConfig, "items", []map[string]interface{}{nigs[0], nigs[1]},
					),
					utilities.TestCheckDataSourceNotHave(
						"fusion_network_interface_group", dsNameConfig, "items", []map[string]interface{}{nigs[2]},
					),
				),
			},
		},
	})
}

func testNetworkInterfaceGroupDataSourceConfig(dsName, availabilityZone, region string) string {
	return fmt.Sprintf(`data "fusion_network_interface_group" "%[1]s" {
		availability_zone = "%[2]s"
		region			  = "%[3]s"
	}`, dsName, availabilityZone, region)
}
