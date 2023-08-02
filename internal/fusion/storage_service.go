/*
Copyright 2022 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package fusion

import (
	context "context"
	"fmt"

	"github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/utilities"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"
)

var storageServiceResourceFunctions *BaseResourceFunctions

// Implements ResourceProvider
type storageServiceProvider struct {
	BaseResourceProvider
}

func schemaStorageService() map[string]*schema.Schema {
	hardwareTypes := []string{
		"flash-array-x", "flash-array-c", "flash-array-x-optane", "flash-array-xl",
	}

	return map[string]*schema.Schema{
		optionName: {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "The name of the Storage Service.",
		},
		optionDisplayName: {
			Type:         schema.TypeString,
			Optional:     true,
			Computed:     true,
			ValidateFunc: validation.StringLenBetween(1, maxDisplayName),
			Description:  "The human-readable name of the Storage Service. If not provided, defaults to I(name).",
		},
		optionHardwareTypes: {
			Type:     schema.TypeSet,
			Required: true,
			MinItems: 1,
			Elem: &schema.Schema{
				Type: schema.TypeString,
				ValidateFunc: validation.All(
					validation.StringInSlice(hardwareTypes, false),
					validation.StringIsNotEmpty,
				),
			},
			Description: "Hardware types to which the Storage Service applies.",
		},
	}
}

// This is our entry point for the Storage Service resource
func resourceStorageService() *schema.Resource {
	p := &storageServiceProvider{BaseResourceProvider{ResourceKind: resourceKindStorageService}}
	storageServiceResourceFunctions = NewBaseResourceFunctions(resourceKindStorageService, p)
	storageServiceResourceFunctions.Resource.Description = "A Storage Service represents a type of storage that shares " +
		"fundamental characteristics like response latency, availability, protocol, and data management features." +
		" Placement Groups select a Storage Service to guarantee consistent snapshots and hardware affinity expectations."
	storageServiceResourceFunctions.Resource.Schema = schemaStorageService()

	return storageServiceResourceFunctions.Resource
}

func (vp *storageServiceProvider) PrepareCreate(ctx context.Context, d *schema.ResourceData) (InvokeWriteAPI, ResourcePost, error) {
	name := rdString(ctx, d, "name")
	hardwareTypes := rdStringSet(ctx, d, "hardware_types")
	displayName := rdStringDefault(ctx, d, "display_name", name)

	body := hmrest.StorageServicePost{
		Name:          name,
		DisplayName:   displayName,
		HardwareTypes: hardwareTypes,
	}

	fn := func(ctx context.Context, client *hmrest.APIClient, body RequestSpec) (*hmrest.Operation, error) {
		op, _, err := client.StorageServicesApi.CreateStorageService(ctx, *body.(*hmrest.StorageServicePost), nil)
		return &op, err
	}
	return fn, &body, nil
}

func (vp *storageServiceProvider) ReadResource(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) error {
	ss, _, err := client.StorageServicesApi.GetStorageServiceById(ctx, d.Id(), nil)
	if err != nil {
		return err
	}

	return vp.loadStorageService(ss, d)
}

func (vp *storageServiceProvider) PrepareDelete(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) (InvokeWriteAPI, error) {
	storageServiceName := rdString(ctx, d, "name")

	fn := func(ctx context.Context, client *hmrest.APIClient, body RequestSpec) (*hmrest.Operation, error) {
		op, _, err := client.StorageServicesApi.DeleteStorageService(ctx, storageServiceName, nil)
		return &op, err
	}
	return fn, nil
}

func (vp *storageServiceProvider) PrepareUpdate(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) (InvokeWriteAPI, []ResourcePatch, error) {
	var patches []ResourcePatch
	storageServiceName := rdString(ctx, d, "name")

	if d.HasChangeExcept("display_name") {
		d.Partial(true) // Do not persist changes to "name" etc.
		return nil, nil, fmt.Errorf("attempting to update an immutable field")
	} else {
		displayName := rdStringDefault(ctx, d, "display_name", storageServiceName)
		tflog.Info(ctx, "Updating", "display_name", displayName)
		patches = append(patches, &hmrest.StorageServicePatch{
			DisplayName: &hmrest.NullableString{Value: displayName},
		})
	}

	fn := func(ctx context.Context, client *hmrest.APIClient, body RequestSpec) (*hmrest.Operation, error) {
		op, _, err := client.StorageServicesApi.UpdateStorageService(ctx, *body.(*hmrest.StorageServicePatch), storageServiceName, nil)
		return &op, err
	}

	return fn, patches, nil
}

func (vp *storageServiceProvider) ImportResource(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) ([]*schema.ResourceData, error) {
	var orderedRequiredGroupNames = []string{resourceGroupNameStorageService}
	// The ID is user provided value - we expect self link
	selfLinkFieldsWithValues, err := utilities.ParseSelfLink(d.Id(), orderedRequiredGroupNames)
	if err != nil {
		return nil, fmt.Errorf("invalid storage_service import path. Expected path in format '/storage-services/<storage-service>'")
	}

	storageService, _, err := client.StorageServicesApi.GetStorageService(ctx, selfLinkFieldsWithValues[resourceGroupNameStorageService], nil)
	if err != nil {
		utilities.TraceError(ctx, err)
		return nil, err
	}

	err = vp.loadStorageService(storageService, d)
	if err != nil {
		return nil, err
	}

	d.SetId(storageService.Id)

	return []*schema.ResourceData{d}, nil
}

func (vp *storageServiceProvider) loadStorageService(ss hmrest.StorageService, d *schema.ResourceData) error {
	hardwareTypes := make([]string, len(ss.HardwareTypes))
	for i, hwType := range ss.HardwareTypes {
		hardwareTypes[i] = hwType.Name
	}
	return getFirstError(
		d.Set(optionName, ss.Name),
		d.Set(optionDisplayName, ss.DisplayName),
		d.Set(optionHardwareTypes, hardwareTypes),
	)
}
