package fusion

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/utilities"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"
)

func TestAccVolume_basic(t *testing.T) {
	if os.Getenv("TF_ACC") == "" {
		t.Skip("Dont run with units tests because it will try to create the context")
	}

	ctx := setupTestCtx(t)

	// Setup resources we need
	hmClient, err := NewHMClient(ctx, testURL, testIssuer, testPrivKey)
	utilities.TraceError(ctx, err)

	ts := testFusionResource{RName: "ts", Name: acctest.RandomWithPrefix("ts-volTest")}
	pg0 := testFusionResource{RName: "pg0", Name: acctest.RandomWithPrefix("pg0-volTest")}
	pg1 := testFusionResource{RName: "pg1", Name: acctest.RandomWithPrefix("pg1-volTest")}
	host0 := testFusionResource{RName: "host0", Name: acctest.RandomWithPrefix("host0-volTest")}
	host1 := testFusionResource{RName: "host1", Name: acctest.RandomWithPrefix("host1-volTest")}
	host2 := testFusionResource{RName: "host2", Name: acctest.RandomWithPrefix("host2-volTest")}
	storageService0Name := acctest.RandomWithPrefix("ss0-volTest")
	storageService1Name := acctest.RandomWithPrefix("ss1-volTest")
	protectionPolicy0Name := acctest.RandomWithPrefix("pp0-volTest")
	protectionPolicy1Name := acctest.RandomWithPrefix("pp1-volTest")
	storageClass0Name := acctest.RandomWithPrefix("sc0-volTest")
	storageClass1Name := acctest.RandomWithPrefix("sc1-volTest")

	eradicate := true

	// Initial state
	volState0 := testVolume{
		RName:                "test_volume",
		Name:                 acctest.RandomWithPrefix("test_vol"),
		DisplayName:          "initial display name",
		TenantSpace:          ts,
		ProtectionPolicyName: protectionPolicy0Name,
		StorageClassName:     storageClass0Name,
		PlacementGroup:       pg0,
		Size:                 1 << 20,
		Eradicate:            &eradicate,
	}

	// Change everything
	volState1 := volState0
	volState1.DisplayName = "changed display name"
	volState1.PlacementGroup = pg1
	volState1.StorageClassName = storageClass1Name
	volState1.Size += 1 << 20
	volState1.Hosts = []testFusionResource{host0}

	// Remove and add hosts at the same time, also change protection policy
	volState2 := volState1
	volState2.Hosts = []testFusionResource{host1, host2}
	volState2.ProtectionPolicyName = protectionPolicy1Name

	// Remove a host, and change some other things back
	volState3 := volState2
	volState3.DisplayName = "changed display name again"
	volState3.Hosts = []testFusionResource{host1}
	volState3.PlacementGroup = pg0
	volState3.StorageClassName = storageClass0Name

	commonConfig := "" +
		testTenantSpaceConfigWithNames(ts.RName, "ts display name", ts.Name, testAccTenant) +
		testPGConfig("", pg0.RName, pg0.Name, "pg display name", region_name, availability_zone_name, storageService0Name, true) +
		testPGConfig("", pg1.RName, pg1.Name, "pg display name", region_name, availability_zone_name, storageService1Name, true) +
		testHostAccessPolicyConfig(host0.RName, host0.Name, "host display name", randIQN(), "linux") +
		testHostAccessPolicyConfig(host1.RName, host1.Name, "host display name", randIQN(), "linux") +
		testHostAccessPolicyConfig(host2.RName, host2.Name, "host display name", randIQN(), "linux") +
		""

	testVolumeStep := func(vol testVolume) resource.TestStep {
		step := resource.TestStep{}

		step.Config = commonConfig + testVolumeConfig(vol)

		r := "fusion_volume." + vol.RName

		step.Check = testCheckVolumeAttributes(r, vol)

		for _, host := range vol.Hosts {
			step.Check = resource.ComposeTestCheckFunc(step.Check,
				resource.TestCheckTypeSetElemAttr(r, "host_names.*", host.Name),
			)
		}

		step.Check = resource.ComposeTestCheckFunc(step.Check, testVolumeExists(r, t))

		return step
	}

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			// Create extra resources
			testSetupVolumeResources(t, ctx, hmClient, protectionPolicy0Name, storageService0Name, storageClass0Name, "flash-array-x")
			testSetupVolumeResources(t, ctx, hmClient, protectionPolicy1Name, storageService1Name, storageClass1Name, "flash-array-c")
		},
		CheckDestroy: func(s *terraform.State) error {
			// Clean up created resources
			testDeleteVolumeResources(t, ctx, hmClient, protectionPolicy0Name, storageService0Name, storageClass0Name)
			testDeleteVolumeResources(t, ctx, hmClient, protectionPolicy1Name, storageService1Name, storageClass1Name)
			return testCheckVolumeDelete(s)
		},
		ProviderFactories: testAccProvidersFactory,
		Steps: []resource.TestStep{
			testVolumeStep(volState0),
			testVolumeStep(volState1),
			testVolumeStep(volState2),
			testVolumeStep(volState3),
		},
	})
}

func TestAccVolume_eradicate(t *testing.T) {
	ctx := setupTestCtx(t)

	// Setup resources we need
	hmClient, err := NewHMClient(ctx, testURL, testIssuer, testPrivKey)
	utilities.TraceError(ctx, err)

	ts := testFusionResource{RName: "ts", Name: acctest.RandomWithPrefix("ts-volTest")}
	pg := testFusionResource{RName: "pg", Name: acctest.RandomWithPrefix("pg-volTest")}

	storageServiceName := acctest.RandomWithPrefix("ss-volTest")
	protectionPolicyName := acctest.RandomWithPrefix("pp-volTest")
	storageClassName := acctest.RandomWithPrefix("sc-volTest")

	commonConfig := "" +
		testTenantSpaceConfigWithNames(ts.RName, "ts display name", ts.Name, testAccTenant) +
		testPGConfig("", pg.RName, pg.Name, "pg display name", region_name, availability_zone_name, storageServiceName, true) +
		""

	eradicate := true

	volState := testVolume{
		RName:                acctest.RandomWithPrefix("test_volume"),
		Name:                 acctest.RandomWithPrefix("test_vol"),
		DisplayName:          "initial display name",
		TenantSpace:          ts,
		ProtectionPolicyName: protectionPolicyName,
		StorageClassName:     storageClassName,
		PlacementGroup:       pg,
		Size:                 1 << 20,
		Eradicate:            &eradicate,
	}

	volState1 := volState
	volState1.Eradicate = nil

	hwType := "flash-array-x"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			// Create extra resources required by volume
			testSetupVolumeResources(t, ctx, hmClient, protectionPolicyName, storageServiceName, storageClassName, hwType)
		},
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy: func(s *terraform.State) error {
			// Clean up created resources
			testDeleteVolumeResources(t, ctx, hmClient, protectionPolicyName, storageServiceName, storageClassName)
			return testCheckVolumeDelete(s)
		},
		Steps: []resource.TestStep{
			{
				// Explicitly set `eradicate_on_delete=true` during volume creation
				Config: commonConfig + testVolumeConfig(volState),
				Check:  testVolumeExists("fusion_volume."+volState.RName, t),
			},
			{
				// Eradicate the volume
				Config: commonConfig,
				Check:  testCheckVolumeDelete,
			},
			{
				// Create volume with `eradicate_on_delete=false`
				Config: commonConfig + testVolumeConfig(volState1),
				Check:  testVolumeExists("fusion_volume."+volState.RName, t),
			},
			{
				// Update volume with `eradicate_on_delete=true` - it will be eradicated after the test finishes
				Config: commonConfig + testVolumeConfig(volState),
				Check:  testVolumeExists("fusion_volume."+volState.RName, t),
			},
		},
	})
}

func TestAccVolume_destroyOnly(t *testing.T) {
	ctx := setupTestCtx(t)

	// Setup resources we need
	hmClient, err := NewHMClient(ctx, testURL, testIssuer, testPrivKey)
	utilities.TraceError(ctx, err)

	ts := testFusionResource{RName: "ts", Name: acctest.RandomWithPrefix("ts-volTest")}
	pg := testFusionResource{RName: "pg", Name: acctest.RandomWithPrefix("pg-volTest")}

	storageServiceName := acctest.RandomWithPrefix("ss-volTest")
	protectionPolicyName := acctest.RandomWithPrefix("pp-volTest")
	storageClassName := acctest.RandomWithPrefix("sc-volTest")

	commonConfig := "" +
		testTenantSpaceConfigWithNames(ts.RName, "ts display name", ts.Name, testAccTenant) +
		testPGConfig("", pg.RName, pg.Name, "pg display name", region_name, availability_zone_name, storageServiceName, true) +
		""

	volState := testVolume{
		RName:                acctest.RandomWithPrefix("test_volume"),
		Name:                 acctest.RandomWithPrefix("test_vol"),
		DisplayName:          "initial display name",
		TenantSpace:          ts,
		ProtectionPolicyName: protectionPolicyName,
		StorageClassName:     storageClassName,
		PlacementGroup:       pg,
		Size:                 1 << 20,
		Eradicate:            nil, // do not eradicate (can be also pointer to variable set to false)
	}

	hwType := "flash-array-x"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			// Create extra resources required by volume
			testSetupVolumeResources(t, ctx, hmClient, protectionPolicyName, storageServiceName, storageClassName, hwType)
		},
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy: func(s *terraform.State) error {
			// Clean up created resources
			testDeleteVolumeResources(t, ctx, hmClient, protectionPolicyName, storageServiceName, storageClassName)

			// Now the volume should be eradicated
			return testCheckVolumeDelete(s)
		},
		Steps: []resource.TestStep{
			{
				// Create volume with set `eradicate_on_delete=false` (implicit)
				Config: commonConfig + testVolumeConfig(volState),
				Check:  testVolumeExists("fusion_volume."+volState.RName, t),
			},
			{
				Config: commonConfig,
				Check: func(s *terraform.State) error {
					// Volume should exist and be destroyed. Not eradicated
					if err := testCheckVolumeDestroy(volState.Name, testAccTenant, ts.Name)(s); err != nil {
						return err
					}

					// Eradicate the volume (test cleanup). This is workaround, because we have to eradicate the volume
					// before we try to delete resources defined in `commonConfig`
					testVolumeDoOperation(t, ctx, hmClient, "volume eradicate")(
						hmClient.VolumesApi.DeleteVolume(ctx, testAccTenant, ts.Name, volState.Name, nil),
					)

					return nil
				},
			},
		},
	})
}

func TestAccVolume_copyFromVolume(t *testing.T) {
	ctx := setupTestCtx(t)

	// Setup resources we need
	hmClient, err := NewHMClient(ctx, testURL, testIssuer, testPrivKey)
	utilities.TraceError(ctx, err)

	ts := testFusionResource{RName: "ts", Name: acctest.RandomWithPrefix("ts-volTest")}
	pg := testFusionResource{RName: "pg", Name: acctest.RandomWithPrefix("pg-volTest")}

	storageServiceName := acctest.RandomWithPrefix("ss-volTest")
	protectionPolicyName := acctest.RandomWithPrefix("pp-volTest")
	storageClassName := acctest.RandomWithPrefix("sc-volTest")

	commonConfig := "" +
		testTenantSpaceConfigWithNames(ts.RName, "ts display name", ts.Name, testAccTenant) +
		testPGConfig("", pg.RName, pg.Name, "pg display name", region_name, availability_zone_name, storageServiceName, true) +
		""

	eradicate := true

	volState := testVolume{
		RName:                acctest.RandomWithPrefix("test_volume"),
		Name:                 acctest.RandomWithPrefix("test_vol"),
		DisplayName:          "initial display name",
		TenantSpace:          ts,
		ProtectionPolicyName: protectionPolicyName,
		StorageClassName:     storageClassName,
		PlacementGroup:       pg,
		Size:                 1 << 20,
		Eradicate:            &eradicate,
	}

	volState1 := volState
	volState1.RName = acctest.RandomWithPrefix("test_volume")
	volState1.Name = acctest.RandomWithPrefix("test_vol")
	volState1.SourceLink = map[string]string{
		"tenant":       testAccTenant,
		"tenant_space": ts.Name,
		"volume":       volState.Name,
	}

	volState2 := volState1
	volState2.SourceLink = nil

	volStateResource := "fusion_volume." + volState1.RName
	hwType := "flash-array-x"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			// Create extra resources required by volume
			testSetupVolumeResources(t, ctx, hmClient, protectionPolicyName, storageServiceName, storageClassName, hwType)
		},
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy: func(s *terraform.State) error {
			// Clean up created resources
			testDeleteVolumeResources(t, ctx, hmClient, protectionPolicyName, storageServiceName, storageClassName)
			return testCheckVolumeDelete(s)
		},
		Steps: []resource.TestStep{
			{
				// Create a volume
				Config: commonConfig + testVolumeConfig(volState),
				Check:  testVolumeExists("fusion_volume."+volState.RName, t),
			},
			{
				// Create another volume that is a copy of the first volume
				Config: commonConfig + testVolumeConfig(volState) + testVolumeConfigCopy(volState1),
				Check: resource.ComposeTestCheckFunc(
					testVolumeExists("fusion_volume."+volState.RName, t),
					testVolumeExists("fusion_volume."+volState1.RName, t),
					testCheckVolumeAttributes(volStateResource, volState1)),
			},
			{
				// Destroy the copied volume
				Config: commonConfig + testVolumeConfig(volState),
			},
			{
				// Create another volume (not a copy)
				Config: commonConfig + testVolumeConfig(volState) + testVolumeConfig(volState2),
				Check: resource.ComposeTestCheckFunc(
					testVolumeExists("fusion_volume."+volState.RName, t),
					testVolumeExists("fusion_volume."+volState2.RName, t),
				),
			},
			{
				// Copy the first volume to the second one (update)
				Config: commonConfig + testVolumeConfig(volState) + testVolumeConfigCopy(volState1),
				Check: resource.ComposeTestCheckFunc(
					testVolumeExists("fusion_volume."+volState.RName, t),
					testVolumeExists("fusion_volume."+volState1.RName, t),
					testCheckVolumeAttributes(volStateResource, volState1),
				),
			},
		},
	})
}

func TestAccVolume_copyFromSnapshot(t *testing.T) {
	ctx := setupTestCtx(t)

	// Setup resources we need
	hmClient, err := NewHMClient(ctx, testURL, testIssuer, testPrivKey)
	utilities.TraceError(ctx, err)

	ts := testFusionResource{RName: "ts", Name: acctest.RandomWithPrefix("ts-volTest")}
	pg := testFusionResource{RName: "pg", Name: acctest.RandomWithPrefix("pg-volTest")}

	storageServiceName := acctest.RandomWithPrefix("ss-volTest")
	protectionPolicyName := acctest.RandomWithPrefix("pp-volTest")
	storageClassName := acctest.RandomWithPrefix("sc-volTest")

	commonConfig := "" +
		testTenantSpaceConfigWithNames(ts.RName, "ts display name", ts.Name, testAccTenant) +
		testPGConfig("", pg.RName, pg.Name, "pg display name", region_name, availability_zone_name, storageServiceName, true) +
		""

	eradicate := true

	volState := testVolume{
		RName:                acctest.RandomWithPrefix("test_volume"),
		Name:                 acctest.RandomWithPrefix("test_vol"),
		DisplayName:          "initial display name",
		TenantSpace:          ts,
		ProtectionPolicyName: protectionPolicyName,
		StorageClassName:     storageClassName,
		PlacementGroup:       pg,
		Size:                 1 << 20,
		Eradicate:            &eradicate,
	}

	snapshotName := acctest.RandomWithPrefix("snapshot")

	volState1 := volState
	volState1.RName = acctest.RandomWithPrefix("test_volume")
	volState1.Name = acctest.RandomWithPrefix("test_vol")
	volState1.SourceLink = map[string]string{
		"tenant":          testAccTenant,
		"tenant_space":    ts.Name,
		"snapshot":        snapshotName,
		"volume_snapshot": volState.Name,
	}

	hwType := "flash-array-x"

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
			// Create extra resources required by volume
			testSetupVolumeResources(t, ctx, hmClient, protectionPolicyName, storageServiceName, storageClassName, hwType)
		},
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy: func(s *terraform.State) error {
			// Clean up created resources
			testDeleteVolumeResources(t, ctx, hmClient, protectionPolicyName, storageServiceName, storageClassName)
			return testCheckVolumeDelete(s)
		},
		Steps: []resource.TestStep{
			{
				// Create a volume
				Config: commonConfig + testVolumeConfig(volState),
				Check: func(s *terraform.State) error {
					if err := testVolumeExists("fusion_volume."+volState.RName, t)(s); err != nil {
						return err
					}

					// TODO: Remove once/if we have a snapshot resource
					snapPost := hmrest.SnapshotPost{
						Name:    snapshotName,
						Volumes: []string{volState.Name},
					}

					testVolumeDoOperation(t, ctx, hmClient, "snapshot-create")(
						hmClient.SnapshotsApi.CreateSnapshot(ctx, snapPost, testAccTenant, ts.Name, nil),
					)

					return nil
				},
			},
			{
				// Create another volume that is a copy of the first volume
				Config: commonConfig + testVolumeConfig(volState) + testVolumeConfigCopy(volState1),
				Check: resource.ComposeTestCheckFunc(
					testVolumeExists("fusion_volume."+volState.RName, t),
					testVolumeExists("fusion_volume."+volState1.RName, t),
					testCheckVolumeAttributes("fusion_volume."+volState1.RName, volState1),
				),
			},
			{
				// Reset the source_link field to force update in the next step
				Config: commonConfig + testVolumeConfig(volState) + testVolumeConfig(volState1),
			},
			{
				// Try to update from snapshot
				Config:      commonConfig + testVolumeConfig(volState) + testVolumeConfigCopy(volState1),
				ExpectError: regexp.MustCompile("cannot copy snapshot to existing volume"),
			},
		},
	})
}

// Verify resource with a direct hmrest call
func testVolumeExists(rName string, t *testing.T) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		volume, ok := s.RootModule().Resources[rName]
		if !ok {
			return fmt.Errorf("resource not found: %s", rName)
		}
		if volume.Type != "fusion_volume" {
			return fmt.Errorf("expected type: fusion_volume. Found: %s", volume.Type)
		}
		attrs := volume.Primary.Attributes

		directVolume, _, err := testAccProvider.Meta().(*hmrest.APIClient).VolumesApi.GetVolume(context.Background(), testAccTenant, attrs["tenant_space_name"], attrs["name"], nil)
		if err != nil {
			return fmt.Errorf("go client retutrned error while searching for %s. Error: %s", attrs["name"], err)
		}
		tfHostNameCount, _ := strconv.Atoi(attrs["host_names.#"])

		directHosts := []string{}
		for _, directHost := range directVolume.HostAccessPolicies {
			directHosts = append(directHosts, directHost.Name)
		}
		sort.Slice(directHosts, func(i, j int) bool { return strings.Compare(directHosts[i], directHosts[j]) < 0 })
		tfHosts := []string{}
		for i := 0; i < tfHostNameCount; i++ {
			tfHosts = append(tfHosts, attrs[fmt.Sprintf("host_names.%d", i)])
		}
		sort.Slice(tfHosts, func(i, j int) bool { return strings.Compare(tfHosts[i], tfHosts[j]) < 0 })

		failed := false

		checkAttr := func(direct, attrName string) {
			if direct != attrs[attrName] {
				t.Errorf("mismatch attr:%s direct:%s tf:%s", attrName, direct, attrs[attrName])
				failed = true
			}
		}

		checkAttr(directVolume.Name, "name")
		checkAttr(directVolume.DisplayName, "display_name")
		checkAttr(directVolume.Tenant.Name, "tenant_name")
		checkAttr(directVolume.TenantSpace.Name, "tenant_space_name")
		checkAttr(directVolume.ProtectionPolicy.Name, "protection_policy_name")
		checkAttr(directVolume.StorageClass.Name, "storage_class_name")
		checkAttr(directVolume.PlacementGroup.Name, "placement_group_name")

		if !reflect.DeepEqual(directHosts, tfHosts) {
			t.Errorf("hosts mismatch")
			for _, h := range directHosts {
				t.Logf("direct host: %s", h)
			}
			for _, h := range tfHosts {
				t.Logf("tf host: %s", h)
			}
			failed = true
		}

		if failed {
			return fmt.Errorf("direct tf mismatch")
		}

		return nil
	}
}

type testVolume struct {
	RName                string
	Name                 string
	DisplayName          string
	ProtectionPolicyName string
	TenantSpace          testFusionResource
	StorageClassName     string
	PlacementGroup       testFusionResource
	Size                 int
	Hosts                []testFusionResource
	Eradicate            *bool
	SourceLink           map[string]string
}

type testFusionResource struct {
	RName string
	Name  string
}

func testVolumeConfigCopy(vol testVolume) string {
	hapList := make([]string, 0)
	for _, host := range vol.Hosts {
		hapList = append(hapList, fmt.Sprintf(`fusion_host_access_policy.%s.name`, host.RName))
	}

	eradicate := ""
	if vol.Eradicate != nil {
		eradicate = fmt.Sprintf("eradicate_on_delete = %t", *vol.Eradicate)
	}

	sourceLink := ""
	if vol.SourceLink != nil {
		for key, value := range vol.SourceLink {
			sourceLink += fmt.Sprintf("%s = \"%s\"\n", key, value)
		}

		sourceLink = fmt.Sprintf(`
		source_link {
			%s
		}
		`, sourceLink)
	}

	return fmt.Sprintf(`
resource "fusion_volume" "%[1]s" {
		name          = "%[2]s"
		display_name  = "%[3]s"
		protection_policy_name = "%[4]s"
		tenant_name        = "%[5]s"
		tenant_space_name  = fusion_tenant_space.%[6]s.name
		storage_class_name = "%[7]s"
		host_names = [%[8]s]
		placement_group_name = fusion_placement_group.%[9]s.name
		%[10]s
		%[11]s
}`, vol.RName, vol.Name, vol.DisplayName, vol.ProtectionPolicyName,
		testAccTenant, vol.TenantSpace.RName, vol.StorageClassName,
		strings.Join(hapList, ","), vol.PlacementGroup.RName, eradicate, sourceLink)
}

func testVolumeConfig(vol testVolume) string {
	hapList := make([]string, 0)
	for _, host := range vol.Hosts {
		hapList = append(hapList, fmt.Sprintf(`fusion_host_access_policy.%s.name`, host.RName))
	}

	eradicate := ""
	if vol.Eradicate != nil {
		eradicate = fmt.Sprintf("eradicate_on_delete = %t", *vol.Eradicate)
	}

	return fmt.Sprintf(`
resource "fusion_volume" "%[1]s" {
		name          = "%[2]s"
		display_name  = "%[3]s"
		protection_policy_name = "%[4]s"
		tenant_name        = "%[5]s"
		tenant_space_name  = fusion_tenant_space.%[6]s.name
		storage_class_name = "%[7]s"
		size          = %[8]d
		host_names = [%[9]s]
		placement_group_name = fusion_placement_group.%[10]s.name
		%[11]s
}`, vol.RName, vol.Name, vol.DisplayName, vol.ProtectionPolicyName,
		testAccTenant, vol.TenantSpace.RName, vol.StorageClassName,
		vol.Size, strings.Join(hapList, ","), vol.PlacementGroup.RName, eradicate)
}

func testCheckVolumeDelete(s *terraform.State) error {
	client := testAccProvider.Meta().(*hmrest.APIClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "fusion_volume" {
			continue
		}
		attrs := rs.Primary.Attributes
		volumeName := attrs["name"]
		tenantName := attrs["tenant_name"]
		tenantSpaceName := attrs["tenant_space_name"]

		_, resp, err := client.VolumesApi.GetVolume(context.Background(), tenantName, tenantSpaceName, volumeName, nil)
		if err != nil && resp.StatusCode == http.StatusNotFound {
			continue
		}

		return fmt.Errorf("volume may still exist. Expected response code 404, got code %d", resp.StatusCode)
	}
	return nil
}

func testCheckVolumeDestroy(name, tenantName, tenantSpaceName string) func(s *terraform.State) error {
	client := testAccProvider.Meta().(*hmrest.APIClient)

	return func(s *terraform.State) error {
		volume, resp, err := client.VolumesApi.GetVolume(context.Background(), tenantName, tenantSpaceName, name, nil)
		if err != nil {
			return err
		}

		if resp.StatusCode == http.StatusNotFound {
			return fmt.Errorf("volume that should be destroyed and exist does not exist")
		}

		if !volume.Destroyed {
			return fmt.Errorf("volume that should be destroyed is not destroyed")
		}

		return nil
	}
}

func testCheckVolumeAttributes(resourceName string, volState testVolume) resource.TestCheckFunc {
	return resource.ComposeTestCheckFunc(
		resource.TestCheckResourceAttr(resourceName, "name", volState.Name),
		resource.TestCheckResourceAttr(resourceName, "display_name", volState.DisplayName),
		resource.TestCheckResourceAttr(resourceName, "tenant_name", testAccTenant),
		resource.TestCheckResourceAttr(resourceName, "tenant_space_name", volState.TenantSpace.Name),
		resource.TestCheckResourceAttr(resourceName, "protection_policy_name", volState.ProtectionPolicyName),
		resource.TestCheckResourceAttr(resourceName, "storage_class_name", volState.StorageClassName),
		resource.TestCheckResourceAttr(resourceName, "placement_group_name", volState.PlacementGroup.Name),
		resource.TestCheckResourceAttr(resourceName, "host_names.#", fmt.Sprintf("%d", len(volState.Hosts))),
	)
}

func testVolumeDoOperation(
	t *testing.T, ctx context.Context, hmClient *hmrest.APIClient, userMessage string,
) func(hmrest.Operation, *http.Response, error) {
	return func(op hmrest.Operation, _ *http.Response, err error) {
		utilities.TraceError(ctx, err)
		if err != nil {
			t.Errorf("%s: %s", userMessage, err)
		}
		succeeded, err := utilities.WaitOnOperation(ctx, &op, hmClient)
		if !succeeded || err != nil {
			t.Errorf("operation failure %s succeeded:%v error:%v", userMessage, succeeded, err)
		}
	}
}

// Will be redundant once we implement more resources
func testSetupVolumeResources(
	t *testing.T,
	ctx context.Context,
	hmClient *hmrest.APIClient,
	protectionPolicyName,
	storageServiceName string,
	storageClassName string,
	hwType string,
) {
	testVolumeDoOperation(t, ctx, hmClient, protectionPolicyName)(
		hmClient.ProtectionPoliciesApi.CreateProtectionPolicy(ctx, hmrest.ProtectionPolicyPost{
			Name: protectionPolicyName,
			Objectives: []hmrest.OneOfProtectionPolicyPostObjectivesItems{
				hmrest.Rpo{Type_: "RPO", Rpo: "PT6H"},
				hmrest.Retention{Type_: "Retention", After: "PT24H"},
			},
		}, nil),
	)

	testVolumeDoOperation(t, ctx, hmClient, storageServiceName)(
		hmClient.StorageServicesApi.CreateStorageService(ctx, hmrest.StorageServicePost{
			Name:          storageServiceName,
			HardwareTypes: []string{hwType},
		}, nil),
	)

	testVolumeDoOperation(t, ctx, hmClient, storageClassName)(
		hmClient.StorageClassesApi.CreateStorageClass(ctx, hmrest.StorageClassPost{
			Name:           storageClassName,
			SizeLimit:      1 << 22,
			BandwidthLimit: 1e9,
			IopsLimit:      100,
		}, storageServiceName, nil),
	)
}

func testDeleteVolumeResources(
	t *testing.T, ctx context.Context, hmClient *hmrest.APIClient, protectionPolicyName, storageServiceName, storageClassName string,
) {
	testVolumeDoOperation(t, ctx, hmClient, protectionPolicyName)(
		hmClient.ProtectionPoliciesApi.DeleteProtectionPolicy(ctx, protectionPolicyName, nil),
	)

	testVolumeDoOperation(t, ctx, hmClient, storageClassName)(
		hmClient.StorageClassesApi.DeleteStorageClass(ctx, storageServiceName, storageClassName, nil),
	)

	testVolumeDoOperation(t, ctx, hmClient, storageServiceName)(
		hmClient.StorageServicesApi.DeleteStorageService(ctx, storageServiceName, nil),
	)
}
