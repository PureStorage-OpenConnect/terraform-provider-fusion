/*
Copyright 2023 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package fusion

import (
	"context"
	"fmt"

	"github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/utilities"
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
			Description:  "The name of the Storage Endpoint.",
		},
		optionDisplayName: {
			Type:         schema.TypeString,
			Optional:     true,
			Computed:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "The human-readable name of the Storage Endpoint. If not provided, defaults to I(name).",
		},
		optionRegion: {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "The name of the Region the Availability Zone is in.",
		},
		optionAvailabilityZone: {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "The name of the Availability Zone for the Storage Endpoint.",
		},
		optionIscsi: {
			Type:         schema.TypeList,
			Optional:     true,
			Description:  "iSCSI options.",
			ExactlyOneOf: []string{optionIscsi, optionCbsAzureIscsi},
			MaxItems:     1,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					optionDiscoveryInterfaces: {
						Type:        schema.TypeSet,
						Required:    true,
						Description: "List of discovery interfaces.",
						Elem: &schema.Resource{
							Schema: map[string]*schema.Schema{
								optionAddress: {
									Type:             schema.TypeString,
									Required:         true,
									ValidateDiagFunc: IsValidCidr,
									Description: "The IPv4 CIDR address to be used in the subnet of the Storage Endpoint." +
										" Only IPv4 is supported at the moment.",
								},
								optionGateway: {
									Type:             schema.TypeString,
									Optional:         true,
									ValidateDiagFunc: IsValidAddress,
									Description:      "The IPv4 address of the subnet gateway.",
								},
								optionNetworkInterfaceGroups: {
									Type:        schema.TypeSet,
									Optional:    true,
									Description: "The list of Network Interface Groups to assign to the address.",
									Elem: &schema.Schema{
										Type:         schema.TypeString,
										ValidateFunc: validation.StringIsNotEmpty,
										Description:  "The name of the Network Interface Group.",
									},
								},
							},
						},
					},
				},
			},
		},
		optionCbsAzureIscsi: {
			Type:         schema.TypeList,
			Optional:     true,
			Description:  "CBS Azure iSCSI.",
			ExactlyOneOf: []string{optionIscsi, optionCbsAzureIscsi},
			MaxItems:     1,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					optionStorageEndpointCollectionIdentity: {
						Type:         schema.TypeString,
						Required:     true,
						ValidateFunc: validation.StringIsNotEmpty,
						Description:  "The Storage Endpoint Collection Identity which belongs to the Azure entities.",
					},
					optionLoadBalancer: {
						Type:         schema.TypeString,
						Required:     true,
						ValidateFunc: validation.StringIsNotEmpty,
						Description:  "The Load Balancer id which gives permissions to CBS array appliations to modify the Load Balancer.",
					},
					optionLoadBalancerAddresses: {
						Type:        schema.TypeList,
						Required:    true,
						Description: "The list of the Load Balancer addresses.",
						MinItems:    cbsAzureIscsiLoadBalancerAddressesAmount,
						MaxItems:    cbsAzureIscsiLoadBalancerAddressesAmount,
						Elem: &schema.Schema{
							Type:             schema.TypeString,
							ValidateDiagFunc: IsValidAddress,
						},
					},
				},
			},
		},
	}
}

// This is our entry point for the Storage Endpoint resource.
func resourceStorageEndpoint() *schema.Resource {
	p := &storageEndpointProvider{BaseResourceProvider{ResourceKind: resourceKindStorageEndpoint}}
	storageEndpointResourceFunctions := NewBaseResourceFunctions(resourceKindStorageEndpoint, p)
	storageEndpointResourceFunctions.Resource.Description = "A Storage Endpoint provides access to storage in an Availability Zone."
	storageEndpointResourceFunctions.Resource.Schema = schemaStorageEndpoint()

	return storageEndpointResourceFunctions.Resource
}

func (p *storageEndpointProvider) PrepareCreate(ctx context.Context, d *schema.ResourceData) (InvokeWriteAPI, ResourcePost, error) {
	name := rdString(ctx, d, optionName)
	displayName := rdStringDefault(ctx, d, optionDisplayName, name)
	region := rdString(ctx, d, optionRegion)
	availabilityZone := rdString(ctx, d, optionAvailabilityZone)

	body := hmrest.StorageEndpointPost{
		Name:        name,
		DisplayName: displayName,
	}

	_, isCbsAzureIscsi := d.GetOk(optionCbsAzureIscsi)
	if isCbsAzureIscsi {
		body.EndpointType = endpointTypeCbsAzureIscsi
		body.CbsAzureIscsi = p.makeStorageEndpointCbsAzureIscsiPost(ctx, d)
	} else {
		body.EndpointType = endpointTypeIscsi
		body.Iscsi = p.makeStorageEndpointIscsiPost(d)
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

	return p.loadStorageEndpoint(se, d)

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

func (p *storageEndpointProvider) ImportResource(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) ([]*schema.ResourceData, error) {
	var orderedRequiredGroupNames = []string{
		resourceGroupNameRegion,
		resourceGroupNameAvailabilityZone,
		resourceGroupNameStorageEndpoint,
	}
	// The ID is user provided value - we expect self link
	selfLinkFieldsWithValues, err := utilities.ParseSelfLink(d.Id(), orderedRequiredGroupNames)
	if err != nil {
		return nil, fmt.Errorf("invalid storage_endpoint import path. Expected path in format '/regions/<region>/availability-zones/<availability-zone>/storage-endpoints/<storage-endpoint>'")
	}

	storageEndpoint, _, err := client.StorageEndpointsApi.GetStorageEndpoint(ctx, selfLinkFieldsWithValues[resourceGroupNameRegion], selfLinkFieldsWithValues[resourceGroupNameAvailabilityZone], selfLinkFieldsWithValues[resourceGroupNameStorageEndpoint], nil)
	if err != nil {
		utilities.TraceError(ctx, err)
		return nil, err
	}

	err = p.loadStorageEndpoint(storageEndpoint, d)
	if err != nil {
		return nil, err
	}

	d.SetId(storageEndpoint.Id)

	return []*schema.ResourceData{d}, nil
}

func (p *storageEndpointProvider) loadStorageEndpoint(se hmrest.StorageEndpoint, d *schema.ResourceData) error {
	err := getFirstError(
		d.Set(optionName, se.Name),
		d.Set(optionDisplayName, se.DisplayName),
		d.Set(optionRegion, se.Region.Name),
		d.Set(optionAvailabilityZone, se.AvailabilityZone.Name),
		d.Set(optionIscsi, nil),
		d.Set(optionCbsAzureIscsi, nil),
	)
	switch se.EndpointType {
	case endpointTypeIscsi:
		err = getFirstError(err, d.Set(optionIscsi, parseStorageEndpointIscsi(se.Iscsi)))
	case endpointTypeCbsAzureIscsi:
		err = getFirstError(err, d.Set(optionCbsAzureIscsi, parseStorageEndpointCbsAzureIscsi(se.CbsAzureIscsi)))
	}
	return err
}

func (p *storageEndpointProvider) makeStorageEndpointIscsiPost(d *schema.ResourceData) *hmrest.StorageEndpointIscsiPost {
	discoveryInterfaceSet := d.Get(p.composeIscsiChildOptionName(optionDiscoveryInterfaces)).(*schema.Set).List()
	discoveryInterfaces := make([]hmrest.StorageEndpointIscsiDiscoveryInterfacePost, len(discoveryInterfaceSet))

	for i, discoveryInterface := range discoveryInterfaceSet {
		discoveryInterfaceMap := discoveryInterface.(map[string]interface{})
		nigSet := discoveryInterfaceMap[optionNetworkInterfaceGroups].(*schema.Set)
		niGroups := make([]string, nigSet.Len())

		for i, group := range nigSet.List() {
			niGroups[i] = group.(string)
		}

		discoveryInterfacePost := hmrest.StorageEndpointIscsiDiscoveryInterfacePost{
			Address: discoveryInterfaceMap[optionAddress].(string),
		}

		if len(niGroups) != 0 {
			discoveryInterfacePost.NetworkInterfaceGroups = niGroups
		}

		if discoveryInterfaceMap[optionGateway] != nil {
			discoveryInterfacePost.Gateway = discoveryInterfaceMap[optionGateway].(string)
		}

		discoveryInterfaces[i] = discoveryInterfacePost
	}

	return &hmrest.StorageEndpointIscsiPost{
		DiscoveryInterfaces: discoveryInterfaces,
	}
}

func (p *storageEndpointProvider) makeStorageEndpointCbsAzureIscsiPost(ctx context.Context, d *schema.ResourceData) *hmrest.StorageEndpointCbsAzureIscsiPost {
	rawAddresses := d.Get(p.composeCbsAzureIscsiChildOptionName(optionLoadBalancerAddresses)).([]interface{})
	addresses := make([]string, 0, len(rawAddresses))
	for _, addr := range rawAddresses {
		addresses = append(addresses, addr.(string))
	}

	return &hmrest.StorageEndpointCbsAzureIscsiPost{
		StorageEndpointCollectionIdentity: rdString(ctx, d, p.composeCbsAzureIscsiChildOptionName(optionStorageEndpointCollectionIdentity)),
		LoadBalancer:                      rdString(ctx, d, p.composeCbsAzureIscsiChildOptionName(optionLoadBalancer)),
		LoadBalancerAddresses:             addresses,
	}
}

func (p *storageEndpointProvider) composeCbsAzureIscsiChildOptionName(option string) string {
	return fmt.Sprintf("%s.0.%s", optionCbsAzureIscsi, option)
}

func (p *storageEndpointProvider) composeIscsiChildOptionName(option string) string {
	return fmt.Sprintf("%s.0.%s", optionIscsi, option)
}

func parseStorageEndpointIscsi(data *hmrest.StorageEndpointIscsi) []map[string]interface{} {
	discoveryInterfaceSet := make([]interface{}, 0, len(data.DiscoveryInterfaces))
	for _, di := range data.DiscoveryInterfaces {
		discoveryInterface := map[string]interface{}{
			optionAddress: di.Address,
			optionGateway: di.Gateway,
		}

		if len(di.NetworkInterfaceGroups) > 0 {
			niGroups := make([]string, 0, len(di.NetworkInterfaceGroups))
			for _, nig := range di.NetworkInterfaceGroups {
				niGroups = append(niGroups, nig.Name)
			}

			discoveryInterface[optionNetworkInterfaceGroups] = niGroups
		}

		discoveryInterfaceSet = append(discoveryInterfaceSet, discoveryInterface)
	}

	return []map[string]interface{}{{
		optionDiscoveryInterfaces: discoveryInterfaceSet,
	}}
}

func parseStorageEndpointCbsAzureIscsi(data *hmrest.StorageEndpointCbsAzureIscsi) []map[string]interface{} {
	return []map[string]interface{}{{
		optionStorageEndpointCollectionIdentity: data.StorageEndpointCollectionIdentity,
		optionLoadBalancer:                      data.LoadBalancer,
		optionLoadBalancerAddresses:             data.LoadBalancerAddresses,
	}}
}
