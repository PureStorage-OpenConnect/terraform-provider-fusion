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

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"
)

// Creates and destroys
func TestAccTenant_basic(t *testing.T) {
	rNameConfig := acctest.RandomWithPrefix("tenant_test")
	rName := "fusion_tenant." + rNameConfig
	tenantName := acctest.RandomWithPrefix("test_tenant")
	displayName := acctest.RandomWithPrefix("tenant-display-name")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckTenantDestroy,
		Steps: []resource.TestStep{
			// Create Tenant and validate it's fields
			{
				Config: testTenantConfig(rNameConfig, tenantName, displayName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(rName, "name", tenantName),
					resource.TestCheckResourceAttr(rName, "display_name", displayName),
					testTenantExists(rName),
				),
			},
		},
	})
}

// Updates display name
func TestAccTenant_update(t *testing.T) {
	rNameConfig := acctest.RandomWithPrefix("tenant_test")
	rName := "fusion_tenant." + rNameConfig
	tenantName := acctest.RandomWithPrefix("test_tenant")
	displayName1 := acctest.RandomWithPrefix("tenant-display-name-1")
	displayName2 := acctest.RandomWithPrefix("tenant-display-name-2")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckTenantDestroy,
		Steps: []resource.TestStep{
			// Create Tenant and validate it's fields
			{
				Config: testTenantConfig(rNameConfig, tenantName, displayName1),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(rName, "name", tenantName),
					resource.TestCheckResourceAttr(rName, "display_name", displayName1),
					testTenantExists(rName),
				),
			},
			// Update the display name, assert that the tf resource got updated, then assert the backend shows the same
			{
				Config: testTenantConfig(rNameConfig, tenantName, displayName2),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(rName, "display_name", displayName2),
					testTenantExists(rName),
				),
			},
			// Can't update certain values
			{
				Config:      testTenantConfig(rNameConfig, "immutable", displayName1),
				ExpectError: regexp.MustCompile("attempting to update an immutable field"),
			},
		},
	})
}

func TestAccTenant_attributes(t *testing.T) {
	rNameConfig := acctest.RandomWithPrefix("tenant_test")
	rName := "fusion_tenant." + rNameConfig
	tenantName := acctest.RandomWithPrefix("test_tenant")
	displayName := acctest.RandomWithPrefix("tenant-display-name")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckTenantDestroy,
		Steps: []resource.TestStep{
			// Missing required fields
			{
				Config:      testTenantConfig(rNameConfig, "", displayName),
				ExpectError: regexp.MustCompile(`Error: expected "name" to not be an empty string`),
			},
			{
				Config:      testTenantConfig(rNameConfig, "bad name", displayName),
				ExpectError: regexp.MustCompile("name must use alphanumeric characters"),
			},
			// Create without display_name then update
			{
				Config: testTenantConfigNoDisplayName(rNameConfig, tenantName),
				Check: resource.ComposeTestCheckFunc(
					testTenantExists(rName),
				),
			},
			{
				Config: testTenantConfig(rNameConfig, tenantName, displayName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(rName, "display_name", displayName),
					testTenantExists(rName),
				),
			},
		},
	})
}

func TestAccTenant_multiple(t *testing.T) {
	rNameConfig1 := acctest.RandomWithPrefix("tenant_test_1")
	rName1 := "fusion_tenant." + rNameConfig1
	tenantName1 := acctest.RandomWithPrefix("test_tenant_1")
	displayName1 := acctest.RandomWithPrefix("tenant-display-name-1")

	rNameConfig2 := acctest.RandomWithPrefix("tenant_test_2")
	rName2 := "fusion_tenant." + rNameConfig2
	tenantName2 := acctest.RandomWithPrefix("test_tenant_2")
	displayName2 := acctest.RandomWithPrefix("tenant-display-name-2")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckTenantDestroy,
		Steps: []resource.TestStep{
			// Sanity check two can be created at once
			{
				Config: testTenantConfig(rNameConfig1, tenantName1, displayName1) + "\n" +
					testTenantConfig(rNameConfig2, tenantName2, displayName2),
				Check: resource.ComposeTestCheckFunc(
					testTenantExists(rName1),
					testTenantExists(rName2),
				),
			},
			// Create two with same name
			{
				Config: testTenantConfig(rNameConfig1, tenantName1, displayName1) + "\n" +
					testTenantConfig(rNameConfig2, tenantName2, displayName2) + "\n" +
					testTenantConfig("conflictRN", tenantName1, "conflictDN"),
				ExpectError: regexp.MustCompile("already exists"),
			},
		},
	})
}

func testTenantExists(rName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		tfTenant, ok := s.RootModule().Resources[rName]
		if !ok {
			return fmt.Errorf("resource not found: %s", rName)
		}
		if tfTenant.Type != "fusion_tenant" {
			return fmt.Errorf("expected type: fusion_tenant. Found: %s", tfTenant.Type)
		}
		attrs := tfTenant.Primary.Attributes

		goclientTenant, _, err := testAccProvider.Meta().(*hmrest.APIClient).TenantsApi.GetTenant(context.Background(), attrs["name"], nil)
		if err != nil {
			return fmt.Errorf("go client retutrned error while searching for %s. Error: %s", attrs["name"], err)
		}
		if strings.Compare(goclientTenant.Name, attrs["name"]) != 0 ||
			strings.Compare(goclientTenant.DisplayName, attrs["display_name"]) != 0 {
			return fmt.Errorf("terraform tenant doesnt match goclients tenant")
		}
		return nil
	}
}

func testCheckTenantDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*hmrest.APIClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "fusion_tenant" {
			continue
		}

		attrs := rs.Primary.Attributes
		tenantName, ok := attrs["name"]
		if !ok {
			continue // Skip data sources
		}

		_, resp, err := client.TenantsApi.GetTenant(context.Background(), tenantName, nil)
		if err != nil && resp.StatusCode == http.StatusNotFound {
			continue
		}

		return fmt.Errorf("tenant may still exist. Expected response code 404, got code %d", resp.StatusCode)
	}
	return nil
}

func testTenantConfig(rName string, tenantName string, displayName string) string {
	return fmt.Sprintf(`
	resource "fusion_tenant" "%[1]s" {
		name          = "%[2]s"
		display_name  = "%[3]s"
	}
	`, rName, tenantName, displayName)
}

func testTenantConfigNoDisplayName(rName string, tenantName string) string {
	return fmt.Sprintf(`
	resource "fusion_tenant" "%[1]s" {
		name = "%[2]s"
	}
	`, rName, tenantName)
}
