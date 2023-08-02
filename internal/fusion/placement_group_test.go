/*
Copyright 2023 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package fusion

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/utilities"
	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"
)

type placementGroupTestConfig struct {
	RName, Name, DisplayName,
	Tenant, TenantSpace,
	Region, AZ,
	StorageService string
}

func generatePlacementGroupTestConfigAndCommonTFConfig(hwTypes []string) (placementGroupTestConfig, string) {
	cfg := placementGroupTestConfig{
		Name:           acctest.RandomWithPrefix("pg-test-name"),
		DisplayName:    acctest.RandomWithPrefix("pg-test-display-name"),
		Tenant:         acctest.RandomWithPrefix("pg-test-tenant"),
		TenantSpace:    acctest.RandomWithPrefix("pg-test-ts"),
		Region:         preexistingRegion,
		AZ:             preexistingAvailabilityZone,
		StorageService: acctest.RandomWithPrefix("pg-test-ss"),
	}
	cfg.RName = "fusion_placement_group." + cfg.Name

	commonTFConfig := testTenantConfig(cfg.Tenant, cfg.Tenant, cfg.Tenant) +
		testTenantSpaceConfigWithRefs(cfg.TenantSpace, cfg.TenantSpace, cfg.TenantSpace, cfg.Tenant) +
		testStorageServiceConfig(cfg.StorageService, cfg.StorageService, cfg.StorageService, hwTypes)

	return cfg, commonTFConfig
}

func TestAccPlacementGroup_basic(t *testing.T) {
	utilities.CheckTestSkip(t)

	arrays := getArraysInPreexistingRegionAndAZ(t)
	if len(arrays) == 0 {
		t.Error("did not find any arrays to test on !")
	}
	hwTypes := getHWTypesFromArrays(arrays)

	cfg1, tfConfig1 := generatePlacementGroupTestConfigAndCommonTFConfig([]string{hwTypes[0]})
	cfg2, tfConfig2 := generatePlacementGroupTestConfigAndCommonTFConfig([]string{arrays[0].HardwareType.Name})
	arrayNameForCreation := arrays[0].Name

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckPlacementGroupDestroy,
		Steps: []resource.TestStep{
			// Create placement group
			{
				Config: tfConfig1 + testPlacementGroupConfigWithRefsNoArray(cfg1.Name, cfg1.Name, cfg1.DisplayName, cfg1.Tenant, cfg1.TenantSpace, cfg1.Region, cfg1.AZ, cfg1.StorageService, false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(cfg1.RName, "name", cfg1.Name),
					resource.TestCheckResourceAttr(cfg1.RName, "display_name", cfg1.DisplayName),
					resource.TestCheckResourceAttr(cfg1.RName, "tenant", cfg1.Tenant),
					resource.TestCheckResourceAttr(cfg1.RName, "tenant_space", cfg1.TenantSpace),
					resource.TestCheckResourceAttr(cfg1.RName, "availability_zone", cfg1.AZ),
					resource.TestCheckResourceAttr(cfg1.RName, "region", cfg1.Region),
					resource.TestCheckResourceAttr(cfg1.RName, "storage_service", cfg1.StorageService),
					checkArrayCorrectnessInPlacementGroup(cfg1.RName, arrays, []string{hwTypes[0]}),
					testPlacementGroupExists(t, cfg1.RName),
				),
			},
			// Create placement group with array field set
			{
				Config: tfConfig2 + testPlacementGroupConfigWithRefs(cfg2.Name, cfg2.Name, cfg2.DisplayName, cfg2.Tenant, cfg2.TenantSpace, cfg2.Region, cfg2.AZ, cfg2.StorageService, arrayNameForCreation, false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(cfg2.RName, "name", cfg2.Name),
					resource.TestCheckResourceAttr(cfg2.RName, "display_name", cfg2.DisplayName),
					resource.TestCheckResourceAttr(cfg2.RName, "tenant", cfg2.Tenant),
					resource.TestCheckResourceAttr(cfg2.RName, "tenant_space", cfg2.TenantSpace),
					resource.TestCheckResourceAttr(cfg2.RName, "availability_zone", cfg2.AZ),
					resource.TestCheckResourceAttr(cfg2.RName, "region", cfg2.Region),
					resource.TestCheckResourceAttr(cfg2.RName, "storage_service", cfg2.StorageService),
					resource.TestCheckResourceAttr(cfg2.RName, "array", arrayNameForCreation),
					testPlacementGroupExists(t, cfg2.RName),
				),
			},
		},
	})
}

func TestAccPlacementGroup_update(t *testing.T) {
	utilities.CheckTestSkip(t)

	arrays := getArraysInPreexistingRegionAndAZ(t)
	if len(arrays) == 0 {
		t.Error("did not find any arrays to test on !")
	}
	hwTypes := getHWTypesFromArrays(arrays)

	cfg, commonConfig := generatePlacementGroupTestConfigAndCommonTFConfig(hwTypes)
	updatedDisplayName := "updated-display-name"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckPlacementGroupDestroy,
		Steps: []resource.TestStep{
			// Create placement group
			{
				Config: commonConfig + testPlacementGroupConfigWithRefsNoArray(cfg.Name, cfg.Name, cfg.DisplayName, cfg.Tenant, cfg.TenantSpace, cfg.Region, cfg.AZ, cfg.StorageService, false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(cfg.RName, "name", cfg.Name),
					resource.TestCheckResourceAttr(cfg.RName, "display_name", cfg.DisplayName),
					resource.TestCheckResourceAttr(cfg.RName, "tenant", cfg.Tenant),
					resource.TestCheckResourceAttr(cfg.RName, "tenant_space", cfg.TenantSpace),
					resource.TestCheckResourceAttr(cfg.RName, "availability_zone", cfg.AZ),
					resource.TestCheckResourceAttr(cfg.RName, "region", cfg.Region),
					resource.TestCheckResourceAttr(cfg.RName, "storage_service", cfg.StorageService),
					resource.TestCheckResourceAttr(cfg.RName, "destroy_snapshots_on_delete", "false"),
					checkArrayCorrectnessInPlacementGroup(cfg.RName, arrays, hwTypes),
					testPlacementGroupExists(t, cfg.RName),
				),
			},
			// Update destroy snapshots on delete
			{
				Config: commonConfig + testPlacementGroupConfigWithRefsNoArray(cfg.Name, cfg.Name, cfg.DisplayName, cfg.Tenant, cfg.TenantSpace, cfg.Region, cfg.AZ, cfg.StorageService, true),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(cfg.RName, "name", cfg.Name),
					resource.TestCheckResourceAttr(cfg.RName, "display_name", cfg.DisplayName),
					resource.TestCheckResourceAttr(cfg.RName, "tenant", cfg.Tenant),
					resource.TestCheckResourceAttr(cfg.RName, "tenant_space", cfg.TenantSpace),
					resource.TestCheckResourceAttr(cfg.RName, "availability_zone", cfg.AZ),
					resource.TestCheckResourceAttr(cfg.RName, "region", cfg.Region),
					resource.TestCheckResourceAttr(cfg.RName, "storage_service", cfg.StorageService),
					resource.TestCheckResourceAttr(cfg.RName, "destroy_snapshots_on_delete", "true"),
					checkArrayCorrectnessInPlacementGroup(cfg.RName, arrays, hwTypes),
					testPlacementGroupExists(t, cfg.RName),
				),
			},
			// Update display name
			{
				Config: commonConfig + testPlacementGroupConfigWithRefsNoArray(cfg.Name, cfg.Name, updatedDisplayName, cfg.Tenant, cfg.TenantSpace, cfg.Region, cfg.AZ, cfg.StorageService, true),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(cfg.RName, "name", cfg.Name),
					resource.TestCheckResourceAttr(cfg.RName, "display_name", updatedDisplayName),
					resource.TestCheckResourceAttr(cfg.RName, "tenant", cfg.Tenant),
					resource.TestCheckResourceAttr(cfg.RName, "tenant_space", cfg.TenantSpace),
					resource.TestCheckResourceAttr(cfg.RName, "availability_zone", cfg.AZ),
					resource.TestCheckResourceAttr(cfg.RName, "region", cfg.Region),
					resource.TestCheckResourceAttr(cfg.RName, "storage_service", cfg.StorageService),
					resource.TestCheckResourceAttr(cfg.RName, "destroy_snapshots_on_delete", "true"),
					checkArrayCorrectnessInPlacementGroup(cfg.RName, arrays, hwTypes),
					testPlacementGroupExists(t, cfg.RName),
				),
			},
			// Update array
			{
				Config: commonConfig + testPlacementGroupConfigWithRefs(cfg.Name, cfg.Name, updatedDisplayName, cfg.Tenant, cfg.TenantSpace, cfg.Region, cfg.AZ, cfg.StorageService, arrays[rand.Intn(len(arrays))].Name, true),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(cfg.RName, "name", cfg.Name),
					resource.TestCheckResourceAttr(cfg.RName, "display_name", updatedDisplayName),
					resource.TestCheckResourceAttr(cfg.RName, "tenant", cfg.Tenant),
					resource.TestCheckResourceAttr(cfg.RName, "tenant_space", cfg.TenantSpace),
					resource.TestCheckResourceAttr(cfg.RName, "availability_zone", cfg.AZ),
					resource.TestCheckResourceAttr(cfg.RName, "region", cfg.Region),
					resource.TestCheckResourceAttr(cfg.RName, "storage_service", cfg.StorageService),
					resource.TestCheckResourceAttr(cfg.RName, "destroy_snapshots_on_delete", "true"),
					checkArrayCorrectnessInPlacementGroup(cfg.RName, arrays, hwTypes),
					testPlacementGroupExists(t, cfg.RName),
				),
			},
		},
	})
}

func TestAccPlacementGroup_attributes(t *testing.T) {
	utilities.CheckTestSkip(t)

	cfg, commonConfig := generatePlacementGroupTestConfigAndCommonTFConfig([]string{"flash-array-x"})
	displayNameTooBig := strings.Repeat("a", 257)
	array := "fakeArray"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckPlacementGroupDestroy,
		Steps: []resource.TestStep{
			{
				Config:      commonConfig + testPlacementGroupConfigWithRefs(cfg.Name, "", cfg.DisplayName, cfg.Tenant, cfg.TenantSpace, cfg.Region, cfg.AZ, cfg.StorageService, array, false),
				ExpectError: regexp.MustCompile(`expected "name" to not be an empty string`),
			},
			{
				Config:      commonConfig + testPlacementGroupConfigWithRefs(cfg.Name, "bad name here", cfg.DisplayName, cfg.Tenant, cfg.TenantSpace, cfg.Region, cfg.AZ, cfg.StorageService, array, false),
				ExpectError: regexp.MustCompile(`name must use alphanumeric characters`),
			},
			{
				Config:      commonConfig + testPlacementGroupConfigWithRefs(cfg.Name, cfg.Name, "", cfg.Tenant, cfg.TenantSpace, cfg.Region, cfg.AZ, cfg.StorageService, array, false),
				ExpectError: regexp.MustCompile(`expected length of display_name to be in the range \(1 - 256\)`),
			},
			{
				Config:      commonConfig + testPlacementGroupConfigWithRefs(cfg.Name, cfg.Name, displayNameTooBig, cfg.Tenant, cfg.TenantSpace, cfg.Region, cfg.AZ, cfg.StorageService, array, false),
				ExpectError: regexp.MustCompile(`expected length of display_name to be in the range \(1 - 256\)`),
			},
			{
				Config:      commonConfig + testPlacementGroupConfigWithValues(cfg.Name, cfg.Name, cfg.DisplayName, "", cfg.TenantSpace, cfg.Region, cfg.AZ, cfg.StorageService, array, false),
				ExpectError: regexp.MustCompile(`expected "tenant" to not be an empty string`),
			},
			{
				Config:      commonConfig + testPlacementGroupConfigWithValues(cfg.Name, cfg.Name, cfg.DisplayName, cfg.Tenant, "", cfg.Region, cfg.AZ, cfg.StorageService, array, false),
				ExpectError: regexp.MustCompile(`expected "tenant_space" to not be an empty string`),
			},
			{
				Config:      commonConfig + testPlacementGroupConfigWithRefs(cfg.Name, cfg.Name, cfg.DisplayName, cfg.Tenant, cfg.TenantSpace, "", cfg.AZ, cfg.StorageService, array, false),
				ExpectError: regexp.MustCompile(`expected "region" to not be an empty string`),
			},
			{
				Config:      commonConfig + testPlacementGroupConfigWithRefs(cfg.Name, cfg.Name, cfg.DisplayName, cfg.Tenant, cfg.TenantSpace, cfg.Region, "", cfg.StorageService, array, false),
				ExpectError: regexp.MustCompile(`expected "availability_zone" to not be an empty string`),
			},
			{
				Config:      commonConfig + testPlacementGroupConfigWithValues(cfg.Name, cfg.Name, cfg.DisplayName, cfg.Tenant, cfg.TenantSpace, cfg.Region, cfg.AZ, "", array, false),
				ExpectError: regexp.MustCompile(`expected "storage_service" to not be an empty string`),
			},
			{ // this is here intentionally as the test fails to destroy state otherwise
				Config: commonConfig,
			},
		},
	})
}

func TestAccPlacementGroup_multiple(t *testing.T) {
	utilities.CheckTestSkip(t)

	cfg1, tfConfig1 := generatePlacementGroupTestConfigAndCommonTFConfig([]string{"flash-array-x"})
	cfg2, tfConfig2 := generatePlacementGroupTestConfigAndCommonTFConfig([]string{"flash-array-x"})

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckPlacementGroupDestroy,
		Steps: []resource.TestStep{
			// Create placement group
			{
				Config: tfConfig1 + testPlacementGroupConfigWithRefsNoArray(cfg1.Name, cfg1.Name, cfg1.DisplayName, cfg1.Tenant, cfg1.TenantSpace, cfg1.Region, cfg1.AZ, cfg1.StorageService, false) +
					tfConfig2 + testPlacementGroupConfigWithRefsNoArray(cfg2.Name, cfg2.Name, cfg2.DisplayName, cfg2.Tenant, cfg2.TenantSpace, cfg2.Region, cfg2.AZ, cfg2.StorageService, false),
				Check: resource.ComposeTestCheckFunc(
					testPlacementGroupExists(t, cfg1.RName),
					testPlacementGroupExists(t, cfg2.RName),
				),
			},
			// Create two with same name
			{
				Config: tfConfig1 + testPlacementGroupConfigWithRefsNoArray(cfg1.Name, cfg1.Name, cfg1.DisplayName, cfg1.Tenant, cfg1.TenantSpace, cfg1.Region, cfg1.AZ, cfg1.StorageService, false) +
					tfConfig2 + testPlacementGroupConfigWithRefsNoArray(cfg2.Name, cfg2.Name, cfg2.DisplayName, cfg2.Tenant, cfg2.TenantSpace, cfg2.Region, cfg2.AZ, cfg2.StorageService, false) +
					testPlacementGroupConfigWithRefsNoArray("conflictRN", cfg1.Name, cfg1.DisplayName, cfg1.Tenant, cfg1.TenantSpace, cfg1.Region, cfg1.AZ, cfg1.StorageService, false),
				ExpectError: regexp.MustCompile("already exists"),
			},
		},
	})
}

func TestAccPlacementGroup_import(t *testing.T) {
	utilities.CheckTestSkip(t)

	arrays := getArraysInPreexistingRegionAndAZ(t)
	if len(arrays) == 0 {
		t.Error("did not find any arrays to test on !")
	}
	hwTypes := getHWTypesFromArrays(arrays)

	cfg, tfConfig := generatePlacementGroupTestConfigAndCommonTFConfig([]string{hwTypes[0]})

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckPlacementGroupDestroy,
		Steps: []resource.TestStep{
			// Create placement group
			{
				Config: tfConfig + testPlacementGroupConfigWithRefsNoArray(cfg.Name, cfg.Name, cfg.DisplayName, cfg.Tenant, cfg.TenantSpace, cfg.Region, cfg.AZ, cfg.StorageService, false),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(cfg.RName, "name", cfg.Name),
					resource.TestCheckResourceAttr(cfg.RName, "display_name", cfg.DisplayName),
					resource.TestCheckResourceAttr(cfg.RName, "tenant", cfg.Tenant),
					resource.TestCheckResourceAttr(cfg.RName, "tenant_space", cfg.TenantSpace),
					resource.TestCheckResourceAttr(cfg.RName, "availability_zone", cfg.AZ),
					resource.TestCheckResourceAttr(cfg.RName, "region", cfg.Region),
					resource.TestCheckResourceAttr(cfg.RName, "storage_service", cfg.StorageService),
					checkArrayCorrectnessInPlacementGroup(cfg.RName, arrays, []string{hwTypes[0]}),
					testPlacementGroupExists(t, cfg.RName),
				),
			},
			{
				ImportState:       true,
				ResourceName:      fmt.Sprintf("fusion_placement_group.%s", cfg.Name),
				ImportStateId:     fmt.Sprintf("/tenants/%[1]s/tenant-spaces/%[2]s/placement-groups/%[3]s", cfg.Tenant, cfg.TenantSpace, cfg.Name),
				ImportStateVerify: true,
				// skipping destroy_snapshots_on_delete field, this field is used as additional parameter for deletion
				// destroy_snapshots_on_delete cannot be imported from harbormaster
				ImportStateVerifyIgnore: []string{optionDestroySnapshotsOnDelete},
			},
			{
				ImportState:   true,
				ResourceName:  fmt.Sprintf("fusion_placement_group.%s", cfg.Name),
				ImportStateId: fmt.Sprintf("/tenants/%[1]s/tenant-spaces/%[2]s/placement-groups/wrong-%[3]s", cfg.Tenant, cfg.TenantSpace, cfg.Name),
				ExpectError:   regexp.MustCompile("Not Found"),
			},
			{
				ImportState:   true,
				ResourceName:  fmt.Sprintf("fusion_placement_group.%s", cfg.Name),
				ImportStateId: fmt.Sprintf("/placement-groups/%[3]s", cfg.Tenant, cfg.TenantSpace, cfg.Name),
				ExpectError:   regexp.MustCompile("invalid placement_group import path. Expected path in format '/tenants/<tenant>/tenant-spaces/<tenant-space>/placement-groups/<placement-group>'"),
			},
		},
	})
}

func testPlacementGroupExists(t *testing.T, rName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		tfPG, ok := s.RootModule().Resources[rName]
		if !ok {
			return fmt.Errorf("resource not found: %s", rName)
		}
		if tfPG.Type != "fusion_placement_group" {
			return fmt.Errorf("expected type: fusion_placement_group. Found: %s", tfPG.Type)
		}
		tfAttrs := tfPG.Primary.Attributes

		clientPG, _, err := testAccProvider.Meta().(*hmrest.APIClient).PlacementGroupsApi.GetPlacementGroupById(context.Background(), tfAttrs["id"], nil)
		if err != nil {
			return fmt.Errorf("go client retutrned error while searching for %s by id: %s. Error: %s", tfAttrs["name"], tfAttrs["id"], err)
		}

		clientAZ, _, err := testAccProvider.Meta().(*hmrest.APIClient).AvailabilityZonesApi.GetAvailabilityZoneById(context.Background(), clientPG.AvailabilityZone.Id, nil)
		if err != nil {
			return fmt.Errorf("go client retutrned error while searching for AZ by id: %s. Error: %s", clientPG.AvailabilityZone.Id, err)
		}

		if !utilities.CheckStrAttribute(t, "name", clientPG.Name, tfAttrs["name"]) ||
			!utilities.CheckStrAttribute(t, "display_name", clientPG.DisplayName, tfAttrs["display_name"]) ||
			!utilities.CheckStrAttribute(t, "tenant", clientPG.Tenant.Name, tfAttrs["tenant"]) ||
			!utilities.CheckStrAttribute(t, "tenant_space", clientPG.TenantSpace.Name, tfAttrs["tenant_space"]) ||
			!utilities.CheckStrAttribute(t, "availability_zone", clientPG.AvailabilityZone.Name, tfAttrs["availability_zone"]) ||
			!utilities.CheckStrAttribute(t, "region", clientAZ.Region.Name, tfAttrs["region"]) ||
			!utilities.CheckStrAttribute(t, "storage_service", clientPG.StorageService.Name, tfAttrs["storage_service"]) ||
			!utilities.CheckStrAttribute(t, "array", clientPG.Array.Name, tfAttrs["array"]) {
			return fmt.Errorf("'fusion_placement_group' stored state in Terraform doesn't match reality")
		}
		return nil
	}
}

func testCheckPlacementGroupDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*hmrest.APIClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "fusion_placement_group" {
			continue
		}
		attrs := rs.Primary.Attributes

		_, resp, err := client.PlacementGroupsApi.GetPlacementGroupById(context.Background(), attrs["id"], nil)
		if err != nil && resp.StatusCode == http.StatusNotFound {
			continue // the PG was destroyed
		}

		return fmt.Errorf("placement group may still exist. Expected response code 404, got code %d", resp.StatusCode)
	}
	return nil
}

func testPlacementGroupConfigWithValues(rName, name, displayName, tenant, tenantSpace, region, availabilityZone, storageService, array string, destroySnap bool) string {
	return fmt.Sprintf(`
	resource "fusion_placement_group" "%[1]s" {
		name                        = "%[2]s"
		display_name                = "%[3]s"
		tenant                      = "%[4]s"
		tenant_space                = "%[5]s"
		region                      = "%[6]s"
		availability_zone           = "%[7]s"
		storage_service             = "%[8]s"
		array                       = "%[9]s"
		destroy_snapshots_on_delete = "%[10]t"
	}
	`, rName, name, displayName, tenant, tenantSpace, region, availabilityZone, storageService, array, destroySnap)
}

func testPlacementGroupConfigWithRefs(rName, name, displayName, tenant, tenantSpace, region, availabilityZone, storageService, array string, destroySnap bool) string {
	return fmt.Sprintf(`
	resource "fusion_placement_group" "%[1]s" {
		name                        = "%[2]s"
		display_name                = "%[3]s"
		tenant                      = fusion_tenant.%[4]s.name
		tenant_space                = fusion_tenant_space.%[5]s.name
		region                      = "%[6]s"
		availability_zone           = "%[7]s"
		storage_service             = fusion_storage_service.%[8]s.name
		array                       = "%[9]s"
		destroy_snapshots_on_delete = "%[10]t"
	}
	`, rName, name, displayName, tenant, tenantSpace, region, availabilityZone, storageService, array, destroySnap)
}

func testPlacementGroupConfigWithRefsNoArray(rName, name, displayName, tenant, tenantSpace, region, availabilityZone, storageService string, destroySnap bool) string {
	return fmt.Sprintf(`
	resource "fusion_placement_group" "%[1]s" {
		name                        = "%[2]s"
		display_name                = "%[3]s"
		tenant                      = fusion_tenant.%[4]s.name
		tenant_space                = fusion_tenant_space.%[5]s.name
		region                      = "%[6]s"
		availability_zone           = "%[7]s"
		storage_service             = fusion_storage_service.%[8]s.name
		destroy_snapshots_on_delete = "%[9]t"
	}
	`, rName, name, displayName, tenant, tenantSpace, region, availabilityZone, storageService, destroySnap)
}

func getArraysInPreexistingRegionAndAZ(t *testing.T) []hmrest.Array {
	if testURL == "" {
		ConfigureApiClientForTests(t)
	}
	ctx := setupTestCtx(t)

	hmClient, err := newTestHMClient(ctx, testURL, testIssuer, testPrivKey, testPrivKeyPassword)
	if err != nil {
		tflog.Error(ctx, "failed to create Fusion API client", "error", err)
		t.Fatalf("NewHMClient(): %v", err)
	}

	arrays, _, err := hmClient.ArraysApi.ListArrays(ctx, preexistingRegion, preexistingAvailabilityZone, nil)
	if err != nil {
		tflog.Error(ctx, "failed to list existing arrays", "region", preexistingRegion, "az", preexistingAvailabilityZone, "error", err)
		t.Fatalf("ListArrays(): %v", err)
	}

	return arrays.Items
}

func getHWTypesFromArrays(arrays []hmrest.Array) []string {
	hwSet := map[string]struct{}{}
	for _, arr := range arrays {
		if _, ok := hwSet[arr.HardwareType.Name]; !ok {
			hwSet[arr.HardwareType.Name] = struct{}{}
		}
	}

	res := make([]string, 0, len(hwSet))
	for hwType := range hwSet {
		res = append(res, hwType)
	}

	return res
}

func filterArrayNamesByHWTypes(arrays []hmrest.Array, hwTypes []string) []string {
	var res []string
	for _, arr := range arrays {
		for _, hwType := range hwTypes {
			if arr.HardwareType.Name == hwType {
				res = append(res, arr.Name)
				break
			}
		}
	}

	return res
}

func checkArrayCorrectnessInPlacementGroup(rName string, arrays []hmrest.Array, hwTypes []string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		if len(hwTypes) == 0 {
			return fmt.Errorf("hardware types can't be empty")
		}

		state, ok := s.RootModule().Resources[rName]
		if !ok {
			return fmt.Errorf("resource not found: %s", rName)
		}

		array, ok := state.Primary.Attributes["array"]
		if !ok {
			return fmt.Errorf("array attribute not found in resource: %s", rName)
		}

		filteredArrays := filterArrayNamesByHWTypes(arrays, hwTypes)
		for _, arr := range filteredArrays {
			if arr == array {
				return nil
			}
		}

		return fmt.Errorf("expected different array name for provided hw types, got: %s", array)
	}
}
