package fusion

import (
	"fmt"
	"testing"

	"github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/utilities"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"

	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"
)

func TestAccSnapshot_e2eDemoTest(t *testing.T) {
	utilities.CheckTestSkip(t)

	ctx := setupTestCtx(t)
	hmClient := testAccPreCheckWithReturningClient(ctx, t)

	arrays := getArraysInPreexistingRegionAndAZ(t)
	arrayHWType := make(map[string]hmrest.Array)
	selectedArrays := []hmrest.Array{}

	// Find two arrays with the same HW type - to allow moving storage class and placement group
	for _, array := range arrays {
		if selectedArray, ok := arrayHWType[array.HardwareType.Name]; ok {
			selectedArrays = []hmrest.Array{selectedArray, array}
			break
		}

		arrayHWType[array.HardwareType.Name] = array
	}

	if len(selectedArrays) < 2 {
		msg := fmt.Sprintf("not enough arrays of the same HW type to run TestAccSnapshot_e2eDemoTest test (required 2, found %d)", len(arrays))
		t.Skip(msg)
		return
	}

	array1, array2 := selectedArrays[0], selectedArrays[1]

	eradicate := true
	snapshotName := acctest.RandomWithPrefix("snapshot")
	protectionPolicyName := acctest.RandomWithPrefix("pp-volTest")
	storageClassName1 := acctest.RandomWithPrefix("sc-volTest")
	storageClassName2 := acctest.RandomWithPrefix("sc-volTest")
	pg, commonConfig := generatePlacementGroupTestConfigAndCommonTFConfig([]string{array1.HardwareType.Name})

	pg1 := pg
	pg1.Name = acctest.RandomWithPrefix("pg")
	pg1.RName = pg1.Name

	// PG on array 1
	pg1Config := testPlacementGroupConfigWithRefs(
		pg.Name, pg.Name, pg.DisplayName, pg.Tenant, pg.TenantSpace, pg.Region, pg.AZ, pg.StorageService, array1.Name, true,
	)

	// PG on array 2 (for live migration)
	pg1ArrayChangeConfig := testPlacementGroupConfigWithRefs(
		pg.Name, pg.Name, pg.DisplayName, pg.Tenant, pg.TenantSpace, pg.Region, pg.AZ, pg.StorageService, array2.Name, true,
	)

	// A new PG
	pg2Config := testPlacementGroupConfigWithRefs(
		pg1.Name, pg1.Name, pg1.DisplayName, pg1.Tenant, pg1.TenantSpace, pg1.Region, pg1.AZ, pg1.StorageService, array1.Name, true,
	)

	commonConfig += testStorageClassConfigNoDisplayName(storageClassName1, storageClassName1, pg.StorageService, testSizeLimit, testIopsLimit, testBandwidthLimit) +
		testStorageClassConfigNoDisplayName(storageClassName2, storageClassName2, pg.StorageService, testSizeLimit, testIopsLimit, testBandwidthLimit) +
		testAccProtectionPolicyConfig(protectionPolicyName, protectionPolicyName, protectionPolicyName, localRPO, localRetention, true)

	volState := testVolume{
		RName:                acctest.RandomWithPrefix("test_volume"),
		Name:                 acctest.RandomWithPrefix("test_vol"),
		DisplayName:          "initial display name",
		Tenant:               pg.Tenant,
		TenantSpace:          pg.TenantSpace,
		ProtectionPolicyName: protectionPolicyName,
		StorageClassName:     storageClassName1,
		PlacementGroup:       pg.Name,
		Size:                 1 << 20,
		StorageService:       pg.StorageService,
		Eradicate:            &eradicate,
	}

	volState1 := volState
	volState1.RName = acctest.RandomWithPrefix("test_volume")
	volState1.Name = acctest.RandomWithPrefix("test_vol")
	volState1.StorageClassName = storageClassName2
	volState1.SourceLink = map[string]string{
		"tenant":          volState.Tenant,
		"tenant_space":    volState.TenantSpace,
		"snapshot":        snapshotName,
		"volume_snapshot": volState.Name,
	}

	volState2 := volState1
	volState2.PlacementGroup = pg1.Name

	iqn := []string{""}

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckVolumeDelete,
		Steps: []resource.TestStep{
			{
				// Create a volume
				Config: pg1Config + commonConfig + testVolumeConfig(volState),
				Check: func(s *terraform.State) error {
					if err := testVolumeExists("fusion_volume."+volState.RName, t)(s); err != nil {
						return err
					}

					// TODO: Remove once/if we have a snapshot resource
					snapPost := hmrest.SnapshotPost{
						Name:           snapshotName,
						PlacementGroup: volState.PlacementGroup,
					}

					testVolumeDoOperation(t, ctx, hmClient, "snapshot-create")(
						hmClient.SnapshotsApi.CreateSnapshot(ctx, snapPost, volState.Tenant, volState.TenantSpace, nil),
					)

					return nil
				},
			},
			{
				// Create another volume that is a copy of the first volume (assign a different storage class) affinity
				Config: pg1Config + commonConfig + testVolumeConfig(volState) + testVolumeConfigCopy(volState1),
				Check: resource.ComposeTestCheckFunc(
					testVolumeExists("fusion_volume."+volState.RName, t),
					testVolumeExists("fusion_volume."+volState1.RName, t),
					testCheckVolumeAttributes("fusion_volume."+volState1.RName, volState1),
					func(s *terraform.State) error {
						volume := s.RootModule().Resources["fusion_volume."+volState1.RName]
						iqn[0] = volume.Primary.Attributes["target_iscsi_iqn"] // Get assigned IQN
						return nil
					},
				),
			},
			{
				// Move the placement group to a different array - anti-affinity (live migration)
				Config: pg1ArrayChangeConfig + commonConfig + testVolumeConfig(volState) + testVolumeConfigCopy(volState1),
				Check: resource.ComposeTestCheckFunc(
					testVolumeExists("fusion_volume."+volState.RName, t),
					testVolumeExists("fusion_volume."+volState1.RName, t),
					testCheckVolumeAttributes("fusion_volume."+volState1.RName, volState1),
					func(s *terraform.State) error {
						return resource.TestCheckResourceAttr("fusion_volume."+volState1.RName, "target_iscsi_iqn", iqn[0])(s)
					},
				),
			},
			{
				// Move the volume to a different array by switching placement group - does not live migrate
				Config: pg2Config + pg1ArrayChangeConfig + commonConfig + testVolumeConfigCopy(volState2) + testVolumeConfig(volState),
				Check: resource.ComposeTestCheckFunc(
					testVolumeExists("fusion_volume."+volState2.RName, t),
					testCheckVolumeAttributes("fusion_volume."+volState2.RName, volState2),
					func(s *terraform.State) error {
						err := resource.TestCheckResourceAttr("fusion_volume."+volState2.RName, "target_iscsi_iqn", iqn[0])(s)
						if err == nil {
							return fmt.Errorf("'target_iscsi_iqn' should not be equal to '%s'", iqn[0])
						}
						return nil
					},
				),
			},
			{
				Config: pg2Config + pg1ArrayChangeConfig + commonConfig,
			},
		},
	})
}
