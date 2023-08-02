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
type availabilityZoneDataSource struct{}

// This is our entry point for the Availability Zone data source
func dataSourceAvailabilityZone() *schema.Resource {
	ds := &availabilityZoneDataSource{}

	dsSchema := map[string]*schema.Schema{
		optionItems: {
			Type:     schema.TypeList,
			Computed: true,
			Elem: &schema.Resource{
				Schema: schemaAvailabilityZone(),
			},
			Description: "List matching Availability Zones.",
		},
		optionRegion: {
			Type:        schema.TypeString,
			Required:    true,
			Description: "Region name.",
		},
	}

	availabilityZoneDataSourceFunctions := NewBaseDataSourceFunctions(resourceKindAvailabilityZone, ds, dsSchema)

	return availabilityZoneDataSourceFunctions.Resource
}

func (ds *availabilityZoneDataSource) ReadDataSource(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) error {
	region := d.Get(optionRegion).(string)

	resp, _, err := client.AvailabilityZonesApi.ListAvailabilityZones(ctx, region, nil)
	if err != nil {
		return err
	}

	azList := make([]map[string]interface{}, 0, resp.Count)

	for _, az := range resp.Items {
		azList = append(azList, map[string]interface{}{
			optionName:        az.Name,
			optionDisplayName: az.DisplayName,
			optionRegion:      az.Region.Name,
		})
	}

	if err := d.Set(optionItems, azList); err != nil {
		return err
	}

	d.SetId(utilities.GetIdForDataSource())

	return nil
}
