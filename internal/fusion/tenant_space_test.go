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

	"github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/utilities"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"
)

// Creates and destroys
func TestAccTenantSpace_basic(t *testing.T) {
	utilities.CheckTestSkip(t)

	rNameConfig := acctest.RandomWithPrefix("tenant_space_test")
	rName := "fusion_tenant_space." + rNameConfig
	displayName1 := acctest.RandomWithPrefix("tenant-space-display-name")
	tenantSpaceName := acctest.RandomWithPrefix("test_ts")

	tenant := acctest.RandomWithPrefix("ts_test_tenant")
	commonConfig := testTenantConfig(tenant, tenant, tenant)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckTenantSpaceDestroy,
		Steps: []resource.TestStep{
			// Create Tenant and validate it's fields
			{
				Config: commonConfig + testTenantSpaceConfigWithRefs(rNameConfig, displayName1, tenantSpaceName, tenant),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(rName, "name", tenantSpaceName),
					resource.TestCheckResourceAttr(rName, "display_name", displayName1),
					resource.TestCheckResourceAttr(rName, "tenant", tenant),
					testTenantSpaceExists(rName),
				),
			},
		},
	})
}

// Updates display name
func TestAccTenantSpace_update(t *testing.T) {
	utilities.CheckTestSkip(t)

	rNameConfig := acctest.RandomWithPrefix("tenant_space_test")
	rName := "fusion_tenant_space." + rNameConfig
	displayName1 := acctest.RandomWithPrefix("tenant-space-display-name")
	displayName2 := acctest.RandomWithPrefix("tenant-space-display-name2")
	displayNameTooBig := strings.Repeat("a", 257)
	tenantSpaceName := acctest.RandomWithPrefix("test_ts")

	tenant := acctest.RandomWithPrefix("ts_test_tenant")
	commonConfig := testTenantConfig(tenant, tenant, tenant)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckTenantSpaceDestroy,
		Steps: []resource.TestStep{
			// Create Tenant and validate it's fields
			{
				Config: commonConfig + testTenantSpaceConfigWithRefs(rNameConfig, displayName1, tenantSpaceName, tenant),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(rName, "name", tenantSpaceName),
					resource.TestCheckResourceAttr(rName, "display_name", displayName1),
					resource.TestCheckResourceAttr(rName, "tenant", tenant),
					testTenantSpaceExists(rName),
				),
			},
			// Update the display name, assert that the tf resource got updated, then assert the backend shows the same
			{
				Config: commonConfig + testTenantSpaceConfigWithRefs(rNameConfig, displayName2, tenantSpaceName, tenant),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(rName, "display_name", displayName2),
					testTenantSpaceExists(rName),
				),
			},
			// Bad display name values
			{
				Config: commonConfig + testTenantSpaceConfigWithRefs(rNameConfig, displayNameTooBig, tenantSpaceName, tenant),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(rName, "display_name", displayName2),
					testTenantSpaceExists(rName),
				),
				ExpectError: regexp.MustCompile(`expected length of display_name to be in the range \(1 - 256\), .?`),
			},

			// Can't update certain values
			{
				Config:      commonConfig + testTenantSpaceConfigWithRefs(rNameConfig, displayName1, "immutable", tenant),
				ExpectError: regexp.MustCompile("attempting to update an immutable field"),
			},
			{
				Config:      commonConfig + testTenantSpaceConfigWithNames(rNameConfig, displayName1, tenantSpaceName, "immutable"),
				ExpectError: regexp.MustCompile("attempting to update an immutable field"),
			},
			// When the test tries to destroy the resources at the end, it does not do a refresh first,
			// and therefore the destroy will fail if the state is invalid. Because of this, we need to manually
			// return the state to a valid config. Note that the "terraform destroy" command does do
			// a refresh first, so this issue only applies to acceptance tests.
			{
				Config: commonConfig + testTenantSpaceConfigWithRefs(rNameConfig, displayName1, tenantSpaceName, tenant),
			},
		},
	})
}

func TestAccTenantSpace_attributes(t *testing.T) {
	utilities.CheckTestSkip(t)

	rNameConfig := acctest.RandomWithPrefix("tenant_space_test")
	rName := "fusion_tenant_space." + rNameConfig
	displayName1 := acctest.RandomWithPrefix("tenant-space-display-name")
	tenantSpaceName := acctest.RandomWithPrefix("tenant-space-name")

	tenant := acctest.RandomWithPrefix("ts_test_tenant")
	commonConfig := testTenantConfig(tenant, tenant, tenant)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckTenantSpaceDestroy,
		Steps: []resource.TestStep{
			// Missing required fields
			{
				Config:      commonConfig + testTenantSpaceConfigWithRefs(rNameConfig, displayName1, "", tenant),
				ExpectError: regexp.MustCompile(`expected "name" to not be an empty string`),
			},
			{
				Config:      commonConfig + testTenantSpaceConfigWithNames(rNameConfig, displayName1, tenantSpaceName, ""),
				ExpectError: regexp.MustCompile(`expected "tenant" to not be an empty string`),
			},
			{
				Config:      commonConfig + testTenantSpaceConfigWithRefs(rNameConfig, displayName1, "bad name here", tenant),
				ExpectError: regexp.MustCompile("name must use alphanumeric characters"),
			},
			// {
			//	Config:      testTenantSpaceConfig(rNameConfig, displayName1, "", ""),
			//	ExpectError: regexp.MustCompile("Error: Name & Tenant Space must be specified"), // TODO: HM-2420 this should be both!
			// },
			// Create without display_name then update
			{
				Config: commonConfig + testTenantSpaceConfigNoDisplayNameWithRefs(rNameConfig, tenantSpaceName, tenant),
				Check: resource.ComposeTestCheckFunc(
					testTenantSpaceExists(rName),
				),
			},
			{
				Config: commonConfig + testTenantSpaceConfigWithRefs(rNameConfig, displayName1, tenantSpaceName, tenant),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(rName, "display_name", displayName1),
					testTenantSpaceExists(rName),
				),
			},
		},
	})
}

func TestAccTenantSpace_multiple(t *testing.T) {
	utilities.CheckTestSkip(t)

	rNameConfig := acctest.RandomWithPrefix("tenant_space_test")
	rName := "fusion_tenant_space." + rNameConfig
	displayName1 := acctest.RandomWithPrefix("tenant-space-display-name")
	tenantSpaceName := acctest.RandomWithPrefix("tenant-space-name")

	rNameConfig2 := acctest.RandomWithPrefix("tenant_space_test2")
	rName2 := "fusion_tenant_space." + rNameConfig
	displayName2 := acctest.RandomWithPrefix("tenant-space-display-name")
	tenantSpaceName2 := acctest.RandomWithPrefix("tenant-space-name")

	tenant := acctest.RandomWithPrefix("ts_test_tenant")
	commonConfig := testTenantConfig(tenant, tenant, tenant)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckTenantSpaceDestroy,
		Steps: []resource.TestStep{
			// Sanity check two can be created at once
			{
				Config: commonConfig + testTenantSpaceConfigWithRefs(rNameConfig, displayName1, tenantSpaceName, tenant) + "\n" +
					testTenantSpaceConfigWithRefs(rNameConfig2, displayName2, tenantSpaceName2, tenant),
				Check: resource.ComposeTestCheckFunc(
					testTenantSpaceExists(rName),
					testTenantSpaceExists(rName2),
				),
			},
			// Create two with same name
			{
				Config: commonConfig + testTenantSpaceConfigWithRefs(rNameConfig, displayName1, tenantSpaceName, tenant) + "\n" +
					testTenantSpaceConfigWithRefs(rNameConfig2, displayName2, tenantSpaceName2, tenant) + "\n" +
					testTenantSpaceConfigWithRefs("conflictRN", "conflictDN", tenantSpaceName, tenant),
				ExpectError: regexp.MustCompile("already exists"),
			},
		},
	})
}

func TestAccTenantSpace_import(t *testing.T) {
	utilities.CheckTestSkip(t)

	rNameConfig := acctest.RandomWithPrefix("tenant_space_test")
	rName := "fusion_tenant_space." + rNameConfig
	displayName := acctest.RandomWithPrefix("tenant-space-display-name")
	tenantSpaceName := acctest.RandomWithPrefix("test_ts")

	tenant := acctest.RandomWithPrefix("ts_test_tenant")
	commonConfig := testTenantConfig(tenant, tenant, tenant)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckTenantSpaceDestroy,
		Steps: []resource.TestStep{
			// Create Tenant Space and validate it's fields
			{
				Config: commonConfig + testTenantSpaceConfigWithRefs(rNameConfig, displayName, tenantSpaceName, tenant),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(rName, "name", tenantSpaceName),
					resource.TestCheckResourceAttr(rName, "display_name", displayName),
					resource.TestCheckResourceAttr(rName, "tenant", tenant),
					testTenantSpaceExists(rName),
				),
			},
			{
				ImportState:       true,
				ResourceName:      fmt.Sprintf("fusion_tenant_space.%s", rNameConfig),
				ImportStateId:     fmt.Sprintf("/tenants/%[1]s/tenant-spaces/%[2]s", tenant, tenantSpaceName),
				ImportStateVerify: true,
			},
			{
				ImportState:   true,
				ResourceName:  fmt.Sprintf("fusion_tenant_space.%s", rNameConfig),
				ImportStateId: fmt.Sprintf("/tenants/%[1]s/tenant-spaces/wrong-%[2]s", tenant, tenantSpaceName),
				ExpectError:   regexp.MustCompile("Not Found"),
			},
			{
				ImportState:   true,
				ResourceName:  fmt.Sprintf("fusion_tenant_space.%s", rNameConfig),
				ImportStateId: fmt.Sprintf("/tenant-spaces/%[2]s", tenant, tenantSpaceName),
				ExpectError:   regexp.MustCompile("invalid tenant_space import path. Expected path in format '/tenants/<tenant>/tenant-spaces/<tenant-space>'"),
			},
		},
	})
}

func testTenantSpaceExists(rName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		tfTenantSpace, ok := s.RootModule().Resources[rName]
		if !ok {
			return fmt.Errorf("resource not found: %s", rName)
		}
		if tfTenantSpace.Type != "fusion_tenant_space" {
			return fmt.Errorf("expected type: fusion_tenant_space. Found: %s", tfTenantSpace.Type)
		}
		attrs := tfTenantSpace.Primary.Attributes

		goclientTenantSpace, _, err := testAccProvider.Meta().(*hmrest.APIClient).TenantSpacesApi.GetTenantSpaceById(context.Background(), attrs["id"], nil)
		if err != nil {
			return fmt.Errorf("go client retutrned error while searching for %s by id: %s. Error: %s", attrs["name"], attrs["id"], err)
		}
		if strings.Compare(goclientTenantSpace.Name, attrs["name"]) != 0 ||
			strings.Compare(goclientTenantSpace.DisplayName, attrs["display_name"]) != 0 ||
			strings.Compare(goclientTenantSpace.Tenant.Name, attrs["tenant"]) != 0 {
			return fmt.Errorf("terraform tenant space doesnt match goclients tenant space")
		}
		return nil
	}
}

func testCheckTenantSpaceDestroy(s *terraform.State) error {

	client := testAccProvider.Meta().(*hmrest.APIClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "fusion_tenant_space" {
			continue
		}
		attrs := rs.Primary.Attributes

		tenantName := attrs["tenant"]
		tenantSpaceName, ok := attrs["name"]
		if !ok {
			continue // Skip data sources
		}

		_, resp, err := client.TenantSpacesApi.GetTenantSpace(context.Background(), tenantName, tenantSpaceName, nil)
		if err != nil && resp.StatusCode == http.StatusNotFound {
			continue
		} else {
			return fmt.Errorf("tenant space may still exist. Expected response code 404, got code %d", resp.StatusCode)
		}
	}
	return nil
}

func testTenantSpaceConfigWithNames(rName string, displayName string, tenantSpaceName string, tenantName string) string {
	return fmt.Sprintf(`
	resource "fusion_tenant_space" "%[1]s" {
		name			= "%[2]s"
		display_name	= "%[3]s"
		tenant			= "%[4]s"
	}
	`, rName, tenantSpaceName, displayName, tenantName)
}

func testTenantSpaceConfigWithRefs(rName string, displayName string, tenantSpaceName string, tenantName string) string {
	return fmt.Sprintf(`
	resource "fusion_tenant_space" "%[1]s" {
		name			= "%[2]s"
		display_name	= "%[3]s"
		tenant			= fusion_tenant.%[4]s.name
	}
	`, rName, tenantSpaceName, displayName, tenantName)
}

func testTenantSpaceConfigNoDisplayNameWithRefs(rName string, tenantSpaceName string, tenantName string) string {
	return fmt.Sprintf(`
	resource "fusion_tenant_space" "%[1]s" {
		name	= "%[2]s"
		tenant	= fusion_tenant.%[3]s.name
	}
	`, rName, tenantSpaceName, tenantName)
}
