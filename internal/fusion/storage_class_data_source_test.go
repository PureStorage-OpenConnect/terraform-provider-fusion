/*
Copyright 2023 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package fusion

import (
	"fmt"
	"strconv"
	"strings"
	"testing"

	"github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/utilities"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

// Contains correct list of StorageClasses
func TestAccStorageClassDataSource_basic(t *testing.T) {
	utilities.CheckTestSkip(t)

	dsNameConfig := acctest.RandomWithPrefix("storage_class_ds_test")
	storageClassesCount := 3
	storageClasses := make([]map[string]interface{}, storageClassesCount)
	storageClassConfigs := make([]string, storageClassesCount)
	storageService := acctest.RandomWithPrefix("storage_sevice_test")
	storageServiceConfig := testStorageServiceConfigNoDisplayName(storageService, storageService, []string{"flash-array-x"})

	for i := 0; i < storageClassesCount; i++ {
		storageClassResourceNameConfig := acctest.RandomWithPrefix("storage_class_test")
		storageClassName := acctest.RandomWithPrefix("test_sc")
		storageClassDisplayName := acctest.RandomWithPrefix("storage_class-display-name")
		storageClassBandwidth := acctest.RandIntRange(1, 10) * 1048576
		storageClassSize := acctest.RandIntRange(1, 10) * 1048576
		storageClassIops := acctest.RandIntRange(100, 100000)

		storageClasses[i] = map[string]interface{}{
			"name":            storageClassName,
			"display_name":    storageClassDisplayName,
			"storage_service": storageService,
			"size_limit":      strconv.Itoa(storageClassSize),
			"iops_limit":      strconv.Itoa(storageClassIops),
			"bandwidth_limit": strconv.Itoa(storageClassBandwidth),
		}

		storageClassConfigs[i] = testStorageClassConfig(storageClassResourceNameConfig, storageClassName,
			storageClassDisplayName, storageService, storageClassSize, storageClassIops, storageClassBandwidth)
	}

	allConfigs := storageServiceConfig + strings.Join(storageClassConfigs, "\n")
	partialConfig := storageServiceConfig + storageClassConfigs[0] + storageClassConfigs[1]

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckStorageClassDestroy,
		Steps: []resource.TestStep{
			// Create n Storage Classes
			{
				Config: allConfigs,
			},
			// Check if they are contained in the data source
			{
				Config: allConfigs + "\n" + testStorageClassDataSourceConfig(dsNameConfig, storageService),
				Check:  utilities.TestCheckDataSource("fusion_storage_class", dsNameConfig, "items", storageClasses),
			},
			{
				Config: partialConfig,
			},
			// Remove one StorageClass. Check if only two of them are contained in the data source
			{
				Config: partialConfig + "\n" + testStorageClassDataSourceConfig(dsNameConfig, storageService),
				Check: resource.ComposeTestCheckFunc(
					utilities.TestCheckDataSource(
						"fusion_storage_class", dsNameConfig, "items", []map[string]interface{}{storageClasses[0], storageClasses[1]},
					),
					utilities.TestCheckDataSourceNotHave(
						"fusion_storage_class", dsNameConfig, "items", []map[string]interface{}{storageClasses[2]},
					),
				),
			},
		},
	})
}

func testStorageClassDataSourceConfig(dsName string, storageService string) string {
	return fmt.Sprintf(`data "fusion_storage_class" "%[1]s" {
		storage_service = "%[2]s"
	}`, dsName, storageService)
}
