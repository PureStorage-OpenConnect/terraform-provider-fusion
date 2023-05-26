/*
Copyright 2022 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package fusion

import (
	context "context"
	"fmt"

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
		"name": {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "The name of the storage service.",
		},
		"display_name": {
			Type:         schema.TypeString,
			Optional:     true,
			Computed:     true,
			ValidateFunc: validation.StringLenBetween(1, maxDisplayName),
			Description:  "The human name of the storage service. If not provided, defaults to I(name).",
		},
		"hardware_types": {
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
			Description: "Hardware types to which the storage service applies.",
		},
	}
}

// This is our entry point for the Storage Service resource
func resourceStorageService() *schema.Resource {
	p := &storageServiceProvider{BaseResourceProvider{ResourceKind: "StorageService"}}
	storageServiceResourceFunctions = NewBaseResourceFunctions("StorageService", p)
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

	hardwareTypes := make([]string, len(ss.HardwareTypes))
	for i, hwType := range ss.HardwareTypes {
		hardwareTypes[i] = hwType.Name
	}

	d.Set("name", ss.Name)
	d.Set("display_name", ss.DisplayName)
	d.Set("hardware_types", hardwareTypes)
	return nil
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
