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
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"
)

// Creates and destroys
func TestAccStorageService_basic(t *testing.T) {
	rNameConfig := acctest.RandomWithPrefix("storage_service_test")
	rName := "fusion_storage_service." + rNameConfig
	displayName1 := acctest.RandomWithPrefix("storage-service-display-name")
	storageServiceName := acctest.RandomWithPrefix("test_ss")
	hardwareTypes := []string{"flash-array-x-optane", "flash-array-x"}

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckStorageServiceDestroy,
		Steps: []resource.TestStep{
			// Create Storage Service and validate it's fields
			{
				Config: testStorageServiceConfig(rNameConfig, storageServiceName, displayName1, hardwareTypes),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(rName, "name", storageServiceName),
					resource.TestCheckResourceAttr(rName, "display_name", displayName1),
					testCheckStorageServiceHardwareTypes(rName, hardwareTypes),
					testStorageServiceExists(rName),
				),
			},
		},
	})
}

// Updates display name
func TestAccStorageService_update(t *testing.T) {
	rNameConfig := acctest.RandomWithPrefix("storage_service_test")
	rName := "fusion_storage_service." + rNameConfig
	displayName1 := acctest.RandomWithPrefix("storage-service-display-name")
	displayName2 := acctest.RandomWithPrefix("storage-service-display-name2")
	storageServiceName := acctest.RandomWithPrefix("test_ss")
	hardwareTypes := []string{"flash-array-xl"}

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckStorageServiceDestroy,
		Steps: []resource.TestStep{
			// Create Storage Service and validate it's fields
			{
				Config: testStorageServiceConfig(rNameConfig, storageServiceName, displayName1, hardwareTypes),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(rName, "name", storageServiceName),
					resource.TestCheckResourceAttr(rName, "display_name", displayName1),
					testStorageServiceExists(rName),
				),
			},
			// Update the display name, assert that the tf resource got updated, then assert the backend shows the same
			{
				Config: testStorageServiceConfig(rNameConfig, storageServiceName, displayName2, hardwareTypes),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(rName, "display_name", displayName2),
					testStorageServiceExists(rName),
				),
			},
			//Can't update certain values
			{
				Config:      testStorageServiceConfig(rNameConfig, "immutable", displayName2, hardwareTypes),
				ExpectError: regexp.MustCompile("attempting to update an immutable field"),
			},
		},
	})
}

func TestAccStorageService_attributes(t *testing.T) {
	rNameConfig := acctest.RandomWithPrefix("storage_service_test")
	rName := "fusion_storage_service." + rNameConfig
	displayName1 := acctest.RandomWithPrefix("storage-service-display-name")
	storageServiceName := acctest.RandomWithPrefix("storage-service-name")
	hardwareTypes := []string{"flash-array-x"}

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckStorageServiceDestroy,
		Steps: []resource.TestStep{
			// TODO: Do name validations in the schema
			{
				Config:      testStorageServiceConfig(rNameConfig, "bad name here", displayName1, hardwareTypes),
				ExpectError: regexp.MustCompile("name must use alphanumeric characters"),
			},
			// Create without display_name then update
			{
				Config: testStorageServiceConfigNoDisplayName(rNameConfig, storageServiceName, hardwareTypes),
				Check: resource.ComposeTestCheckFunc(
					testStorageServiceExists(rName),
				),
			},
			{
				Config: testStorageServiceConfig(rNameConfig, storageServiceName, displayName1, hardwareTypes),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(rName, "display_name", displayName1),
					testStorageServiceExists(rName),
				),
			},
		},
	})
}

func TestAccStorageService_multiple(t *testing.T) {
	rNameConfig := acctest.RandomWithPrefix("storage_service_test")
	rName := "fusion_storage_service." + rNameConfig
	displayName1 := acctest.RandomWithPrefix("storage-service-display-name")
	storageServiceName := acctest.RandomWithPrefix("storage-service-name")

	rNameConfig2 := acctest.RandomWithPrefix("storage_service_test2")
	rName2 := "fusion_storage_service." + rNameConfig
	displayName2 := acctest.RandomWithPrefix("storage-service-display-name")
	storageServiceName2 := acctest.RandomWithPrefix("storage-service-name")

	hardware_types := []string{"flash-array-c"}

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckStorageServiceDestroy,
		Steps: []resource.TestStep{
			// Sanity check two can be created at once
			{
				Config: testStorageServiceConfig(rNameConfig, storageServiceName, displayName1, hardware_types) + "\n" +
					testStorageServiceConfig(rNameConfig2, storageServiceName2, displayName2, hardware_types),
				Check: resource.ComposeTestCheckFunc(
					testStorageServiceExists(rName),
					testStorageServiceExists(rName2),
				),
			},
			// Create two with same name
			{
				Config: testStorageServiceConfig(rNameConfig, storageServiceName, displayName1, hardware_types) + "\n" +
					testStorageServiceConfig(rNameConfig2, storageServiceName2, displayName2, hardware_types) + "\n" +
					testStorageServiceConfig("conflictRN", storageServiceName, "conflictDN", hardware_types),
				ExpectError: regexp.MustCompile("already exists"),
			},
		},
	})
}

func testStorageServiceExists(rName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		tfStorageService, ok := s.RootModule().Resources[rName]
		if !ok {
			return fmt.Errorf("resource not found: %s", rName)
		}
		if tfStorageService.Type != "fusion_storage_service" {
			return fmt.Errorf("expected type: fusion_storage_service. Found: %s", tfStorageService.Type)
		}
		attrs := tfStorageService.Primary.Attributes

		goclientStorageService, _, err := testAccProvider.Meta().(*hmrest.APIClient).StorageServicesApi.GetStorageService(context.Background(), attrs["name"], nil)
		if err != nil {
			return fmt.Errorf("go client returned error while searching for %s. Error: %s", attrs["name"], err)
		}

		if strings.Compare(goclientStorageService.Name, attrs["name"]) != 0 ||
			strings.Compare(goclientStorageService.DisplayName, attrs["display_name"]) != 0 {
			return fmt.Errorf("terraform storage service doesn't match goclients storage service")
		}
		return nil
	}
}

func testCheckStorageServiceDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*hmrest.APIClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "fusion_storage_service" {
			continue
		}
		attrs := rs.Primary.Attributes
		storageServiceName, ok := attrs["name"]
		if !ok {
			continue // Skip data sources
		}

		_, resp, err := client.StorageServicesApi.GetStorageService(context.Background(), storageServiceName, nil)
		if err != nil && resp.StatusCode == http.StatusNotFound {
			continue
		} else {
			return fmt.Errorf("storage service may still exist. Expected response code 404, got code %d", resp.StatusCode)
		}
	}
	return nil
}

func testStorageServiceConfig(rName string, storageServiceName string, displayName string, hardwareTypes []string) string {
	return fmt.Sprintf(`
	resource "fusion_storage_service" "%[1]s" {
		name      		= "%[2]s"
		display_name    = "%[3]s"
		hardware_types	= [%[4]s]
	}
	`, rName, storageServiceName, displayName, testStorageServiceGetHardwareTypeString(hardwareTypes))
}

func testStorageServiceConfigNoDisplayName(rName string, storageServiceName string, hardwareTypes []string) string {
	return fmt.Sprintf(`
	resource "fusion_storage_service" "%[1]s" {
		name          	= "%[2]s"
		hardware_types	= [%[3]s]
	}
	`, rName, storageServiceName, testStorageServiceGetHardwareTypeString(hardwareTypes))
}

func testStorageServiceGetHardwareTypeString(hardwareTypes []string) string {
	quotedHW := make([]string, len(hardwareTypes))

	for i, hwType := range hardwareTypes {
		quotedHW[i] = fmt.Sprintf("\"%s\"", hwType)
	}

	return strings.Join(quotedHW, ",")
}

func testCheckStorageServiceHardwareTypes(resourceName string, hardwareTypes []string) resource.TestCheckFunc {
	// Test length
	testChecks := []resource.TestCheckFunc{
		resource.TestCheckResourceAttr(resourceName, "hardware_types.#", strconv.Itoa(len(hardwareTypes))),
	}

	// Test that all hw types are present
	for _, hwType := range hardwareTypes {
		testChecks = append(testChecks, resource.TestCheckTypeSetElemAttr(resourceName, "hardware_types.*", hwType))
	}

	return resource.ComposeAggregateTestCheckFunc(testChecks...)
}
