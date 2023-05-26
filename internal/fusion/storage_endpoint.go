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

	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"
)

// Implements ResourceProvider
type storageEndpointProvider struct {
	BaseResourceProvider
}

func schemaStorageEndpoint() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		optionName: {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "The name of the storage endpoint.",
		},
		optionDisplayName: {
			Type:         schema.TypeString,
			Optional:     true,
			Computed:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "The human name of the storage endpoint. If not provided, defaults to I(name).",
		},
		optionRegion: {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "The name of the region the availability zone is in.",
		},
		optionAvailabilityZone: {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "The name of the availability zone for the storage endpoint.",
		},
		optionIscsi: {
			Type:        schema.TypeSet,
			Required:    true,
			Description: "List of discovery interfaces.",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					optionAddress: {
						Type:             schema.TypeString,
						Required:         true,
						ValidateDiagFunc: IsValidPrefix,
						Description: "IP address to be used in the subnet of the storage endpoint." +
							" IP address must include a CIDR notation." +
							" Only IPv4 is supported at the moment.",
					},
					optionGateway: {
						Type:             schema.TypeString,
						Optional:         true,
						ValidateDiagFunc: IsValidAddress,
						Description:      "Address of the subnet gateway.",
					},
					optionNetworkInterfaceGroups: {
						Type:        schema.TypeSet,
						Optional:    true,
						Description: "List of network interface groups to assign to the address.",
						Elem: &schema.Schema{
							Type:         schema.TypeString,
							ValidateFunc: validation.StringIsNotEmpty,
							Description:  "The name of the network interface group.",
						},
					},
				},
			},
		},
	}
}

// This is our entry point for the Storage Endpoint resource.
func resourceStorageEndpoint() *schema.Resource {
	p := &storageEndpointProvider{BaseResourceProvider{ResourceKind: "StorageEndpoint"}}
	storageEndpointResourceFunctions := NewBaseResourceFunctions("StorageEndpoint", p)
	storageEndpointResourceFunctions.Resource.Schema = schemaStorageEndpoint()

	return storageEndpointResourceFunctions.Resource
}

func (p *storageEndpointProvider) PrepareCreate(ctx context.Context, d *schema.ResourceData) (InvokeWriteAPI, ResourcePost, error) {
	name := rdString(ctx, d, optionName)
	displayName := rdStringDefault(ctx, d, optionDisplayName, name)
	region := rdString(ctx, d, optionRegion)
	availabilityZone := rdString(ctx, d, optionAvailabilityZone)

	iscsiSet := d.Get(optionIscsi).(*schema.Set)
	discoveryInterfaces := make([]hmrest.StorageEndpointIscsiDiscoveryInterfacePost, iscsiSet.Len())

	for i, iscsi := range iscsiSet.List() {
		iscsiMap := iscsi.(map[string]interface{})
		nigSet := iscsiMap[optionNetworkInterfaceGroups].(*schema.Set)
		niGroups := make([]string, nigSet.Len())

		for i, group := range nigSet.List() {
			niGroups[i] = group.(string)
		}

		discoveryInterfacePost := hmrest.StorageEndpointIscsiDiscoveryInterfacePost{
			Address: iscsiMap[optionAddress].(string),
		}

		if len(niGroups) != 0 {
			discoveryInterfacePost.NetworkInterfaceGroups = niGroups
		}

		if iscsiMap[optionGateway] != nil {
			discoveryInterfacePost.Gateway = iscsiMap[optionGateway].(string)
		}

		discoveryInterfaces[i] = discoveryInterfacePost
	}

	// Currently there is only one endpoint type, which is implied by the presence of the iscsi field
	endpointType := optionIscsi

	body := hmrest.StorageEndpointPost{
		Name:         name,
		DisplayName:  displayName,
		EndpointType: endpointType,
		Iscsi: &hmrest.StorageEndpointIscsiPost{
			DiscoveryInterfaces: discoveryInterfaces,
		},
	}

	fn := func(ctx context.Context, client *hmrest.APIClient, body RequestSpec) (*hmrest.Operation, error) {
		op, _, err := client.StorageEndpointsApi.CreateStorageEndpoint(
			ctx, *body.(*hmrest.StorageEndpointPost), region, availabilityZone, nil,
		)

		return &op, err
	}
	return fn, &body, nil
}

func (p *storageEndpointProvider) ReadResource(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) error {
	se, _, err := client.StorageEndpointsApi.GetStorageEndpointById(ctx, d.Id(), nil)
	if err != nil {
		return err
	}

	iscsiSet := make([]interface{}, 0)
	for _, discoveryInterface := range se.Iscsi.DiscoveryInterfaces {
		niGroups := make([]string, 0)

		for _, niGroup := range discoveryInterface.NetworkInterfaceGroups {
			niGroups = append(niGroups, niGroup.Name)
		}

		iscsi := map[string]interface{}{
			optionAddress:                discoveryInterface.Address,
			optionGateway:                discoveryInterface.Gateway,
			optionNetworkInterfaceGroups: niGroups,
		}

		iscsiSet = append(iscsiSet, iscsi)
	}

	d.Set(optionName, se.Name)
	d.Set(optionDisplayName, se.DisplayName)
	d.Set(optionRegion, se.Region.Name)
	d.Set(optionAvailabilityZone, se.AvailabilityZone.Name)
	d.Set(optionIscsi, iscsiSet)

	return nil
}

func (p *storageEndpointProvider) PrepareDelete(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) (InvokeWriteAPI, error) {
	name := rdString(ctx, d, optionName)
	region := rdString(ctx, d, optionRegion)
	availabilityZone := rdString(ctx, d, optionAvailabilityZone)

	fn := func(ctx context.Context, client *hmrest.APIClient, body RequestSpec) (*hmrest.Operation, error) {
		op, _, err := client.StorageEndpointsApi.DeleteStorageEndpoint(ctx, region, availabilityZone, name, nil)
		return &op, err
	}
	return fn, nil
}

func (p *storageEndpointProvider) PrepareUpdate(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) (InvokeWriteAPI, []ResourcePatch, error) {
	var patches []ResourcePatch

	name := rdString(ctx, d, optionName)
	region := rdString(ctx, d, optionRegion)
	availabilityZone := rdString(ctx, d, optionAvailabilityZone)

	if d.HasChangeExcept(optionDisplayName) {
		d.Partial(true)

		// Set iscsi to its previous value (Partial does not work with sets)
		iscsi, _ := d.GetChange(optionIscsi)
		d.Set(optionIscsi, iscsi)

		return nil, nil, fmt.Errorf("attempting to update an immutable field")
	}

	displayName := rdStringDefault(ctx, d, optionDisplayName, name)
	tflog.Info(ctx, "Updating", "display_name", displayName)
	patches = append(patches, &hmrest.StorageEndpointPatch{
		DisplayName: &hmrest.NullableString{Value: displayName},
	})

	fn := func(ctx context.Context, client *hmrest.APIClient, body RequestSpec) (*hmrest.Operation, error) {
		op, _, err := client.StorageEndpointsApi.UpdateStorageEndpoint(ctx, *body.(*hmrest.StorageEndpointPatch), region, availabilityZone, name, nil)
		return &op, err
	}

	return fn, patches, nil
}
