/*
Copyright 2022 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package fusion

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"testing"

	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
)

func TestAccHostAccessPolicy_basic(t *testing.T) {
	rNameConfig := fmt.Sprintf("host_access_policy_%d", acctest.RandIntRange(0, 1000))
	rName := "fusion_host_access_policy." + rNameConfig
	displayName := acctest.RandomWithPrefix("host-access-policy-display-name")
	hostAccessPolicyName := acctest.RandomWithPrefix("test_hap")
	iqn := randIQN()

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckHAPDestroy,
		Steps: []resource.TestStep{
			// Create Host Access Policy and validate it's fields
			{
				Config: testHostAccessPolicyConfig(rNameConfig, hostAccessPolicyName, displayName, iqn, "linux"),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(rName, "name", hostAccessPolicyName),
					resource.TestCheckResourceAttr(rName, "display_name", displayName),
					resource.TestCheckResourceAttr(rName, "iqn", iqn),
					resource.TestCheckResourceAttr(rName, "personality", "linux"),
					testHostAccessPolicyExists(rName),
				),
			},
		},
	})
}

func TestAccHostAccessPolicy_RequiredAttributes(t *testing.T) {
	rNameConfig := fmt.Sprintf("host_access_policy_%d", acctest.RandIntRange(0, 1000))
	displayName := acctest.RandomWithPrefix("host-access-policy-display-name")
	hostAccessPolicyName := acctest.RandomWithPrefix("test_hap")
	iqn := randIQN()

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckHAPDestroy,
		Steps: []resource.TestStep{
			// IQN attribute value is empty
			{
				Config:      testHostAccessPolicyConfig(rNameConfig, hostAccessPolicyName, displayName, "", "linux"),
				ExpectError: regexp.MustCompile("Error: iqn must be specified"),
			},
			// Personality attribute value is empty
			{
				Config:      testHostAccessPolicyConfig(rNameConfig, hostAccessPolicyName, displayName, iqn, ""),
				ExpectError: regexp.MustCompile("Error: personality must be specified"),
			},
		},
	})
}

func testHostAccessPolicyConfig(rName string, hostAccessPolicyName string, displayName string, iqn string, personality string) string {
	return fmt.Sprintf(`
	resource "fusion_host_access_policy" "%[1]s" {
		name          = "%[2]s"
		display_name  = "%[3]s"
		iqn           = "%[4]s"
		personality   = "%[5]s"
	}
	`, rName, hostAccessPolicyName, displayName, iqn, personality)
}

func testHostAccessPolicyExists(rName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		tfHostAccessPolicy, ok := s.RootModule().Resources[rName]
		if !ok {
			return fmt.Errorf("Resource not found: %s", rName)
		}
		if tfHostAccessPolicy.Type != "fusion_host_access_policy" {
			return fmt.Errorf("Expected type: fusion_host_access_policy. Found: %s", tfHostAccessPolicy.Type)
		}
		attrs := tfHostAccessPolicy.Primary.Attributes

		goclientHostAccessPolicy, _, err := testAccProvider.Meta().(*hmrest.APIClient).HostAccessPoliciesApi.GetHostAccessPolicy(context.Background(), attrs["name"], nil)
		if err != nil {
			return fmt.Errorf("Go client retutrned error while searching for %s. Error: %s", attrs["name"], err)
		}
		if strings.Compare(goclientHostAccessPolicy.Name, attrs["name"]) != 0 ||
			strings.Compare(goclientHostAccessPolicy.DisplayName, attrs["display_name"]) != 0 {
			return fmt.Errorf("Terraform host access policy doesnt match goclients host access policy")
		}
		return nil
	}
}

func testCheckHAPDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*hmrest.APIClient)
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "fusion_host_access_policy" {
			continue
		}
		attrs := rs.Primary.Attributes
		hostAccessPolicyName := attrs["name"]

		_, resp, err := client.HostAccessPoliciesApi.GetHostAccessPolicy(context.Background(), hostAccessPolicyName, nil)
		if err != nil && resp.StatusCode == http.StatusNotFound {
			continue
		} else {
			return fmt.Errorf("Host access policy exist. Expected response code 404, got code %d", resp.StatusCode)
		}
	}
	return nil
}

func randIQN() string {
	return fmt.Sprintf("iqn.year-mo.org.debian:XX:%d", acctest.RandIntRange(100000000000, 200000000000))
}
