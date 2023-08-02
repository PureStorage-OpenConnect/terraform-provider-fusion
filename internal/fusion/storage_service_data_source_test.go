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

// Contains correct list of Storage Services
func TestAccStorageServiceDataSource_basic(t *testing.T) {
	utilities.CheckTestSkip(t)

	dsNameConfig := acctest.RandomWithPrefix("storage_service_ds_test")
	storageServiceCount := 3
	storageServices := make([]map[string]interface{}, storageServiceCount)
	storageServiceConfigs := make([]string, storageServiceCount)

	for i := 0; i < storageServiceCount; i++ {
		ssResourceNameConfig := acctest.RandomWithPrefix("storage_service_test")
		ssName := acctest.RandomWithPrefix("test_ss")
		ssDisplayName := acctest.RandomWithPrefix("storage-service-display-name")
		hwTypes := []string{"flash-array-x-optane", "flash-array-x"}

		storageServices[i] = map[string]interface{}{
			"name":           ssName,
			"display_name":   ssDisplayName,
			"hardware_types": hwTypes,
		}

		storageServiceConfigs[i] = testStorageServiceConfig(ssResourceNameConfig, ssName, ssDisplayName, hwTypes)
	}

	allStorageServicesConfig := strings.Join(storageServiceConfigs, "\n")
	partialStorageServicesConfig := storageServiceConfigs[0] + storageServiceConfigs[1]

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckStorageServiceDestroy,
		Steps: []resource.TestStep{
			// Create n storage services
			{
				Config: allStorageServicesConfig,
			},
			// Check if they are contained in the data source
			{
				Config: allStorageServicesConfig + "\n" + testStorageServiceDataSourceConfig(dsNameConfig),
				Check: utilities.TestCheckDataSource(
					"fusion_storage_service", dsNameConfig, "items", storageServices,
				),
			},
			{
				Config: partialStorageServicesConfig,
			},
			// Remove one storage service. Check if only two of them are contained in the data source
			{
				Config: partialStorageServicesConfig + "\n" + testStorageServiceDataSourceConfig(dsNameConfig),
				Check: resource.ComposeTestCheckFunc(
					utilities.TestCheckDataSource(
						"fusion_storage_service", dsNameConfig, "items", []map[string]interface{}{
							storageServices[0], storageServices[1],
						},
					),
					utilities.TestCheckDataSourceNotHave(
						"fusion_storage_service", dsNameConfig, "items", []map[string]interface{}{
							storageServices[2],
						},
					),
				),
			},
		},
	})
}

func testStorageServiceDataSourceConfig(dsName string) string {
	return fmt.Sprintf(`data "fusion_storage_service" "%[1]s" {}`, dsName)
}
