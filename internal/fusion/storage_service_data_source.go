/*
Copyright 2023 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package fusion

import (
	context "context"

	"github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/utilities"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"
)

// Implements DataSource
type storageServiceDataSource struct{}

// This is our entry point for the Storage Service data source
func dataSourceStorageService() *schema.Resource {
	ds := &storageServiceDataSource{}

	dsSchema := map[string]*schema.Schema{
		"items": {
			Type:     schema.TypeList,
			Computed: true,
			Elem: &schema.Resource{
				Schema: schemaStorageService(),
			},
			Description: "List of matching Storage Services.",
		},
	}

	storageServiceDataSourceFunctions := NewBaseDataSourceFunctions(resourceKindStorageService, ds, dsSchema)

	return storageServiceDataSourceFunctions.Resource
}

func (ds *storageServiceDataSource) ReadDataSource(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) error {
	resp, _, err := client.StorageServicesApi.ListStorageServices(ctx, nil)
	if err != nil {
		return err
	}

	storageServicesList := make([]map[string]interface{}, resp.Count)

	for i, ss := range resp.Items {
		hardwareTypes := make([]string, len(ss.HardwareTypes))
		for i, hwType := range ss.HardwareTypes {
			hardwareTypes[i] = hwType.Name
		}

		storageServicesList[i] = map[string]interface{}{
			"name":           ss.Name,
			"display_name":   ss.DisplayName,
			"hardware_types": hardwareTypes,
		}
	}

	if err := d.Set("items", storageServicesList); err != nil {
		return err
	}

	d.SetId(utilities.GetIdForDataSource())

	return nil
}
