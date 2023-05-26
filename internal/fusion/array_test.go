/*
Copyright 2023 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package fusion

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"testing"

	"github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/utilities"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"
)

/*
	Array tests by default poach arrays to test from region 'pure-us-west', availability zone 'az1'
	instead of having them configurable. While a bit unorthodox, this is designed to work
	out of the box on freshly deployed control plane to simplify test suite deployment.
	To deterministically use specific array setup, set following env variables:
	TF_ACC_USE_ENV_ARRAYS=1
	TF_ACC_ARRAY_APPLIANCE_ID_1=...
	TF_ACC_ARRAY_HOST_NAME_1=...
	TF_ACC_HARDWARE_TYPE_1=...
	TF_ACC_ARRAY_APPLIANCE_ID_2=...
	TF_ACC_ARRAY_HOST_NAME_2=...
	TF_ACC_HARDWARE_TYPE_2=...
	TF_ACC_ARRAY_APPLIANCE_ID_3=...
	TF_ACC_ARRAY_HOST_NAME_3=...
	TF_ACC_HARDWARE_TYPE_3=...
	Arrays in these env variables must not be preregistered with the control plane.
	XXX: These tests currently do not work with doubleagents due to HM-5548.
	XXX: Test on testbed instead.
*/

func TestAccArray_all(t *testing.T) {
	// The reason all array-related tests (including its Data Source) are under single
	// test entry point is that the target arrays need to be poached first and
	// poaching is SLOW (90s/round ATM).
	// Having tests separately would mean having to do separate poaching for each
	// test and you are likely to sooner colonize Mars than for the tests to complete.
	if os.Getenv(resource.TestEnvVar) == "" {
		t.Skipf("Acceptance tests skipped unless env '%s' set", resource.TestEnvVar)
		return
	}

	arrays, release := FindArraysForTests(t, 3)
	arraysToTestOn = arrays
	defer func() {
		release()
		arraysToTestOn = nil
	}()

	t.Run("TestAccArray_basic", testAccArray_basic)
	t.Run("TestAccArray_attributes", testAccArray_attributes)
	t.Run("TestAccArray_immutables", testAccArray_immutables)
	t.Run("TestAccArray_multiple", testAccArray_multiple)
	t.Run("TestAccArray_update", testAccArray_update)
	t.Run("TestAccArrayDataSource_basic", testAccArrayDataSource_basic)
}

var arraysToTestOn []ArrayTestData

func testAccArray_basic(t *testing.T) {
	array := arraysToTestOn[0]

	tfName := acctest.RandomWithPrefix("array_test_tf_array")
	arrayName := acctest.RandomWithPrefix("array_test_fs_array")
	displayName := acctest.RandomWithPrefix("array_test_display_name")
	availabilityZone := acctest.RandomWithPrefix("array_test_az")
	region := acctest.RandomWithPrefix("array_test_region")

	commonConfig := "" +
		testRegionConfig(region, region, region) +
		testAvailabilityZoneConfig(availabilityZone, availabilityZone, availabilityZone, region)

	printArrayConfig := func() string {
		return fmt.Sprintf(`
		resource "fusion_array" "%[1]s" {
			name      			= "%[2]s"
			display_name    	= "%[3]s"
			hardware_type		= "%[4]s"
			region				= fusion_region.%[5]s.name
			availability_zone	= fusion_availability_zone.%[6]s.name
			appliance_id		= "%[7]s"
			host_name			= "%[8]s"
			maintenance_mode	= false
			unavailable_mode	= false
		}
		`, tfName, arrayName, displayName, array.HardwareType, region, availabilityZone, array.ApplianceId, array.HostName)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckArrayDestroy,
		Steps: []resource.TestStep{
			{
				Config: commonConfig + printArrayConfig(),
				Check: resource.ComposeTestCheckFunc(
					testArrayExists(tfName, t),
				),
			},
		},
	})
}

func testAccArray_update(t *testing.T) {
	// ATM can patch: display_name, host_name, maintenance_mode, unavailable_mode
	// testing against doubleagent, so hard to test host_name changes - leave it out
	// XXX: also impossible to test unavailable_mode as backend cannot revert unavailable_mode right now
	array := arraysToTestOn[0]

	tfName := acctest.RandomWithPrefix("array_test_tf_array")
	arrayName := acctest.RandomWithPrefix("array_test_fs_array")
	displayName1 := acctest.RandomWithPrefix("array_test_display_name1")
	displayName2 := acctest.RandomWithPrefix("array_test_display_name2")
	availabilityZone := acctest.RandomWithPrefix("array_test_az")
	region := acctest.RandomWithPrefix("array_test_region")
	maintenanceMode1 := false
	maintenanceMode2 := false
	// TODO FIX: not testable right now as unavailable mode cannot be reverted
	unavailableMode1 := false
	//unavailableMode2 := false

	commonConfig := "" +
		testRegionConfig(region, region, region) +
		testAvailabilityZoneConfig(availabilityZone, availabilityZone, availabilityZone, region)

	printArrayConfig := func(displayName string, maintenanceMode, unavailableMode bool) string {
		return testArrayConfig(tfName, arrayName, displayName, array.HardwareType, region, availabilityZone, array.ApplianceId, array.HostName, maintenanceMode, unavailableMode)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckArrayDestroy,
		Steps: []resource.TestStep{
			// Initial create
			{
				Config: commonConfig + printArrayConfig(displayName1, maintenanceMode1, unavailableMode1),
				Check:  testArrayExists(tfName, t),
			},
			// Change display name
			{
				Config: commonConfig + printArrayConfig(displayName2, maintenanceMode1, unavailableMode1),
				Check:  testArrayExists(tfName, t),
			},
			// Change maintenance mode
			{
				Config: commonConfig + printArrayConfig(displayName2, maintenanceMode2, unavailableMode1),
				Check:  testArrayExists(tfName, t),
			},
			// Change maintenance mode back
			{
				Config: commonConfig + printArrayConfig(displayName2, maintenanceMode1, unavailableMode1),
				Check:  testArrayExists(tfName, t),
			},
			// // Change unavailable mode
			// {
			// 	Config: commonConfig + printArrayConfig(displayName2, maintenanceMode1, unavailableMode1),
			// 	Check:  testArrayExists(tfName, t),
			// },
			// // Change unavailable mode back
			// {
			// 	Config: commonConfig + printArrayConfig(displayName2, maintenanceMode1, unavailableMode2),
			// 	Check:  testArrayExists(tfName, t),
			// },
		},
	})
}

func testAccArray_attributes(t *testing.T) {
	array := arraysToTestOn[0]

	tfName := acctest.RandomWithPrefix("array_test_tf_array")
	arrayName := acctest.RandomWithPrefix("array_test_fs_array")
	displayName := acctest.RandomWithPrefix("array_test_display_name")
	availabilityZone := acctest.RandomWithPrefix("array_test_az")
	region := acctest.RandomWithPrefix("array_test_region")
	maintenanceMode := false
	unavailableMode := false

	commonConfig := "" +
		testRegionConfig(region, region, region) +
		testAvailabilityZoneConfig(availabilityZone, availabilityZone, availabilityZone, region)

	printArrayConfig := func(arrayName string, displayName *string) string {
		displayNameLine := ""
		if displayName != nil {
			displayNameLine = fmt.Sprintf(`display_name    	= "%s"`, *displayName)
		}
		return fmt.Sprintf(`
		resource "fusion_array" "%[1]s" {
			name      			= "%[2]s"
			%[3]s
			hardware_type		= "%[4]s"
			region				= fusion_region.%[5]s.name
			availability_zone	= fusion_availability_zone.%[6]s.name
			appliance_id		= "%[7]s"
			host_name			= "%[8]s"
			maintenance_mode	= false
			unavailable_mode	= false
		}
		`, tfName, arrayName, displayNameLine, array.HardwareType, region, availabilityZone, array.ApplianceId, array.HostName, maintenanceMode, unavailableMode)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckArrayDestroy,
		Steps: []resource.TestStep{
			// TODO: Do name validations in the schema
			{
				Config:      commonConfig + printArrayConfig("bad name here", &displayName),
				ExpectError: regexp.MustCompile("name must use alphanumeric characters"),
			},
			// Create without display_name then update
			{
				Config: commonConfig + printArrayConfig(arrayName, nil),
				Check: resource.ComposeTestCheckFunc(
					testArrayExists(tfName, t),
				),
			},
			{
				Config: commonConfig + printArrayConfig(arrayName, &displayName),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("fusion_array."+tfName, "display_name", displayName),
					testArrayExists(tfName, t),
				),
			},
		},
	})
}

func testAccArray_immutables(t *testing.T) {
	array := arraysToTestOn[0]

	tfName := acctest.RandomWithPrefix("array_test_tf_array")
	arrayName := acctest.RandomWithPrefix("array_test_fs_array")
	availabilityZone := acctest.RandomWithPrefix("array_test_az")
	region := acctest.RandomWithPrefix("array_test_region")

	commonConfig := "" +
		testRegionConfig(region, region, region) +
		testAvailabilityZoneConfig(availabilityZone, availabilityZone, availabilityZone, region)

	printArrayConfigWithRefs := func(arrayName, region, availabilityZone, applianceId, hardwareType string) string {
		return fmt.Sprintf(`
		resource "fusion_array" "%[1]s" {
			name      			= "%[2]s"
			display_name		= "%[2]s"
			hardware_type		= "%[3]s"
			region				= fusion_region.%[4]s.name
			availability_zone	= fusion_availability_zone.%[5]s.name
			appliance_id		= "%[6]s"
			host_name			= "%[7]s"
			maintenance_mode	= false
			unavailable_mode	= false
		}
		`, tfName, arrayName, hardwareType, region, availabilityZone, applianceId, array.HostName)
	}

	printArrayConfigWithNames := func(arrayName, region, availabilityZone, applianceId, hardwareType string) string {
		return fmt.Sprintf(`
		resource "fusion_array" "%[1]s" {
			name      			= "%[2]s"
			display_name		= "%[2]s"
			hardware_type		= "%[3]s"
			region				= "%[4]s"
			availability_zone	= "%[5]s"
			appliance_id		= "%[6]s"
			host_name			= "%[7]s"
			maintenance_mode	= false
			unavailable_mode	= false
		}
		`, tfName, arrayName, hardwareType, region, availabilityZone, applianceId, array.HostName)
	}

	immutableErrRegex := regexp.MustCompile("immutable field")

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckArrayDestroy,
		Steps: []resource.TestStep{
			{
				Config: commonConfig + printArrayConfigWithRefs(arrayName, region, availabilityZone, array.ApplianceId, array.HardwareType),
				Check: resource.ComposeTestCheckFunc(
					testArrayExists(tfName, t),
				),
			},
			{
				Config:      commonConfig + printArrayConfigWithNames(arrayName, "someRandomRegion", availabilityZone, array.ApplianceId, array.HardwareType),
				ExpectError: immutableErrRegex,
			},
			{
				Config:      commonConfig + printArrayConfigWithNames(arrayName, region, "someRandomAz", array.ApplianceId, array.HardwareType),
				ExpectError: immutableErrRegex,
			},
			{
				Config:      commonConfig + printArrayConfigWithNames(arrayName, region, availabilityZone, "someRandomApplianceId", array.HardwareType),
				ExpectError: immutableErrRegex,
			},
			{
				Config:      commonConfig + printArrayConfigWithNames(arrayName, region, availabilityZone, array.ApplianceId, "flash-array-x-optane"),
				ExpectError: immutableErrRegex,
			},
			{ // this is here intentionally as the test is otherwise flaky
				Config: commonConfig + printArrayConfigWithRefs(arrayName, region, availabilityZone, array.ApplianceId, array.HardwareType),
				Check: resource.ComposeTestCheckFunc(
					testArrayExists(tfName, t),
				),
			},
		},
	})
}

func testAccArray_multiple(t *testing.T) {
	arrays := arraysToTestOn

	tf1Name := acctest.RandomWithPrefix("array_test_tf_array1")
	array1Name := acctest.RandomWithPrefix("array_test_fs_array1")
	tf2Name := acctest.RandomWithPrefix("array_test_tf_array2")
	tf3Name := acctest.RandomWithPrefix("array_test_tf_array3")
	array2Name := acctest.RandomWithPrefix("array_test_fs_array2")
	availabilityZone := acctest.RandomWithPrefix("array_test_az")
	region := acctest.RandomWithPrefix("array_test_region")

	commonConfig := "" +
		testRegionConfig(region, region, region) +
		testAvailabilityZoneConfig(availabilityZone, availabilityZone, availabilityZone, region)

	printArrayConfig := func(tfName, arrayName, hardwareType, hostName, applianceId string) string {
		return testArrayConfig(tfName, arrayName, arrayName, hardwareType, region, availabilityZone, applianceId, hostName, false, false)
	}

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckArrayDestroy,
		Steps: []resource.TestStep{
			// Sanity check two can be created at once
			{
				Config: commonConfig +
					printArrayConfig(tf1Name, array1Name, arrays[0].HardwareType, arrays[0].HostName, arrays[0].ApplianceId) +
					printArrayConfig(tf2Name, array2Name, arrays[1].HardwareType, arrays[1].HostName, arrays[1].ApplianceId),
				Check: resource.ComposeTestCheckFunc(
					testArrayExists(tf1Name, t),
					testArrayExists(tf2Name, t),
				),
			},
			// Try to reuse array
			{
				Config: commonConfig +
					printArrayConfig(tf1Name, array1Name, arrays[0].HardwareType, arrays[0].HostName, arrays[0].ApplianceId) +
					printArrayConfig(tf2Name, array2Name, arrays[1].HardwareType, arrays[1].HostName, arrays[1].ApplianceId) +
					printArrayConfig(tf3Name, array2Name, arrays[1].HardwareType, arrays[1].HostName, arrays[1].ApplianceId),
				ExpectError: regexp.MustCompile("already exists"),
			},
		},
	})
}

func testArrayExists(rName string, t *testing.T) resource.TestCheckFunc {
	fullName := "fusion_array." + rName
	return func(s *terraform.State) error {
		tfArray, ok := s.RootModule().Resources[fullName]
		if !ok {
			return fmt.Errorf("resource not found: %s", fullName)
		}
		if tfArray.Type != "fusion_array" {
			return fmt.Errorf("expected type: fusion_array. Found: %s", tfArray.Type)
		}
		savedArray := tfArray.Primary.Attributes
		arrayName := savedArray["name"]
		availabilityZoneName := savedArray["availability_zone"]
		regionName := savedArray["region"]

		foundArray, _, err := testAccProvider.Meta().(*hmrest.APIClient).ArraysApi.GetArray(context.Background(), regionName, availabilityZoneName, arrayName, nil)
		if err != nil {
			return fmt.Errorf("go client returned error while searching for %s. Error: %s", savedArray["name"], err)
		}

		if !utilities.CheckStrAttribute(t, "display_name", foundArray.DisplayName, savedArray["display_name"]) ||
			!utilities.CheckStrAttribute(t, "hardware_type", foundArray.HardwareType.Name, savedArray["hardware_type"]) ||
			!utilities.CheckStrAttribute(t, "appliance_id", foundArray.ApplianceId, savedArray["appliance_id"]) ||
			!utilities.CheckStrAttribute(t, "host_name", foundArray.HostName, savedArray["host_name"]) ||
			!utilities.CheckStrAttribute(t, "apartment_id", foundArray.ApartmentId, savedArray["apartment_id"]) ||
			!utilities.CheckBoolAttribute(t, "maintenance_mode", foundArray.MaintenanceMode, savedArray["maintenance_mode"]) ||
			!utilities.CheckBoolAttribute(t, "unavailable_mode", foundArray.UnavailableMode, savedArray["unavailable_mode"]) {
			return fmt.Errorf("'fusion_array' stored state for in Terraform doesn't match reality")
		}

		return nil
	}
}

func testCheckArrayDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*hmrest.APIClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "fusion_array" {
			continue
		}
		attrs := rs.Primary.Attributes
		arrayName := attrs["name"]
		availabilityZoneName := attrs["availability_zone_name"]
		regionName := attrs["region"]

		_, resp, err := client.ArraysApi.GetArray(context.Background(), regionName, availabilityZoneName, arrayName, nil)
		if err == nil || resp.StatusCode != http.StatusNotFound {
			return fmt.Errorf("array may still exist. Expected response code 404, got code %d", resp.StatusCode)
		}
	}
	return nil
}

func testAccArrayDataSource_basic(t *testing.T) {
	srcArrays := arraysToTestOn

	dsTfName := acctest.RandomWithPrefix("array_test_ds")
	availabilityZone := acctest.RandomWithPrefix("array_test_az")
	region := acctest.RandomWithPrefix("array_test_region")
	arrayCount := 3
	arrays := make([]map[string]interface{}, 0, arrayCount)
	arrayConfigs := make([]string, 0, arrayCount)

	commonConfig := "" +
		testRegionConfig(region, region, region) +
		testAvailabilityZoneConfig(availabilityZone, availabilityZone, availabilityZone, region)

	for i := 0; i < arrayCount; i++ {
		tfName := acctest.RandomWithPrefix("array_test_tf_array")
		arrayName := acctest.RandomWithPrefix("array_test_fs_array")
		displayName := acctest.RandomWithPrefix("array_test_display_name")

		arrays = append(arrays, map[string]interface{}{
			"name":              arrayName,
			"display_name":      displayName,
			"availability_zone": availabilityZone,
			"region":            region,
			"appliance_id":      srcArrays[i].ApplianceId,
			"host_name":         srcArrays[i].HostName,
			"hardware_type":     srcArrays[i].HardwareType,
			"maintenance_mode":  false,
			"unavailable_mode":  false,
		})

		arrayConfigs = append(arrayConfigs, testArrayConfig(tfName, arrayName, displayName, srcArrays[i].HardwareType, region, availabilityZone, srcArrays[i].ApplianceId, srcArrays[i].HostName, false, false))
	}

	allArraysConfig := commonConfig + strings.Join(arrayConfigs, "\n")
	partialArraysConfig := commonConfig + arrayConfigs[0] + arrayConfigs[1]

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckArrayDestroy,
		Steps: []resource.TestStep{
			// Create n arrays
			{
				Config: allArraysConfig,
			},
			// Check if they are contained in the data source
			{
				Config: allArraysConfig + "\n" + testArrayDataSourceConfig(dsTfName, region, availabilityZone),
				Check: utilities.TestCheckDataSource(
					"fusion_array", dsTfName, "items", arrays,
				),
			},
			// Remove one array. Check if only two of them are contained in the data source
			{
				Config: partialArraysConfig + "\n" + testArrayDataSourceConfig(dsTfName, region, availabilityZone),
				Check: utilities.TestCheckDataSource(
					"fusion_array", dsTfName, "items", []map[string]interface{}{
						arrays[0], arrays[1],
					},
				),
			},
		},
	})
}

func testArrayConfig(tfName, arrayName, displayName, hardwareType, region, availabilityZone, applianceId, hostName string, maintenanceMode, unavailableMode bool) string {
	return fmt.Sprintf(`
		resource "fusion_array" "%[1]s" {
			name      			= "%[2]s"
			display_name    	= "%[3]s"
			hardware_type		= "%[4]s"
			region				= fusion_region.%[5]s.name
			availability_zone	= fusion_availability_zone.%[6]s.name
			appliance_id		= "%[7]s"
			host_name			= "%[8]s"
			maintenance_mode	= %[9]t
			unavailable_mode	= %[10]t
		}
		`, tfName, arrayName, displayName, hardwareType, region, availabilityZone, applianceId, hostName, maintenanceMode, unavailableMode)
}

func testArrayDataSourceConfig(dsName, region, availabilityZone string) string {
	return fmt.Sprintf(`data "fusion_array" "%[1]s" {
		region				= fusion_region.%[2]s.name
		availability_zone	= fusion_availability_zone.%[3]s.name
	}`, dsName, region, availabilityZone)
}
