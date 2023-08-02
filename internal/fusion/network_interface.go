/*
Copyright 2023 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package fusion

import (
	"context"
	"fmt"
	"strconv"

	"github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/utilities"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"
)

// these are network interface-specific subpaths
var networkInterfaceWwnKey = fmt.Sprintf("%s.0.%s", optionFc, optionWwn)
var networkInterfaceAddressKey = fmt.Sprintf("%s.0.%s", optionEth, optionAddress)

type networkInterfaceProvider struct {
	BaseResourceProvider
}

func schemaNetworkInterface() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		optionName: {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "The name of the Network Interface.",
		},
		optionDisplayName: {
			Type:         schema.TypeString,
			Optional:     true,
			Computed:     true,
			ValidateFunc: validation.StringLenBetween(1, maxDisplayName),
			Description:  "The human-readable name of the Network Interface. If not provided, defaults to I(name).",
		},
		optionRegion: {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "The name of Region of the array to which the Network Interface is attached.",
		},
		optionAvailabilityZone: {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "The name of Availability Zone of the Array to which the Network Interface is attached.",
		},
		optionArray: {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "The name of the Array to which the Network Interface is attached.",
		},
		optionServices: {
			Type:     schema.TypeSet,
			Computed: true,
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
			Description: "List of services provided by this Network Interface.",
		},
		optionEnabled: {
			Type:        schema.TypeBool,
			Required:    true,
			Description: "Whether the Network Interface is in use.",
		},
		optionMaxSpeed: {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The configured speed of the Network Interface. Typically maximum speed of underlying hardware. Measured in bits per second.",
		},
		optionNetworkInterfaceGroup: {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The name of Network Interface Group assigned to the Network Interface.",
		},
		optionInterfaceType: {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringInSlice(interfaceTypes, false),
			Description:  "The hardware type of the Network Interface.",
		},
		optionEth: {
			Type:         schema.TypeList,
			Optional:     true,
			ExactlyOneOf: []string{optionEth, optionFc},
			MaxItems:     1,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					optionAddress: {
						Type:             schema.TypeString,
						Required:         true,
						ValidateDiagFunc: IsValidOptionalCidr,
						Description:      "The address with subnet in CIDR notation. Currently only IPv4 addresses with subnet mask are supported.",
					},
					optionGateway: {
						Type:        schema.TypeString,
						Computed:    true,
						Description: fmt.Sprintf("Address of '%s' subnet gateway.", optionAddress),
					},
					optionMtu: {
						Type:        schema.TypeInt,
						Computed:    true,
						Description: "MTU of the subnet.",
					},
					optionVlan: {
						Type:        schema.TypeInt,
						Computed:    true,
						Description: "VLAN ID assigned to the Network Interface.",
					},
					optionMac: {
						Type:        schema.TypeString,
						Computed:    true,
						Description: "MAC Address of the underlying hardware device.",
					},
				},
			},
		},
		optionFc: {
			Type:         schema.TypeList,
			Optional:     true,
			ExactlyOneOf: []string{optionEth, optionFc},
			MaxItems:     1,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					optionWwn: {
						Type:     schema.TypeString,
						Required: true,
						// TODO add validation once backend stops ignoring this field
						Description: "FC WWN (World Wide Name) of the underlying Fibre Channel port.",
					},
				},
			},
		},
	}
}

func resourceNetworkInterface() *schema.Resource {
	p := &networkInterfaceProvider{BaseResourceProvider{ResourceKind: "NetworkInterface"}}

	networkInterfaceResourceFunctions := NewBaseResourceFunctions("NetworkInterface", p)
	networkInterfaceResourceFunctions.Resource.Description = "A Network Interface of an Array for use by Pure Fusion."
	networkInterfaceResourceFunctions.Resource.Schema = schemaNetworkInterface()

	return networkInterfaceResourceFunctions.Resource
}

func (p *networkInterfaceProvider) PrepareCreate(ctx context.Context, d *schema.ResourceData) (InvokeWriteAPI, ResourcePost, error) {
	// TODO: Right now there is no way to directly emit diagnostics without significant refactoring.
	// Once `ResourceProvider` is refactored, emit warning about the network interface not really
	// being created as its lifecycle is directly tied to the array.

	name := rdString(ctx, d, optionName)
	tflog.Warn(ctx, "Network Interface cannot be really created as it is tied to its array and must already exist to modify it", "network_interface", name)

	fn := func(ctx context.Context, client *hmrest.APIClient, body RequestSpec) (*hmrest.Operation, error) {
		return patchNetworkInterface(ctx, client, d)
	}
	return fn, nil, nil
}

func (p *networkInterfaceProvider) ReadResource(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) error {
	ni, _, err := client.NetworkInterfacesApi.GetNetworkInterfaceById(ctx, d.Id(), nil)
	if err != nil {
		return err
	}

	return p.loadNetworkInterface(ni, d)
}

func (p *networkInterfaceProvider) PrepareDelete(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) (InvokeWriteAPI, error) {
	// TODO: Right now there is no way to directly emit diagnostics without significant refactoring.
	// Once `ResourceProvider` is refactored, emit warning about the network interface not really
	// being destroyed as its lifecycle is directly tied to the array.
	name := rdString(ctx, d, optionName)
	tflog.Warn(ctx, "Network Interface is not really being destroyed and will be only forgotten by Terraform as it is tied to its array", "network_interface", name)

	return DummyInvokeWriteAPI, nil
}

func (p *networkInterfaceProvider) PrepareUpdate(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) (InvokeWriteAPI, []ResourcePatch, error) {
	if err := utilities.CheckImmutableFieldsExcept(ctx, d, optionDisplayName, optionEnabled, optionNetworkInterfaceGroup, optionEth, optionFc); err != nil {
		d.Partial(true)
		return nil, nil, err
	}
	_, err := patchNetworkInterface(ctx, client, d)
	if err != nil {
		return nil, nil, err
	}
	return DummyInvokeWriteAPI, []ResourcePatch{}, nil
}

func (p *networkInterfaceProvider) ImportResource(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) ([]*schema.ResourceData, error) {
	var orderedRequiredGroupNames = []string{
		resourceGroupNameRegion,
		resourceGroupNameAvailabilityZone,
		resourceGroupNameArray,
		resourceGroupNameNetworkInterface,
	}
	// The ID is user provided value - we expect self link
	selfLinkFieldsWithValues, err := utilities.ParseSelfLink(d.Id(), orderedRequiredGroupNames)
	if err != nil {
		return nil, fmt.Errorf("invalid interface import path. Expected path in format '/regions/<region>/availability-zones/<availability-zone>/arrays/<array>/network-interfaces/<network-interface>'")
	}

	networkInterface, _, err := client.NetworkInterfacesApi.GetNetworkInterface(ctx, selfLinkFieldsWithValues[resourceGroupNameRegion], selfLinkFieldsWithValues[resourceGroupNameAvailabilityZone], selfLinkFieldsWithValues[resourceGroupNameArray], selfLinkFieldsWithValues[resourceGroupNameNetworkInterface], nil)
	if err != nil {
		utilities.TraceError(ctx, err)
		return nil, err
	}

	err = p.loadNetworkInterface(networkInterface, d)
	if err != nil {
		return nil, err
	}

	d.SetId(networkInterface.Id)

	return []*schema.ResourceData{d}, nil
}

func (p *networkInterfaceProvider) loadNetworkInterface(ni hmrest.NetworkInterface, d *schema.ResourceData) error {
	err := getFirstError(
		d.Set(optionName, ni.Name),
		d.Set(optionDisplayName, ni.DisplayName),
		d.Set(optionRegion, ni.Region.Name),
		d.Set(optionAvailabilityZone, ni.AvailabilityZone.Name),
		d.Set(optionArray, ni.Array.Name),
		d.Set(optionEnabled, ni.Enabled),
		d.Set(optionMaxSpeed, strconv.FormatInt(ni.MaxSpeed, 10)),
		d.Set(optionInterfaceType, ni.InterfaceType),
		d.Set(optionEth, nil),
		d.Set(optionFc, nil),
		d.Set(optionNetworkInterfaceGroup, nil),
	)
	if ni.Services == nil {
		ni.Services = []string{}
	}
	err = getFirstError(err, d.Set(optionServices, ni.Services))
	if ni.NetworkInterfaceGroup != nil {
		err = getFirstError(err, d.Set(optionNetworkInterfaceGroup, ni.NetworkInterfaceGroup.Name))
	}
	switch ni.InterfaceType {
	case optionEth:
		err = getFirstError(err,
			d.Set(optionEth, []map[string]interface{}{{
				optionAddress: ni.Eth.Address,
				optionGateway: ni.Eth.Gateway,
				optionMtu:     ni.Eth.Mtu,
				optionVlan:    ni.Eth.Vlan,
				optionMac:     ni.Eth.MacAddress,
			}}),
		)
	case optionFc:
		err = getFirstError(err,
			d.Set(optionFc, []map[string]interface{}{{
				optionWwn: ni.Fc.Wwn,
			}}))
	}
	return err
}

func patchNetworkInterface(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) (op *hmrest.Operation, err error) {
	region := rdString(ctx, d, optionRegion)
	availabilityZone := rdString(ctx, d, optionAvailabilityZone)
	array := rdString(ctx, d, optionArray)
	name := rdString(ctx, d, optionName)

	ni, _, err := client.NetworkInterfacesApi.GetNetworkInterface(ctx, region, availabilityZone, array, name, nil)
	if err != nil {
		tflog.Error(ctx, "network interface does not exist", optionRegion, region, optionAvailabilityZone, availabilityZone, optionArray, array, "network_interface", name, "error", err)
		return nil, fmt.Errorf("network interface %s does not exist and cannot be created by Terraform: %w", name, err)
	}

	if err := validateAdditionalNetworkInterfaceReqs(ctx, d, ni); err != nil {
		return nil, err
	}

	var patch hmrest.NetworkInterfacePatch
	hasChanges := false

	displayName := rdStringDefault(ctx, d, optionDisplayName, name)
	if ni.DisplayName != displayName {
		tflog.Debug(ctx, "Updating network interface", "network_interface", name, optionDisplayName, displayName)
		patch.DisplayName = &hmrest.NullableString{Value: displayName}
		hasChanges = true
	}

	enabled := d.Get(optionEnabled).(bool)
	if ni.Enabled != enabled {
		tflog.Debug(ctx, "Updating network interface", "network_interface", name, optionEnabled, enabled)
		patch.Enabled = &hmrest.NullableBoolean{Value: enabled}
		hasChanges = true
	}

	niGroup := ""
	if ni.NetworkInterfaceGroup != nil {
		niGroup = ni.NetworkInterfaceGroup.Name
	}

	group := rdString(ctx, d, optionNetworkInterfaceGroup)
	if niGroup != group {
		tflog.Debug(ctx, "Updating network interface", "network_interface", name, optionNetworkInterfaceGroup, group)
		patch.NetworkInterfaceGroup = &hmrest.NullableString{
			Value: group,
		}
		hasChanges = true
	}

	switch ni.InterfaceType {
	case optionEth:
		// Ethernet
		patch.Eth = &hmrest.NetworkInterfacePatchEth{}
		address := rdString(ctx, d, networkInterfaceAddressKey)
		addressChanged := address != ni.Eth.Address
		if addressChanged {
			tflog.Debug(ctx, "Updating network interface", "network_interface", name, optionAddress, address)
			patch.Eth.Address = &hmrest.NullableString{
				Value: address,
			}
			hasChanges = true
		}
		// XXX: FCs are not supported yet
	}

	if hasChanges {
		tflog.Debug(ctx, "Patching network_interface", "network_interface", name, "patch", patch)
		op, _, err := client.NetworkInterfacesApi.UpdateNetworkInterface(ctx, patch, region, availabilityZone, array, name, nil)
		if err != nil {
			tflog.Error(ctx, "failed to patch network interface", "network_interface", ni.Name, "error", err)
			d.Partial(true)
			return &op, fmt.Errorf("failed to patch network interface '%s': %w", ni.Name, err)
		}
		// await the update here and return fake op instead since BaseResourceProvider.PrepareCreate() expects Create() op which can be a bit different
		utilities.TraceOperation(ctx, &op, "Patching Network Interface")

		succeeded, err := utilities.WaitOnOperation(ctx, &op, client)
		if err != nil {
			d.Partial(true)
			return &op, err
		}
		if !succeeded {
			d.Partial(true)
			return &op, fmt.Errorf("failed to patch Network Interface '%s': %s (%s)", ni.Name, err, op.Error_.Message)
		}
	}

	return MakeDummyCreateOperation(ni.Id), nil
}

func validateAdditionalNetworkInterfaceReqs(ctx context.Context, d *schema.ResourceData, ni hmrest.NetworkInterface) error {
	nifg := d.Get(optionNetworkInterfaceGroup)
	addr := d.Get(networkInterfaceAddressKey)
	wwn := d.Get(networkInterfaceWwnKey)
	niType := d.Get(optionInterfaceType)
	nifgEmpty := nifg == ""
	addrEmpty := addr == ""
	if niType != ni.InterfaceType {
		d.Partial(true)
		tflog.Error(ctx, "network interface is not of expected type", "network_interface", ni.Name, "config", niType, "remote", ni.InterfaceType)
		return fmt.Errorf("network interface '%v' has type mismatch, config: %v, remote: %v", ni.Name, niType, ni.InterfaceType)
	}
	if !nifgEmpty && niType != optionEth {
		d.Partial(true)
		tflog.Error(ctx, "Network Interface field 'network_interface_group' can be set only on interfaces of type 'eth'", "network_interface", ni.Name, "network_interface_type", ni.InterfaceType)
		return fmt.Errorf("network interface '%s' of type '%s' cannot have field 'network_interface_group', that can be set only for 'eth' interfaces", ni.Name, ni.InterfaceType)
	}
	if nifgEmpty != addrEmpty {
		d.Partial(true)
		tflog.Error(ctx, "Network Interface must have 'eth.address' and 'network_interface_group' set or cleared together", "network_interface", ni.Name)
		return fmt.Errorf("network interface '%s' must have 'eth.address' and 'network_interface_group' set or cleared together", ni.Name)
	}
	if niType == optionFc && ni.Fc.Wwn != wwn {
		d.Partial(true)
		tflog.Error(ctx, "Fiber Channel Network Interfaces are read only right now, to specify them in config, you must match their WWN", "network_interface", ni.Name, "wwn_in_config", wwn, "wwn_in_interface", ni.Fc.Wwn)
		return fmt.Errorf(" Fiber Channel Network Interface '%s' is read-only, its WWN must be '%s', but is set to '%s'", ni.Name, ni.Fc.Wwn, wwn)
	}

	return nil
}
