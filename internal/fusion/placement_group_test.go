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

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"
)

const (
	region_name            = "pure-us-west"
	availability_zone_name = "az1"
)

func TestAccPlacementGroup_basic(t *testing.T) {
	rNameConfig := acctest.RandomWithPrefix("placementgroup")
	rName := "fusion_placement_group." + rNameConfig
	displayName := acctest.RandomWithPrefix("placement-group-display-name")
	placementGroupName := acctest.RandomWithPrefix("test_pg")
	tsName := acctest.RandomWithPrefix("ts-pgtest")

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckPGDestroy,
		Steps: []resource.TestStep{
			// Create placement group
			{
				Config: testPGConfigWithTS("", rNameConfig, placementGroupName, displayName, region_name, availability_zone_name, testAccStorageService, tsName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(rName, "name", placementGroupName),
					resource.TestCheckResourceAttr(rName, "display_name", displayName),
					resource.TestCheckResourceAttr(rName, "region_name", region_name),
					resource.TestCheckResourceAttr(rName, "availability_zone_name", availability_zone_name),
					resource.TestCheckResourceAttr(rName, "storage_service_name", testAccStorageService),
					testPlacementGroupExists(rName, tsName),
				),
			},
		},
	})
}

func TestAccPlacementGroup_EmptyAttributeValues(t *testing.T) {
	rNameConfig := acctest.RandomWithPrefix("placementgroup")
	displayName := acctest.RandomWithPrefix("placement-group-display-name")
	placementGroupName := acctest.RandomWithPrefix("test_pg")
	tsName := acctest.RandomWithPrefix("ts-pgtest")

	tenant := acctest.RandomWithPrefix("tenant")
	commonConfig := testTenantConfig(tenant, tenant, tenant) +
		testTenantSpaceConfigWithRefs("ts", "ts_dn", tsName, tenant)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckPGDestroy,
		Steps: []resource.TestStep{
			// storage_service_name is empty
			{
				Config:      commonConfig + testPGConfig("", rNameConfig, placementGroupName, displayName, region_name, availability_zone_name, "", false),
				ExpectError: regexp.MustCompile("Error: storage_service must be specified"),
			},
			// region_name is empty
			{
				Config:      commonConfig + testPGConfig("", rNameConfig, placementGroupName, displayName, "", availability_zone_name, testAccStorageService, false),
				ExpectError: regexp.MustCompile("Error: region must be specified"),
			},
			// availability_zone_name is empty
			{
				Config:      commonConfig + testPGConfig("", rNameConfig, placementGroupName, displayName, region_name, "", testAccStorageService, false),
				ExpectError: regexp.MustCompile("Error: availability_zone must be specified"),
			},
		},
	})
}

func TestAccPlacementGroup_MissingAttributes(t *testing.T) {
	rNameConfig := acctest.RandomWithPrefix("placementgroup")
	displayName := acctest.RandomWithPrefix("placement-group-display-name")
	placementGroupName := acctest.RandomWithPrefix("test_pg")
	tsName := acctest.RandomWithPrefix("ts-pgtest")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckPGDestroy,
		Steps: []resource.TestStep{
			// storage_service_name is missing
			{
				Config:      testPGConfigWithTS("storage_service_name", rNameConfig, placementGroupName, displayName, region_name, availability_zone_name, "", tsName),
				ExpectError: regexp.MustCompile(`The argument "storage_service_name" is required, but no definition was found`),
			},
			// region_name is missing
			{
				Config:      testPGConfigWithTS("region_name", rNameConfig, placementGroupName, displayName, "", availability_zone_name, testAccStorageService, tsName),
				ExpectError: regexp.MustCompile(`The argument "region_name" is required, but no definition was found`),
			},
			// availability_zone_name is missing
			{
				Config:      testPGConfigWithTS("availability_zone_name", rNameConfig, placementGroupName, displayName, region_name, "", testAccStorageService, tsName),
				ExpectError: regexp.MustCompile(`The argument "availability_zone_name" is required, but no definition was`),
			},
		},
	})
}
func testPGConfig(skipAttribute string, pgName string, placementGroupName string, displayName string, regionName string, availabilityZone string, storageService string, destroySnap bool) string {
	resourceConfiguration := fmt.Sprintf(`
	resource "fusion_placement_group" "%[1]s" {
		name                         = "%[2]s"
		display_name                 = "%[3]s"
		tenant_space_name            = fusion_tenant_space.ts.name
		tenant_name                  = fusion_tenant_space.ts.tenant
		region_name                  = "%[4]s"
		availability_zone_name       = "%[5]s"
		storage_service_name         = "%[6]s"
        destroy_snapshots_on_delete  = "%[7]t"
	}
	`, pgName, placementGroupName, displayName, regionName, availabilityZone, storageService, destroySnap)

	if skipAttribute == "" {
		return resourceConfiguration
	}
	newConfiguration := ""
	regexPattern := regexp.MustCompile("^" + skipAttribute)
	for _, stringLine := range strings.Split(resourceConfiguration, "\n") {
		matched := regexPattern.MatchString(strings.TrimSpace(stringLine))
		if matched {
			// Do not include attribute into a new configuration we want to skip
			continue
		}
		newConfiguration += stringLine + "\n"
	}
	return newConfiguration
}

// Defaults tenant_space_name and destroy_snapshots_on_delete
func testPGConfigWithTS(skipAttribute string, pgName string, placementGroupName string, displayName string, regionName string, availabilityZone string, storageService string, tsName string) string {
	tenant := acctest.RandomWithPrefix("tenant")
	resourceConfiguration := testTenantConfigNoDisplayName(tenant, tenant) +
		testTenantSpaceConfigWithRefs("ts", "ts_dn", tsName, tenant) +
		testPGConfig(skipAttribute, pgName, placementGroupName, displayName, regionName, availabilityZone, storageService, false)
	return resourceConfiguration
}

func testPlacementGroupExists(rName string, tsName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		tfPlacementGroup, ok := s.RootModule().Resources[rName]
		if !ok {
			return fmt.Errorf("Resource not found: %s", rName)
		}
		if tfPlacementGroup.Type != "fusion_placement_group" {
			return fmt.Errorf("Expected type: fusion_placement_group. Found: %s", tfPlacementGroup.Type)
		}
		attrs := tfPlacementGroup.Primary.Attributes

		goclientPlacementGroup, _, err := testAccProvider.Meta().(*hmrest.APIClient).PlacementGroupsApi.GetPlacementGroup(context.Background(), attrs["tenant_name"], tsName, attrs["name"], nil)
		if err != nil {
			return fmt.Errorf("Go client retutrned error while searching for %s. Error: %s", attrs["name"], err)
		}
		if strings.Compare(goclientPlacementGroup.Name, attrs["name"]) != 0 ||
			strings.Compare(goclientPlacementGroup.DisplayName, attrs["display_name"]) != 0 {
			return fmt.Errorf("Terraform placement group doesnt match goclients placement group")
		}
		return nil
	}
}

func testCheckPGDestroy(s *terraform.State) error {

	client := testAccProvider.Meta().(*hmrest.APIClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "fusion_placement_group" {
			continue
		}
		attrs := rs.Primary.Attributes

		tenant := attrs["tenant"]
		ts := attrs["tenant_space"]
		name := attrs["name"]

		_, resp, err := client.PlacementGroupsApi.GetPlacementGroup(context.Background(), tenant, ts, name, nil)
		if err != nil && resp.StatusCode == http.StatusNotFound {
			continue // the PG was destroyed
		} else {
			return fmt.Errorf("placement group may still exist. Expected response code 404, got code %d", resp.StatusCode)
		}
	}
	return nil
}
