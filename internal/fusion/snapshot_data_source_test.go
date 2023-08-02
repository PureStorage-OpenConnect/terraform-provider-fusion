/*
Copyright 2023 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package fusion

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/utilities"
	"github.com/antihax/optional"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"
)

func TestAccSnapshotDataSource_basic(t *testing.T) {
	utilities.CheckTestSkip(t)

	snapshotsCount := 3
	volumesCount := 3

	// Setup resources
	cfg, stringCfg := generatePlacementGroupTestConfigAndCommonTFConfig([]string{"flash-array-x", "flash-array-c", "flash-array-x-optane", "flash-array-xl"})

	scName := acctest.RandomWithPrefix("storage_class-snapTest")
	scConfig := testStorageClassConfigNoDisplayName(scName, scName, cfg.StorageService, 1048576, 1000, 1048576)
	pgConfig := testPlacementGroupConfigWithRefsNoArray(cfg.Name, cfg.Name, cfg.DisplayName, cfg.Tenant, cfg.TenantSpace, cfg.Region, cfg.AZ, cfg.StorageService, true)

	ppName := acctest.RandomWithPrefix("test-protection-policy")
	protectionPolicyConfig := testAccProtectionPolicyConfig(ppName, ppName, ppName, localRPO, localRetention, true)

	// Volumes for snapshot pg filter test
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
			Tenant:           cfg.Tenant,
			TenantSpace:      cfg.TenantSpace,
			PlacementGroup:   cfg.Name,
			Size:             1048576,
			Eradicate:        &eradicate,
		}
		volumeConfigs[i] = testVolumeConfig(vol)
	}

	// Volume for snapshot volume filter
	additionalVolumeName := acctest.RandomWithPrefix("volume")
	additionalVolumeConfig := testVolumeConfig(testVolume{
		RName:            additionalVolumeName,
		Name:             additionalVolumeName,
		DisplayName:      additionalVolumeName,
		StorageClassName: scName,
		Tenant:           cfg.Tenant,
		TenantSpace:      cfg.TenantSpace,
		PlacementGroup:   cfg.Name,
		Size:             1048576,
		Eradicate:        &eradicate,
	})
	// Volume for snapshot protection policy filter
	protectionPolicyVolumeName := acctest.RandomWithPrefix("volume")
	protectionPolicyVolumeConfig := testVolumeConfig(testVolume{
		RName:                protectionPolicyVolumeName,
		Name:                 protectionPolicyVolumeName,
		DisplayName:          protectionPolicyVolumeName,
		StorageClassName:     scName,
		Tenant:               cfg.Tenant,
		TenantSpace:          cfg.TenantSpace,
		PlacementGroup:       cfg.Name,
		Size:                 1048576,
		Eradicate:            &eradicate,
		ProtectionPolicyName: ppName,
	})

	// Create snapshot names
	snapshotNames := make([]string, snapshotsCount)
	for i := 0; i < snapshotsCount; i++ {
		snapshotNames[i] = "snapshot" + fmt.Sprint(i)
	}

	// Data sources names
	dsSnapPgName := acctest.RandomWithPrefix("snapshot-data-source")
	dsSnapVolName := acctest.RandomWithPrefix("snapshot-data-source")
	dsSnapName := acctest.RandomWithPrefix("snapshot-data-source")
	dsSnapPpName := acctest.RandomWithPrefix("snapshot-data-source")

	commonConfig := stringCfg + pgConfig + scConfig + strings.Join(volumeConfigs, "\n")

	ctx := setupTestCtx(t)
	// Create hm client
	client := testAccPreCheckWithReturningClient(ctx, t)
	checkDataSourceIsCreatedByPlacementGroupStep := func() resource.TestStep {
		step := resource.TestStep{}
		snapshots := make([]map[string]interface{}, snapshotsCount)

		step.PreConfig = func() {
			snapshotsList, err := testCreateSnapshotsListWithPlacementGroup(ctx, snapshotNames, cfg.Tenant, cfg.TenantSpace, cfg.Name, client)
			if err != nil {
				t.Fatal(err)
			}
			copy(snapshots, snapshotsList)
		}
		step.Config = commonConfig + "\n" + testSnapshotDataSourceConfigWithPlacementGroup(dsSnapPgName, cfg.Tenant, cfg.TenantSpace, cfg.Name)
		step.Check = utilities.TestCheckDataSourceExact("fusion_snapshot", dsSnapPgName, "items", snapshots)
		return step
	}

	checkDataSourceIsCreatedByVolumeStep := func() resource.TestStep {
		step := resource.TestStep{}
		snapshotName := acctest.RandomWithPrefix("snapshot")
		snapshot := make(map[string]interface{})

		// Create volume snapshot
		step.PreConfig = func() {
			snapshots, err := testCreateSnapshotsWithVolumes(ctx, []string{snapshotName}, cfg.Tenant, cfg.TenantSpace, []string{additionalVolumeName}, client)
			if err != nil {
				t.Fatal(err)
			}
			for k, v := range snapshots[0] {
				snapshot[k] = v
			}
		}
		step.Config = commonConfig + additionalVolumeConfig + "\n" + testSnapshotDataSourceConfigWithVolume(dsSnapVolName, cfg.Tenant, cfg.TenantSpace, additionalVolumeName)
		step.Check = utilities.TestCheckDataSourceExact("fusion_snapshot", dsSnapVolName, "items", []map[string]interface{}{snapshot})
		return step
	}

	checkDataSourceIsCreatedByVolumeAndPGStep := func() resource.TestStep {
		step := resource.TestStep{}
		snapshots := make([]map[string]interface{}, snapshotsCount+1)

		// Create volume snapshot
		step.PreConfig = func() {
			snapshotsList, _, err := client.SnapshotsApi.ListSnapshots(ctx, cfg.Tenant, cfg.TenantSpace, nil)
			if err != nil {
				t.Fatal(err)
			}
			for i, item := range snapshotsList.Items {
				snapshots[i] = map[string]interface{}{
					"name":         item.Name,
					"tenant":       cfg.Tenant,
					"tenant_space": cfg.TenantSpace,
					"destroyed":    false,
				}
			}
		}
		step.Config = commonConfig + additionalVolumeConfig + "\n" + testSnapshotDataSourceConfigWithoutFilters(dsSnapName, cfg.Tenant, cfg.TenantSpace)
		step.Check = utilities.TestCheckDataSourceExact("fusion_snapshot", dsSnapName, "items", snapshots)
		return step
	}

	checkDataSourceIsCreatedByProtectionPolicyStep := func() resource.TestStep {
		step := resource.TestStep{}
		snapshot := map[string]interface{}{
			"tenant":            cfg.Tenant,
			"tenant_space":      cfg.TenantSpace,
			"protection_policy": ppName,
			"destroyed":         false,
		}
		step.PreConfig = func() {
			opts := hmrest.SnapshotsApiListSnapshotsOpts{}
			opts.Volume = optional.NewString(protectionPolicyVolumeName)
			snapshots, _, err := client.SnapshotsApi.ListSnapshots(ctx, cfg.Tenant, cfg.TenantSpace, &opts)
			if err != nil {
				t.Fatal(err)
			}
			snapshot["name"] = snapshots.Items[0].Name
		}

		step.Config = commonConfig + protectionPolicyConfig + protectionPolicyVolumeConfig + "\n" + testSnapshotDataSourceConfigWithProtectionPolicy(dsSnapPpName, cfg.Tenant, cfg.TenantSpace, ppName)
		step.Check = utilities.TestCheckDataSourceExact("fusion_snapshot", dsSnapPpName, "items", []map[string]interface{}{snapshot})
		return step
	}

	resource.ParallelTest(t, resource.TestCase{
		ProviderFactories: testAccProvidersFactory,
		Steps: []resource.TestStep{
			{
				// Create infrastructure before creating snapshots
				Config: commonConfig,
			},
			checkDataSourceIsCreatedByPlacementGroupStep(),
			{
				// Create another volume
				Config: commonConfig + additionalVolumeConfig,
			},
			checkDataSourceIsCreatedByVolumeStep(),
			checkDataSourceIsCreatedByVolumeAndPGStep(),
			{
				// Create volume linked to protection policy
				Config: commonConfig + protectionPolicyConfig + protectionPolicyVolumeConfig,
			},
			checkDataSourceIsCreatedByProtectionPolicyStep(),
		},
	})

}

func testCreateSnapshotsWithVolumes(ctx context.Context, snapshotNames []string, tenant, tenantSpace string, volumes []string, client *hmrest.APIClient) ([]map[string]interface{}, error) {
	snapshots := make([]map[string]interface{}, len(snapshotNames))

	for i, snapshotName := range snapshotNames {
		postSnapshot := hmrest.SnapshotPost{
			Name:    snapshotName,
			Volumes: volumes,
		}
		err := createSnapshot(ctx, &postSnapshot, tenant, tenantSpace, client)
		if err != nil {
			return []map[string]interface{}{}, err
		}

		snapshots[i] = map[string]interface{}{
			"name":         snapshotName,
			"tenant":       tenant,
			"tenant_space": tenantSpace,
			"destroyed":    false,
		}
	}

	return snapshots, nil
}

func testCreateSnapshotsListWithPlacementGroup(ctx context.Context, snapshotNames []string, tenant, tenantSpace, placementGroup string, client *hmrest.APIClient) ([]map[string]interface{}, error) {
	snapshots := make([]map[string]interface{}, len(snapshotNames))

	for i, snapshotName := range snapshotNames {
		postSnapshot := hmrest.SnapshotPost{
			Name:           snapshotName,
			PlacementGroup: placementGroup,
		}
		err := createSnapshot(ctx, &postSnapshot, tenant, tenantSpace, client)
		if err != nil {
			return []map[string]interface{}{}, err
		}

		snapshots[i] = map[string]interface{}{
			"name":         snapshotName,
			"tenant":       tenant,
			"tenant_space": tenantSpace,
			"destroyed":    false,
		}
	}

	return snapshots, nil
}

func testSnapshotDataSourceConfigWithVolume(dsName, tenant, tenantSpace, volume string) string {
	return fmt.Sprintf(`data "fusion_snapshot" "%[1]s" {
		tenant = fusion_tenant.%[2]s.name
		tenant_space = fusion_tenant_space.%[3]s.name
		volume = fusion_volume.%[4]s.name
	}`, dsName, tenant, tenantSpace, volume)
}

func testSnapshotDataSourceConfigWithPlacementGroup(dsName, tenant, tenantSpace, pg string) string {
	return fmt.Sprintf(`data "fusion_snapshot" "%[1]s" {
		tenant = fusion_tenant.%[2]s.name
		tenant_space = fusion_tenant_space.%[3]s.name
		placement_group = fusion_placement_group.%[4]s.name
	}`, dsName, tenant, tenantSpace, pg)
}

func testSnapshotDataSourceConfigWithProtectionPolicy(dsName, tenant, tenantSpace, protectionPolicy string) string {
	return fmt.Sprintf(`data "fusion_snapshot" "%[1]s" {
		tenant = fusion_tenant.%[2]s.name
		tenant_space = fusion_tenant_space.%[3]s.name
		protection_policy_id = fusion_protection_policy.%[4]s.id
	}`, dsName, tenant, tenantSpace, protectionPolicy)
}

func testSnapshotDataSourceConfigWithoutFilters(dsName, tenant, tenantSpace string) string {
	return fmt.Sprintf(`data "fusion_snapshot" "%[1]s" {
		tenant = fusion_tenant.%[2]s.name
		tenant_space = fusion_tenant_space.%[3]s.name
	}`, dsName, tenant, tenantSpace)
}
