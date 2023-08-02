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
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"
)

type storageEndpointTestConfig struct {
	RName, Name, DisplayName,
	Region, AZ string
}

func generateStorageEndpointTestConfigAndCommonTFConfig() (storageEndpointTestConfig, string) {
	cfg := storageEndpointTestConfig{
		Name:        acctest.RandomWithPrefix("se-test-name"),
		DisplayName: acctest.RandomWithPrefix("se-test-display-name"),
		Region:      acctest.RandomWithPrefix("se-test-region"),
		AZ:          acctest.RandomWithPrefix("se-test-az"),
	}
	cfg.RName = "fusion_storage_endpoint." + cfg.Name

	commonTFConfig := testRegionConfigNoDisplayName(cfg.Region, cfg.Region) +
		testAvailabilityZoneConfigRef(cfg.AZ, cfg.AZ, cfg.Region)

	return cfg, commonTFConfig
}

// Creates and destroys
func TestAccStorageEndpoint_basic(t *testing.T) {
	utilities.CheckTestSkip(t)

	cfg1, tfConfig1 := generateStorageEndpointTestConfigAndCommonTFConfig()
	cfg2, tfConfig2 := generateStorageEndpointTestConfigAndCommonTFConfig()

	nigName := acctest.RandomWithPrefix("nig_se_test")
	nigsConfig := testNetworkInterfaceGroupConfigRef(nigName, nigName, cfg1.AZ, cfg1.Region, "10.21.200.1", "10.21.200.0/24")

	discoveryInterfaces := []map[string]interface{}{
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

	iscsi := []map[string]interface{}{{
		"discovery_interfaces": discoveryInterfaces,
	}}

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckStorageEndpointDestroy,
		Steps: []resource.TestStep{
			// Create Storage Endpoint with Iscsi and validate it's fields
			{
				Config: tfConfig1 + nigsConfig + testStorageEndpointConfig(cfg1.Name, cfg1.Name, cfg1.DisplayName, cfg1.Region, cfg1.AZ, iscsi),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(cfg1.RName, "name", cfg1.Name),
					resource.TestCheckResourceAttr(cfg1.RName, "display_name", cfg1.DisplayName),
					resource.TestCheckResourceAttr(cfg1.RName, "region", cfg1.Region),
					resource.TestCheckResourceAttr(cfg1.RName, "availability_zone", cfg1.AZ),
					testCheckStorageEndpointIscsiDiscoveryInterfaces(cfg1.RName, discoveryInterfaces),
					testStorageEndpointExistsIscsi(cfg1.RName, iscsi),
				),
			},
			// Create Storage Endpoint with Cbs Azure Iscsi and validate it's fields
			{
				Config: tfConfig2 + testStorageEndpointConfigWithCbsAzureIscsi(cfg2.Name, cfg2.Name, cfg2.DisplayName, cfg2.Region, cfg2.AZ, "identity", "lb-id", `["127.0.0.1","127.0.0.2"]`),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(cfg2.RName, "name", cfg2.Name),
					resource.TestCheckResourceAttr(cfg2.RName, "display_name", cfg2.DisplayName),
					resource.TestCheckResourceAttr(cfg2.RName, "region", cfg2.Region),
					resource.TestCheckResourceAttr(cfg2.RName, "availability_zone", cfg2.AZ),
					resource.TestCheckResourceAttr(cfg2.RName, "cbs_azure_iscsi.0.storage_endpoint_collection_identity", "identity"),
					resource.TestCheckResourceAttr(cfg2.RName, "cbs_azure_iscsi.0.load_balancer", "lb-id"),
					resource.TestCheckResourceAttr(cfg2.RName, "cbs_azure_iscsi.0.load_balancer_addresses.0", "127.0.0.1"),
					resource.TestCheckResourceAttr(cfg2.RName, "cbs_azure_iscsi.0.load_balancer_addresses.1", "127.0.0.2"),
					testStorageEndpointExistsCbsAzureIscsi(cfg2.RName),
				),
			},
		},
	})
}

func TestAccStorageEndpoint_update(t *testing.T) {
	utilities.CheckTestSkip(t)

	cfg, tfConfig := generateStorageEndpointTestConfigAndCommonTFConfig()
	discoveryInterfaces := []map[string]interface{}{
		{
			"address": "10.21.200.121/24",
		},
	}

	iscsi := []map[string]interface{}{{
		"discovery_interfaces": discoveryInterfaces,
	}}

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckStorageEndpointDestroy,
		Steps: []resource.TestStep{
			// Create Storage Endpoint and validate it's fields
			{
				Config: tfConfig + testStorageEndpointConfigNoDisplayName(cfg.Name, cfg.Name, cfg.Region, cfg.AZ, iscsi),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(cfg.RName, "name", cfg.Name),
					resource.TestCheckResourceAttr(cfg.RName, "display_name", cfg.Name),
					resource.TestCheckResourceAttr(cfg.RName, "region", cfg.Region),
					resource.TestCheckResourceAttr(cfg.RName, "availability_zone", cfg.AZ),
					testCheckStorageEndpointIscsiDiscoveryInterfaces(cfg.RName, discoveryInterfaces),
					testStorageEndpointExistsIscsi(cfg.RName, iscsi),
				),
			},
			// Update display name
			{
				Config: tfConfig + testStorageEndpointConfig(cfg.Name, cfg.Name, cfg.DisplayName, cfg.Region, cfg.AZ, iscsi),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(cfg.RName, "name", cfg.Name),
					resource.TestCheckResourceAttr(cfg.RName, "display_name", cfg.DisplayName),
					resource.TestCheckResourceAttr(cfg.RName, "region", cfg.Region),
					resource.TestCheckResourceAttr(cfg.RName, "availability_zone", cfg.AZ),
					testCheckStorageEndpointIscsiDiscoveryInterfaces(cfg.RName, discoveryInterfaces),
					testStorageEndpointExistsIscsi(cfg.RName, iscsi),
				),
			},
			// Can't update immutable fields
			{
				Config:      tfConfig + testStorageEndpointConfig(cfg.Name, "immutable", cfg.DisplayName, cfg.Region, cfg.AZ, iscsi),
				ExpectError: regexp.MustCompile("attempting to update an immutable field"),
			},
		},
	})
}

func TestAccStorageEndpoint_multiple(t *testing.T) {
	utilities.CheckTestSkip(t)

	cfg1, tfConfig1 := generateStorageEndpointTestConfigAndCommonTFConfig()
	iscsi := []map[string]interface{}{
		{
			"discovery_interfaces": []map[string]interface{}{
				{"address": "10.21.200.121/24"},
			},
		},
	}

	cfg2, tfConfig2 := generateStorageEndpointTestConfigAndCommonTFConfig()
	iscsi2 := []map[string]interface{}{
		{
			"discovery_interfaces": []map[string]interface{}{
				{
					"address": "10.21.200.122/24",
					"gateway": "10.21.200.1",
				},
			},
		},
	}

	commonConfig := tfConfig1 + tfConfig2
	seConfig := testStorageEndpointConfig(cfg1.Name, cfg1.Name, cfg1.DisplayName, cfg1.Region, cfg1.AZ, iscsi)
	seConfig2 := testStorageEndpointConfig(cfg2.Name, cfg2.Name, cfg2.DisplayName, cfg2.Region, cfg2.AZ, iscsi2)
	seConfig3 := testStorageEndpointConfig("conflictR", cfg1.Name, cfg1.DisplayName, cfg1.Region, cfg1.AZ, iscsi2)

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckStorageEndpointDestroy,
		Steps: []resource.TestStep{
			// Multiple SE can be created
			{
				Config: commonConfig + seConfig + seConfig2,
				Check: resource.ComposeTestCheckFunc(
					testStorageEndpointExistsIscsi(cfg1.RName, iscsi),
					testStorageEndpointExistsIscsi(cfg2.RName, iscsi2),
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

func TestAccStorageEndpoint_import(t *testing.T) {
	utilities.CheckTestSkip(t)

	cfg, tfConfig := generateStorageEndpointTestConfigAndCommonTFConfig()

	nigName := acctest.RandomWithPrefix("nig_se_test")
	nigsConfig := testNetworkInterfaceGroupConfigRef(nigName, nigName, cfg.AZ, cfg.Region, "10.21.200.1", "10.21.200.0/24")

	discoveryInterfaces := []map[string]interface{}{
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

	iscsi := []map[string]interface{}{{
		"discovery_interfaces": discoveryInterfaces,
	}}

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckStorageEndpointDestroy,
		Steps: []resource.TestStep{
			// Create Storage Endpoint with Iscsi and validate it's fields
			{
				Config: tfConfig + nigsConfig + testStorageEndpointConfig(cfg.Name, cfg.Name, cfg.DisplayName, cfg.Region, cfg.AZ, iscsi),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(cfg.RName, "name", cfg.Name),
					resource.TestCheckResourceAttr(cfg.RName, "display_name", cfg.DisplayName),
					resource.TestCheckResourceAttr(cfg.RName, "region", cfg.Region),
					resource.TestCheckResourceAttr(cfg.RName, "availability_zone", cfg.AZ),
					testStorageEndpointExistsIscsi(cfg.RName, iscsi),
				),
			},
			{
				ImportState:       true,
				ResourceName:      fmt.Sprintf("fusion_storage_endpoint.%s", cfg.Name),
				ImportStateId:     fmt.Sprintf("/regions/%[1]s/availability-zones/%[2]s/storage-endpoints/%[3]s", cfg.Region, cfg.AZ, cfg.Name),
				ImportStateVerify: true,
			},
			{
				ImportState:   true,
				ResourceName:  fmt.Sprintf("fusion_storage_endpoint.%s", cfg.Name),
				ImportStateId: fmt.Sprintf("/regions/%[1]s/availability-zones/%[2]s/storage-endpoints/wrong-%[3]s", cfg.Region, cfg.AZ, cfg.Name),
				ExpectError:   regexp.MustCompile("Not Found"),
			},
			{
				ImportState:   true,
				ResourceName:  fmt.Sprintf("fusion_storage_endpoint.%s", cfg.Name),
				ImportStateId: fmt.Sprintf("/storage-endpoints/%[3]s", cfg.Region, cfg.AZ, cfg.Name),
				ExpectError:   regexp.MustCompile("invalid storage_endpoint import path. Expected path in format '/regions/<region>/availability-zones/<availability-zone>/storage-endpoints/<storage-endpoint>'"),
			},
		},
	})
}

func testStorageEndpointExistsIscsi(rName string, iscsi []map[string]interface{}) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		tfResource, actualSe, err := testStorageEndpointExistsBase(s, rName)
		if err != nil {
			return err
		}

		attrs := tfResource.Primary.Attributes

		if strings.Compare(actualSe.Name, attrs["name"]) != 0 ||
			strings.Compare(actualSe.DisplayName, attrs["display_name"]) != 0 ||
			!testStorageEndpointMatchDiscoveryInterfaces(iscsi[0]["discovery_interfaces"].([]map[string]interface{}), actualSe.Iscsi.DiscoveryInterfaces) {
			return fmt.Errorf("terraform storage endpoint doesn't match goclients storage endpoint")
		}
		return nil
	}
}

func testStorageEndpointExistsCbsAzureIscsi(rName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		tfResource, actualSe, err := testStorageEndpointExistsBase(s, rName)
		if err != nil {
			return err
		}

		attrs := tfResource.Primary.Attributes

		if strings.Compare(actualSe.Name, attrs["name"]) != 0 ||
			strings.Compare(actualSe.DisplayName, attrs["display_name"]) != 0 ||
			strings.Compare(actualSe.CbsAzureIscsi.LoadBalancer, attrs["cbs_azure_iscsi.0.load_balancer"]) != 0 ||
			strings.Compare(actualSe.CbsAzureIscsi.StorageEndpointCollectionIdentity, attrs["cbs_azure_iscsi.0.storage_endpoint_collection_identity"]) != 0 {
			return fmt.Errorf("terraform storage endpoint doesn't match goclients storage endpoint")
		}
		return nil
	}
}

func testStorageEndpointExistsBase(s *terraform.State, rName string) (*terraform.ResourceState, hmrest.StorageEndpoint, error) {
	tfResource, ok := s.RootModule().Resources[rName]
	if !ok {
		return nil, hmrest.StorageEndpoint{}, fmt.Errorf("resource not found: %s", rName)
	}

	if tfResource.Type != "fusion_storage_endpoint" {
		return nil, hmrest.StorageEndpoint{}, fmt.Errorf("expected type: fusion_storage_endpoint. Found: %s", tfResource.Type)
	}

	attrs := tfResource.Primary.Attributes
	actualSe, _, err := testAccProvider.Meta().(*hmrest.APIClient).StorageEndpointsApi.GetStorageEndpointById(
		context.Background(),
		attrs["id"],
		nil,
	)

	if err != nil {
		return nil,
			hmrest.StorageEndpoint{},
			fmt.Errorf("go client returned error while searching for %s by id: %s. Error: %s", attrs["name"], attrs["id"], err)
	}

	return tfResource, actualSe, nil
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

func testStorageEndpointDiscoveryInterfacesConfig(discIntsList []map[string]interface{}) string {
	discIntsStrings := make([]string, 0)

	for _, discInt := range discIntsList {
		gatewayField := ""
		if gateway, ok := discInt["gateway"]; ok {
			gatewayField = fmt.Sprintf("gateway = \"%s\"", gateway)
		}

		nigsField := ""
		if nigList, ok := discInt["network_interface_groups"]; ok {
			nigs := make([]string, 0)

			for _, nig := range nigList.([]interface{}) {
				nigs = append(nigs, fmt.Sprintf("fusion_network_interface_group.%s.name", nig))
			}

			nigsField = fmt.Sprintf("network_interface_groups = [%s]", strings.Join(nigs, ","))
		}

		discIntsString := fmt.Sprintf(`
		discovery_interfaces {
			address = "%[1]s"
			%[2]s
			%[3]s
		}
		`, discInt["address"], gatewayField, nigsField)

		discIntsStrings = append(discIntsStrings, discIntsString)
	}

	return strings.Join(discIntsStrings, "")
}

func testStorageEndpointConfig(
	rName, storageEndpointName, displayName, region, availabilityZone string, iscsi []map[string]interface{},
) string {
	discoveryInterfaces := iscsi[0]["discovery_interfaces"].([]map[string]interface{})
	return fmt.Sprintf(`
	resource "fusion_storage_endpoint" "%[1]s" {
		name				= "%[2]s"
		display_name		= "%[3]s"
		region				= fusion_region.%[4]s.name
		availability_zone	= fusion_availability_zone.%[5]s.name
                iscsi {
		        %[6]s
                }
	}
	`, rName, storageEndpointName, displayName, region, availabilityZone, testStorageEndpointDiscoveryInterfacesConfig(discoveryInterfaces))
}

func testStorageEndpointConfigNoDisplayName(
	rName, storageEndpointName, region, availabilityZone string, iscsi []map[string]interface{},
) string {
	discoveryInterfaces := iscsi[0]["discovery_interfaces"].([]map[string]interface{})
	return fmt.Sprintf(`
	resource "fusion_storage_endpoint" "%[1]s" {
		name				= "%[2]s"
		region				= fusion_region.%[3]s.name
		availability_zone	= fusion_availability_zone.%[4]s.name
                iscsi {
		        %[5]s
                }
	}
	`, rName, storageEndpointName, region, availabilityZone, testStorageEndpointDiscoveryInterfacesConfig(discoveryInterfaces))
}

func testStorageEndpointConfigWithCbsAzureIscsi(
	rName, storageEndpointName, displayName, region, availabilityZone, collectionIdentity, loadBalancer, loadBalancerAddresses string,
) string {
	return fmt.Sprintf(`
	resource "fusion_storage_endpoint" "%[1]s" {
		name				= "%[2]s"
		display_name		= "%[3]s"
		region				= fusion_region.%[4]s.name
		availability_zone	= fusion_availability_zone.%[5]s.name
		cbs_azure_iscsi {
			storage_endpoint_collection_identity = "%[6]s"
			load_balancer                        = "%[7]s"
			load_balancer_addresses              = %[8]s
		}
	}
	`, rName, storageEndpointName, displayName, region, availabilityZone, collectionIdentity, loadBalancer, loadBalancerAddresses)
}

func testCheckStorageEndpointIscsiDiscoveryInterfaces(resourceName string, discoveryInterfaces []map[string]interface{}) resource.TestCheckFunc {
	// Test length
	testChecks := []resource.TestCheckFunc{
		resource.TestCheckResourceAttr(resourceName, "iscsi.0.discovery_interfaces.#", strconv.Itoa(len(discoveryInterfaces))),
	}

	// Checks that correct discovery interface blocks are present (ignores nig)
	for _, discoveryInterface := range discoveryInterfaces {
		discoveryInterfaceCompareMap := make(map[string]string)

		for key, value := range discoveryInterface {
			if strValue, ok := value.(string); ok {
				discoveryInterfaceCompareMap[key] = strValue
			}
		}

		testChecks = append(testChecks, resource.TestCheckTypeSetElemNestedAttrs(resourceName, "iscsi.0.discovery_interfaces.*", discoveryInterfaceCompareMap))
	}

	return resource.ComposeAggregateTestCheckFunc(testChecks...)
}

func testStorageEndpointMatchDiscoveryInterfaces(expectedDiscInts []map[string]interface{}, discInts []hmrest.StorageEndpointIscsiDiscoveryInterface) bool {
	if len(expectedDiscInts) != len(discInts) {
		return false
	}

	for _, expectedDiscInt := range expectedDiscInts {
		found := false

		for _, iscsi := range discInts {
			if iscsi.Address != expectedDiscInt["address"] {
				continue
			}

			if v, ok := expectedDiscInt["gateway"]; ok && v != iscsi.Gateway {
				continue
			}

			if v, ok := expectedDiscInt["network_interface_groups"]; ok {
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
