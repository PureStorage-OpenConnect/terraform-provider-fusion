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
type networkInterfaceGroupDataSource struct{}

// This is our entry point for the Network Interface Group data source
func dataSourceNetworkInterfaceGroup() *schema.Resource {
	ds := &networkInterfaceGroupDataSource{}

	dsSchema := map[string]*schema.Schema{
		optionAvailabilityZone: {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "The name of the Availability Zone for the Network Interface Group.",
		},
		optionRegion: {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "The name of Region for the Network Interface Group.",
		},
		optionItems: {
			Type:     schema.TypeList,
			Computed: true,
			Elem: &schema.Resource{
				Schema: schemaNetworkInterfaceGroup(),
			},
			Description: "List of matching Network Interface Groups.",
		},
	}

	networkInterfaceGroupDataSourceFunctions := NewBaseDataSourceFunctions("NetworkInterfaceGroup", ds, dsSchema)

	return networkInterfaceGroupDataSourceFunctions.Resource
}

func (ds *networkInterfaceGroupDataSource) ReadDataSource(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) error {
	region := rdString(ctx, d, optionRegion)
	availabilityZone := rdString(ctx, d, optionAvailabilityZone)

	resp, _, err := client.NetworkInterfaceGroupsApi.ListNetworkInterfaceGroups(ctx, region, availabilityZone, nil)
	if err != nil {
		return err
	}

	networkInterfaceGroupsList := make([]map[string]interface{}, 0, resp.Count)

	for _, nig := range resp.Items {
		networkInterfaceGroupsList = append(networkInterfaceGroupsList, map[string]interface{}{
			optionName:             nig.Name,
			optionDisplayName:      nig.DisplayName,
			optionAvailabilityZone: nig.AvailabilityZone.Name,
			optionRegion:           nig.Region.Name,
			optionGroupType:        nig.GroupType,
			optionEth: []map[string]interface{}{{
				optionGateway: nig.Eth.Gateway,
				optionPrefix:  nig.Eth.Prefix,
				optionMtu:     nig.Eth.Mtu,
			}},
		})
	}

	if err := d.Set(optionItems, networkInterfaceGroupsList); err != nil {
		return err
	}

	d.SetId(utilities.GetIdForDataSource())

	return nil
}
