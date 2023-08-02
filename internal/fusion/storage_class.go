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

// Implements ResourceProvider
type storageClassProvider struct {
	BaseResourceProvider
}

func schemaStorageClass() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		optionName: {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "The name of the Storage Class.",
		},
		optionDisplayName: {
			Type:         schema.TypeString,
			Optional:     true,
			Computed:     true,
			ValidateFunc: validation.StringLenBetween(1, maxDisplayName),
			Description:  "The human-readable name of the Storage Class. If not provided, defaults to I(name).",
		},
		optionStorageService: {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "Storage Service in which the Storage Class is created.",
		},
		optionSizeLimit: {
			Type:             schema.TypeString,
			Optional:         true,
			ValidateDiagFunc: utilities.DataUnitsBeetween(storageClassSizeMin, storageClassSizeMax, 1024),
			DiffSuppressFunc: utilities.GetDiffSuppressForDataUnits(1024),
			Default:          "4P",
			Description: `Maximum Volume size limit of Storage Class. 
			- Volume size limit in M, G, T or P units.
			- Must be between 1MB and 4PB.
			- If not provided at creation, this will default to 4PB.`,
		},
		optionIopsLimit: {
			Type:     schema.TypeString,
			Optional: true,
			ValidateDiagFunc: utilities.AllValid(
				utilities.AllowedDataUnitSuffix('K', 'M'),
				utilities.DataUnitsBeetween(storageClassIopsMin, storageClassIopsMax, 1000),
			),
			Default:          "100M",
			DiffSuppressFunc: utilities.GetDiffSuppressForDataUnits(1000),
			Description: `Maximum IOPS limit of Storage Class.
			- The IOPs limit - use value or K or M.
			K will mean 1000.
			M will mean 1000000.
			- Must be between 100 and 100000000
			- If not provided at creation, this will default to 100M.`,
		},
		optionBandwidthLimit: {
			Type:     schema.TypeString,
			Optional: true,
			ValidateDiagFunc: utilities.AllValid(
				utilities.AllowedDataUnitSuffix('M', 'G'),
				utilities.DataUnitsBeetween(storageClassBandwidthMin, storageClassBandwidthMax, 1024),
			),
			Default:          "512G",
			DiffSuppressFunc: utilities.GetDiffSuppressForDataUnits(1024),
			Description: `Maximum Bandwidth Limit of Storage Class.
			- The bandwidth limit in M or G units.
			M will set MB/s.
			G will set GB/s.
			- Must be between 1MB/s and 512GB/s.
			- If not provided at creation, this will default to 512GB/s.`,
		},
	}
}

func resourceStorageClass() *schema.Resource {
	p := &storageClassProvider{BaseResourceProvider{ResourceKind: resourceKindStorageClass}}
	storageClassResourceFunctions := NewBaseResourceFunctions(resourceKindStorageClass, p)
	storageClassResourceFunctions.Resource.Description = `A Storage Class (e.g. "Gold") is published by the AZ Admin.` +
		` It is a coarse grained tier, perhaps just one per Storage Service and QoS.` +
		` It specifies a Max IOPS / GB, bandwidth and size; as well as Storage Service.`
	storageClassResourceFunctions.Schema = schemaStorageClass()

	return storageClassResourceFunctions.Resource
}

func (p *storageClassProvider) PrepareCreate(ctx context.Context, d *schema.ResourceData) (InvokeWriteAPI, ResourcePost, error) {
	name := rdString(ctx, d, optionName)
	displayName := rdStringDefault(ctx, d, optionDisplayName, name)
	storageService := rdString(ctx, d, optionStorageService)
	sizeLimit, _ := utilities.ConvertDataUnitsToInt64(rdString(ctx, d, optionSizeLimit), 1024)
	iopsLimit, _ := utilities.ConvertDataUnitsToInt64(rdString(ctx, d, optionIopsLimit), 1000)
	bandwidthLimit, _ := utilities.ConvertDataUnitsToInt64(rdString(ctx, d, optionBandwidthLimit), 1024)

	body := hmrest.StorageClassPost{
		Name:           name,
		DisplayName:    displayName,
		SizeLimit:      sizeLimit,
		IopsLimit:      iopsLimit,
		BandwidthLimit: bandwidthLimit,
	}

	fn := func(ctx context.Context, client *hmrest.APIClient, body RequestSpec) (*hmrest.Operation, error) {
		op, _, err := client.StorageClassesApi.CreateStorageClass(ctx, *body.(*hmrest.StorageClassPost), storageService, nil)
		return &op, err
	}
	return fn, &body, nil
}

func (p *storageClassProvider) ReadResource(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) error {
	sc, _, err := client.StorageClassesApi.GetStorageClassById(ctx, d.Id(), nil)
	if err != nil {
		return err
	}

	return p.loadStorageClass(sc, d)
}

func (p *storageClassProvider) PrepareUpdate(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) (InvokeWriteAPI, []ResourcePatch, error) {
	var patches []ResourcePatch

	storageClassName := rdString(ctx, d, optionName)
	storageServiceName := rdString(ctx, d, optionStorageService)
	if d.HasChangeExcept(optionDisplayName) {
		d.Partial(true)
		return nil, nil, fmt.Errorf("attempting to update an immutable field")
	}

	displayName := rdStringDefault(ctx, d, optionDisplayName, storageClassName)
	tflog.Info(ctx, "Updating", optionDisplayName, displayName)
	patches = append(patches, &hmrest.StorageClassPatch{
		DisplayName: &hmrest.NullableString{Value: displayName},
	})

	fn := func(ctx context.Context, client *hmrest.APIClient, body RequestSpec) (*hmrest.Operation, error) {
		op, _, err := client.StorageClassesApi.UpdateStorageClass(ctx, *body.(*hmrest.StorageClassPatch), storageServiceName, storageClassName, nil)
		return &op, err
	}

	return fn, patches, nil
}

func (p *storageClassProvider) PrepareDelete(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) (InvokeWriteAPI, error) {
	storageClassName := rdString(ctx, d, optionName)
	storageServiceName := rdString(ctx, d, optionStorageService)

	fn := func(ctx context.Context, client *hmrest.APIClient, body RequestSpec) (*hmrest.Operation, error) {
		op, _, err := client.StorageClassesApi.DeleteStorageClass(ctx, storageServiceName, storageClassName, nil)
		return &op, err
	}
	return fn, nil
}

func (p *storageClassProvider) ImportResource(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) ([]*schema.ResourceData, error) {
	var orderedRequiredGroupNames = []string{
		resourceGroupNameStorageService,
		resourceGroupNameStorageClass,
	}
	// The ID is user provided value - we expect self link
	selfLinkFieldsWithValues, err := utilities.ParseSelfLink(d.Id(), orderedRequiredGroupNames)
	if err != nil {
		return nil, fmt.Errorf("invalid storage_class import path. Expected path in format '/storage-services/<storage-service>/storage-classes/<storage-class>'")
	}

	storageClass, _, err := client.StorageClassesApi.GetStorageClass(ctx, selfLinkFieldsWithValues[resourceGroupNameStorageService], selfLinkFieldsWithValues[resourceGroupNameStorageClass], nil)
	if err != nil {
		utilities.TraceError(ctx, err)
		return nil, err
	}

	err = p.loadStorageClass(storageClass, d)
	if err != nil {
		return nil, err
	}

	d.SetId(storageClass.Id)

	return []*schema.ResourceData{d}, nil
}

func (p *storageClassProvider) loadStorageClass(sc hmrest.StorageClass, d *schema.ResourceData) error {
	return getFirstError(
		d.Set(optionName, sc.Name),
		d.Set(optionDisplayName, sc.DisplayName),
		d.Set(optionStorageService, sc.StorageService.Name),
		d.Set(optionSizeLimit, strconv.FormatInt(sc.SizeLimit, 10)),
		d.Set(optionIopsLimit, strconv.FormatInt(sc.IopsLimit, 10)),
		d.Set(optionBandwidthLimit, strconv.FormatInt(sc.BandwidthLimit, 10)),
	)
}
