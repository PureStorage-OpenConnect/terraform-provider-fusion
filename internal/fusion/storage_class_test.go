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

	"github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/utilities"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"
)

var (
	testBandwidthLimit = 1048576
	testIopsLimit      = 100
	testSizeLimit      = 1048576
)

// Creates and destroys
func TestAccStorageClass_basic(t *testing.T) {
	rNameConfig := acctest.RandomWithPrefix("storage_class_test")
	rName := "fusion_storage_class." + rNameConfig
	displayName := acctest.RandomWithPrefix("storage-class-display-name")
	storageClassName := acctest.RandomWithPrefix("storage-class-name")
	storageServiceName := testAccStorageService

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckStorageClassDestroy,
		Steps: []resource.TestStep{
			// Create Storage Class and validate it's fields
			{
				Config: testStorageClassConfig(rNameConfig, storageClassName, displayName, storageServiceName, testSizeLimit, testIopsLimit, testBandwidthLimit),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(rName, "name", storageClassName),
					resource.TestCheckResourceAttr(rName, "display_name", displayName),
					resource.TestCheckResourceAttr(rName, "storage_service", storageServiceName),
					resource.TestCheckResourceAttr(rName, "size_limit", strconv.Itoa(testSizeLimit)),
					resource.TestCheckResourceAttr(rName, "iops_limit", strconv.Itoa(testIopsLimit)),
					resource.TestCheckResourceAttr(rName, "bandwidth_limit", strconv.Itoa(testBandwidthLimit)),
					testStorageClassExists(rName),
				),
			},
		},
	})
}

func TestAccStorageClass_units(t *testing.T) {
	rNameConfig := acctest.RandomWithPrefix("storage_class_test")
	rName := "fusion_storage_class." + rNameConfig
	displayName1 := acctest.RandomWithPrefix("storage-class-display-name")
	displayName2 := acctest.RandomWithPrefix("storage-class-display-name")
	storageClassName := acctest.RandomWithPrefix("storage-class-name")
	storageServiceName := testAccStorageService
	unitsBandwidthLimit := "100G"
	unitsIopsLimit := "100K"
	unitsSizeLimit := "1G"
	numericBandwidthLimit := 100 * 1024 * 1024 * 1024
	numericIopsLimit := 100 * 1000
	numericSizeLimit := 1 * 1024 * 1024 * 1024

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckStorageClassDestroy,
		Steps: []resource.TestStep{
			// Create Storage Class and validate it's fields
			{
				Config: testStorageClassConfigWithUnits(rNameConfig, storageClassName, displayName1, storageServiceName, unitsSizeLimit, unitsIopsLimit, unitsBandwidthLimit),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(rName, "name", storageClassName),
					resource.TestCheckResourceAttr(rName, "display_name", displayName1),
					resource.TestCheckResourceAttr(rName, "storage_service", storageServiceName),
					resource.TestCheckResourceAttr(rName, "size_limit", strconv.Itoa(numericSizeLimit)),
					resource.TestCheckResourceAttr(rName, "iops_limit", strconv.Itoa(numericIopsLimit)),
					resource.TestCheckResourceAttr(rName, "bandwidth_limit", strconv.Itoa(numericBandwidthLimit)),
					testStorageClassExists(rName),
				),
			},
			// Check if update works with numeric values
			{
				Config: testStorageClassConfig(rNameConfig, storageClassName, displayName2, storageServiceName, numericSizeLimit, numericIopsLimit, numericBandwidthLimit),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(rName, "name", storageClassName),
					resource.TestCheckResourceAttr(rName, "display_name", displayName2),
					resource.TestCheckResourceAttr(rName, "storage_service", storageServiceName),
					resource.TestCheckResourceAttr(rName, "size_limit", strconv.Itoa(numericSizeLimit)),
					resource.TestCheckResourceAttr(rName, "iops_limit", strconv.Itoa(numericIopsLimit)),
					resource.TestCheckResourceAttr(rName, "bandwidth_limit", strconv.Itoa(numericBandwidthLimit)),
					testStorageClassExists(rName),
				),
			},
			// Check if update works with unit values
			{
				Config: testStorageClassConfigWithUnits(rNameConfig, storageClassName, displayName1, storageServiceName, unitsSizeLimit, unitsIopsLimit, unitsBandwidthLimit),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(rName, "name", storageClassName),
					resource.TestCheckResourceAttr(rName, "display_name", displayName1),
					resource.TestCheckResourceAttr(rName, "storage_service", storageServiceName),
					resource.TestCheckResourceAttr(rName, "size_limit", strconv.Itoa(numericSizeLimit)),
					resource.TestCheckResourceAttr(rName, "iops_limit", strconv.Itoa(numericIopsLimit)),
					resource.TestCheckResourceAttr(rName, "bandwidth_limit", strconv.Itoa(numericBandwidthLimit)),
					testStorageClassExists(rName),
				),
			},
		},
	})
}

// Updates display name
func TestAccStorageClass_update(t *testing.T) {
	rNameConfig := acctest.RandomWithPrefix("storage_class_test")
	rName := "fusion_storage_class." + rNameConfig
	displayName1 := acctest.RandomWithPrefix("storage-class-display-name")
	displayName2 := acctest.RandomWithPrefix("storage-class-display-name")
	storageClassName := acctest.RandomWithPrefix("storage-class-name")
	storageServiceName := testAccStorageService

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckStorageClassDestroy,
		Steps: []resource.TestStep{
			// Create Storage Class and validate it's fields
			{
				Config: testStorageClassConfig(rNameConfig, storageClassName, displayName1, storageServiceName, testSizeLimit, testIopsLimit, testBandwidthLimit),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(rName, "name", storageClassName),
					resource.TestCheckResourceAttr(rName, "display_name", displayName1),
					resource.TestCheckResourceAttr(rName, "storage_service", storageServiceName),
					resource.TestCheckResourceAttr(rName, "size_limit", strconv.Itoa(testSizeLimit)),
					resource.TestCheckResourceAttr(rName, "iops_limit", strconv.Itoa(testIopsLimit)),
					resource.TestCheckResourceAttr(rName, "bandwidth_limit", strconv.Itoa(testBandwidthLimit)),
					testStorageClassExists(rName),
				),
			},
			// Update the display name, assert that the tf resource got updated, then assert the backend shows the same
			{
				Config: testStorageClassConfig(rNameConfig, storageClassName, displayName2, storageServiceName, testSizeLimit, testIopsLimit, testBandwidthLimit),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(rName, "name", storageClassName),
					resource.TestCheckResourceAttr(rName, "display_name", displayName2),
					resource.TestCheckResourceAttr(rName, "storage_service", storageServiceName),
					resource.TestCheckResourceAttr(rName, "size_limit", strconv.Itoa(testSizeLimit)),
					resource.TestCheckResourceAttr(rName, "iops_limit", strconv.Itoa(testIopsLimit)),
					resource.TestCheckResourceAttr(rName, "bandwidth_limit", strconv.Itoa(testBandwidthLimit)),
					testStorageClassExists(rName),
				),
			},
			//Can't update certain values
			{
				Config:      testStorageClassConfig(rNameConfig, storageClassName, displayName2, "immutable", testSizeLimit, testIopsLimit, testBandwidthLimit),
				ExpectError: regexp.MustCompile("attempting to update an immutable field"),
			},
			{
				Config:      testStorageClassConfig(rNameConfig, "immutable", displayName2, storageServiceName, testSizeLimit, testIopsLimit, testBandwidthLimit),
				ExpectError: regexp.MustCompile("attempting to update an immutable field"),
			},
			{
				Config:      testStorageClassConfig(rNameConfig, storageClassName, displayName2, storageServiceName, testSizeLimit+1024, testIopsLimit, testBandwidthLimit),
				ExpectError: regexp.MustCompile("attempting to update an immutable field"),
			},
			{
				Config:      testStorageClassConfig(rNameConfig, storageClassName, displayName2, storageServiceName, testSizeLimit, testIopsLimit+10, testBandwidthLimit),
				ExpectError: regexp.MustCompile("attempting to update an immutable field"),
			},
			{
				Config:      testStorageClassConfig(rNameConfig, storageClassName, displayName2, storageServiceName, testSizeLimit, testIopsLimit, testBandwidthLimit+10),
				ExpectError: regexp.MustCompile("attempting to update an immutable field"),
			},
		},
	})
}

func TestAccStorageClass_attributes(t *testing.T) {
	rNameConfig1 := acctest.RandomWithPrefix("storage_class_test")
	rName1 := "fusion_storage_class." + rNameConfig1

	rNameConfig2 := acctest.RandomWithPrefix("storage_class_test")
	rName2 := "fusion_storage_class." + rNameConfig2

	displayName1 := acctest.RandomWithPrefix("storage-class-display-name")
	displayName2 := acctest.RandomWithPrefix("storage-class-display-name")
	storageClassName1 := acctest.RandomWithPrefix("storage-class-name")
	storageClassName2 := acctest.RandomWithPrefix("storage-class-name")
	storageServiceName := testAccStorageService

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckStorageClassDestroy,
		Steps: []resource.TestStep{
			// TODO: Do name validations in the schema
			{
				Config:      testStorageClassConfig(rNameConfig1, "bad name", displayName1, storageServiceName, testSizeLimit, testIopsLimit, testBandwidthLimit),
				ExpectError: regexp.MustCompile("name must use alphanumeric characters"),
			},
			// Create without display_name then update
			{
				Config: testStorageClassConfigNoDisplayName(rNameConfig1, storageClassName1, storageServiceName, testSizeLimit, testIopsLimit, testBandwidthLimit),
				Check: resource.ComposeTestCheckFunc(
					testStorageClassExists(rName1),
				),
			},
			{
				Config: testStorageClassConfig(rNameConfig1, storageClassName1, displayName1, storageServiceName, testSizeLimit, testIopsLimit, testBandwidthLimit),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(rName1, "display_name", displayName1),
					testStorageClassExists(rName1),
				),
			},
			// Create without units
			{
				Config: testStorageClassConfigWithoutUnits(rNameConfig2, storageClassName2, displayName2, storageServiceName),
				Check: resource.ComposeTestCheckFunc(
					testStorageClassExists(rName2),
					resource.TestCheckResourceAttr(rName2, "name", storageClassName2),
					resource.TestCheckResourceAttr(rName2, "display_name", displayName2),
					resource.TestCheckResourceAttr(rName2, "storage_service", storageServiceName),
					resource.TestCheckResourceAttr(rName2, "size_limit", "4503599627370496"),
					resource.TestCheckResourceAttr(rName2, "iops_limit", "100000000"),
					resource.TestCheckResourceAttr(rName2, "bandwidth_limit", "549755813888"),
				),
			},
		},
	})
}

func TestAccStorageClass_multiple(t *testing.T) {
	rNameConfig := acctest.RandomWithPrefix("storage_service_test")
	rName := "fusion_storage_class." + rNameConfig
	displayName1 := acctest.RandomWithPrefix("storage-service-display-name")
	storageClassName := acctest.RandomWithPrefix("storage-class-name")
	storageServiceName := testAccStorageService

	rNameConfig2 := acctest.RandomWithPrefix("storage_class_test2")
	rName2 := "fusion_storage_class." + rNameConfig
	displayName2 := acctest.RandomWithPrefix("storage-class-display-name")
	storageClassName2 := acctest.RandomWithPrefix("storage-class-name")

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckStorageClassDestroy,
		Steps: []resource.TestStep{
			// Sanity check two can be created at once
			{
				Config: testStorageClassConfig(rNameConfig, storageClassName, displayName1, storageServiceName, testSizeLimit, testIopsLimit, testBandwidthLimit) + "\n" +
					testStorageClassConfig(rNameConfig2, storageClassName2, displayName2, storageServiceName, testSizeLimit+512, testIopsLimit+10, testBandwidthLimit+10),
				Check: resource.ComposeTestCheckFunc(
					testStorageClassExists(rName),
					testStorageClassExists(rName2),
				),
			},
			// Create two with same name
			{
				Config: testStorageClassConfig(rNameConfig, storageClassName, displayName1, storageServiceName, testSizeLimit, testIopsLimit, testBandwidthLimit) + "\n" +
					testStorageClassConfig(rNameConfig2, storageClassName2, displayName2, storageServiceName, testSizeLimit+512, testIopsLimit+10, testBandwidthLimit+10) + "\n" +
					testStorageClassConfig("conflictRN", storageClassName, "conflictDN", storageServiceName, testSizeLimit, testIopsLimit, testBandwidthLimit),
				ExpectError: regexp.MustCompile("already exists"),
			},
		},
	})
}

func testStorageClassExists(rName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		tfStorageClass, ok := s.RootModule().Resources[rName]
		if !ok {
			return fmt.Errorf("resource not found: %s", rName)
		}
		if tfStorageClass.Type != "fusion_storage_class" {
			return fmt.Errorf("expected type: fusion_storage_class. Found: %s", tfStorageClass.Type)
		}
		attrs := tfStorageClass.Primary.Attributes

		goclientStorageClass, _, err := testAccProvider.Meta().(*hmrest.APIClient).StorageClassesApi.GetStorageClassById(context.Background(), attrs["id"], nil)
		if err != nil {
			return fmt.Errorf("go client returned error while searching for %s. Error: %s", attrs["name"], err)
		}

		bandwidthLimit, err := utilities.ConvertDataUnitsToInt64(attrs["bandwidth_limit"], 1024)
		if err != nil {
			return fmt.Errorf("resource has bad parameter. Error: %s", err)
		}

		iopsLimit, err := utilities.ConvertDataUnitsToInt64(attrs["iops_limit"], 1000)
		if err != nil {
			return fmt.Errorf("resource has bad parameter. Error: %s", err)
		}

		sizeLimit, err := utilities.ConvertDataUnitsToInt64(attrs["size_limit"], 1024)
		if err != nil {
			return fmt.Errorf("resource has bad parameter. Error: %s", err)
		}

		if strings.Compare(goclientStorageClass.Name, attrs["name"]) != 0 ||
			strings.Compare(goclientStorageClass.DisplayName, attrs["display_name"]) != 0 ||
			strings.Compare(goclientStorageClass.StorageService.Name, attrs["storage_service"]) != 0 ||
			goclientStorageClass.BandwidthLimit != bandwidthLimit ||
			goclientStorageClass.IopsLimit != iopsLimit ||
			goclientStorageClass.SizeLimit != sizeLimit {
			return fmt.Errorf("terraform storage class doesn't match goclients storage class")
		}
		return nil
	}
}

func testCheckStorageClassDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*hmrest.APIClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "fusion_storage_class" {
			continue
		}
		attrs := rs.Primary.Attributes

		_, resp, err := client.StorageClassesApi.GetStorageClassById(context.Background(), attrs["id"], nil)
		if err != nil && resp.StatusCode == http.StatusNotFound {
			continue
		} else {
			return fmt.Errorf("storage class may still exist. Expected response code 404, got code %d", resp.StatusCode)
		}
	}
	return nil
}

func testStorageClassConfigWithoutUnits(rName, storageClassName, displayName, storageServiceName string) string {
	return fmt.Sprintf(`
	resource "fusion_storage_class" "%[1]s" {
		name			= "%[2]s"
		display_name	= "%[3]s"
		storage_service	= "%[4]s"
	}
	`, rName, storageClassName, displayName, storageServiceName)
}

func testStorageClassConfig(rName, storageClassName, displayName, storageServiceName string, sizeLimit, iopsLimit, bandwidthLimit int) string {
	return fmt.Sprintf(`
	resource "fusion_storage_class" "%[1]s" {
		name			= "%[2]s"
		display_name	= "%[3]s"
		storage_service	= "%[4]s"
		size_limit		= "%[5]d"
		iops_limit		= "%[6]d"
		bandwidth_limit	= "%[7]d"
	}
	`, rName, storageClassName, displayName, storageServiceName, sizeLimit, iopsLimit, bandwidthLimit)
}

func testStorageClassConfigWithUnits(rName, storageClassName, displayName, storageServiceName, sizeLimit, iopsLimit, bandwidthLimit string) string {
	return fmt.Sprintf(`
	resource "fusion_storage_class" "%[1]s" {
		name			= "%[2]s"
		display_name	= "%[3]s"
		storage_service	= "%[4]s"
		size_limit		= "%[5]s"
		iops_limit		= "%[6]s"
		bandwidth_limit	= "%[7]s"
	}
	`, rName, storageClassName, displayName, storageServiceName, sizeLimit, iopsLimit, bandwidthLimit)
}

func testStorageClassConfigNoDisplayName(rName, storageClassName, storageServiceName string, sizeLimit, iopsLimit, bandwidthLimit int) string {
	return fmt.Sprintf(`
	resource "fusion_storage_class" "%[1]s" {
		name      		= "%[2]s"
		storage_service	= "%[3]s"
		size_limit      = "%[4]d"
		iops_limit      = "%[5]d"
		bandwidth_limit = "%[6]d"
	}
	`, rName, storageClassName, storageServiceName, sizeLimit, iopsLimit, bandwidthLimit)
}
