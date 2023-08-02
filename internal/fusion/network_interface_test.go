/*
Copyright 2023 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package fusion

import (
	"context"
	"fmt"
	"regexp"
	"testing"

	"github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/utilities"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"
)

// TODO: add FC management once it is supported

var nisToTestOn *NetworkInterfacesTestData

var skip = func(t *testing.T) {
	t.SkipNow()
}

func TestAccNetworkInterface_all(t *testing.T) {
	skip(t)

	nis, release := FindNetworkInterfacesForTests(t, 0, 2)
	nisToTestOn = &nis
	defer func() {
		release()
		nisToTestOn = nil
	}()
	// import tests should be before other test suits in case there will be other test suits which require any modifications to interface
	t.Run("testAccNetworkInterfaceDataSource_import", testAccNetworkInterface_import)
	t.Run("testAccNetworkInterface_update", testAccNetworkInterface_update)
	t.Run("testAccNetworkInterfaceDataSource_basic", testAccNetworkInterfaceDataSource_basic)
}

func testAccNetworkInterface_update(t *testing.T) {
	// network interfaces do not really have create/delete parts, so bundle everything
	// in one big test (update)

	tfName := acctest.RandomWithPrefix("ni_test_tf_ni")
	tfPath := "fusion_network_interface." + tfName
	displayName1 := acctest.RandomWithPrefix("ni_test_display_name")
	displayName2 := acctest.RandomWithPrefix("ni_test_display_name")

	niAz, niRegion, niArray, niName, _, niWwn, _, niEnabled := getIfaceProperties(nisToTestOn.FcInterfaces[0])

	// So right now freshly deployed control planes comes up with arrays that have an Eth address, but no network interface group.
	// This is tolerated if the array is freshly registered, but not allowed when anything is changed; the address and group have
	// to be set or cleared together, which means the interfaces cannot be returned to their original state. Also, changing the
	// address seemed to break a testbed and the provider is not written in a way to be able to ignore some properties, that would
	// make the implementation much more complex.
	// These two things together are pretty problematic, as one shouldn't change assigned address, but also then can't change anything
	// without adding a network interface group, which has to be appropriate for given interface address, so the group would have
	// to be dynamically generated to suit the addresses, which are dynamic. This would make for pretty complicated tests, which
	// are already complex due to the fact that there is fixed number of dynamic network interfaces tied to fixed number of
	// available dynamic arrays.
	//
	// For now, leave problem of testing Eth interfaces to future us (sorry future us) and test only FC display name, that
	// will serve at least as a smoke test to verify the network interfaces are not completely broken.
	// TODO: Figure out how to test changing Eth addresses and network interface groups on both testbeds and doubleagents without
	// breaking either.

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		Steps: []resource.TestStep{
			// create initial and check
			{
				Config: testNetworkInterfaceConfig(tfName, niName, displayName1, niRegion, niAz, niArray, "", "fc", "", niWwn, niEnabled),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(tfPath, "name", niName),
					resource.TestCheckResourceAttr(tfPath, "display_name", displayName1),
					testNetworkInterfaceExists(tfPath),
				),
			},
			// change display name
			{
				Config: testNetworkInterfaceConfig(tfName, niName, displayName2, niRegion, niAz, niArray, "", "fc", "", niWwn, niEnabled),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(tfPath, "name", niName),
					resource.TestCheckResourceAttr(tfPath, "display_name", displayName2),
					testNetworkInterfaceExists(tfPath),
				),
			},
		},
	})
}

func testAccNetworkInterface_import(t *testing.T) {
	// import feature doesn't require any update steps before tests, so we just import existing network interface
	tfName := acctest.RandomWithPrefix("ni_test_tf_ni")
	tfPath := "fusion_network_interface." + tfName

	// using not updated interface
	niAz, niRegion, niArray, niName, _, _, _, _ := getIfaceProperties(nisToTestOn.FcInterfaces[1])

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		Steps: []resource.TestStep{
			{
				ImportState:      true,
				ResourceName:     tfPath,
				ImportStateId:    fmt.Sprintf("/regions/%[1]s/availability-zones/%[2]s/arrays/%[3]s/network-interfaces/%[4]s", niRegion, niAz, niArray, niName),
				ImportStateCheck: testIfaceImportStateCheck(nisToTestOn.FcInterfaces[1]),
				Config:           testEmptyNetworkInterfaceConfigForImport(tfName),
			},
			{
				ImportState:   true,
				ResourceName:  tfPath,
				ImportStateId: fmt.Sprintf("/regions/%[1]s/availability-zones/%[2]s/arrays/%[3]s/network-interfaces/wrong-%[4]s", niRegion, niAz, niArray, niName),
				ExpectError:   regexp.MustCompile("Not Found"),
				Config:        testEmptyNetworkInterfaceConfigForImport(tfName),
			},
			{
				ImportState:   true,
				ResourceName:  tfPath,
				ImportStateId: fmt.Sprintf("/network-interfaces/%[4]s", niRegion, niAz, niArray, niName),
				ExpectError:   regexp.MustCompile("invalid interface import path. Expected path in format '/regions/<region>/availability-zones/<availability-zone>/arrays/<array>/network-interfaces/<network-interface>'"),
				Config:        testEmptyNetworkInterfaceConfigForImport(tfName),
			},
		},
	})
}

func testAccNetworkInterfaceDataSource_basic(t *testing.T) {
	dsNameConfig := acctest.RandomWithPrefix("ni_test_ds")

	niTfName1 := acctest.RandomWithPrefix("ni_test_tf_ni")
	niTfName2 := acctest.RandomWithPrefix("ni_test_tf_ni")
	niTfDisplayName1 := acctest.RandomWithPrefix("ni_test_display_name")
	niTfDisplayName2 := acctest.RandomWithPrefix("ni_test_display_name")

	ni1Az, ni1Region, ni1Array, ni1Name, _, ni1Wwn, _, ni1Enabled := getIfaceProperties(nisToTestOn.FcInterfaces[0])
	ni2Az, ni2Region, ni2Array, ni2Name, _, ni2Wwn, _, ni2Enabled := getIfaceProperties(nisToTestOn.FcInterfaces[1])

	networkInterfaces := []map[string]interface{}{
		{
			"name":              ni1Name,
			"display_name":      niTfDisplayName1,
			"region":            ni1Region,
			"availability_zone": ni1Az,
			"array":             ni1Array,
			"enabled":           ni1Enabled,
			"interface_type":    "fc",
			"fc": []interface{}{
				map[string]interface{}{
					"wwn": ni1Wwn,
				},
			},
		},
		{
			"name":              ni2Name,
			"display_name":      niTfDisplayName2,
			"region":            ni2Region,
			"availability_zone": ni2Az,
			"array":             ni2Array,
			"enabled":           ni2Enabled,
			"interface_type":    "fc",
			"fc": []interface{}{
				map[string]interface{}{
					"wwn": ni2Wwn,
				},
			},
		},
	}

	interfaceConfig1 := testNetworkInterfaceConfig(niTfName1, ni1Name, niTfDisplayName1, ni1Region, ni1Az, ni1Array, "", "fc", "", ni1Wwn, ni1Enabled)
	interfaceConfig2 := testNetworkInterfaceConfig(niTfName2, ni2Name, niTfDisplayName2, ni2Region, ni2Az, ni2Array, "", "fc", "", ni2Wwn, ni2Enabled)

	resource.Test(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		Steps: []resource.TestStep{
			// Precreate interfaces
			{
				Config: interfaceConfig1 + interfaceConfig2,
			},
			// Check if they are contained in the data source
			{
				Config: interfaceConfig1 + interfaceConfig2 + testNetworkInterfaceDataSourceConfig(dsNameConfig, ni1Region, ni1Az, ni1Array),
				Check:  utilities.TestCheckDataSource("fusion_network_interface", dsNameConfig, "items", networkInterfaces),
			},
		},
	})
}

func testNetworkInterfaceExists(rName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		tfNetworkInterface, ok := s.RootModule().Resources[rName]
		if !ok {
			return fmt.Errorf("resource not found: %s", rName)
		}
		if tfNetworkInterface.Type != "fusion_network_interface" {
			return fmt.Errorf("expected type: fusion_network_interface. Found: %s", tfNetworkInterface.Type)
		}
		attrs := tfNetworkInterface.Primary.Attributes

		remote, _, err := testAccProvider.Meta().(*hmrest.APIClient).NetworkInterfacesApi.GetNetworkInterfaceById(context.Background(), attrs["id"], nil)
		if err != nil {
			return fmt.Errorf("go client returned error while searching for %s with %s. Error: %s", attrs["name"], attrs["id"], err)
		}

		errRef := &err
		checkAttr := func(key string, remote interface{}) bool {
			strRemote := fmt.Sprintf("%v", remote)
			attr := attrs[key]
			if strRemote != attr {
				*errRef = fmt.Errorf("mismatch attr: '%s' remote: '%s' tf config: '%s'", key, strRemote, attr)
				return false
			}
			return true
		}

		remoteNifg := ""
		if remote.NetworkInterfaceGroup != nil {
			remoteNifg = remote.NetworkInterfaceGroup.Name
		}
		if !checkAttr("name", remote.Name) ||
			!checkAttr("display_name", remote.DisplayName) ||
			!checkAttr("region", remote.Region.Name) ||
			!checkAttr("availability_zone", remote.AvailabilityZone.Name) ||
			!checkAttr("array", remote.Array.Name) ||
			!checkAttr("enabled", remote.Enabled) ||
			!checkAttr("network_interface_group", remoteNifg) {
			return err
		}

		switch remote.InterfaceType {
		case "eth":
			if !checkAttr("eth.0.address", remote.Eth.Address) {
				return err
			}
		case "fc":
			if !checkAttr("fc.0.wwn", remote.Fc.Wwn) {
				return err
			}
		}

		return nil
	}
}

func testNetworkInterfaceConfig(tfName, name, displayName, region, availabilityZone, array, networkInterfaceGroup string, ifaceType, ethAddress, fcWwn string, enabled bool) string {
	nifgLine := ""
	ethBlock := ""
	fcBlock := ""
	switch ifaceType {
	case "eth":
		if networkInterfaceGroup != "" {
			nifgLine = fmt.Sprintf(`network_interface_group = fusion_network_interface_group.%[1]s.name`, networkInterfaceGroup)
		} else {
			nifgLine = `network_interface_group = ""`
		}
		ethBlock = fmt.Sprintf(`eth {
			address = "%[1]s"
		}`, ethAddress)
	case "fc":
		fcBlock = fmt.Sprintf(`fc {
			wwn = "%[1]s"
		}`, fcWwn)
	}
	return fmt.Sprintf(`
	resource "fusion_network_interface" "%[1]s" {
		name				= "%[2]s"
		display_name		= "%[3]s"
		region 				= "%[4]s"
		availability_zone 	= "%[5]s"
		array				= "%[6]s"
		enabled				= %[7]t
		interface_type      = "%[8]s"
		%[9]s
		%[10]s
		%[11]s
	}
	`, tfName, name, displayName, region, availabilityZone, array, enabled, ifaceType, nifgLine, ethBlock, fcBlock)
}

func testNetworkInterfaceDataSourceConfig(dsName, region, availabilityZone, array string) string {
	return fmt.Sprintf(`data "fusion_network_interface" "%[1]s" {
		region = "%[2]s"
		availability_zone = "%[3]s"
		array = "%[4]s"
	}`, dsName, region, availabilityZone, array)
}

func getIfaceProperties(iface hmrest.NetworkInterface) (az, region, array, ifaceName, address, wwn, group string, enabled bool) {

	ifaceName = iface.Name
	group = ""
	if iface.NetworkInterfaceGroup != nil {
		group = iface.NetworkInterfaceGroup.Name
	}
	az = iface.AvailabilityZone.Name
	region = iface.Region.Name
	array = iface.Array.Name
	if iface.Eth != nil {
		address = iface.Eth.Address
	}
	if iface.Fc != nil {
		wwn = iface.Fc.Wwn
	}

	enabled = iface.Enabled

	return az, region, array, ifaceName, address, wwn, group, enabled
}

func testIfaceImportStateCheck(iface hmrest.NetworkInterface) resource.ImportStateCheckFunc {
	return func(is []*terraform.InstanceState) error {
		if len(is) != 1 {
			return fmt.Errorf("unexpected amount of imported states for interface, required 1")
		}
		state := is[0].Attributes
		if state[optionName] != iface.Name || state[optionDisplayName] != iface.DisplayName || state[optionRegion] != iface.Region.Name || state[optionAvailabilityZone] != iface.AvailabilityZone.Name || state[optionArray] != iface.Array.Name {
			return fmt.Errorf("imported state is not correct")
		}
		return nil
	}
}

func testEmptyNetworkInterfaceConfigForImport(tfName string) string {
	return fmt.Sprintf(`
	resource "fusion_network_interface" "%[1]s" {
	}
	`, tfName)
}
