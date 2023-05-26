/*
Copyright 2023 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package fusion

import (
	"context"

	"github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/utilities"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"
)

// Implements DataSource
type storageEndpointDataSource struct{}

// This is our entry point for the Storage Endpoint data source
func dataSourceStorageEndpoint() *schema.Resource {
	ds := &storageEndpointDataSource{}

	dsSchema := map[string]*schema.Schema{
		optionItems: {
			Type:     schema.TypeList,
			Computed: true,
			Elem: &schema.Resource{
				Schema: schemaStorageEndpoint(),
			},
		},
		optionRegion: {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringIsNotEmpty,
		},
		optionAvailabilityZone: {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringIsNotEmpty,
		},
	}

	storageEndpointDataSourceFunctions := NewBaseDataSourceFunctions("StorageEndpoint", ds, dsSchema)

	return storageEndpointDataSourceFunctions.Resource
}

func (ds *storageEndpointDataSource) ReadDataSource(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) error {
	resp, _, err := client.StorageEndpointsApi.ListStorageEndpoints(ctx, rdString(ctx, d, optionRegion), rdString(ctx, d, optionAvailabilityZone), nil)
	if err != nil {
		return err
	}

	storageEndpointsList := make([]map[string]interface{}, resp.Count)

	for i, se := range resp.Items {
		iscsiSet := make([]interface{}, 0)
		for _, discoveryInterface := range se.Iscsi.DiscoveryInterfaces {
			niGroups := make([]string, 0)

			for _, niGroup := range discoveryInterface.NetworkInterfaceGroups {
				niGroups = append(niGroups, niGroup.Name)
			}

			iscsi := map[string]interface{}{
				optionAddress: discoveryInterface.Address,
				optionGateway: discoveryInterface.Gateway,
			}

			if len(niGroups) != 0 {
				iscsi[optionNetworkInterfaceGroups] = niGroups
			}

			iscsiSet = append(iscsiSet, iscsi)
		}

		storageEndpointsList[i] = map[string]interface{}{
			optionName:             se.Name,
			optionDisplayName:      se.DisplayName,
			optionRegion:           se.Region.Name,
			optionAvailabilityZone: se.AvailabilityZone.Name,
			optionIscsi:            iscsiSet,
		}
	}

	if err := d.Set(optionItems, storageEndpointsList); err != nil {
		return err
	}

	d.SetId(utilities.GetIdForDataSource())

	return nil
}
