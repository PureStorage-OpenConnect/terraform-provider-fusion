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
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"

	"github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/utilities"
)

var (
	localRPO       = "20"
	localRetention = "5D"
)

func TestAccProtectionPolicy_basic(t *testing.T) {
	utilities.CheckTestSkip(t)

	rNameConfig := acctest.RandomWithPrefix("fusion_protection_policy_test")
	rName := "fusion_protection_policy." + rNameConfig
	policyName := acctest.RandomWithPrefix("pp-name")
	displayName := acctest.RandomWithPrefix("pp-display-name")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testAccCheckProtectionPolicyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccProtectionPolicyConfig(rNameConfig, policyName, displayName, localRPO, localRetention, true),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(rName, "name", policyName),
					resource.TestCheckResourceAttr(rName, "display_name", displayName),
					resource.TestCheckResourceAttr(rName, "local_rpo", localRPO),
					resource.TestCheckResourceAttr(rName, "local_retention", "7200"),
					testAccCheckProtectionPolicyExists(rName),
				),
			},
		},
	})
}

func TestAccProtectionPolicy_update(t *testing.T) {
	utilities.CheckTestSkip(t)

	rNameConfig := acctest.RandomWithPrefix("fusion_protection_policy_test")
	rName := "fusion_protection_policy." + rNameConfig
	policyName := acctest.RandomWithPrefix("pp-name")
	displayName := acctest.RandomWithPrefix("pp-display-name")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testAccCheckProtectionPolicyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccProtectionPolicyConfig(rNameConfig, policyName, displayName, localRPO, localRetention, true),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(rName, "name", policyName),
					resource.TestCheckResourceAttr(rName, "display_name", displayName),
					resource.TestCheckResourceAttr(rName, "local_rpo", localRPO),
					resource.TestCheckResourceAttr(rName, "local_retention", "7200"),
					resource.TestCheckResourceAttr(rName, "destroy_snapshots_on_delete", "true"),
					testAccCheckProtectionPolicyExists(rName),
				),
			},
			{
				Config: testAccProtectionPolicyConfig(rNameConfig, policyName, displayName, localRPO, localRetention, false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(rName, "name", policyName),
					resource.TestCheckResourceAttr(rName, "display_name", displayName),
					resource.TestCheckResourceAttr(rName, "local_rpo", localRPO),
					resource.TestCheckResourceAttr(rName, "local_retention", "7200"),
					resource.TestCheckResourceAttr(rName, "destroy_snapshots_on_delete", "false"),
					testAccCheckProtectionPolicyExists(rName),
				),
			},
		},
	})
}

func TestAccProtectionPolicy_attributes(t *testing.T) {
	utilities.CheckTestSkip(t)

	rNameConfig := acctest.RandomWithPrefix("fusion_protection_policy_test")
	policyName := acctest.RandomWithPrefix("pp-name")
	displayName := acctest.RandomWithPrefix("pp-display-name")
	displayNameTooBig := strings.Repeat("a", 257)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testAccCheckProtectionPolicyDestroy,
		Steps: []resource.TestStep{
			// Values should not pass validations
			{
				Config:      testAccProtectionPolicyConfig(rNameConfig, "", displayName, localRPO, localRetention, true),
				ExpectError: regexp.MustCompile(`expected "name" to not be an empty string`),
			},
			{
				Config:      testAccProtectionPolicyConfig(rNameConfig, "bad name here", displayName, localRPO, localRetention, true),
				ExpectError: regexp.MustCompile("name must use alphanumeric characters"),
			},
			{
				Config:      testAccProtectionPolicyConfig(rNameConfig, policyName, displayNameTooBig, localRPO, localRetention, true),
				ExpectError: regexp.MustCompile(`expected length of display_name to be in the range \(1 - 256\), got .{257}`),
			},
			{
				Config:      testAccProtectionPolicyConfig(rNameConfig, policyName, "", localRPO, localRetention, true),
				ExpectError: regexp.MustCompile(`expected length of display_name to be in the range \(1 - 256\), .?`),
			},
			{
				Config:      testAccProtectionPolicyConfig(rNameConfig, policyName, displayName, "1", localRetention, true),
				ExpectError: regexp.MustCompile(`expected local_rpo to be at least \(10\), got 1`),
			},
			{
				Config:      testAccProtectionPolicyConfig(rNameConfig, policyName, displayName, localRPO, "", true),
				ExpectError: regexp.MustCompile("Bad local retention"),
			},
			{
				Config:      testAccProtectionPolicyConfig(rNameConfig, policyName, displayName, localRPO, "0", true),
				ExpectError: regexp.MustCompile("Bad local retention"),
			},
		},
	})
}

func TestAccProtectionPolicy_multiple(t *testing.T) {
	utilities.CheckTestSkip(t)

	rNameConfig1 := acctest.RandomWithPrefix("fusion_protection_policy_test")
	rName1 := "fusion_protection_policy." + rNameConfig1
	policyName1 := acctest.RandomWithPrefix("pp-name")
	displayName1 := acctest.RandomWithPrefix("pp-display-name")

	rNameConfig2 := acctest.RandomWithPrefix("fusion_protection_policy_test")
	rName2 := "fusion_protection_policy." + rNameConfig2
	policyName2 := acctest.RandomWithPrefix("pp-name")
	displayName2 := acctest.RandomWithPrefix("pp-display-name")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testAccCheckProtectionPolicyDestroy,
		Steps: []resource.TestStep{
			// Sanity check two can be created at once
			{
				Config: testAccProtectionPolicyConfig(rNameConfig1, policyName1, displayName1, localRPO, localRetention, true) + "\n" +
					testAccProtectionPolicyConfig(rNameConfig2, policyName2, displayName2, localRPO, localRetention, true),
				Check: resource.ComposeTestCheckFunc(
					testAccCheckProtectionPolicyExists(rName1),
					testAccCheckProtectionPolicyExists(rName2),
				),
			},
			// Create two with same name
			{
				Config: testAccProtectionPolicyConfig(rNameConfig1, policyName1, displayName1, localRPO, localRetention, true) + "\n" +
					testAccProtectionPolicyConfig(rNameConfig2, policyName2, displayName2, localRPO, localRetention, true) + "\n" +
					testAccProtectionPolicyConfig("conflictRN", policyName1, "conflictDN", localRPO, localRetention, true),
				ExpectError: regexp.MustCompile("already exists"),
			},
		},
	})
}

func TestAccProtectionPolicy_import(t *testing.T) {
	utilities.CheckTestSkip(t)

	rNameConfig := acctest.RandomWithPrefix("fusion_protection_policy_test")
	rName := "fusion_protection_policy." + rNameConfig
	policyName := acctest.RandomWithPrefix("pp-name")
	displayName := acctest.RandomWithPrefix("pp-display-name")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testAccCheckProtectionPolicyDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccProtectionPolicyConfig(rNameConfig, policyName, displayName, localRPO, localRetention, true),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(rName, "name", policyName),
					resource.TestCheckResourceAttr(rName, "display_name", displayName),
					resource.TestCheckResourceAttr(rName, "local_rpo", localRPO),
					resource.TestCheckResourceAttr(rName, "local_retention", "7200"),
					testAccCheckProtectionPolicyExists(rName),
				),
			},
			{
				ImportState:       true,
				ResourceName:      fmt.Sprintf("fusion_protection_policy.%s", rNameConfig),
				ImportStateId:     fmt.Sprintf("/protection-policies/%s", policyName),
				ImportStateVerify: true,
				// skipping destroy_snapshots_on_delete field, this field is used as additional parameter for deletion
				// destroy_snapshots_on_delete cannot be imported from harbormaster
				ImportStateVerifyIgnore: []string{optionDestroySnapshotsOnDelete},
			},
			{
				ImportState:   true,
				ResourceName:  fmt.Sprintf("fusion_protection_policy.%s", rNameConfig),
				ImportStateId: fmt.Sprintf("/protection-policies/wrong-%s", policyName),
				ExpectError:   regexp.MustCompile("Not Found"),
			},
			{
				ImportState:   true,
				ResourceName:  fmt.Sprintf("fusion_protection_policy.%s", rNameConfig),
				ImportStateId: fmt.Sprintf("/%s", policyName),
				ExpectError:   regexp.MustCompile("invalid protection_policy import path. Expected path in format '/protection-policies/<protection-policy>'"),
			},
		},
	})
}

func testAccCheckProtectionPolicyExists(rName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		tfResource, ok := s.RootModule().Resources[rName]
		if !ok {
			return fmt.Errorf("resource not found: %s", rName)
		}
		if tfResource.Type != "fusion_protection_policy" {
			return fmt.Errorf("expected type: fusion_protection_policy. Found: %s", tfResource.Type)
		}
		savedPolicy := tfResource.Primary.Attributes
		policyName := savedPolicy["name"]
		policyId := savedPolicy["id"]

		foundPolicy, _, err := testAccProvider.Meta().(*hmrest.APIClient).ProtectionPoliciesApi.GetProtectionPolicyById(context.Background(), policyId, nil)
		if err != nil {
			return fmt.Errorf("go client returned error while searching for %s by id: %s. Error: %s", policyName, policyId, err)
		}

		var errs error
		checkAttr := func(client, attrName string) {
			if client != savedPolicy[attrName] {
				errs = multierror.Append(errs, fmt.Errorf("mismatch attr: %s client: %s tf: %s", attrName, client, savedPolicy[attrName]))
			}
		}

		checkAttr(foundPolicy.Name, "name")
		checkAttr(foundPolicy.DisplayName, "display_name")

		for _, obj := range foundPolicy.Objectives {
			if rpo, ok := obj.(*hmrest.Rpo); ok {
				foundRPOValue, _ := utilities.StringISO8601MinutesToInt(rpo.Rpo)
				if strconv.Itoa(foundRPOValue) != savedPolicy["local_rpo"] {
					errs = multierror.Append(errs, fmt.Errorf("mismatch attr: %s client: %d tf: %s", "local_rpo", foundRPOValue, savedPolicy["local_rpo"]))
				}
				continue
			}

			if retention, ok := obj.(*hmrest.Retention); ok {
				foundRetentionValue, _ := utilities.StringISO8601MinutesToInt(retention.After)
				if strconv.Itoa(foundRetentionValue) != savedPolicy["local_retention"] {
					errs = multierror.Append(errs, fmt.Errorf("mismatch attr: %s client: %d tf: %s", "local_retention", foundRetentionValue, savedPolicy["local_retention"]))
				}
				continue
			}
		}

		if errs != nil {
			return multierror.Append(fmt.Errorf("terraform protection policy resource doesnt match clients protection policy"), errs)
		}

		return nil
	}
}

func testAccCheckProtectionPolicyDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*hmrest.APIClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "fusion_protection_policy" {
			continue
		}
		attrs := rs.Primary.Attributes
		name, ok := attrs["name"]
		if !ok {
			continue // Skip data sources
		}

		_, resp, err := client.ProtectionPoliciesApi.GetProtectionPolicy(context.Background(), name, nil)
		if err != nil && resp.StatusCode == http.StatusNotFound {
			continue
		}
		return fmt.Errorf("protection policy may still exist. Expected response code 404, got code %d", resp.StatusCode)
	}
	return nil
}

func testAccProtectionPolicyConfig(rName, name, displayName, localRPO, localRetention string, destroySnap bool) string {
	return fmt.Sprintf(`
	resource "fusion_protection_policy" "%[1]s" {
		name			= "%[2]s"
		display_name	= "%[3]s"
		local_rpo		= "%[4]s"
		local_retention = "%[5]s"
		destroy_snapshots_on_delete = "%[6]t"
	}
	`, rName, name, displayName, localRPO, localRetention, destroySnap)
}
