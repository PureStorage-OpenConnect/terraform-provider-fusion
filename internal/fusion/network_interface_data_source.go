/*
Copyright 2023 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package fusion

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/utilities"
	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"
)

type networkInterfaceDataSource struct{}

func dataSourceNetworkInterface() *schema.Resource {
	ds := &networkInterfaceDataSource{}

	niSchema := schemaNetworkInterface()
	// ExactlyOneOf and similar cross-field references break when schema gets nested
	// because the referenced paths change and they shouldn't be needed in data source
	// because data source does not work with direct user input
	niSchema[optionEth].ExactlyOneOf = nil
	niSchema[optionFc].ExactlyOneOf = nil

	dsSchema := map[string]*schema.Schema{
		optionRegion: {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "Region of the array to which the Network Interface is attached.",
		},
		optionAvailabilityZone: {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "Availability zone of the array to which the Network Interface is attached.",
		},
		optionArray: {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "Name of the array to which the Network Interface is attached.",
		},
		optionItems: {
			Type:     schema.TypeList,
			Computed: true,
			Elem: &schema.Resource{
				Schema: niSchema,
			},
			Description: "List of matching Network Interfaces.",
		},
	}

	networkInterfaceDataSourceFunctions := NewBaseDataSourceFunctions("NetworkInterface", ds, dsSchema)

	return networkInterfaceDataSourceFunctions.Resource
}

func (ds *networkInterfaceDataSource) ReadDataSource(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) error {
	region := rdString(ctx, d, optionRegion)
	availabilityZone := rdString(ctx, d, optionAvailabilityZone)
	array := rdString(ctx, d, optionArray)
	resp, _, err := client.NetworkInterfacesApi.ListNetworkInterfaces(ctx, region, availabilityZone, array, nil)
	if err != nil {
		return err
	}

	networkInterfacesList := make([]map[string]interface{}, 0, resp.Count)

	for _, ni := range resp.Items {
		services := ni.Services
		if ni.Services == nil {
			services = []string{}
		}
		iface := map[string]interface{}{
			optionName:             ni.Name,
			optionDisplayName:      ni.DisplayName,
			optionRegion:           ni.Region.Name,
			optionAvailabilityZone: ni.AvailabilityZone.Name,
			optionArray:            ni.Array.Name,
			optionInterfaceType:    ni.InterfaceType,
			optionServices:         services,
			optionEnabled:          ni.Enabled,
			optionMaxSpeed:         strconv.FormatInt(ni.MaxSpeed, 10),
		}
		if ni.NetworkInterfaceGroup != nil {
			iface[optionNetworkInterfaceGroup] = ni.NetworkInterfaceGroup.Name
		}
		switch ni.InterfaceType {
		case optionEth:
			iface[optionEth] = []map[string]interface{}{{
				optionAddress: ni.Eth.Address,
				optionGateway: ni.Eth.Gateway,
				optionMtu:     ni.Eth.Mtu,
				optionVlan:    ni.Eth.Vlan,
				optionMac:     ni.Eth.MacAddress,
			}}
		case optionFc:
			iface[optionFc] = []map[string]interface{}{{
				optionWwn: ni.Fc.Wwn,
			}}
		}

		networkInterfacesList = append(networkInterfacesList, iface)
	}

	if err := d.Set(optionItems, networkInterfacesList); err != nil {
		return err
	}

	d.SetId(utilities.GetIdForDataSource())

	return nil
}
