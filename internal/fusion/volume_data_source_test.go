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

func TestAccVolumeDataSource_basic(t *testing.T) {
	utilities.CheckTestSkip(t)

	dsTfNameConfig := acctest.RandomWithPrefix("volume_test_ds")
	tenant := acctest.RandomWithPrefix("volume_test_tenant")
	tenantSpace := acctest.RandomWithPrefix("volume_test_tenant_space")
	storageService := acctest.RandomWithPrefix("volume_test_storage_service")
	storageClass := acctest.RandomWithPrefix("volume_test_storage_class")
	placementGroup := acctest.RandomWithPrefix("volume_test_protection_group")
	eradicate := true
	volumeCount := 3
	volumes := make([]map[string]interface{}, 0, volumeCount)
	volumeConfigs := make([]string, 0, volumeCount)

	commonConfig := "" +
		testTenantConfig(tenant, tenant, tenant) +
		testTenantSpaceConfigWithRefs(tenantSpace, tenantSpace, tenantSpace, tenant) +
		testStorageServiceConfig(storageService, storageService, storageService, hwTypes) +
		testStorageClassConfigNoDisplayName(storageClass, storageClass, storageService, 2*1024*1024, 10000, 2*1024*1024) +
		testPlacementGroupConfigWithRefsNoArray(placementGroup, placementGroup, placementGroup, tenant, tenantSpace, preexistingRegion, preexistingAvailabilityZone, storageService, true)

	for i := 0; i < volumeCount; i++ {
		tfName := acctest.RandomWithPrefix("volume_test_tf_array")
		volumeName := acctest.RandomWithPrefix("volume_test_fs_array")
		displayName := acctest.RandomWithPrefix("volume_test_display_name")

		volumes = append(volumes, map[string]interface{}{
			"name":            volumeName,
			"display_name":    displayName,
			"tenant":          tenant,
			"tenant_space":    tenantSpace,
			"storage_class":   storageClass,
			"placement_group": placementGroup,
		})

		volumeConfigs = append(volumeConfigs, testVolumeConfig(testVolume{
			RName:                tfName,
			Name:                 volumeName,
			DisplayName:          displayName,
			ProtectionPolicyName: "",
			Tenant:               tenant,
			TenantSpace:          tenantSpace,
			StorageClassName:     storageClass,
			PlacementGroup:       placementGroup,
			Size:                 1024 * 1024,
			Eradicate:            &eradicate,
		}))
	}

	allVolumesConfig := commonConfig + strings.Join(volumeConfigs, "\n")
	partialVolumesConfig := commonConfig + volumeConfigs[0] + volumeConfigs[1]

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckVolumeDelete,
		Steps: []resource.TestStep{
			// Create n volumes
			{
				Config: allVolumesConfig,
			},
			// Check if they are contained in the data source
			{
				Config: allVolumesConfig + "\n" + testVolumeDataSourceConfig(dsTfNameConfig, tenant, tenantSpace),
				Check: utilities.TestCheckDataSourceExact(
					"fusion_volume", dsTfNameConfig, "items", volumes,
				),
			},
			{
				Config: partialVolumesConfig,
			},
			// Remove one volume. Check if only two of them are contained in the data source
			{
				Config: partialVolumesConfig + "\n" + testVolumeDataSourceConfig(dsTfNameConfig, tenant, tenantSpace),
				Check: utilities.TestCheckDataSourceExact(
					"fusion_volume", dsTfNameConfig, "items", []map[string]interface{}{
						volumes[0], volumes[1],
					},
				),
			},
		},
	})
}

func testVolumeDataSourceConfig(dsName string, tenant string, tenantSpace string) string {
	return fmt.Sprintf(`data "fusion_volume" "%[1]s" {
		tenant        = fusion_tenant.%[2]s.name
		tenant_space  = fusion_tenant_space.%[3]s.name
	}`, dsName, tenant, tenantSpace)
}
