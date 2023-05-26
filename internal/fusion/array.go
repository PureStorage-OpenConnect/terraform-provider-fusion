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

func createArraySchema() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		optionName: {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringIsNotEmpty,
		},
		optionDisplayName: {
			Type:         schema.TypeString,
			Optional:     true,
			Computed:     true,
			ValidateFunc: validation.StringLenBetween(1, maxDisplayName),
		},
		optionAvailabilityZone: {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringIsNotEmpty,
		},
		optionRegion: {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringIsNotEmpty,
		},
		optionApplianceId: {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringIsNotEmpty,
		},
		optionHostName: {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringIsNotEmpty,
		},
		optionHardwareType: {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringInSlice(HwTypes, false),
		},
		optionMaintenanceMode: {
			Type:     schema.TypeBool,
			Optional: true,
			Computed: true,
		},
		optionUnavailableMode: {
			Type:     schema.TypeBool,
			Optional: true,
			Computed: true,
		},
		optionApartmentId: {
			Type:         schema.TypeString,
			Optional:     true,
			Computed:     true,
			ValidateFunc: validation.StringIsNotEmpty,
		},
	}
}

func resourceArray() *schema.Resource {
	ap := &arrayProvider{BaseResourceProvider{ResourceKind: "Array"}}
	array := NewBaseResourceFunctions("Array", ap)

	array.Resource.Schema = createArraySchema()

	return array.Resource
}

type arrayProvider struct {
	BaseResourceProvider
}

func (p *arrayProvider) PrepareCreate(ctx context.Context, d *schema.ResourceData) (InvokeWriteAPI, ResourcePost, error) {
	name := rdString(ctx, d, optionName)
	region := rdString(ctx, d, optionRegion)
	availabilityZone := rdString(ctx, d, optionAvailabilityZone)

	body := hmrest.ArrayPost{
		Name:         name,
		DisplayName:  rdStringDefault(ctx, d, optionDisplayName, name),
		ApartmentId:  rdString(ctx, d, optionApartmentId),
		HostName:     rdString(ctx, d, optionHostName),
		HardwareType: rdString(ctx, d, optionHardwareType),
		ApplianceId:  rdString(ctx, d, optionApplianceId),
	}

	fn := func(ctx context.Context, client *hmrest.APIClient, body RequestSpec) (*hmrest.Operation, error) {
		op, _, err := client.ArraysApi.CreateArray(ctx, *body.(*hmrest.ArrayPost), region, availabilityZone, nil)
		if err != nil {
			return &op, err
		}

		succeeded, err := utilities.WaitOnOperation(ctx, &op, client)
		if err != nil {
			return &op, err
		}

		if !succeeded {
			tflog.Error(ctx, "REST create array failed", "error_message", op.Error_.Message,
				"PureCode", op.Error_.PureCode, "HttpCode", op.Error_.HttpCode)

			return &op, utilities.NewRestErrorFromOperation(&op)
		}

		lastOp := &op

		d.SetId(op.Result.Resource.Id)

		// create does not support setting maintenance mode / unavailable mode so update them as well as
		// the user would otherwise have to do `terraform apply` twice to really get the desired state

		array, _, err := client.ArraysApi.GetArrayById(ctx, op.Result.Resource.Id, nil)
		if err != nil {
			return lastOp, err
		}

		var patches []hmrest.ArrayPatch
		maintenanceMode := d.Get(optionMaintenanceMode).(bool)
		if array.MaintenanceMode != maintenanceMode {
			patches = append(patches, hmrest.ArrayPatch{
				MaintenanceMode: &hmrest.NullableBoolean{
					Value: maintenanceMode,
				},
			})
		}

		unavailableMode := d.Get(optionUnavailableMode).(bool)
		if array.UnavailableMode != unavailableMode {
			patches = append(patches, hmrest.ArrayPatch{
				MaintenanceMode: &hmrest.NullableBoolean{
					Value: array.UnavailableMode,
				},
			})
		}

		for i, p := range patches {
			ctx := tflog.With(ctx, "patch_idx", i)
			tflog.Debug(ctx, "Starting operation to apply a patch", "patch_op", "arrayUpdate", "patch_num", i, "patch", p)
			op, _, err := client.ArraysApi.UpdateArray(ctx, p, region, availabilityZone, array.Name, nil)
			utilities.TraceOperation(ctx, &op, "Applying Array Patch")
			if err != nil {
				return &op, err
			}

			succeeded, err := utilities.WaitOnOperation(ctx, &op, client)
			if err != nil {
				return &op, err
			}
			if !succeeded {
				return &op, fmt.Errorf("operation failed Message:%s ID:%s", op.Error_.Message, op.Id)
			}

			lastOp = &op
		}

		return lastOp, nil
	}

	return fn, &body, nil
}

func (p *arrayProvider) ReadResource(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) error {
	array, _, err := client.ArraysApi.GetArrayById(ctx, d.Id(), nil)
	if err != nil {
		return err
	}

	d.Set(optionName, array.Name)
	d.Set(optionDisplayName, array.DisplayName)
	d.Set(optionAvailabilityZone, array.AvailabilityZone.Name)
	d.Set(optionRegion, array.Region.Name)
	d.Set(optionApplianceId, array.ApplianceId)
	d.Set(optionHostName, array.HostName)
	d.Set(optionHardwareType, array.HardwareType.Name)
	d.Set(optionApartmentId, array.ApartmentId)
	d.Set(optionMaintenanceMode, array.MaintenanceMode)
	d.Set(optionUnavailableMode, array.UnavailableMode)
	return nil
}

func (p *arrayProvider) PrepareUpdate(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) (InvokeWriteAPI, []ResourcePatch, error) {
	if err := utilities.CheckImmutableFieldsExcept(ctx, d,
		optionDisplayName,
		optionHostName,
		optionMaintenanceMode,
		optionUnavailableMode); err != nil {
		return nil, nil, err
	}

	arrayName := d.Get(optionName).(string)
	region := d.Get(optionRegion).(string)
	availabilityZone := d.Get(optionAvailabilityZone).(string)

	var patches []ResourcePatch

	if d.HasChange(optionDisplayName) {
		displayName := rdStringDefault(ctx, d, optionDisplayName, arrayName)
		utilities.TracePatch(ctx, "array", arrayName, optionDisplayName, displayName, len(patches))
		patches = append(patches, &hmrest.ArrayPatch{
			DisplayName: &hmrest.NullableString{Value: displayName},
		})
	}

	if d.HasChange(optionHostName) {
		hostName := d.Get(optionHostName).(string)
		utilities.TracePatch(ctx, "array", arrayName, optionHostName, hostName, len(patches))
		patches = append(patches, &hmrest.ArrayPatch{
			HostName: &hmrest.NullableString{Value: hostName},
		})
	}

	if d.HasChange(optionMaintenanceMode) {
		maintenanceMode := d.Get(optionMaintenanceMode).(bool)
		utilities.TracePatch(ctx, "array", arrayName, optionMaintenanceMode, maintenanceMode, len(patches))
		patches = append(patches, &hmrest.ArrayPatch{
			MaintenanceMode: &hmrest.NullableBoolean{Value: maintenanceMode},
		})
	}

	if d.HasChange(optionUnavailableMode) {
		unavailableMode := d.Get(optionUnavailableMode).(bool)
		utilities.TracePatch(ctx, "array", arrayName, optionUnavailableMode, unavailableMode, len(patches))
		patches = append(patches, &hmrest.ArrayPatch{
			UnavailableMode: &hmrest.NullableBoolean{Value: unavailableMode},
		})
	}

	fn := func(ctx context.Context, client *hmrest.APIClient, body RequestSpec) (*hmrest.Operation, error) {
		op, _, err := client.ArraysApi.UpdateArray(ctx, *body.(*hmrest.ArrayPatch), region, availabilityZone, arrayName, nil)
		return &op, err
	}

	return fn, patches, nil
}

func (p *arrayProvider) PrepareDelete(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) (InvokeWriteAPI, error) {
	arrayName := d.Get(optionName).(string)
	region := d.Get(optionRegion).(string)
	availabilityZone := d.Get(optionAvailabilityZone).(string)

	fn := func(ctx context.Context, client *hmrest.APIClient, body RequestSpec) (*hmrest.Operation, error) {
		op, _, err := client.ArraysApi.DeleteArray(ctx, region, availabilityZone, arrayName, nil)
		return &op, err
	}
	return fn, nil
}
