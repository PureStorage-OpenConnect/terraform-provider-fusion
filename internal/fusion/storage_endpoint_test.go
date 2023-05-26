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

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"
)

// Creates and destroys
func TestAccStorageEndpoint_basic(t *testing.T) {
	rNameConfig := acctest.RandomWithPrefix("storage_endpoint_test")
	rName := "fusion_storage_endpoint." + rNameConfig
	displayName := acctest.RandomWithPrefix("se-display-name")
	storageEndpointName := acctest.RandomWithPrefix("test_se")

	nigName := acctest.RandomWithPrefix("nig_se_test")
	regionName := acctest.RandomWithPrefix("region_se_test")
	azName := acctest.RandomWithPrefix("se_az")

	regionConfig := testRegionConfigNoDisplayName(regionName, regionName)
	azConfig := testAvailabilityZoneConfigRef(azName, azName, regionName)
	nigsConfig := testNetworkInterfaceGroupConfigRef(nigName, nigName, azName, regionName, "10.21.200.1", "10.21.200.0/24")

	iscsi := []map[string]interface{}{
		{
			"address": "10.21.200.121/24",
		},
		{
			"address": "10.21.200.122/24",
			"gateway": "10.21.200.1",
		},
		{
			"address":                  "10.21.200.123/24",
			"network_interface_groups": []interface{}{nigName},
		},
		{
			"address":                  "10.21.200.124/24",
			"gateway":                  "10.21.200.1",
			"network_interface_groups": []interface{}{nigName},
		},
	}

	commonConfig := regionConfig + azConfig + nigsConfig

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckStorageEndpointDestroy,
		Steps: []resource.TestStep{
			// Create Storage Endpoint and validate it's fields
			{
				Config: commonConfig + testStorageEndpointConfig(rNameConfig, storageEndpointName, displayName, regionName, azName, iscsi),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(rName, "name", storageEndpointName),
					resource.TestCheckResourceAttr(rName, "display_name", displayName),
					resource.TestCheckResourceAttr(rName, "region", regionName),
					resource.TestCheckResourceAttr(rName, "availability_zone", azName),
					testCheckStorageEndpointIscsi(rName, iscsi),
					testStorageEndpointExists(rName, iscsi),
				),
			},
		},
	})
}

func TestAccStorageEndpoint_update(t *testing.T) {
	rNameConfig := acctest.RandomWithPrefix("storage_endpoint_test")
	rName := "fusion_storage_endpoint." + rNameConfig
	displayName := acctest.RandomWithPrefix("se-display-name")
	storageEndpointName := acctest.RandomWithPrefix("test_se")

	regionName := acctest.RandomWithPrefix("region_se_test")
	azName := acctest.RandomWithPrefix("se_az")

	regionConfig := testRegionConfigNoDisplayName(regionName, regionName)
	azConfig := testAvailabilityZoneConfigRef(azName, azName, regionName)

	iscsi := []map[string]interface{}{
		{
			"address": "10.21.200.121/24",
		},
	}

	commonConfig := regionConfig + azConfig

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckStorageEndpointDestroy,
		Steps: []resource.TestStep{
			// Create Storage Endpoint and validate it's fields
			{
				Config: commonConfig + testStorageEndpointConfigNoDisplayName(rNameConfig, storageEndpointName, regionName, azName, iscsi),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(rName, "name", storageEndpointName),
					resource.TestCheckResourceAttr(rName, "display_name", storageEndpointName),
					resource.TestCheckResourceAttr(rName, "region", regionName),
					resource.TestCheckResourceAttr(rName, "availability_zone", azName),
					testCheckStorageEndpointIscsi(rName, iscsi),
					testStorageEndpointExists(rName, iscsi),
				),
			},
			// Update display name
			{
				Config: commonConfig + testStorageEndpointConfig(rNameConfig, storageEndpointName, displayName, regionName, azName, iscsi),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(rName, "name", storageEndpointName),
					resource.TestCheckResourceAttr(rName, "display_name", displayName),
					resource.TestCheckResourceAttr(rName, "region", regionName),
					resource.TestCheckResourceAttr(rName, "availability_zone", azName),
					testCheckStorageEndpointIscsi(rName, iscsi),
					testStorageEndpointExists(rName, iscsi),
				),
			},
			// Can't update immutable fields
			{
				Config:      commonConfig + testStorageEndpointConfig(rNameConfig, "immutable", displayName, regionName, azName, iscsi),
				ExpectError: regexp.MustCompile("attempting to update an immutable field"),
			},
		},
	})
}

func TestAccStorageEndpoint_multiple(t *testing.T) {
	rNameConfig := acctest.RandomWithPrefix("storage_endpoint_test")
	rName := "fusion_storage_endpoint." + rNameConfig
	storageEndpointName := acctest.RandomWithPrefix("test_se")

	rNameConfig2 := acctest.RandomWithPrefix("storage_endpoint_test")
	rName2 := "fusion_storage_endpoint." + rNameConfig2
	storageEndpointName2 := acctest.RandomWithPrefix("test_se")

	displayName := acctest.RandomWithPrefix("se-display-name")

	regionName := acctest.RandomWithPrefix("region_se_test")
	azName := acctest.RandomWithPrefix("se_az")
	azName2 := acctest.RandomWithPrefix("se_az")

	regionConfig := testRegionConfigNoDisplayName(regionName, regionName)
	azConfig := testAvailabilityZoneConfigRef(azName, azName, regionName)
	azConfig2 := testAvailabilityZoneConfigRef(azName2, azName2, regionName)

	iscsi := []map[string]interface{}{
		{
			"address": "10.21.200.121/24",
		},
	}

	iscsi2 := []map[string]interface{}{
		{
			"address": "10.21.200.122/24",
			"gateway": "10.21.200.1",
		},
	}

	commonConfig := regionConfig + azConfig + azConfig2
	seConfig := testStorageEndpointConfig(rNameConfig, storageEndpointName, displayName, regionName, azName, iscsi)
	seConfig2 := testStorageEndpointConfig(rNameConfig2, storageEndpointName2, displayName, regionName, azName2, iscsi2)
	seConfig3 := testStorageEndpointConfig("conflictR", storageEndpointName, displayName, regionName, azName, iscsi2)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckStorageEndpointDestroy,
		Steps: []resource.TestStep{
			// Multiple SE can be created
			{
				Config: commonConfig + seConfig + seConfig2,
				Check: resource.ComposeTestCheckFunc(
					testStorageEndpointExists(rName, iscsi),
					testStorageEndpointExists(rName2, iscsi2),
				),
			},
			// Cannot create the same SE twice
			{
				Config:      commonConfig + seConfig + seConfig2 + seConfig3,
				ExpectError: regexp.MustCompile("already exists"),
			},
		},
	})
}

func testStorageEndpointExists(rName string, iscsi []map[string]interface{}) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		tfStorageEndpoint, ok := s.RootModule().Resources[rName]
		if !ok {
			return fmt.Errorf("resource not found: %s", rName)
		}

		if tfStorageEndpoint.Type != "fusion_storage_endpoint" {
			return fmt.Errorf("expected type: fusion_storage_endpoint. Found: %s", tfStorageEndpoint.Type)
		}

		attrs := tfStorageEndpoint.Primary.Attributes
		goclientStorageEndpoint, _, err := testAccProvider.Meta().(*hmrest.APIClient).StorageEndpointsApi.GetStorageEndpoint(
			context.Background(),
			attrs["region"],
			attrs["availability_zone"],
			attrs["name"],
			nil,
		)

		if err != nil {
			return fmt.Errorf("go client returned error while searching for %s. Error: %s", attrs["name"], err)
		}

		if strings.Compare(goclientStorageEndpoint.Name, attrs["name"]) != 0 ||
			strings.Compare(goclientStorageEndpoint.DisplayName, attrs["display_name"]) != 0 ||
			!testStorageEndpointMatchIscsi(iscsi, goclientStorageEndpoint.Iscsi.DiscoveryInterfaces) {
			return fmt.Errorf("terraform storage endpoint doesn't match goclients storage endpoint")
		}
		return nil
	}
}

func testCheckStorageEndpointDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*hmrest.APIClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "fusion_storage_endpoint" {
			continue
		}
		attrs := rs.Primary.Attributes

		storageEndpointName := attrs["name"]
		storageEndpointRegion := attrs["region"]
		storageEndpointAvailabilityZone := attrs["availability_zone"]

		_, resp, err := client.StorageEndpointsApi.GetStorageEndpoint(
			context.Background(),
			storageEndpointRegion,
			storageEndpointAvailabilityZone,
			storageEndpointName,
			nil,
		)
		if err != nil && resp.StatusCode == http.StatusNotFound {
			continue
		}

		return fmt.Errorf("storage endpoint may still exist. Expected response code 404, got code %d", resp.StatusCode)
	}
	return nil
}

func testStorageEndpointIscsiConfig(iscsiList []map[string]interface{}) string {
	iscsiStrings := make([]string, 0)

	for _, iscsi := range iscsiList {
		gatewayField := ""
		if gateway, ok := iscsi["gateway"]; ok {
			gatewayField = fmt.Sprintf("gateway = \"%s\"", gateway)
		}

		nigsField := ""
		if nigList, ok := iscsi["network_interface_groups"]; ok {
			nigs := make([]string, 0)

			for _, nig := range nigList.([]interface{}) {
				nigs = append(nigs, fmt.Sprintf("fusion_network_interface_group.%s.name", nig))
			}

			nigsField = fmt.Sprintf("network_interface_groups = [%s]", strings.Join(nigs, ","))
		}

		iscsiString := fmt.Sprintf(`
		iscsi {
			address = "%[1]s"
			%[2]s
			%[3]s
		}
		`, iscsi["address"], gatewayField, nigsField)

		iscsiStrings = append(iscsiStrings, iscsiString)
	}

	return strings.Join(iscsiStrings, "")
}

func testStorageEndpointConfig(
	rName, storageEndpointName, displayName, region, availabilityZone string, iscsi []map[string]interface{},
) string {
	return fmt.Sprintf(`
	resource "fusion_storage_endpoint" "%[1]s" {
		name				= "%[2]s"
		display_name		= "%[3]s"
		region				= fusion_region.%[4]s.name
		availability_zone	= fusion_availability_zone.%[5]s.name
		%[6]s
	}
	`, rName, storageEndpointName, displayName, region, availabilityZone, testStorageEndpointIscsiConfig(iscsi))
}

func testStorageEndpointConfigNoDisplayName(
	rName, storageEndpointName, region, availabilityZone string, iscsi []map[string]interface{},
) string {
	return fmt.Sprintf(`
	resource "fusion_storage_endpoint" "%[1]s" {
		name				= "%[2]s"
		region				= fusion_region.%[3]s.name
		availability_zone	= fusion_availability_zone.%[4]s.name
		%[5]s
	}
	`, rName, storageEndpointName, region, availabilityZone, testStorageEndpointIscsiConfig(iscsi))
}

func testCheckStorageEndpointIscsi(resourceName string, iscsis []map[string]interface{}) resource.TestCheckFunc {
	// Test length
	testChecks := []resource.TestCheckFunc{
		resource.TestCheckResourceAttr(resourceName, "iscsi.#", strconv.Itoa(len(iscsis))),
	}

	// Checks that correct iscsi blocks are present (ignores nig)
	for _, iscsi := range iscsis {
		iscsiCompareMap := make(map[string]string)

		for key, value := range iscsi {
			if strValue, ok := value.(string); ok {
				iscsiCompareMap[key] = strValue
			}
		}

		testChecks = append(testChecks, resource.TestCheckTypeSetElemNestedAttrs(resourceName, "iscsi.*", iscsiCompareMap))
	}

	return resource.ComposeAggregateTestCheckFunc(testChecks...)
}

func testStorageEndpointMatchIscsi(expectedIscsis []map[string]interface{}, iscsis []hmrest.StorageEndpointIscsiDiscoveryInterface) bool {
	if len(expectedIscsis) != len(iscsis) {
		return false
	}

	for _, expectedIscsi := range expectedIscsis {
		found := false

		for _, iscsi := range iscsis {
			if iscsi.Address != expectedIscsi["address"] {
				continue
			}

			if v, ok := expectedIscsi["gateway"]; ok && v != iscsi.Gateway {
				continue
			}

			if v, ok := expectedIscsi["network_interface_groups"]; ok {
				names := make([]interface{}, 0)
				for _, nig := range iscsi.NetworkInterfaceGroups {
					names = append(names, nig.Name)
				}

				expectedSet := schema.NewSet(schema.HashString, v.([]interface{}))
				actualSet := schema.NewSet(schema.HashString, names)

				if !expectedSet.Equal(actualSet) {
					continue
				}
			}

			found = true
			break
		}

		if !found {
			return false
		}
	}

	return true
}
