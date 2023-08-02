/*
Copyright 2023 Pure Storage Inc
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

	"github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/utilities"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"
)

func TestAccHostAccessPolicy_basic(t *testing.T) {
	utilities.CheckTestSkip(t)

	rNameConfig := acctest.RandomWithPrefix("host_access_policy")
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
	utilities.CheckTestSkip(t)

	rNameConfig := acctest.RandomWithPrefix("host_access_policy")
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
				ExpectError: regexp.MustCompile("Error: Invalid IQN"),
			},
			// Personality attribute value is empty
			{
				Config:      testHostAccessPolicyConfig(rNameConfig, hostAccessPolicyName, displayName, iqn, ""),
				ExpectError: regexp.MustCompile("Error: expected personality to be one of"),
			},
		},
	})
}

func TestAccHostAccessPolicy_import(t *testing.T) {
	utilities.CheckTestSkip(t)

	rNameConfig := acctest.RandomWithPrefix("host_access_policy")
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
			{
				ImportState:       true,
				ResourceName:      fmt.Sprintf("fusion_host_access_policy.%s", rNameConfig),
				ImportStateId:     fmt.Sprintf("/host-access-policies/%s", hostAccessPolicyName),
				ImportStateVerify: true,
			},
			{
				ImportState:   true,
				ResourceName:  fmt.Sprintf("fusion_host_access_policy.%s", rNameConfig),
				ImportStateId: fmt.Sprintf("/host-access-policies/wrong-%s", hostAccessPolicyName),
				ExpectError:   regexp.MustCompile("Not Found"),
			},
			{
				ImportState:   true,
				ResourceName:  fmt.Sprintf("fusion_host_access_policy.%s", rNameConfig),
				ImportStateId: fmt.Sprintf("/wrong-%s", hostAccessPolicyName),
				ExpectError:   regexp.MustCompile("invalid host_access_policy import path. Expected path in format '/host-access-policies/<host-access-policy>'"),
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

		goclientHostAccessPolicy, _, err := testAccProvider.Meta().(*hmrest.APIClient).HostAccessPoliciesApi.GetHostAccessPolicyById(context.Background(), attrs["id"], nil)
		if err != nil {
			return fmt.Errorf("Go client returned error while searching for %s by id: %s. Error: %s", attrs["name"], attrs["id"], err)
		}
		if strings.Compare(goclientHostAccessPolicy.Name, attrs["name"]) != 0 ||
			strings.Compare(goclientHostAccessPolicy.DisplayName, attrs["display_name"]) != 0 ||
			strings.Compare(goclientHostAccessPolicy.Iqn, attrs["iqn"]) != 0 ||
			strings.Compare(goclientHostAccessPolicy.Personality, attrs["personality"]) != 0 {
			return fmt.Errorf("Terraform host access policy doesn't match goclients host access policy")
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
		hostAccessPolicyName, ok := attrs["name"]
		if !ok {
			continue // Skip data sources
		}

		_, resp, err := client.HostAccessPoliciesApi.GetHostAccessPolicy(context.Background(), hostAccessPolicyName, nil)
		if err != nil && resp.StatusCode == http.StatusNotFound {
			continue
		}

		return fmt.Errorf("Host access policy exists. Expected response code 404, got code %d", resp.StatusCode)
	}
	return nil
}

func randIQN() string {
	return fmt.Sprintf("iqn.year-mo.org.debian:XX:%d", acctest.RandIntRange(100000000000, 200000000000))
}
