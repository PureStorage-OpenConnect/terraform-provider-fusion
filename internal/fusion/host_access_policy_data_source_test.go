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

// Contains correct list of Host Access Policies
func TestAccHostAccessPolicyDataSource_basic(t *testing.T) {
	utilities.CheckTestSkip(t)

	dsNameConfig := acctest.RandomWithPrefix("host_access_policy_ds_test")
	numOfHostAccessPolicies := 3
	hostAccessPoliciesConfigs := make([]string, numOfHostAccessPolicies)
	hostAccessPolicies := make([]map[string]interface{}, numOfHostAccessPolicies)

	for i := 0; i < numOfHostAccessPolicies; i++ {
		configName := acctest.RandomWithPrefix("host_access_policy")
		displayName := acctest.RandomWithPrefix("host-access-policy-display-name")
		hostAccessPolicyName := acctest.RandomWithPrefix("test_hap")
		iqn := randIQN()
		hostAccessPolicies[i] = map[string]interface{}{
			"name":         hostAccessPolicyName,
			"display_name": displayName,
			"iqn":          iqn,
			"personality":  "linux",
		}
		hostAccessPoliciesConfigs[i] = testHostAccessPolicyConfig(configName, hostAccessPolicyName, displayName, iqn, "linux")
	}

	allConfigs := strings.Join(hostAccessPoliciesConfigs, "\n")
	partialConfig := hostAccessPoliciesConfigs[0] + hostAccessPoliciesConfigs[1]

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckHAPDestroy,
		Steps: []resource.TestStep{
			// Create n HAPs
			{
				Config: allConfigs,
			},
			// Check if they are contained in the data source
			{
				Config: allConfigs + "\n" + testHostAccessPolicyDataSourceConfig(dsNameConfig),
				Check:  utilities.TestCheckDataSource("fusion_host_access_policy", dsNameConfig, "items", hostAccessPolicies),
			},
			{
				Config: partialConfig,
			},
			// Remove one host access policy. Check if only two of them are contained in the data source
			{
				Config: partialConfig + "\n" + testHostAccessPolicyDataSourceConfig(dsNameConfig),
				Check: resource.ComposeTestCheckFunc(
					utilities.TestCheckDataSource(
						"fusion_host_access_policy", dsNameConfig, "items", []map[string]interface{}{hostAccessPolicies[0], hostAccessPolicies[1]},
					),
					utilities.TestCheckDataSourceNotHave(
						"fusion_host_access_policy", dsNameConfig, "items", []map[string]interface{}{hostAccessPolicies[2]},
					),
				),
			},
		},
	})
}

func testHostAccessPolicyDataSourceConfig(dsName string) string {
	return fmt.Sprintf(`data "fusion_host_access_policy" "%[1]s" {}`, dsName)
}
