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
	"github.com/hashicorp/go-multierror"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"
)

const (
	gateway   = "127.0.0.1"
	prefix    = "127.0.0.1/32"
	groupType = "eth"
)

func TestAccNetworkInterfaceGroup_basic(t *testing.T) {
	utilities.CheckTestSkip(t)

	rNameConfig1 := acctest.RandomWithPrefix("network_interface_group_test")
	rNameConfig2 := acctest.RandomWithPrefix("network_interface_group_test")
	rName1 := "fusion_network_interface_group." + rNameConfig1
	rName2 := "fusion_network_interface_group." + rNameConfig2
	nigName1 := acctest.RandomWithPrefix("nig-name-1")
	nigName2 := acctest.RandomWithPrefix("nig-name-2")
	displayName1 := acctest.RandomWithPrefix("display-name")
	mtu1 := strconv.Itoa(acctest.RandIntRange(1280, 9216))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckNetworkInterfaceGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testNetworkInterfaceGroupConfig(rNameConfig1, nigName1, displayName1, preexistingAvailabilityZone, preexistingRegion, groupType, gateway, prefix, mtu1),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(rName1, "name", nigName1),
					resource.TestCheckResourceAttr(rName1, "display_name", displayName1),
					resource.TestCheckResourceAttr(rName1, "availability_zone", preexistingAvailabilityZone),
					resource.TestCheckResourceAttr(rName1, "region", preexistingRegion),
					resource.TestCheckResourceAttr(rName1, "group_type", groupType),
					resource.TestCheckResourceAttr(rName1, "eth.0.gateway", gateway),
					resource.TestCheckResourceAttr(rName1, "eth.0.prefix", prefix),
					resource.TestCheckResourceAttr(rName1, "eth.0.mtu", mtu1),
					testNetworkInterfaceGroupExists(rName1),
				),
			},
			// Default values are set for optional fields
			{
				Config: testNetworkInterfaceGroupConfigNoOptionalValues(rNameConfig2, nigName2, preexistingAvailabilityZone, preexistingRegion, gateway, prefix),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(rName2, "name", nigName2),
					resource.TestCheckResourceAttr(rName2, "display_name", nigName2),
					resource.TestCheckResourceAttr(rName2, "availability_zone", preexistingAvailabilityZone),
					resource.TestCheckResourceAttr(rName2, "region", preexistingRegion),
					resource.TestCheckResourceAttr(rName2, "group_type", "eth"),
					resource.TestCheckResourceAttr(rName2, "eth.0.gateway", gateway),
					resource.TestCheckResourceAttr(rName2, "eth.0.prefix", prefix),
					resource.TestCheckResourceAttr(rName2, "eth.0.mtu", "1500"),
					testNetworkInterfaceGroupExists(rName2),
				),
			},
		},
	})
}

func TestAccNetworkInterfaceGroup_update(t *testing.T) {
	utilities.CheckTestSkip(t)

	rNameConfig := acctest.RandomWithPrefix("network_interface_group_test")
	rName := "fusion_network_interface_group." + rNameConfig
	nigName := acctest.RandomWithPrefix("nig-name")
	displayName1 := acctest.RandomWithPrefix("display-name")
	displayName2 := acctest.RandomWithPrefix("display-name")
	mtu := strconv.Itoa(acctest.RandIntRange(1280, 9216))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckNetworkInterfaceGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testNetworkInterfaceGroupConfig(rNameConfig, nigName, displayName1, preexistingAvailabilityZone, preexistingRegion, groupType, gateway, prefix, mtu),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(rName, "name", nigName),
					resource.TestCheckResourceAttr(rName, "display_name", displayName1),
					resource.TestCheckResourceAttr(rName, "availability_zone", preexistingAvailabilityZone),
					resource.TestCheckResourceAttr(rName, "group_type", groupType),
					resource.TestCheckResourceAttr(rName, "region", preexistingRegion),
					resource.TestCheckResourceAttr(rName, "eth.0.gateway", gateway),
					resource.TestCheckResourceAttr(rName, "eth.0.prefix", prefix),
					resource.TestCheckResourceAttr(rName, "eth.0.mtu", mtu),
					testNetworkInterfaceGroupExists(rName),
				),
			},
			// Update the display name, assert that the tf resource got updated, then assert the backend shows the same
			{
				Config: testNetworkInterfaceGroupConfig(rNameConfig, nigName, displayName2, preexistingAvailabilityZone, preexistingRegion, groupType, gateway, prefix, mtu),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(rName, "display_name", displayName2),
					testNetworkInterfaceGroupExists(rName),
				),
			},
			// Can update display_name only
			{
				Config:      testNetworkInterfaceGroupConfig(rNameConfig, nigName, displayName1, "immutable", preexistingRegion, groupType, gateway, prefix, mtu),
				ExpectError: regexp.MustCompile("attempting to update an immutable field"),
			},
		},
	})
}

func TestAccNetworkInterfaceGroup_attributes(t *testing.T) {
	utilities.CheckTestSkip(t)

	rNameConfig := acctest.RandomWithPrefix("network_interface_group_test")
	nigName := acctest.RandomWithPrefix("nig-name")
	displayName := acctest.RandomWithPrefix("display-name")
	mtu := strconv.Itoa(acctest.RandIntRange(1280, 9216))

	displayNameTooBig := strings.Repeat("a", 257)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckNetworkInterfaceGroupDestroy,
		Steps: []resource.TestStep{
			// Missing required fields
			{
				Config:      testNetworkInterfaceGroupConfig(rNameConfig, "", displayName, preexistingAvailabilityZone, preexistingRegion, groupType, gateway, prefix, mtu),
				ExpectError: regexp.MustCompile(`expected "name" to not be an empty string`),
			},
			{
				Config:      testNetworkInterfaceGroupConfig(rNameConfig, "bad name here", displayName, preexistingAvailabilityZone, preexistingRegion, groupType, gateway, prefix, mtu),
				ExpectError: regexp.MustCompile("name must use alphanumeric characters"),
			},
			{
				Config:      testNetworkInterfaceGroupConfig(rNameConfig, nigName, "", preexistingAvailabilityZone, preexistingRegion, groupType, gateway, prefix, mtu),
				ExpectError: regexp.MustCompile(`expected "display_name" to not be an empty string`),
			},
			{
				Config:      testNetworkInterfaceGroupConfig(rNameConfig, nigName, displayNameTooBig, preexistingAvailabilityZone, preexistingRegion, groupType, gateway, prefix, mtu),
				ExpectError: regexp.MustCompile("display_name must be at most 256 characters"),
			},
			// Values should not pass validations
			{
				Config:      testNetworkInterfaceGroupConfig(rNameConfig, nigName, displayName, preexistingAvailabilityZone, preexistingRegion, groupType, gateway, prefix, "1"),
				ExpectError: regexp.MustCompile("mtu must be between 1280 and 9216"),
			},
			{
				Config:      testNetworkInterfaceGroupConfig(rNameConfig, nigName, displayName, preexistingAvailabilityZone, preexistingRegion, groupType, gateway, prefix, "10000"),
				ExpectError: regexp.MustCompile("mtu must be between 1280 and 9216"),
			},
			// Prefix and gateway should be valid
			{
				Config:      testNetworkInterfaceGroupConfig(rNameConfig, nigName, displayName, preexistingAvailabilityZone, preexistingRegion, groupType, "not gateway", prefix, mtu),
				ExpectError: regexp.MustCompile(`Bad address`),
			},
			{
				Config:      testNetworkInterfaceGroupConfig(rNameConfig, nigName, displayName, preexistingAvailabilityZone, preexistingRegion, groupType, gateway, "not prefix", mtu),
				ExpectError: regexp.MustCompile(`Bad CIDR`),
			},
			{
				Config:      testNetworkInterfaceGroupConfig(rNameConfig, nigName, displayName, preexistingAvailabilityZone, preexistingRegion, groupType, "127.0.0.2", prefix, mtu),
				ExpectError: regexp.MustCompile(`"gateway" must be an address in subnet "prefix"`),
			},
		},
	})
}

func TestAccNetworkInterfaceGroup_multiple(t *testing.T) {
	utilities.CheckTestSkip(t)

	rNameConfig1 := acctest.RandomWithPrefix("network_interface_group_test")
	rName1 := "fusion_network_interface_group." + rNameConfig1
	nigName1 := acctest.RandomWithPrefix("nig-name")
	displayName1 := acctest.RandomWithPrefix("nig-display-name")
	mtu1 := strconv.Itoa(acctest.RandIntRange(1280, 9216))

	rNameConfig2 := acctest.RandomWithPrefix("network_interface_group_test")
	rName2 := "fusion_network_interface_group." + rNameConfig1
	nigName2 := acctest.RandomWithPrefix("nig-name")
	displayName2 := acctest.RandomWithPrefix("nig-display-name")
	mtu2 := strconv.Itoa(acctest.RandIntRange(1280, 9216))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckNetworkInterfaceGroupDestroy,
		Steps: []resource.TestStep{
			// Sanity check two can be created at once
			{
				Config: testNetworkInterfaceGroupConfig(rNameConfig1, nigName1, displayName1, preexistingAvailabilityZone, preexistingRegion, groupType, gateway, prefix, mtu1) + "\n" +
					testNetworkInterfaceGroupConfig(rNameConfig2, nigName2, displayName2, preexistingAvailabilityZone, preexistingRegion, groupType, gateway, prefix, mtu2),
				Check: resource.ComposeTestCheckFunc(
					testNetworkInterfaceGroupExists(rName1),
					testNetworkInterfaceGroupExists(rName2),
				),
			},
			// Create two with same name
			{
				Config: testNetworkInterfaceGroupConfig(rNameConfig1, nigName1, displayName1, preexistingAvailabilityZone, preexistingRegion, groupType, gateway, prefix, mtu1) + "\n" +
					testNetworkInterfaceGroupConfig(rNameConfig2, nigName2, displayName2, preexistingAvailabilityZone, preexistingRegion, groupType, gateway, prefix, mtu2) + "\n" +
					testNetworkInterfaceGroupConfig("conflictRN", nigName1, "conflictDN", preexistingAvailabilityZone, preexistingRegion, groupType, gateway, prefix, mtu1),
				ExpectError: regexp.MustCompile("already exists"),
			},
		},
	})
}

func TestAccNetworkInterfaceGroup_import(t *testing.T) {
	utilities.CheckTestSkip(t)

	rNameConfig := acctest.RandomWithPrefix("network_interface_group_test")
	rName := "fusion_network_interface_group." + rNameConfig
	nigName := acctest.RandomWithPrefix("nig-name")
	displayName := acctest.RandomWithPrefix("display-name")
	mtu := strconv.Itoa(acctest.RandIntRange(1280, 9216))

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckNetworkInterfaceGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config: testNetworkInterfaceGroupConfig(rNameConfig, nigName, displayName, preexistingAvailabilityZone, preexistingRegion, groupType, gateway, prefix, mtu),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(rName, "name", nigName),
					resource.TestCheckResourceAttr(rName, "display_name", displayName),
					resource.TestCheckResourceAttr(rName, "availability_zone", preexistingAvailabilityZone),
					resource.TestCheckResourceAttr(rName, "region", preexistingRegion),
					resource.TestCheckResourceAttr(rName, "group_type", groupType),
					resource.TestCheckResourceAttr(rName, "eth.0.gateway", gateway),
					resource.TestCheckResourceAttr(rName, "eth.0.prefix", prefix),
					resource.TestCheckResourceAttr(rName, "eth.0.mtu", mtu),
					testNetworkInterfaceGroupExists(rName),
				),
			},
			{
				ImportState:       true,
				ResourceName:      fmt.Sprintf("fusion_network_interface_group.%s", rNameConfig),
				ImportStateId:     fmt.Sprintf("/regions/%[1]s/availability-zones/%[2]s/network-interface-groups/%[3]s", preexistingRegion, preexistingAvailabilityZone, nigName),
				ImportStateVerify: true,
			},
			{
				ImportState:   true,
				ResourceName:  fmt.Sprintf("fusion_network_interface_group.%s", rNameConfig),
				ImportStateId: fmt.Sprintf("/regions/%[1]s/availability-zones/%[2]s/network-interface-groups/wrong-%[3]s", preexistingRegion, preexistingAvailabilityZone, nigName),
				ExpectError:   regexp.MustCompile("Not Found"),
			},
			{
				ImportState:   true,
				ResourceName:  fmt.Sprintf("fusion_network_interface_group.%s", rNameConfig),
				ImportStateId: fmt.Sprintf("/network-interface-groups/%[3]s", preexistingRegion, preexistingAvailabilityZone, nigName),
				ExpectError:   regexp.MustCompile("invalid network_interface_group import path. Expected path in format '/regions/<region>/availability-zones/<availability-zone>/network-interface-groups/<network-interface-group>'"),
			},
		},
	})
}

func testNetworkInterfaceGroupExists(rName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		resource, ok := s.RootModule().Resources[rName]
		if !ok {
			return fmt.Errorf("resource not found: %s", rName)
		}
		if resource.Type != "fusion_network_interface_group" {
			return fmt.Errorf("expected type: fusion_network_interface_group. Found: %s", resource.Type)
		}
		attrs := resource.Primary.Attributes

		client, _, err := testAccProvider.Meta().(*hmrest.APIClient).NetworkInterfaceGroupsApi.GetNetworkInterfaceGroupById(context.Background(), attrs["id"], nil)
		if err != nil {
			return fmt.Errorf("go client returned error while searching for %s by id: %s. Error: %s", attrs["name"], attrs["id"], err)
		}

		var errs error
		checkAttr := func(client, attrName string) {
			if client != attrs[attrName] {
				errs = multierror.Append(errs, fmt.Errorf("mismatch attr: %s client: %s tf: %s", attrName, client, attrs[attrName]))
			}
		}

		checkAttr(client.Name, "name")
		checkAttr(client.DisplayName, "display_name")
		checkAttr(client.AvailabilityZone.Name, "availability_zone")
		checkAttr(client.Region.Name, "region")
		checkAttr(client.GroupType, "group_type")
		checkAttr(client.Eth.Gateway, "eth.0.gateway")
		checkAttr(client.Eth.Prefix, "eth.0.prefix")

		mtu, err := strconv.Atoi(attrs["eth.0.mtu"])
		if err != nil {
			errs = multierror.Append(errs, fmt.Errorf("mtu conversion error: %w mtu: %s", err, attrs["mtu"]))
		}
		if client.Eth.Mtu != int32(mtu) {
			errs = multierror.Append(errs, fmt.Errorf("mismatch attr:mtu client: %d tf: %d", client.Eth.Mtu, mtu))
		}

		if errs != nil {
			return multierror.Append(fmt.Errorf("terraform network interface group resource doesnt match clients network interface group"), errs)
		}

		return nil
	}
}

func testCheckNetworkInterfaceGroupDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*hmrest.APIClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "fusion_network_interface_group" {
			continue
		}
		attrs := rs.Primary.Attributes

		name, ok := attrs["name"]
		if !ok {
			continue // Skip data sources
		}

		region := attrs["region"]
		az := attrs["availability_zone"]

		_, resp, err := client.NetworkInterfaceGroupsApi.GetNetworkInterfaceGroup(context.Background(), region, az, name, nil)
		if err != nil && resp.StatusCode == http.StatusNotFound {
			continue
		}
		return fmt.Errorf("network interface group may still exist. Expected response code 404, got code %d", resp.StatusCode)
	}
	return nil
}

func testNetworkInterfaceGroupConfig(rName, nigName, displayName, availabilityZone, region, groupType, gateway, prefix, mtu string) string {
	return fmt.Sprintf(`
	resource "fusion_network_interface_group" "%[1]s" {
		name          	  = "%[2]s"
		display_name  	  = "%[3]s"
		availability_zone = "%[4]s"
		region			  = "%[5]s"
		group_type		  = "%[6]s"
		eth {
			gateway  		  = "%[7]s"
			prefix  		  = "%[8]s"
			mtu				  = "%[9]s"
		}
	}
	`, rName, nigName, displayName, availabilityZone, region, groupType, gateway, prefix, mtu)
}

func testNetworkInterfaceGroupConfigNoOptionalValues(rName, nigName, availabilityZone, region, gateway, prefix string) string {
	return fmt.Sprintf(`
	resource "fusion_network_interface_group" "%[1]s" {
		name          	  = "%[2]s"
		availability_zone = "%[3]s"
		region			  = "%[4]s"
		eth {
			gateway  		  = "%[5]s"
			prefix  		  = "%[6]s"
		}
	}
	`, rName, nigName, availabilityZone, region, gateway, prefix)
}

func testNetworkInterfaceGroupConfigRef(rName, nigName, availabilityZone, region, gateway, prefix string) string {
	return fmt.Sprintf(`
	resource "fusion_network_interface_group" "%[1]s" {
		name          	  = "%[2]s"
		availability_zone = fusion_availability_zone.%[3]s.name
		region			  = fusion_region.%[4]s.name
		eth {
			gateway  		  = "%[5]s"
			prefix  		  = "%[6]s"
		}
	}
	`, rName, nigName, availabilityZone, region, gateway, prefix)
}
