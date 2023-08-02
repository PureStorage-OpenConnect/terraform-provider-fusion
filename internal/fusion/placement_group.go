/*
Copyright 2023 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package fusion

import (
	"context"
	"fmt"

	"github.com/antihax/optional"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/utilities"
	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"
)

// Implements ResourceProvider
type placementGroupProvider struct {
	BaseResourceProvider
}

func schemaPlacementGroup() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		optionName: {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "The name of the Placement Group.",
		},
		optionDisplayName: {
			Type:         schema.TypeString,
			Optional:     true,
			Computed:     true,
			ValidateFunc: validation.StringLenBetween(1, maxDisplayName),
			Description:  "The human-readable name of the Placement Group. If not provided, defaults to I(name).",
		},
		optionTenant: {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "The name of the Tenant.",
		},
		optionTenantSpace: {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "The name of the Tenant Space.",
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
			Description:  "The name of the Availability Zone the Placement Group is in.",
		},
		optionStorageService: {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "The name of the Storage Service to create the Placement Group for.",
		},
		optionArray: {
			Type:         schema.TypeString,
			Optional:     true,
			Computed:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "The name of the Array to place the Placement Group to. Changing it (i.e. manual migration) is an elevated operation.",
		},
		optionDestroySnapshotsOnDelete: {
			Type:     schema.TypeBool,
			Optional: true,
			Default:  false,
			Description: "Before deleting placement group, snapshots within the Placement Group will be deleted. " +
				"If `false` then any snapshots will need to be deleted as a separate step before removing the Placement Group",
		},
	}
}

// This is our entry point for the Placement Group resource
func resourcePlacementGroup() *schema.Resource {
	p := &placementGroupProvider{BaseResourceProvider{ResourceKind: resourceKindPlacementGroup}}
	placementGroupResourceFunctions := NewBaseResourceFunctions(resourceKindPlacementGroup, p)
	placementGroupResourceFunctions.Resource.Description = "A Network Interface of an Array for use by Pure Fusion."
	placementGroupResourceFunctions.Resource.Schema = schemaPlacementGroup()

	return placementGroupResourceFunctions.Resource
}

func (p *placementGroupProvider) PrepareCreate(ctx context.Context, d *schema.ResourceData) (InvokeWriteAPI, ResourcePost, error) {
	name := rdString(ctx, d, optionName)
	tenantName := rdString(ctx, d, optionTenant)
	tenantSpaceName := rdString(ctx, d, optionTenantSpace)
	array := rdString(ctx, d, optionArray)

	body := hmrest.PlacementGroupPost{
		Name:             name,
		DisplayName:      rdStringDefault(ctx, d, optionDisplayName, name),
		Region:           rdString(ctx, d, optionRegion),
		AvailabilityZone: rdString(ctx, d, optionAvailabilityZone),
		StorageService:   rdString(ctx, d, optionStorageService),
	}

	fn := func(ctx context.Context, client *hmrest.APIClient, body RequestSpec) (*hmrest.Operation, error) {
		op, _, err := client.PlacementGroupsApi.CreatePlacementGroup(ctx, *body.(*hmrest.PlacementGroupPost), tenantName, tenantSpaceName, nil)
		if err != nil {
			return &op, err
		}

		succeeded, err := utilities.WaitOnOperation(ctx, &op, client)
		if err != nil {
			return &op, err
		}

		if !succeeded {
			tflog.Error(ctx, "REST create placement_group failed", "error_message", op.Error_.Message, "PureCode", op.Error_.PureCode, "HttpCode", op.Error_.HttpCode)
			return &op, utilities.NewRestErrorFromOperation(&op)
		}

		d.SetId(op.Result.Resource.Id)

		if array != "" {
			patch := hmrest.PlacementGroupPatch{
				Array: &hmrest.NullableString{Value: array},
			}
			op, _, err = client.PlacementGroupsApi.UpdatePlacementGroup(ctx, patch, tenantName, tenantSpaceName, name, nil)
		}

		return &op, err
	}
	return fn, &body, nil
}

func (p *placementGroupProvider) ReadResource(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) error {
	pg, _, err := client.PlacementGroupsApi.GetPlacementGroupById(ctx, d.Id(), nil)
	if err != nil {
		return err
	}
	return p.loadPlacementGroup(ctx, pg, client, d)
}

func (p *placementGroupProvider) PrepareDelete(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) (InvokeWriteAPI, error) {
	placementGroupName := rdString(ctx, d, optionName)
	tenantName := rdString(ctx, d, optionTenant)
	tenantSpaceName := rdString(ctx, d, optionTenantSpace)
	destroySnaps := d.Get(optionDestroySnapshotsOnDelete).(bool)

	fn := func(ctx context.Context, client *hmrest.APIClient, body RequestSpec) (*hmrest.Operation, error) {
		if destroySnaps {
			tflog.Debug(ctx, "Destroying relevant snapshots if they exist", optionTenant, tenantName, optionTenantSpace, tenantSpaceName)
			snapshots, _, err := client.SnapshotsApi.ListSnapshots(ctx, tenantName, tenantSpaceName, &hmrest.SnapshotsApiListSnapshotsOpts{
				PlacementGroup: optional.NewString(placementGroupName),
			})
			if err != nil {
				tflog.Error(ctx, "Failed listing snapshots", optionTenant, tenantName, optionTenantSpace, tenantSpaceName)
				utilities.TraceError(ctx, err)
				return nil, err
			}
			if len(snapshots.Items) > 0 {
				tflog.Info(ctx, "Deleting Snapshots in order to delete Placement Group", "placement_group", placementGroupName)
				deleteSnapshots(ctx, &snapshots, client)
			} else {
				tflog.Debug(ctx, "No snapshots found", optionTenant, tenantName, optionTenantSpace, tenantSpaceName)
			}
		}

		op, _, err := client.PlacementGroupsApi.DeletePlacementGroup(ctx, tenantName, tenantSpaceName, placementGroupName, nil)
		return &op, err
	}
	return fn, nil
}

func (p *placementGroupProvider) PrepareUpdate(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) (InvokeWriteAPI, []ResourcePatch, error) {
	if err := utilities.CheckImmutableFieldsExcept(ctx, d, optionDisplayName, optionArray, optionDestroySnapshotsOnDelete); err != nil {
		return nil, nil, err
	}

	name := rdString(ctx, d, optionName)
	tenantName := rdString(ctx, d, optionTenant)
	tenantSpaceName := rdString(ctx, d, optionTenantSpace)

	var patches []ResourcePatch

	if d.HasChange(optionDisplayName) {
		displayName := rdStringDefault(ctx, d, optionDisplayName, name)
		utilities.TracePatch(ctx, "placement_group", name, optionDisplayName, displayName, len(patches))
		patches = append(patches, &hmrest.PlacementGroupPatch{
			DisplayName: &hmrest.NullableString{Value: displayName},
		})
	}

	if d.HasChange(optionArray) {
		array := rdString(ctx, d, optionArray)
		utilities.TracePatch(ctx, "placement_group", name, optionArray, array, len(patches))
		patches = append(patches, &hmrest.PlacementGroupPatch{
			Array: &hmrest.NullableString{Value: array},
		})
	}

	fn := func(ctx context.Context, client *hmrest.APIClient, body RequestSpec) (*hmrest.Operation, error) {
		op, _, err := client.PlacementGroupsApi.UpdatePlacementGroup(ctx, *body.(*hmrest.PlacementGroupPatch), tenantName, tenantSpaceName, name, nil)
		return &op, err
	}

	return fn, patches, nil
}

func (p *placementGroupProvider) ImportResource(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) ([]*schema.ResourceData, error) {
	var orderedRequiredGroupNames = []string{
		resourceGroupNameTenant,
		resourceGroupNameTenantSpace,
		resourceGroupNamePlacementGroup,
	}
	// The ID is user provided value - we expect self link
	selfLinkFieldsWithValues, err := utilities.ParseSelfLink(d.Id(), orderedRequiredGroupNames)
	if err != nil {
		return nil, fmt.Errorf("invalid placement_group import path. Expected path in format '/tenants/<tenant>/tenant-spaces/<tenant-space>/placement-groups/<placement-group>'")
	}

	placementGroup, _, err := client.PlacementGroupsApi.GetPlacementGroup(ctx, selfLinkFieldsWithValues[resourceGroupNameTenant], selfLinkFieldsWithValues[resourceGroupNameTenantSpace], selfLinkFieldsWithValues[resourceGroupNamePlacementGroup], nil)
	if err != nil {
		utilities.TraceError(ctx, err)
		return nil, err
	}

	err = p.loadPlacementGroup(ctx, placementGroup, client, d)
	if err != nil {
		return nil, err
	}

	d.SetId(placementGroup.Id)

	return []*schema.ResourceData{d}, nil
}

func (p *placementGroupProvider) loadPlacementGroup(ctx context.Context, pg hmrest.PlacementGroup, client *hmrest.APIClient, d *schema.ResourceData) error {

	az, _, err := client.AvailabilityZonesApi.GetAvailabilityZoneById(ctx, pg.AvailabilityZone.Id, nil)
	if err != nil {
		return err
	}

	return getFirstError(
		d.Set(optionName, pg.Name),
		d.Set(optionDisplayName, pg.DisplayName),
		d.Set(optionTenant, pg.Tenant.Name),
		d.Set(optionTenantSpace, pg.TenantSpace.Name),
		d.Set(optionAvailabilityZone, pg.AvailabilityZone.Name),
		d.Set(optionStorageService, pg.StorageService.Name),
		d.Set(optionArray, pg.Array.Name),
		d.Set(optionRegion, az.Region.Name),
	)
}
