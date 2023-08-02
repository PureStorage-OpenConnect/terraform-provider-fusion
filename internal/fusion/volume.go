/*
Copyright 2023 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package fusion

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/utilities"
	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"
)

// This is our entry point for the Volume resource. Get it movin'
func schemaVolume() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		optionName: {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "The name of the Volume.",
		},
		optionDisplayName: {
			Type:         schema.TypeString,
			Optional:     true,
			Computed:     true,
			ValidateFunc: validation.StringLenBetween(1, maxDisplayName),
			Description:  "The human-readable name of the Volume. If not provided, defaults to I(name).",
		},
		optionSize: {
			Type:             schema.TypeString,
			Optional:         true,
			Computed:         true,
			ValidateDiagFunc: utilities.DataUnitsBeetween(volumeSizeMin, volumeSizeMax, 1024),
			DiffSuppressFunc: utilities.GetDiffSuppressForDataUnits(1024),
			ConflictsWith:    []string{optionSourceLink},
			Description: `The Volume size in M, G, T or P units.
			- Volume size in M, G, T or P units.
			- Must be between 1MB and 4PB.`,
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
		optionStorageClass: {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "The name of the Storage Class.",
		},
		optionPlacementGroup: {
			Type:        schema.TypeString,
			Required:    true,
			Description: "The name of the Placement Group. WARNING: Changing this value will cause a new IQN number to be generated and will disrupt initiator access to this Volume.",
		},
		optionProtectionPolicy: {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The name of the Protection Policy.",
		},
		optionHostAccessPolicies: {
			Type:     schema.TypeSet,
			Optional: true,
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
			Description: "The list of Host Access Policies to connect the Volume to.",
		},
		optionCreatedAt: {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "The time that the operation was created, in milliseconds since the Unix epoch.",
		},
		optionSerialNumber: {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The serial number of the Volume.",
		},
		optionTargetIscsiIqn: {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The IQN of the iSCSI target.",
		},
		optionTargetIscsiAddresses: {
			Type:     schema.TypeSet,
			Computed: true,
			Elem: &schema.Schema{
				Type:        schema.TypeString,
				Description: "The address of the iSCSI target.",
			},
		},
		optionEradicateOnDelete: {
			Type:        schema.TypeBool,
			Optional:    true,
			Default:     false,
			Description: "Eradicate the Volume when the Volume is deleted.",
		},
		optionSourceLink: {
			Type:          schema.TypeList,
			Optional:      true,
			MaxItems:      1,
			ConflictsWith: []string{optionSize},
			Description:   "The link to copy data from.",
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
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
					optionSnapshot: {
						Type:          schema.TypeString,
						Optional:      true,
						ValidateFunc:  validation.StringIsNotEmpty,
						Description:   "The Snapshot name.",
						RequiredWith:  []string{getSourceLinkItem(optionVolumeSnapshot)},
						ConflictsWith: []string{getSourceLinkItem(optionVolume)},
					},
					optionVolumeSnapshot: {
						Type:          schema.TypeString,
						Optional:      true,
						ValidateFunc:  validation.StringIsNotEmpty,
						Description:   "The name of the Volume Snapshot.",
						RequiredWith:  []string{getSourceLinkItem(optionSnapshot)},
						ConflictsWith: []string{getSourceLinkItem(optionVolume)},
					},
					optionVolume: {
						Type:          schema.TypeString,
						Optional:      true,
						ValidateFunc:  validation.StringIsNotEmpty,
						Description:   "The name of the Volume.",
						ConflictsWith: []string{getSourceLinkItem(optionVolumeSnapshot), getSourceLinkItem(optionSnapshot)},
					},
				},
			},
		},
	}

}

func resourceVolume() *schema.Resource {
	vp := &volumeProvider{BaseResourceProvider{ResourceKind: resourceKindVolume}}
	volumeResourceFunctions := NewBaseResourceFunctions(resourceKindVolume, vp)

	volumeResourceFunctions.Resource.Description = "A Volume represents a container that manages the storage space " +
		"on the array. After a Volume has been created, establish a Host-Volume connection so that the Host can read data " +
		"from and write data to the Volume."
	volumeResourceFunctions.Resource.Schema = schemaVolume()

	return volumeResourceFunctions.Resource
}

// Implements ResourceProvider
type volumeProvider struct {
	BaseResourceProvider
}

func (vp *volumeProvider) PrepareCreate(ctx context.Context, d *schema.ResourceData) (InvokeWriteAPI, ResourcePost, error) {
	tenantName := rdString(ctx, d, optionTenant)
	tenantSpaceName := rdString(ctx, d, optionTenantSpace)
	name := rdString(ctx, d, optionName)
	displayName := rdStringDefault(ctx, d, optionDisplayName, name)
	size, _ := utilities.ConvertDataUnitsToInt64(rdString(ctx, d, optionSize), 1024)

	body := hmrest.VolumePost{
		Name:             name,
		DisplayName:      displayName,
		Size:             size,
		StorageClass:     rdString(ctx, d, optionStorageClass),
		PlacementGroup:   rdString(ctx, d, optionPlacementGroup),
		ProtectionPolicy: rdString(ctx, d, optionProtectionPolicy),
	}

	if _, ok := d.GetOk(optionSourceLink); ok {
		body.SourceLink = vp.getSourceLink(ctx, d)
		body.Size = 0
	}

	fn := func(ctx context.Context, client *hmrest.APIClient, body RequestSpec) (*hmrest.Operation, error) {
		op, _, err := client.VolumesApi.CreateVolume(ctx, *body.(*hmrest.VolumePost), tenantName, tenantSpaceName, nil)
		if err != nil {
			return &op, err
		}

		succeeded, err := utilities.WaitOnOperation(ctx, &op, client)
		if err != nil {
			return &op, err
		}

		if !succeeded {
			tflog.Error(ctx, "REST create volume failed", "error_message", op.Error_.Message,
				"PureCode", op.Error_.PureCode, "HttpCode", op.Error_.HttpCode)

			return &op, utilities.NewRestErrorFromOperation(&op)
		}

		d.SetId(op.Result.Resource.Id)

		hosts := strings.Join(rdStringSet(ctx, d, optionHostAccessPolicies), ",")
		patch := hmrest.VolumePatch{
			HostAccessPolicies: &hmrest.NullableString{Value: hosts},
		}

		// Add hosts to the volume (cannot be done via POST)
		tflog.Debug(ctx, "Starting operation to apply a (create) patch", "patch_op", "volumeUpdate", "patch", patch)
		op, _, err = client.VolumesApi.UpdateVolume(ctx, patch, tenantName, tenantSpaceName, op.Result.Resource.Name, nil)
		utilities.TraceOperation(ctx, &op, "Applying Volume Patch")
		if err != nil {
			return &op, err
		}

		succeeded, err = utilities.WaitOnOperation(ctx, &op, client)
		if err != nil {
			return &op, err
		}
		if !succeeded {
			return &op, fmt.Errorf("operation failed Message:%s ID:%s", op.Error_.Message, op.Id)
		}

		return &op, nil
	}

	return fn, &body, nil
}

func (vp *volumeProvider) ReadResource(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) error {
	vol, _, err := client.VolumesApi.GetVolumeById(ctx, d.Id(), nil)
	if err != nil {
		return err
	}

	return vp.loadVolume(vol, d)
}

// ColumeProvider.PrepareUpdate will update the attributes of the volume.
//
// If a new size is provided, it must be larger than the current size.  Only
// extending volumes is supported at this time, since truncating volumes can
// lead to data loss.
func (vp *volumeProvider) PrepareUpdate(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) (InvokeWriteAPI, []ResourcePatch, error) {
	if err := utilities.CheckImmutableFields(ctx, d, optionName, optionTenant, optionTenantSpace); err != nil {
		return nil, nil, err
	}

	volumeName := d.Get(optionName).(string)
	tenantSpaceName := d.Get(optionTenantSpace).(string)
	tenantName := d.Get(optionTenant).(string)

	var patches []ResourcePatch

	if d.HasChange(optionDisplayName) {
		displayName := rdStringDefault(ctx, d, optionDisplayName, volumeName)
		tflog.Trace(ctx, "update",
			"resource", "volume",
			"parameter", optionDisplayName,
			"to", displayName,
			"patch_idx", len(patches),
		)
		patches = append(patches, &hmrest.VolumePatch{
			DisplayName: &hmrest.NullableString{Value: displayName},
		})
	}

	if d.HasChange(optionProtectionPolicy) {
		protectionPolicyName := d.Get(optionProtectionPolicy).(string)
		tflog.Trace(ctx, "update",
			"resource", "volume",
			"parameter", optionProtectionPolicy,
			"to", protectionPolicyName,
			"patch_idx", len(patches),
		)
		patches = append(patches, &hmrest.VolumePatch{
			ProtectionPolicy: &hmrest.NullableString{Value: protectionPolicyName},
		})
	}

	// if there is a change to placement groups, then we need to remove the hosts and then re-add them
	reAddHosts := false
	if d.HasChange(optionPlacementGroup) {
		reAddHosts = true
		tflog.Trace(ctx, "update",
			"resource", "volume",
			"parameter", optionHostAccessPolicies,
			"to", "",
			"patch_idx", len(patches),
			"message", "temporary removal of hosts for placement_groups_name change",
		)
		patches = append(patches, &hmrest.VolumePatch{
			HostAccessPolicies: &hmrest.NullableString{Value: ""},
		})
	}

	if d.HasChange(optionStorageClass) || d.HasChange(optionPlacementGroup) {
		patch := &hmrest.VolumePatch{}
		if d.HasChange(optionStorageClass) {
			storageClassName := d.Get(optionStorageClass).(string)
			tflog.Trace(ctx, "update",
				"resource", "volume",
				"parameter", optionStorageClass,
				"to", storageClassName,
				"patch_idx", len(patches),
			)
			patch.StorageClass = &hmrest.NullableString{Value: storageClassName}
		}
		if d.HasChange(optionPlacementGroup) {
			placementGroupName := d.Get(optionPlacementGroup).(string)
			tflog.Trace(ctx, "update",
				"resource", "volume",
				"parameter", optionPlacementGroup,
				"to", placementGroupName,
				"patch_idx", len(patches),
			)
			patch.PlacementGroup = &hmrest.NullableString{Value: placementGroupName}
		}
		patches = append(patches, patch)
	}

	if d.HasChange(optionHostAccessPolicies) || reAddHosts {
		s := strings.Join(rdStringSet(ctx, d, optionHostAccessPolicies), ",")

		tflog.Trace(ctx, "update",
			"resource", "volume",
			"parameter", optionHostAccessPolicies,
			"to", s,
			"patch_idx", len(patches),
			"readded", reAddHosts,
		)
		patches = append(patches, &hmrest.VolumePatch{
			HostAccessPolicies: &hmrest.NullableString{Value: s},
		})
	}

	if _, ok := d.GetOk(optionSize); ok && d.HasChange(optionSize) {
		size, _ := utilities.ConvertDataUnitsToInt64(rdString(ctx, d, optionSize), 1024)

		tflog.Trace(ctx, "update",
			"resource", "volume",
			"parameter", optionSize,
			"to", size,
			"patch_idx", len(patches),
		)

		patches = append(patches, &hmrest.VolumePatch{
			Size: &hmrest.NullableSize{Value: size},
		})
	}

	// The source_link is present and has been changed
	if _, ok := d.GetOk(optionSourceLink); ok && d.HasChange(optionSourceLink) {
		if _, ok := d.GetOk(getSourceLinkItem(optionSnapshot)); ok {
			return nil, patches, errors.New("cannot copy snapshot to existing volume")
		}

		sourceLink := vp.getSourceLink(ctx, d)
		patches = append(patches, &hmrest.VolumePatch{
			SourceLink: &hmrest.NullableString{Value: sourceLink},
		})
	}

	fn := func(ctx context.Context, client *hmrest.APIClient, body RequestSpec) (*hmrest.Operation, error) {
		op, _, err := client.VolumesApi.UpdateVolume(ctx, *body.(*hmrest.VolumePatch), tenantName, tenantSpaceName, volumeName, nil)
		return &op, err
	}

	return fn, patches, nil
}

func (vp *volumeProvider) PrepareDelete(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) (InvokeWriteAPI, error) {
	volumeName := d.Get(optionName).(string)
	tenantSpaceName := d.Get(optionTenantSpace).(string)
	tenantName := d.Get(optionTenant).(string)
	eradicate := d.Get(optionEradicateOnDelete).(bool)

	fn := func(ctx context.Context, client *hmrest.APIClient, body RequestSpec) (*hmrest.Operation, error) {
		tflog.Trace(ctx, "removing host assignments before deleting volume")
		op, _, err := client.VolumesApi.UpdateVolume(ctx, hmrest.VolumePatch{
			HostAccessPolicies: &hmrest.NullableString{Value: ""},
		}, tenantName, tenantSpaceName, volumeName, nil)
		utilities.TraceError(ctx, err)
		if err != nil {
			return &op, err
		}

		succeeded, err := utilities.WaitOnOperation(ctx, &op, client)
		if err != nil {
			return &op, err
		}
		if !succeeded {
			tflog.Error(ctx, "failed removing host assignments")
			return &op, fmt.Errorf("failed to clear out host assignments as part of deleting volume")
		}
		tflog.Trace(ctx, "done removing host assignments")

		tflog.Trace(ctx, "destroying volume")
		op, _, err = client.VolumesApi.UpdateVolume(ctx, hmrest.VolumePatch{
			Destroyed: &hmrest.NullableBoolean{Value: true},
		}, tenantName, tenantSpaceName, volumeName, nil)

		// Do not eradicate the volume - return the operation for patching the volume (destroyed=true)
		if !eradicate {
			return &op, err
		}

		utilities.TraceError(ctx, err)
		if err != nil {
			return &op, err
		}

		// Wait for patching the volume (destroyed=true)
		succeeded, err = utilities.WaitOnOperation(ctx, &op, client)
		if err != nil {
			return &op, err
		}
		if !succeeded {
			tflog.Error(ctx, "failed destroying volume")
			return &op, fmt.Errorf("failed destroying volume")
		}
		tflog.Trace(ctx, "done destroying volume")

		op, _, err = client.VolumesApi.DeleteVolume(ctx, tenantName, tenantSpaceName, volumeName, nil)
		return &op, err
	}
	return fn, nil
}

func (vp *volumeProvider) ImportResource(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) ([]*schema.ResourceData, error) {
	orderedRequiredGroupNames := []string{
		resourceGroupNameTenant,
		resourceGroupNameTenantSpace,
		resourceGroupNameVolume,
	}
	// The ID is user provided value - we expect self link
	parsedSelfLink, err := utilities.ParseSelfLink(d.Id(), orderedRequiredGroupNames)
	if err != nil {
		return nil, fmt.Errorf("invalid volume import path. Expected path in format '/tenants/<tenant>/tenant-spaces/<tenant-space>/volumes/<volume>'")
	}

	volume, _, err := client.VolumesApi.GetVolume(ctx, parsedSelfLink[resourceGroupNameTenant], parsedSelfLink[resourceGroupNameTenantSpace], parsedSelfLink[resourceGroupNameVolume], nil)
	if err != nil {
		utilities.TraceError(ctx, err)
		return nil, err
	}

	err = vp.loadVolume(volume, d)
	if err != nil {
		return nil, err
	}

	// Set the ID to the real volume ID
	d.SetId(volume.Id)

	// Volume is not destroyed, no need to recover it
	if !volume.Destroyed {
		return []*schema.ResourceData{d}, nil
	}

	if err := vp.recoverVolume(ctx, volume, client, d); err != nil {
		return nil, err
	}

	return []*schema.ResourceData{d}, nil
}

func (vp *volumeProvider) getSourceLink(ctx context.Context, d *schema.ResourceData) string {
	tenant := rdString(ctx, d, getSourceLinkItem(optionTenant))
	ts := rdString(ctx, d, getSourceLinkItem(optionTenantSpace))

	volume, isVolCopy := d.GetOk(getSourceLinkItem(optionVolume))
	if isVolCopy {
		// Copy from volume
		return fmt.Sprintf("/tenants/%s/tenant-spaces/%s/volumes/%s", tenant, ts, volume.(string))
	}

	// Copy from snapshot
	snapshot := rdString(ctx, d, getSourceLinkItem(optionSnapshot))
	volumeSnapshot := rdString(ctx, d, getSourceLinkItem(optionVolumeSnapshot))

	return fmt.Sprintf(
		"/tenants/%s/tenant-spaces/%s/snapshots/%s/volume-snapshots/%s", tenant, ts, snapshot, volumeSnapshot,
	)
}

func (vp *volumeProvider) loadVolume(volume hmrest.Volume, d *schema.ResourceData) error {
	hostNames := []string{}
	for _, hap := range volume.HostAccessPolicies {
		hostNames = append(hostNames, hap.Name)
	}
	err := getFirstError(
		d.Set(optionHostAccessPolicies, hostNames),
		d.Set(optionTenant, volume.Tenant.Name),
		d.Set(optionTenantSpace, volume.TenantSpace.Name),
		d.Set(optionStorageClass, volume.StorageClass.Name),
		d.Set(optionPlacementGroup, volume.PlacementGroup.Name),
		d.Set(optionName, volume.Name),
		d.Set(optionDisplayName, volume.DisplayName),
		d.Set(optionSize, strconv.FormatInt(volume.Size, 10)),
		d.Set(optionSerialNumber, volume.SerialNumber),
		d.Set(optionCreatedAt, volume.CreatedAt),
		d.Set(optionProtectionPolicy, nil),
		d.Set(optionTargetIscsiIqn, nil),
		d.Set(optionTargetIscsiAddresses, nil),
	)
	if volume.ProtectionPolicy != nil {
		err = getFirstError(err, d.Set(optionProtectionPolicy, volume.ProtectionPolicy.Name))
	}

	if volume.Target != nil && volume.Target.Iscsi != nil {
		err = getFirstError(err,
			d.Set(optionTargetIscsiIqn, volume.Target.Iscsi.Iqn),
			d.Set(optionTargetIscsiAddresses, volume.Target.Iscsi.Addresses),
		)
	}
	return err
}

func (vp *volumeProvider) recoverVolume(
	ctx context.Context, volume hmrest.Volume, client *hmrest.APIClient, d *schema.ResourceData,
) error {
	body := hmrest.VolumePatch{Destroyed: &hmrest.NullableBoolean{Value: false}}
	op, _, err := client.VolumesApi.UpdateVolume(ctx, body, volume.Tenant.Name, volume.TenantSpace.Name, volume.Name, nil)
	if err != nil {
		utilities.TraceError(ctx, err)
		return err
	}

	succeeded, err := utilities.WaitOnOperation(ctx, &op, client)
	if err != nil {
		utilities.TraceError(ctx, err)
		return err
	}

	if !succeeded {
		tflog.Error(ctx, "REST recover failed", "error_message", op.Error_.Message, "PureCode", op.Error_.PureCode, "HttpCode", op.Error_.HttpCode)
		return errors.New(op.Error_.Message)
	}

	return nil
}

func getSourceLinkItem(optionName string) string {
	return fmt.Sprintf("%s.0.%s", optionSourceLink, optionName)
}
