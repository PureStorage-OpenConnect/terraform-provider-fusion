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
type regionDataSource struct{}

// This is our entry point for the Region data source
func dataSourceRegion() *schema.Resource {
	ds := &regionDataSource{}

	dsSchema := map[string]*schema.Schema{
		optionItems: {
			Type:     schema.TypeList,
			Computed: true,
			Elem: &schema.Resource{
				Schema: schemaRegion(),
			},
		},
	}

	regionDataSourceFunctions := NewBaseDataSourceFunctions("Region", ds, dsSchema)

	return regionDataSourceFunctions.Resource
}

func (ds *regionDataSource) ReadDataSource(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) error {
	resp, _, err := client.RegionsApi.ListRegions(ctx, nil)
	if err != nil {
		return err
	}

	regionList := make([]map[string]interface{}, 0, resp.Count)

	for _, region := range resp.Items {
		regionList = append(regionList, map[string]interface{}{
			optionName:        region.Name,
			optionDisplayName: region.DisplayName,
		})
	}

	if err := d.Set(optionItems, regionList); err != nil {
		return err
	}

	d.SetId(utilities.GetIdForDataSource())

	return nil
}
