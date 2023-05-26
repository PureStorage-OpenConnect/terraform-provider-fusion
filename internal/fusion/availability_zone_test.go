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
func TestAccAvailabilityZone_basic(t *testing.T) {
	rNameConfig := acctest.RandomWithPrefix("az_test")
	rName := "fusion_availability_zone." + rNameConfig
	displayName1 := acctest.RandomWithPrefix("az-display-name")
	azName := acctest.RandomWithPrefix("az-test")
	region := acctest.RandomWithPrefix("az_test_region")

	commonConfig := testRegionConfig(region, region, region)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckAvailabilityZoneDestroy,
		Steps: []resource.TestStep{
			// Create Availability Zone and validate it's fields
			{
				Config: commonConfig + testAvailabilityZoneConfig(rNameConfig, azName, displayName1, region),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(rName, "name", azName),
					resource.TestCheckResourceAttr(rName, "display_name", displayName1),
					resource.TestCheckResourceAttr(rName, "region", region),
					testAvailabilityZoneExists(rName),
				),
			},
		},
	})
}

func TestAccAvailabilityZone_update(t *testing.T) {
	rNameConfig := acctest.RandomWithPrefix("az_test")
	rName := "fusion_availability_zone." + rNameConfig
	displayName1 := acctest.RandomWithPrefix("az-display-name")
	azName := acctest.RandomWithPrefix("test_ts")
	region := acctest.RandomWithPrefix("az_test_region")

	commonConfig := testRegionConfig(region, region, region)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckAvailabilityZoneDestroy,
		Steps: []resource.TestStep{
			// Create AZ and validate its fields
			{
				Config: commonConfig + testAvailabilityZoneConfig(rNameConfig, azName, displayName1, region),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(rName, "name", azName),
					resource.TestCheckResourceAttr(rName, "display_name", displayName1),
					resource.TestCheckResourceAttr(rName, "region", region),
					testAvailabilityZoneExists(rName),
				),
			},
			// AZ does not support update
			{
				Config:      commonConfig + testAvailabilityZoneConfig(rNameConfig, "immutable", displayName1, region),
				ExpectError: regexp.MustCompile("unsupported operation: update"),
			},
			// TODO: Remove this step once the HM-5438 bug is resolved
			{
				Config: commonConfig + testAvailabilityZoneConfig(rNameConfig, azName, displayName1, region),
			},
		},
	})
}

func TestAccAvailabilityZone_multiple(t *testing.T) {
	rNameConfig := acctest.RandomWithPrefix("az_test")
	rName := "fusion_availability_zone." + rNameConfig
	displayName1 := acctest.RandomWithPrefix("az-display-name")
	azName := acctest.RandomWithPrefix("az-name")

	rNameConfig2 := acctest.RandomWithPrefix("az_test2")
	rName2 := "fusion_availability_zone." + rNameConfig
	displayName2 := acctest.RandomWithPrefix("az-display-name")
	azName2 := acctest.RandomWithPrefix("az-name")
	region := acctest.RandomWithPrefix("az_test_region")

	commonConfig := testRegionConfig(region, region, region)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckAvailabilityZoneDestroy,
		Steps: []resource.TestStep{
			// Sanity check two can be created at once
			{
				Config: commonConfig +
					testAvailabilityZoneConfig(rNameConfig, azName, displayName1, region) + "\n" +
					testAvailabilityZoneConfigNoDisplayName(rNameConfig2, azName2, region),
				Check: resource.ComposeTestCheckFunc(
					testAvailabilityZoneExists(rName),
					testAvailabilityZoneExists(rName2),
				),
			},
			// Create two with same name
			{
				Config: commonConfig +
					testAvailabilityZoneConfig(rNameConfig, azName, displayName1, region) + "\n" +
					testAvailabilityZoneConfig(rNameConfig2, azName2, displayName2, region) + "\n" +
					testAvailabilityZoneConfig("conflictRN", azName, "conflictDN", region),
				ExpectError: regexp.MustCompile("already exists"),
			},
		},
	})
}

func testAvailabilityZoneExists(rName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		tfAvailabilityZone, ok := s.RootModule().Resources[rName]
		if !ok {
			return fmt.Errorf("resource not found: %s", rName)
		}
		if tfAvailabilityZone.Type != "fusion_availability_zone" {
			return fmt.Errorf("expected type: fusion_availability_zone. Found: %s", tfAvailabilityZone.Type)
		}
		attrs := tfAvailabilityZone.Primary.Attributes

		goclientAvailabilityZone, _, err := testAccProvider.Meta().(*hmrest.APIClient).AvailabilityZonesApi.GetAvailabilityZoneById(context.Background(), attrs["id"], nil)
		if err != nil {
			return fmt.Errorf("go client returned error while searching for %s. Error: %s", attrs["name"], err)
		}
		if strings.Compare(goclientAvailabilityZone.Name, attrs["name"]) != 0 ||
			strings.Compare(goclientAvailabilityZone.DisplayName, attrs["display_name"]) != 0 ||
			strings.Compare(goclientAvailabilityZone.Region.Name, attrs["region"]) != 0 {
			return fmt.Errorf("terraform availability one doesn't match goclients availability zone")
		}
		return nil
	}
}

func testCheckAvailabilityZoneDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*hmrest.APIClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "fusion_availability_zone" {
			continue
		}
		attrs := rs.Primary.Attributes

		regionName := attrs["region"]
		azName := attrs["name"]

		_, resp, err := client.AvailabilityZonesApi.GetAvailabilityZone(context.Background(), regionName, azName, nil)
		if err != nil && resp.StatusCode == http.StatusNotFound {
			continue
		}

		return fmt.Errorf("availability zone may still exist. Expected response code 404, got code %d", resp.StatusCode)
	}
	return nil
}

func testAvailabilityZoneConfig(rName, azName, displayName, region string) string {
	return fmt.Sprintf(`
	resource "fusion_availability_zone" "%[1]s" {
		name			= "%[2]s"
		display_name	= "%[3]s"
		region			= fusion_region.%[4]s.name
	}
	`, rName, azName, displayName, region)
}

func testAvailabilityZoneConfigNoDisplayName(rName, azName, region string) string {
	return fmt.Sprintf(`
	resource "fusion_availability_zone" "%[1]s" {
		name	= "%[2]s"
		region	= fusion_region.%[3]s.name
	}
	`, rName, azName, region)
}

func testAvailabilityZoneConfigRef(rName, azName, region string) string {
	return fmt.Sprintf(`
	resource "fusion_availability_zone" "%[1]s" {
		name	= "%[2]s"
		region	= fusion_region.%[3]s.name
	}
	`, rName, azName, region)
}
