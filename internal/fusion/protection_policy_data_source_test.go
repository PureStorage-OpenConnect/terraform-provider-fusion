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

func TestAccProtectionPolicyDataSource_basic(t *testing.T) {
	utilities.CheckTestSkip(t)

	protectionPoliciesCount := 3
	dsNameConfig := acctest.RandomWithPrefix("protection_policy_ds_test")
	protectionPolicies := make([]map[string]interface{}, protectionPoliciesCount)
	protectionPoliciesConfigs := make([]string, protectionPoliciesCount)

	for i := 0; i < protectionPoliciesCount; i++ {
		protectionPolicyResourceNameConfig := acctest.RandomWithPrefix("protection_policy_test")
		protectionPolicyName := acctest.RandomWithPrefix("protection_policy_name_test")
		protectionPolicyDisplayName := acctest.RandomWithPrefix("protection_policy_name-display-name")
		localRPO := "100"
		localRetention := "100"

		protectionPolicies[i] = map[string]interface{}{
			"name":            protectionPolicyName,
			"display_name":    protectionPolicyDisplayName,
			"local_rpo":       localRPO,
			"local_retention": localRetention,
		}

		protectionPoliciesConfigs[i] = testAccProtectionPolicyConfig(protectionPolicyResourceNameConfig,
			protectionPolicyName, protectionPolicyDisplayName, localRPO, localRetention, true)
	}

	allConfigs := strings.Join(protectionPoliciesConfigs, "\n")

	partialConfig := strings.Join(protectionPoliciesConfigs[:2], "\n")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testAccCheckProtectionPolicyDestroy,
		Steps: []resource.TestStep{
			{
				Config: allConfigs,
			},
			{
				Config: allConfigs + testProtectionPolicyDataSourceConfig(dsNameConfig),
				Check:  utilities.TestCheckDataSource("fusion_protection_policy", dsNameConfig, "items", protectionPolicies),
			},
			{
				Config: partialConfig,
			},
			{
				Config: partialConfig + testProtectionPolicyDataSourceConfig(dsNameConfig),
				Check: resource.ComposeTestCheckFunc(
					utilities.TestCheckDataSource("fusion_protection_policy", dsNameConfig, "items", protectionPolicies[:2]),
					utilities.TestCheckDataSourceNotHave("fusion_protection_policy", dsNameConfig, "items", protectionPolicies[2:]),
				),
			},
		},
	})
}

func testProtectionPolicyDataSourceConfig(dsName string) string {
	return fmt.Sprintf(`data "fusion_protection_policy" "%[1]s" {}`, dsName)
}
