/*
Copyright 2022 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package fusion

import (
	"context"
	"fmt"

	"github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/utilities"
	"github.com/antihax/optional"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"
)

var placementGroupResourceFunctions *BaseResourceFunctions

// This is our entry point for the Storage Class resource. Get it movin'
func resourcePlacementGroup() *schema.Resource {
	vp := &placementGroupProvider{BaseResourceProvider{ResourceKind: "PlacementGroup"}}
	placementGroupResourceFunctions = NewBaseResourceFunctions("PlacementGroup", vp)

	placementGroupResourceFunctions.Resource.Schema = map[string]*schema.Schema{
		"name": {
			Type:     schema.TypeString,
			Required: true,
		},
		"display_name": {
			Type:     schema.TypeString,
			Optional: true,
			Computed: true,
		},
		"tenant_name": {
			Type:     schema.TypeString,
			Required: true,
		},
		"tenant_space_name": {
			Type:     schema.TypeString,
			Required: true,
		},
		"region_name": {
			Type:     schema.TypeString,
			Required: true,
		},
		"availability_zone_name": {
			Type:     schema.TypeString,
			Required: true,
		},
		"storage_service_name": {
			Type:     schema.TypeString,
			Required: true,
		},
		"destroy_snapshots_on_delete": {
			Type:     schema.TypeBool,
			Optional: true,
			Default:  false,
			Description: "Before deleting placement group, snapshots within the placement group will be deleted. " +
				"If `false` then any snapshots will need to be deleted as a separate step before removing the placement group",
		},
	}

	return placementGroupResourceFunctions.Resource
}

// Implements ResourceProvider
type placementGroupProvider struct {
	BaseResourceProvider
}

func (vp *placementGroupProvider) PrepareCreate(ctx context.Context, d *schema.ResourceData) (InvokeWriteAPI, ResourcePost, error) {
	name := rdString(ctx, d, "name")
	displayName := rdStringDefault(ctx, d, "display_name", name)
	tenantName := rdString(ctx, d, "tenant_name")
	tenantSpaceName := rdString(ctx, d, "tenant_space_name")
	storageService := rdString(ctx, d, "storage_service_name")

	regionName := rdString(ctx, d, "region_name")
	availabilityZone := rdString(ctx, d, "availability_zone_name")

	tflog.Debug(ctx, "PlacementGroup.CreateResource()", "ts", tenantSpaceName, "name", name)

	body := hmrest.PlacementGroupPost{
		Name:             name,
		DisplayName:      displayName,
		Region:           regionName,
		AvailabilityZone: availabilityZone,
		StorageService:   storageService,
	}

	fn := func(ctx context.Context, client *hmrest.APIClient, body RequestSpec) (*hmrest.Operation, error) {
		op, _, err := client.PlacementGroupsApi.CreatePlacementGroup(ctx, *body.(*hmrest.PlacementGroupPost), tenantName, tenantSpaceName, nil)
		return &op, err
	}
	return fn, &body, nil
}

func (vp *placementGroupProvider) ReadResource(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) error {
	tflog.Debug(ctx, "PlacementGroup.ReadResource()", "id", d.Id())
	pg, _, err := client.PlacementGroupsApi.GetPlacementGroupById(ctx, d.Id(), nil)
	if err != nil {
		return err
	}

	d.Set("name", pg.Name)
	d.Set("display_name", pg.DisplayName)
	d.Set("tenant_name", pg.Tenant.Name)
	d.Set("tenant_space_name", pg.TenantSpace.Name)
	d.Set("availability_zone_name", pg.AvailabilityZone.Name)
	d.Set("storage_service_name", pg.StorageService.Name)

	az, _, err := client.AvailabilityZonesApi.GetAvailabilityZoneById(ctx, pg.AvailabilityZone.Id, nil)
	if err != nil {
		return err
	}
	d.Set("region_name", az.Region.Name)

	return nil
}

func (vp *placementGroupProvider) PrepareDelete(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) (InvokeWriteAPI, error) {
	placementGroupName := rdString(ctx, d, "name")
	tenantName := rdString(ctx, d, "tenant_name")
	tenantSpaceName := rdString(ctx, d, "tenant_space_name")
	destroySnaps := d.Get("destroy_snapshots_on_delete").(bool)

	fn := func(ctx context.Context, client *hmrest.APIClient, body RequestSpec) (*hmrest.Operation, error) {
		if destroySnaps {
			tflog.Debug(ctx, "Destroying relevant snapshots if they exist", "tenant_name", tenantName, "tenant_space_name", tenantSpaceName)
			snapshots, _, err := client.SnapshotsApi.ListSnapshots(ctx, tenantName, tenantSpaceName, &hmrest.SnapshotsApiListSnapshotsOpts{
				PlacementGroup: optional.NewString(placementGroupName),
			})
			if err != nil {
				tflog.Error(ctx, "Failed listing snapshots", "tenant_name", tenantName, "tenant_space_name", tenantSpaceName)
				utilities.TraceError(ctx, err)
				return nil, err
			}
			if len(snapshots.Items) > 0 {
				tflog.Info(ctx, "Deleting Snapshots in order to delete Placement Group", "placement_group", placementGroupName)
				var patches []ResourcePatch
				for _, snap := range snapshots.Items {
					tflog.Trace(ctx, "Constructing Patch to Delete Snapshot", "name", snap.Name)
					patches = append(patches, snap.Name)
				}

				fn := func(ctx context.Context, client *hmrest.APIClient, body RequestSpec) (*hmrest.Operation, error) {
					patchBody := hmrest.SnapshotPatch{Destroyed: &hmrest.NullableBoolean{Value: true}}
					op, _, err := client.SnapshotsApi.UpdateSnapshot(ctx, patchBody, tenantName, tenantSpaceName, body.(string), nil)
					if err != nil {
						tflog.Error(ctx, "failed destroying a snapshot as part of deleting placement group",
							"snapshot_name", body.(string))
					}
					return &op, err
				}

				if err := executePatches(ctx, fn, patches, client, "destroySnap"); err != nil {
					return nil, err
				}

				fn = func(ctx context.Context, client *hmrest.APIClient, body RequestSpec) (*hmrest.Operation, error) {
					op, _, err := client.SnapshotsApi.DeleteSnapshot(ctx, tenantName, tenantSpaceName, body.(string), nil)
					if err != nil {
						tflog.Error(ctx, "failed deleting a snapshot as part of deleting placement group",
							"snapshot_name", body.(string))
					}
					return &op, err
				}

				if err = executePatches(ctx, fn, patches, client, "deleteSnap"); err != nil {
					return nil, err
				}
			} else {
				tflog.Debug(ctx, "No snapshots found", "tenant_name", tenantName, "tenant_space_name", tenantSpaceName)
			}
		}

		op, _, err := client.PlacementGroupsApi.DeletePlacementGroup(ctx, tenantName, tenantSpaceName, placementGroupName, nil)
		return &op, err
	}
	return fn, nil
}

func (vp *placementGroupProvider) PrepareUpdate(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) (InvokeWriteAPI, []ResourcePatch, error) {

	var patches []ResourcePatch // []*hmrest.TenantSpacePatch

	placementGroupName := rdString(ctx, d, "name")
	tenantName := rdString(ctx, d, "tenant_name")
	tenantSpaceName := rdString(ctx, d, "tenant_space_name")

	if d.HasChangeExcept("display_name") {
		return nil, nil, fmt.Errorf("attempting to update an immutable field")
	} else if d.HasChange("display_name") {
		displayName := d.Get("display_name").(string)
		tflog.Info(ctx, "Updating", "display_name", displayName)
		patches = append(patches, &hmrest.PlacementGroupPatch{
			DisplayName: &hmrest.NullableString{Value: displayName},
		})
	}
	fn := func(ctx context.Context, client *hmrest.APIClient, body RequestSpec) (*hmrest.Operation, error) {
		op, _, err := client.PlacementGroupsApi.UpdatePlacementGroup(ctx, *body.(*hmrest.PlacementGroupPatch), tenantName, tenantSpaceName, placementGroupName, nil)
		return &op, err
	}
	return fn, patches, nil
}
