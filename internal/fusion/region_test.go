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

// Creates and destroys
func TestAccRegion_basic(t *testing.T) {
	rNameConfig := acctest.RandomWithPrefix("region_test")
	rName := "fusion_region." + rNameConfig
	displayName1 := acctest.RandomWithPrefix("region-display-name")
	regionName := acctest.RandomWithPrefix("test_region")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckRegionDestroy,
		Steps: []resource.TestStep{
			// Create Region and validate it's fields
			{
				Config: testRegionConfig(rNameConfig, regionName, displayName1),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(rName, "name", regionName),
					resource.TestCheckResourceAttr(rName, "display_name", displayName1),
					testRegionExists(rName),
				),
			},
		},
	})
}

// Updates display name
func TestAccRegion_update(t *testing.T) {
	rNameConfig := acctest.RandomWithPrefix("region_test")
	rName := "fusion_region." + rNameConfig
	displayName1 := acctest.RandomWithPrefix("region-display-name")
	displayName2 := acctest.RandomWithPrefix("region-display-name2")
	regionName := acctest.RandomWithPrefix("test_region")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckRegionDestroy,
		Steps: []resource.TestStep{
			// Create Region and validate it's fields
			{
				Config: testRegionConfig(rNameConfig, regionName, displayName1),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(rName, "name", regionName),
					resource.TestCheckResourceAttr(rName, "display_name", displayName1),
					testRegionExists(rName),
				),
			},
			// Update the display name, assert that the tf resource got updated, then assert the backend shows the same
			{
				Config: testRegionConfig(rNameConfig, regionName, displayName2),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(rName, "display_name", displayName2),
					testRegionExists(rName),
				),
			},
			//Can't update certain values
			{
				Config:      testRegionConfig(rNameConfig, "immutable", displayName1),
				ExpectError: regexp.MustCompile("attempting to update an immutable field"),
			},
		},
	})
}

func TestAccRegion_attributes(t *testing.T) {
	rNameConfig := acctest.RandomWithPrefix("region_test")
	rName := "fusion_region." + rNameConfig
	displayName1 := acctest.RandomWithPrefix("region-display-name")
	regionName := acctest.RandomWithPrefix("region-name")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckRegionDestroy,
		Steps: []resource.TestStep{
			{
				Config:      testRegionConfig(rNameConfig, "bad name here", displayName1),
				ExpectError: regexp.MustCompile("name must use alphanumeric characters"),
			},
			// Create without display_name then update
			{
				Config: testRegionConfigNoDisplayName(rNameConfig, regionName),
				Check: resource.ComposeTestCheckFunc(
					testRegionExists(rName),
				),
			},
			{
				Config: testRegionConfig(rNameConfig, regionName, displayName1),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(rName, "display_name", displayName1),
					testRegionExists(rName),
				),
			},
		},
	})
}

func TestAccRegion_multiple(t *testing.T) {
	rNameConfig := acctest.RandomWithPrefix("region_test")
	rName := "fusion_region." + rNameConfig
	displayName1 := acctest.RandomWithPrefix("region-display-name")
	regionName := acctest.RandomWithPrefix("region-name")

	rNameConfig2 := acctest.RandomWithPrefix("region_test2")
	rName2 := "fusion_region." + rNameConfig
	displayName2 := acctest.RandomWithPrefix("region-display-name")
	regionName2 := acctest.RandomWithPrefix("region-name")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckRegionDestroy,
		Steps: []resource.TestStep{
			// Sanity check two can be created at once
			{
				Config: testRegionConfig(rNameConfig, regionName, displayName1) + "\n" +
					testRegionConfig(rNameConfig2, regionName2, displayName2),
				Check: resource.ComposeTestCheckFunc(
					testRegionExists(rName),
					testRegionExists(rName2),
				),
			},
			// Create two with same name
			{
				Config: testRegionConfig(rNameConfig, regionName, displayName1) + "\n" +
					testRegionConfig(rNameConfig2, regionName2, displayName2) + "\n" +
					testRegionConfig("conflictRN", regionName, "conflictDN"),
				ExpectError: regexp.MustCompile("already exists"),
			},
		},
	})
}

func testRegionExists(rName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		tfRegion, ok := s.RootModule().Resources[rName]
		if !ok {
			return fmt.Errorf("resource not found: %s", rName)
		}
		if tfRegion.Type != "fusion_region" {
			return fmt.Errorf("expected type: fusion_region. Found: %s", tfRegion.Type)
		}
		attrs := tfRegion.Primary.Attributes

		goclientRegion, _, err := testAccProvider.Meta().(*hmrest.APIClient).RegionsApi.GetRegion(context.Background(), attrs["name"], nil)
		if err != nil {
			return fmt.Errorf("go client returned error while searching for %s. Error: %s", attrs["name"], err)
		}
		if strings.Compare(goclientRegion.Name, attrs["name"]) != 0 ||
			strings.Compare(goclientRegion.DisplayName, attrs["display_name"]) != 0 {
			return fmt.Errorf("terraform region doesn't match goclients region")
		}
		return nil
	}
}

func testCheckRegionDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*hmrest.APIClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "fusion_region" {
			continue
		}
		attrs := rs.Primary.Attributes
		regionName, ok := attrs["name"]
		if !ok {
			continue // Skip data sources
		}

		_, resp, err := client.RegionsApi.GetRegion(context.Background(), regionName, nil)
		if err != nil && resp.StatusCode == http.StatusNotFound {
			continue
		} else {
			return fmt.Errorf("region may still exist. Expected response code 404, got code %d", resp.StatusCode)
		}
	}
	return nil
}

func testRegionConfig(rName string, regionName string, displayName string) string {
	return fmt.Sprintf(`
	resource "fusion_region" "%[1]s" {
		name          = "%[2]s"
		display_name  = "%[3]s"
	}
	`, rName, regionName, displayName)
}

func testRegionConfigNoDisplayName(rName string, regionName string) string {
	return fmt.Sprintf(`
	resource "fusion_region" "%[1]s" {
		name          = "%[2]s"
	}
	`, rName, regionName)
}
