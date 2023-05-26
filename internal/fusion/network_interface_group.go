/*
Copyright 2023 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package fusion

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/utilities"
	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"
)

const (
	networkInterfaceGroupGroupTypeETH = "eth"
)

type networkInterfaceGroupProvider struct {
	BaseResourceProvider
}

func schemaNetworkInterfaceGroup() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		optionName: {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "The name of the network interface group.",
		},
		optionDisplayName: {
			Type:         schema.TypeString,
			Optional:     true,
			Computed:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "The human name of the network interface group. If not provided, defaults to I(name).",
		},
		optionAvailabilityZone: {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "The name of the availability zone for the network interface group.",
		},
		optionRegion: {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "Region for the network interface group.",
		},
		optionGroupType: {
			Type:         schema.TypeString,
			Optional:     true,
			Default:      networkInterfaceGroupGroupTypeETH,
			ValidateFunc: validation.StringInSlice([]string{networkInterfaceGroupGroupTypeETH}, false),
			Description:  "The type of network interface group.",
		},
		optionGroupTypeEth: {
			Type:     schema.TypeList,
			Required: true,
			MaxItems: 1,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					optionGateway: {
						Type:             schema.TypeString,
						Required:         true,
						ValidateDiagFunc: IsValidAddress,
						Description:      "Address of the subnet gateway. Currently must be a valid IPv4 address.",
					},
					optionPrefix: {
						Type:             schema.TypeString,
						Required:         true,
						ValidateDiagFunc: IsValidPrefix,
						Description:      "Network prefix in CIDR notation. Required to create a new network interface group. Currently only IPv4 addresses with subnet mask are supported.",
					},
					optionMtu: {
						Type:        schema.TypeInt,
						Optional:    true,
						Default:     1500,
						Description: "MTU setting for the subnet.",
					},
				},
			},
		},
	}
}

func resourceNetworkInterfaceGroup() *schema.Resource {
	p := &networkInterfaceGroupProvider{BaseResourceProvider{ResourceKind: "NetworkInterfaceGroup"}}

	networkInterfaceGroupResourceFunctions := NewBaseResourceFunctions("NetworkInterfaceGroup", p)
	networkInterfaceGroupResourceFunctions.Resource.Schema = schemaNetworkInterfaceGroup()

	return networkInterfaceGroupResourceFunctions.Resource
}

func (p *networkInterfaceGroupProvider) PrepareCreate(ctx context.Context, d *schema.ResourceData) (InvokeWriteAPI, ResourcePost, error) {
	name := rdString(ctx, d, optionName)
	displayName := rdStringDefault(ctx, d, optionDisplayName, name)
	availabilityZone := rdString(ctx, d, optionAvailabilityZone)
	region := rdString(ctx, d, optionRegion)
	groupType := rdString(ctx, d, optionGroupType)

	body := hmrest.NetworkInterfaceGroupPost{
		Name:        name,
		DisplayName: displayName,
		GroupType:   groupType,
	}

	if groupType == networkInterfaceGroupGroupTypeETH {
		gateway := rdString(ctx, d, p.composeEthChildOptionName(optionGateway))
		prefix := rdString(ctx, d, p.composeEthChildOptionName(optionPrefix))
		mtu := rdInt(d, p.composeEthChildOptionName(optionMtu))

		if !utilities.IsAddressInPrefix(gateway, prefix) {
			return nil, nil, fmt.Errorf(`"gateway" must be an address in subnet "prefix"`)
		}

		body.Eth = &hmrest.NetworkInterfaceGroupEthPost{
			Prefix:  prefix,
			Gateway: gateway,
			Mtu:     int32(mtu),
		}
	}

	fn := func(ctx context.Context, client *hmrest.APIClient, body RequestSpec) (*hmrest.Operation, error) {
		op, _, err := client.NetworkInterfaceGroupsApi.CreateNetworkInterfaceGroup(ctx, *body.(*hmrest.NetworkInterfaceGroupPost), region, availabilityZone, nil)
		return &op, err
	}
	return fn, &body, nil
}

func (p *networkInterfaceGroupProvider) ReadResource(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) error {
	nig, _, err := client.NetworkInterfaceGroupsApi.GetNetworkInterfaceGroupById(ctx, d.Id(), nil)
	if err != nil {
		return err
	}

	d.Set(optionName, nig.Name)
	d.Set(optionDisplayName, nig.DisplayName)
	d.Set(optionAvailabilityZone, nig.AvailabilityZone.Name)
	d.Set(optionRegion, nig.Region.Name)
	d.Set(optionGroupType, nig.GroupType)
	d.Set(optionGroupTypeEth, []map[string]interface{}{{
		optionGateway: nig.Eth.Gateway,
		optionPrefix:  nig.Eth.Prefix,
		optionMtu:     nig.Eth.Mtu,
	}})

	return nil
}

func (p *networkInterfaceGroupProvider) PrepareDelete(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) (InvokeWriteAPI, error) {
	name := rdString(ctx, d, optionName)
	availabilityZone := rdString(ctx, d, optionAvailabilityZone)
	region := rdString(ctx, d, optionRegion)

	fn := func(ctx context.Context, client *hmrest.APIClient, body RequestSpec) (*hmrest.Operation, error) {
		op, _, err := client.NetworkInterfaceGroupsApi.DeleteNetworkInterfaceGroup(ctx, region, availabilityZone, name, nil)
		return &op, err
	}
	return fn, nil
}

func (p *networkInterfaceGroupProvider) PrepareUpdate(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) (InvokeWriteAPI, []ResourcePatch, error) {
	if d.HasChangeExcept(optionDisplayName) {
		d.Partial(true)
		return nil, nil, fmt.Errorf("attempting to update an immutable field")
	}

	name := rdString(ctx, d, optionName)
	displayName := rdStringDefault(ctx, d, optionDisplayName, name)
	availabilityZone := rdString(ctx, d, optionAvailabilityZone)
	region := rdString(ctx, d, optionRegion)

	tflog.Info(ctx, "Updating", optionDisplayName, displayName)
	patches := []ResourcePatch{
		&hmrest.NetworkInterfaceGroupPatch{
			DisplayName: &hmrest.NullableString{Value: displayName},
		},
	}

	fn := func(ctx context.Context, client *hmrest.APIClient, body RequestSpec) (*hmrest.Operation, error) {
		op, _, err := client.NetworkInterfaceGroupsApi.UpdateNetworkInterfaceGroup(ctx, *body.(*hmrest.NetworkInterfaceGroupPatch), region, availabilityZone, name, nil)
		return &op, err
	}
	return fn, patches, nil
}

func (p *networkInterfaceGroupProvider) composeEthChildOptionName(option string) string {
	return optionGroupTypeEth + ".0." + option
}
