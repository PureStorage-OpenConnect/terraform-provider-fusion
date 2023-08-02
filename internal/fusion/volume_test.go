package fusion

import (
	"context"
	"fmt"
	"net/http"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	"github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/utilities"
	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"
)

func TestAccVolume_basic(t *testing.T) {
	utilities.CheckTestSkip(t)

	tenant := acctest.RandomWithPrefix("tenant-volTest")
	ts := acctest.RandomWithPrefix("ts-volTest")
	pg0 := acctest.RandomWithPrefix("pg0-volTest")
	pg1 := acctest.RandomWithPrefix("pg1-volTest")
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
		Tenant:               tenant,
		TenantSpace:          ts,
		ProtectionPolicyName: protectionPolicy0Name,
		StorageClassName:     storageClass0Name,
		PlacementGroup:       pg0,
		Size:                 1 << 20,
		Hosts:                []testFusionResource{host0},
		Eradicate:            &eradicate,
	}

	// Change everything
	volState1 := volState0
	volState1.DisplayName = "changed display name"
	volState1.PlacementGroup = pg1
	volState1.StorageClassName = storageClass1Name
	volState1.Size += 1 << 20
	volState1.Hosts = []testFusionResource{}

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
		testTenantConfig(tenant, tenant, "tenant display name") +
		testTenantSpaceConfigWithRefs(ts, "ts display name", ts, tenant) +
		testPlacementGroupConfigWithRefsNoArray(pg0, pg0, "pg display name", tenant, ts, preexistingRegion, preexistingAvailabilityZone, storageService0Name, true) +
		testPlacementGroupConfigWithRefsNoArray(pg1, pg1, "pg display name", tenant, ts, preexistingRegion, preexistingAvailabilityZone, storageService1Name, true) +
		testHostAccessPolicyConfig(host0.RName, host0.Name, "host display name", randIQN(), "linux") +
		testHostAccessPolicyConfig(host1.RName, host1.Name, "host display name", randIQN(), "linux") +
		testHostAccessPolicyConfig(host2.RName, host2.Name, "host display name", randIQN(), "linux") +
		testStorageServiceConfigNoDisplayName(storageService0Name, storageService0Name, []string{"flash-array-x"}) +
		testStorageClassConfigNoDisplayName(storageClass0Name, storageClass0Name, storageService0Name, 2*testSizeLimit, testIopsLimit, testBandwidthLimit) +
		testAccProtectionPolicyConfig(protectionPolicy0Name, protectionPolicy0Name, protectionPolicy0Name, localRPO, localRetention, true) +
		testStorageServiceConfigNoDisplayName(storageService1Name, storageService1Name, []string{"flash-array-c"}) +
		testStorageClassConfigNoDisplayName(storageClass1Name, storageClass1Name, storageService1Name, 2*testSizeLimit, testIopsLimit, testBandwidthLimit) +
		testAccProtectionPolicyConfig(protectionPolicy1Name, protectionPolicy1Name, protectionPolicy1Name, localRPO, localRetention, true)

	testVolumeStep := func(vol testVolume) resource.TestStep {
		step := resource.TestStep{}

		step.Config = commonConfig + testVolumeConfig(vol)

		r := "fusion_volume." + vol.RName

		step.Check = testCheckVolumeAttributes(r, vol)

		for _, host := range vol.Hosts {
			step.Check = resource.ComposeTestCheckFunc(step.Check,
				resource.TestCheckTypeSetElemAttr(r, "host_access_policies.*", host.Name),
			)
		}

		step.Check = resource.ComposeTestCheckFunc(step.Check, testVolumeExists(r, t))

		return step
	}

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		CheckDestroy:      testCheckVolumeDelete,
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
	utilities.CheckTestSkip(t)

	eradicate := true
	volState, commonConfig := generateVolumeTestConfigAndCommonTFConfig(&eradicate, []string{"flash-array-x"}, nil)

	volState1 := volState
	volState1.Eradicate = nil

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckVolumeDelete,
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
	utilities.CheckTestSkip(t)

	volState, commonConfig := generateVolumeTestConfigAndCommonTFConfig(nil, []string{"flash-array-x"}, nil)
	ctx := setupTestCtx(t)
	hmClient := testAccPreCheckWithReturningClient(ctx, t)

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckVolumeDelete,
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
					if err := testCheckVolumeDestroy(volState.Name, volState.Tenant, volState.TenantSpace)(s); err != nil {
						return err
					}
					// Eradicate the volume (test cleanup). This is workaround, because we have to eradicate the volume
					// before we try to delete resources defined in `commonConfig`
					testVolumeDoOperation(t, ctx, hmClient, "volume eradicate")(
						hmClient.VolumesApi.DeleteVolume(ctx, volState.Tenant, volState.TenantSpace, volState.Name, nil),
					)

					return nil
				},
			},
		},
	})
}

func TestAccVolume_copyFromVolume(t *testing.T) {
	utilities.CheckTestSkip(t)

	eradicate := true
	volState, commonConfig := generateVolumeTestConfigAndCommonTFConfig(&eradicate, []string{"flash-array-x"}, nil)

	volState1 := volState
	volState1.RName = acctest.RandomWithPrefix("test_volume")
	volState1.Name = acctest.RandomWithPrefix("test_vol")
	volState1.SourceLink = map[string]string{
		"tenant":       volState.Tenant,
		"tenant_space": volState.TenantSpace,
		"volume":       volState.Name,
	}

	volState2 := volState1
	volState2.SourceLink = nil

	volStateResource := "fusion_volume." + volState1.RName

	resource.ParallelTest(t, resource.TestCase{
		PreCheck: func() {
			testAccPreCheck(t)
		},
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckVolumeDelete,
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
	utilities.CheckTestSkip(t)

	eradicate := true
	volState, commonConfig := generateVolumeTestConfigAndCommonTFConfig(&eradicate, []string{"flash-array-x"}, nil)

	snapshotName := acctest.RandomWithPrefix("snapshot")
	ctx := setupTestCtx(t)
	hmClient := testAccPreCheckWithReturningClient(ctx, t)

	volState1 := volState
	volState1.RName = acctest.RandomWithPrefix("test_volume")
	volState1.Name = acctest.RandomWithPrefix("test_vol")
	volState1.SourceLink = map[string]string{
		"tenant":          volState.Tenant,
		"tenant_space":    volState.TenantSpace,
		"snapshot":        snapshotName,
		"volume_snapshot": volState.Name,
	}

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckVolumeDelete,
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
						hmClient.SnapshotsApi.CreateSnapshot(ctx, snapPost, volState.Tenant, volState.TenantSpace, nil),
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

func TestAccVolume_recovery(t *testing.T) {
	utilities.CheckTestSkip(t)

	eradicate := true
	volState, commonConfig := generateVolumeTestConfigAndCommonTFConfig(&eradicate, []string{"flash-array-x"}, nil)
	ctx := setupTestCtx(t)
	hmClient := testAccPreCheckWithReturningClient(ctx, t)

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckVolumeDelete,
		Steps: []resource.TestStep{
			{
				// Create a volume
				Config: commonConfig + testVolumeConfig(volState),
				Check:  testVolumeExists("fusion_volume."+volState.RName, t),
			},
			{
				// Run `terraform import "fusion_volume.<rname>" /tenants/<tenant>/tenant-spaces/<ts>/volumes/<volume>`
				// Manually destroy the volume, so that it can be recovered
				// The imported state is compared the the state in the previous step - we cannot delete the volume by
				// removing it from the config
				PreConfig: func() {
					// Destroy the volume
					body := hmrest.VolumePatch{Destroyed: &hmrest.NullableBoolean{Value: true}}
					testVolumeDoOperation(t, ctx, hmClient, "volume delete")(
						hmClient.VolumesApi.UpdateVolume(ctx, body, volState.Tenant, volState.TenantSpace, volState.Name, nil),
					)
				},
				ImportState:             true,
				ImportStateId:           fmt.Sprintf("/tenants/%s/tenant-spaces/%s/volumes/%s", volState.Tenant, volState.TenantSpace, volState.Name),
				ResourceName:            "fusion_volume." + volState.RName,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"eradicate_on_delete"},
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

		directVolume, _, err := testAccProvider.Meta().(*hmrest.APIClient).VolumesApi.GetVolumeById(context.Background(), attrs["id"], nil)
		if err != nil {
			return fmt.Errorf("go client returned error while searching for %s by id: %s. Error: %s", attrs["name"], attrs["id"], err)
		}
		tfHostNameCount, _ := strconv.Atoi(attrs["host_access_policies.#"])

		directHosts := []string{}
		for _, directHost := range directVolume.HostAccessPolicies {
			directHosts = append(directHosts, directHost.Name)
		}
		sort.Slice(directHosts, func(i, j int) bool { return strings.Compare(directHosts[i], directHosts[j]) < 0 })
		tfHosts := []string{}
		for i := 0; i < tfHostNameCount; i++ {
			tfHosts = append(tfHosts, attrs[fmt.Sprintf("host_access_policies.%d", i)])
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
		checkAttr(directVolume.Tenant.Name, "tenant")
		checkAttr(directVolume.TenantSpace.Name, "tenant_space")
		checkAttr(directVolume.ProtectionPolicy.Name, "protection_policy")
		checkAttr(directVolume.StorageClass.Name, "storage_class")
		checkAttr(directVolume.PlacementGroup.Name, "placement_group")

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
	Tenant               string
	TenantSpace          string
	StorageClassName     string
	PlacementGroup       string
	Size                 int
	Hosts                []testFusionResource
	Eradicate            *bool
	SourceLink           map[string]string
	StorageService       string
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

	hapField := ""
	if len(hapList) != 0 {
		hapField = fmt.Sprintf("host_access_policies = [%s]", strings.Join(hapList, ","))
	}

	return fmt.Sprintf(`
resource "fusion_volume" "%[1]s" {
		name          = "%[2]s"
		display_name  = "%[3]s"
		protection_policy = fusion_protection_policy.%[4]s.name
		tenant        = fusion_tenant.%[5]s.name
		tenant_space  = fusion_tenant_space.%[6]s.name
		storage_class = fusion_storage_class.%[7]s.name
		%[8]s
		placement_group = fusion_placement_group.%[9]s.name
		%[10]s
		%[11]s
}`, vol.RName, vol.Name, vol.DisplayName, vol.ProtectionPolicyName,
		vol.Tenant, vol.TenantSpace, vol.StorageClassName,
		hapField, vol.PlacementGroup, eradicate, sourceLink)
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

	protectionPolicy := ""
	if vol.ProtectionPolicyName != "" {
		protectionPolicy = fmt.Sprintf("protection_policy = fusion_protection_policy.%[1]s.name", vol.ProtectionPolicyName)
	}

	hapField := ""
	if len(hapList) != 0 {
		hapField = fmt.Sprintf("host_access_policies = [%s]", strings.Join(hapList, ","))
	}

	return fmt.Sprintf(`
resource "fusion_volume" "%[1]s" {
		name          = "%[2]s"
		display_name  = "%[3]s"
		%[4]s
		tenant        = fusion_tenant.%[5]s.name
		tenant_space  = fusion_tenant_space.%[6]s.name
		storage_class = fusion_storage_class.%[7]s.name
		size          = %[8]d
		%[9]s
		placement_group = fusion_placement_group.%[10]s.name
		%[11]s
}`, vol.RName, vol.Name, vol.DisplayName, protectionPolicy,
		vol.Tenant, vol.TenantSpace, vol.StorageClassName,
		vol.Size, hapField, vol.PlacementGroup, eradicate)
}

func testCheckVolumeDelete(s *terraform.State) error {
	for _, rs := range s.RootModule().Resources {
		client := testAccProvider.Meta().(*hmrest.APIClient)

		if rs.Type != "fusion_volume" {
			continue
		}
		attrs := rs.Primary.Attributes
		tenantName := attrs["tenant"]
		tenantSpaceName := attrs["tenant_space"]
		volumeName, ok := attrs["name"]
		if !ok {
			continue // Skip data sources
		}

		_, resp, err := client.VolumesApi.GetVolume(context.Background(), tenantName, tenantSpaceName, volumeName, nil)
		if err != nil && resp.StatusCode == http.StatusNotFound {
			continue
		}

		return fmt.Errorf("volume may still exist. Expected response code 404, got code %d", resp.StatusCode)
	}
	return nil
}

func testCheckVolumeDestroy(name, tenantName, tenantSpaceName string) func(s *terraform.State) error {
	return func(s *terraform.State) error {
		client := testAccProvider.Meta().(*hmrest.APIClient)

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
		resource.TestCheckResourceAttr(resourceName, "tenant", volState.Tenant),
		resource.TestCheckResourceAttr(resourceName, "tenant_space", volState.TenantSpace),
		resource.TestCheckResourceAttr(resourceName, "protection_policy", volState.ProtectionPolicyName),
		resource.TestCheckResourceAttr(resourceName, "storage_class", volState.StorageClassName),
		resource.TestCheckResourceAttr(resourceName, "placement_group", volState.PlacementGroup),
		resource.TestCheckResourceAttr(resourceName, "host_access_policies.#", fmt.Sprintf("%d", len(volState.Hosts))),
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

func generateVolumeTestConfigAndCommonTFConfig(eradicate *bool, hwTypes []string, array *string) (testVolume, string) {
	protectionPolicyName := acctest.RandomWithPrefix("pp-volTest")
	storageClassName := acctest.RandomWithPrefix("sc-volTest")

	cfg, commonConfig := generatePlacementGroupTestConfigAndCommonTFConfig(hwTypes)

	if array == nil {
		commonConfig += testPlacementGroupConfigWithRefsNoArray(
			cfg.Name, cfg.Name, cfg.DisplayName, cfg.Tenant, cfg.TenantSpace, cfg.Region, cfg.AZ, cfg.StorageService, true,
		)
	} else {
		commonConfig += testPlacementGroupConfigWithRefs(
			cfg.Name, cfg.Name, cfg.DisplayName, cfg.Tenant, cfg.TenantSpace, cfg.Region, cfg.AZ, cfg.StorageService, *array, true,
		)
	}

	commonConfig += testStorageClassConfigNoDisplayName(storageClassName, storageClassName, cfg.StorageService, testSizeLimit, testIopsLimit, testBandwidthLimit) +
		testAccProtectionPolicyConfig(protectionPolicyName, protectionPolicyName, protectionPolicyName, localRPO, localRetention, true)

	volCfg := testVolume{
		RName:                acctest.RandomWithPrefix("test_volume"),
		Name:                 acctest.RandomWithPrefix("test_vol"),
		DisplayName:          "initial display name",
		Tenant:               cfg.Tenant,
		TenantSpace:          cfg.TenantSpace,
		ProtectionPolicyName: protectionPolicyName,
		StorageClassName:     storageClassName,
		PlacementGroup:       cfg.Name,
		Size:                 1 << 20,
		StorageService:       cfg.StorageService,
	}

	if eradicate != nil {
		volCfg.Eradicate = eradicate
	}

	return volCfg, commonConfig
}
