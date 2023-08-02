/*
Copyright 2023 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package fusion

import (
	"fmt"
	"strings"
	"testing"

	"github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/utilities"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccVolumeSnapshotDataSource_basic(t *testing.T) {
	utilities.CheckTestSkip(t)

	// Setup resources
	pgCfg, stringCfg := generatePlacementGroupTestConfigAndCommonTFConfig([]string{"flash-array-x", "flash-array-c", "flash-array-x-optane", "flash-array-xl"})

	scName := acctest.RandomWithPrefix("storage_class-snapTest")

	scConfig := testStorageClassConfigNoDisplayName(scName, scName, pgCfg.StorageService, 1048576, 1000, 1048576)
	pgConfig := testPlacementGroupConfigWithRefsNoArray(pgCfg.Name, pgCfg.Name, pgCfg.DisplayName, pgCfg.Tenant, pgCfg.TenantSpace, pgCfg.Region, pgCfg.AZ, pgCfg.StorageService, true)

	// Volumes for snapshot
	volumesCount := 2
	volumeSnapshots := make([]map[string]interface{}, volumesCount)
	volumeNames := make([]string, volumesCount)
	volumeConfigs := make([]string, volumesCount)
	eradicate := true
	for i := 0; i < volumesCount; i++ {
		volumeNames[i] = acctest.RandomWithPrefix("volume")
		vol := testVolume{
			RName:            volumeNames[i],
			Name:             volumeNames[i],
			DisplayName:      volumeNames[i],
			StorageClassName: scName,
			Tenant:           pgCfg.Tenant,
			TenantSpace:      pgCfg.TenantSpace,
			PlacementGroup:   pgCfg.Name,
			Size:             1048576,
			Eradicate:        &eradicate,
		}
		volumeConfigs[i] = testVolumeConfig(vol)
	}
	additionalVolumeName := acctest.RandomWithPrefix("volume")
	additionalVolumeConfig := testVolumeConfig(testVolume{
		RName:            additionalVolumeName,
		Name:             additionalVolumeName,
		DisplayName:      additionalVolumeName,
		StorageClassName: scName,
		Tenant:           pgCfg.Tenant,
		TenantSpace:      pgCfg.TenantSpace,
		PlacementGroup:   pgCfg.Name,
		Size:             1048576,
		Eradicate:        &eradicate,
	})

	snapshotName1 := acctest.RandomWithPrefix("snapshot")
	snapshotName2 := acctest.RandomWithPrefix("snapshot")

	// Data sources names
	dsVolumeSnapshot := acctest.RandomWithPrefix("volume-snapshot-data-source")
	dsVolumeSnapshotAdditonal := acctest.RandomWithPrefix("volume-snapshot-data-source")
	dsVolumeSnapshotVolume := acctest.RandomWithPrefix("volume-snapshot-data-source")
	dsVolumeSnapshotPg := acctest.RandomWithPrefix("volume-snapshot-data-source")

	commonConfig := stringCfg + pgConfig + scConfig + strings.Join(volumeConfigs, "\n")

	ctx := setupTestCtx(t)
	// Create hm client
	client := testAccPreCheckWithReturningClient(ctx, t)

	// Create snapshot and check if data source has 2 volume snapshots
	checkDataSourceHasAllVolumeSnapshotsStep := func() resource.TestStep {
		step := resource.TestStep{}
		// Create snapshot before testing data source
		step.PreConfig = func() {
			_, err := testCreateSnapshotsListWithPlacementGroup(ctx, []string{snapshotName1}, pgCfg.Tenant, pgCfg.TenantSpace, pgCfg.Name, client)
			if err != nil {
				t.Fatal(err)
			}
		}
		// Data source must have these volume_snapshots
		for i, volumeName := range volumeNames {
			volumeSnapshots[i] = map[string]interface{}{
				"tenant":          pgCfg.Tenant,
				"tenant_space":    pgCfg.TenantSpace,
				"snapshot":        snapshotName1,
				"name":            volumeName,
				"placement_group": pgCfg.Name,
				"size":            "1048576",
			}
		}

		step.Config = commonConfig + "\n" + testVolumeSnapshotDataSourceConfig(dsVolumeSnapshot, pgCfg.Tenant, pgCfg.TenantSpace, snapshotName1)
		step.Check = utilities.TestCheckDataSourceExact("fusion_volume_snapshot", dsVolumeSnapshot, "items", volumeSnapshots)
		return step
	}

	// Create new snapshot and check if previous data source still has 2 volume snapshots inside
	checkNewAndOldDataSourceSnapshotStep := func() resource.TestStep {
		step := resource.TestStep{}

		// Create volume volumeSnapshot
		step.PreConfig = func() {
			_, err := testCreateSnapshotsWithVolumes(ctx, []string{snapshotName2}, pgCfg.Tenant, pgCfg.TenantSpace, []string{additionalVolumeName}, client)
			if err != nil {
				t.Fatal(err)
			}
		}
		volumeSnapshot := map[string]interface{}{
			"name":            additionalVolumeName,
			"volume":          additionalVolumeName,
			"tenant":          pgCfg.Tenant,
			"tenant_space":    pgCfg.TenantSpace,
			"snapshot":        snapshotName2,
			"placement_group": pgCfg.Name,
		}

		step.Config = commonConfig + "\n" +
			testVolumeSnapshotDataSourceConfig(dsVolumeSnapshot, pgCfg.Tenant, pgCfg.TenantSpace, snapshotName1) + "\n" +
			testVolumeSnapshotDataSourceConfig(dsVolumeSnapshotAdditonal, pgCfg.Tenant, pgCfg.TenantSpace, snapshotName2)

		step.Check = resource.ComposeTestCheckFunc(
			utilities.TestCheckDataSourceExact("fusion_volume_snapshot", dsVolumeSnapshot, "items", volumeSnapshots),
			utilities.TestCheckDataSourceExact("fusion_volume_snapshot", dsVolumeSnapshotAdditonal, "items", []map[string]interface{}{volumeSnapshot}),
		)

		return step
	}

	// Test step for Volume ID filter
	checkDataSourceIsFilteredByVolumeIdStep := func() resource.TestStep {
		step := resource.TestStep{}

		step.Config = commonConfig + "\n" +
			testVolumeSnapshotDataSourceVolumeIdFilterConfig(dsVolumeSnapshotVolume, pgCfg.Tenant, pgCfg.TenantSpace, snapshotName1, volumeNames[0])
		snapshot := map[string]interface{}{
			"name":            volumeNames[0],
			"tenant":          pgCfg.Tenant,
			"tenant_space":    pgCfg.TenantSpace,
			"snapshot":        snapshotName1,
			"placement_group": pgCfg.Name,
		}
		step.Check = utilities.TestCheckDataSourceExact("fusion_volume_snapshot", dsVolumeSnapshotVolume, "items", []map[string]interface{}{snapshot})

		return step
	}

	checkDataSourceIsFilteredByPGIdStep := func() resource.TestStep {
		step := resource.TestStep{}

		step.Config = commonConfig + "\n" +
			testVolumeSnapshotDataSourcePlacementGroupIdFilterConfig(dsVolumeSnapshotPg, pgCfg.Tenant, pgCfg.TenantSpace, snapshotName1, pgCfg.Name)

		step.Check = utilities.TestCheckDataSourceExact("fusion_volume_snapshot", dsVolumeSnapshotPg, "items", volumeSnapshots)

		return step
	}

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testAccProvidersFactory,
		Steps: []resource.TestStep{
			{
				// Create infrastructure before creating Snapshots
				Config: commonConfig,
			},
			checkDataSourceHasAllVolumeSnapshotsStep(),
			{
				// Create another volume
				Config: commonConfig + additionalVolumeConfig,
			},
			checkNewAndOldDataSourceSnapshotStep(),
			checkDataSourceIsFilteredByVolumeIdStep(),
			checkDataSourceIsFilteredByPGIdStep(),
		},
	})

}

func testVolumeSnapshotDataSourceConfig(dsName, tenant, tenantSpace, snapshot string) string {
	return fmt.Sprintf(`data "fusion_volume_snapshot" "%[1]s" {
		tenant = fusion_tenant.%[2]s.name
		tenant_space = fusion_tenant_space.%[3]s.name
		snapshot = "%[4]s"
	}`, dsName, tenant, tenantSpace, snapshot)
}

func testVolumeSnapshotDataSourceVolumeIdFilterConfig(dsName, tenant, tenantSpace, snapshot, volumeRes string) string {
	return fmt.Sprintf(`data "fusion_volume_snapshot" "%[1]s" {
		tenant = fusion_tenant.%[2]s.name
		tenant_space = fusion_tenant_space.%[3]s.name
		snapshot = "%[4]s"
		volume_id = fusion_volume.%[5]s.id
	}`, dsName, tenant, tenantSpace, snapshot, volumeRes)
}

func testVolumeSnapshotDataSourcePlacementGroupIdFilterConfig(dsName, tenant, tenantSpace, snapshot, pgRes string) string {
	return fmt.Sprintf(`data "fusion_volume_snapshot" "%[1]s" {
		tenant = fusion_tenant.%[2]s.name
		tenant_space = fusion_tenant_space.%[3]s.name
		snapshot = "%[4]s"
		placement_group_id = fusion_placement_group.%[5]s.id
	}`, dsName, tenant, tenantSpace, snapshot, pgRes)
}
