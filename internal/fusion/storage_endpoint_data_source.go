/*
Copyright 2023 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package fusion

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/utilities"
	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"
)

// Implements DataSource
type storageEndpointDataSource struct{}

// This is our entry point for the Storage Endpoint data source
func dataSourceStorageEndpoint() *schema.Resource {
	ds := &storageEndpointDataSource{}

	seSchema := schemaStorageEndpoint()
	seSchema[optionIscsi].ExactlyOneOf = nil
	seSchema[optionCbsAzureIscsi].ExactlyOneOf = nil

	dsSchema := map[string]*schema.Schema{
		optionItems: {
			Type:     schema.TypeList,
			Computed: true,
			Elem: &schema.Resource{
				Schema: seSchema,
			},
			Description: "List of matching Storage Endpoints.",
		},
		optionRegion: {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "The name of the Region in which this Storage Endpoint is located.",
		},
		optionAvailabilityZone: {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "The name of the Availability Zone in which this Storage Endpoint is located.",
		},
	}

	storageEndpointDataSourceFunctions := NewBaseDataSourceFunctions(resourceKindStorageEndpoint, ds, dsSchema)

	return storageEndpointDataSourceFunctions.Resource
}

func (ds *storageEndpointDataSource) ReadDataSource(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) error {
	resp, _, err := client.StorageEndpointsApi.ListStorageEndpoints(ctx, rdString(ctx, d, optionRegion), rdString(ctx, d, optionAvailabilityZone), nil)
	if err != nil {
		return err
	}

	storageEndpointsList := make([]map[string]interface{}, 0, resp.Count)

	for _, se := range resp.Items {
		storageEndpointOut := map[string]interface{}{
			optionName:             se.Name,
			optionDisplayName:      se.DisplayName,
			optionRegion:           se.Region.Name,
			optionAvailabilityZone: se.AvailabilityZone.Name,
		}

		switch se.EndpointType {
		case endpointTypeIscsi:
			storageEndpointOut[optionIscsi] = parseStorageEndpointIscsi(se.Iscsi)
		case endpointTypeCbsAzureIscsi:
			storageEndpointOut[optionCbsAzureIscsi] = parseStorageEndpointCbsAzureIscsi(se.CbsAzureIscsi)
		}

		storageEndpointsList = append(storageEndpointsList, storageEndpointOut)
	}

	if err := d.Set(optionItems, storageEndpointsList); err != nil {
		return err
	}

	d.SetId(utilities.GetIdForDataSource())

	return nil
}
