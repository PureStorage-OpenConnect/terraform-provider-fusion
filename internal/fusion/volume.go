/*
Copyright 2022 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package fusion

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"
	"github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/utilities"
)

var volumeResourceFunctions *BaseResourceFunctions

// This is our entry point for the Volume resource. Get it movin'
func resourceVolume() *schema.Resource {
	vp := &volumeProvider{BaseResourceProvider{ResourceKind: "Volume"}}
	volumeResourceFunctions = NewBaseResourceFunctions("Volume", vp)

	volumeResourceFunctions.Resource.Schema = map[string]*schema.Schema{
		"name": {
			Type:     schema.TypeString,
			Required: true,
		},
		"display_name": {
			Type:     schema.TypeString,
			Optional: true,
			Computed: true,
		},
		"size": {
			Type:     schema.TypeInt,
			Required: true,
		},
		"tenant_name": {
			Type:     schema.TypeString,
			Required: true,
		},
		"tenant_space_name": {
			Type:     schema.TypeString,
			Required: true,
		},
		"storage_class_name": {
			Type:     schema.TypeString,
			Required: true,
		},
		"placement_group_name": {
			Type:        schema.TypeString,
			Required:    true,
			Description: "WARNING: Changing this value will cause a new IQN number to be generated and will disrupt initator access to this volume",
		},
		"protection_policy_name": {
			Type:     schema.TypeString,
			Optional: true,
		},
		"host_names": {
			Type:     schema.TypeSet,
			Required: true,
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
		},
		"created_at": {
			Type:     schema.TypeInt,
			Computed: true,
		},
		"serial_number": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"target_iscsi_iqn": {
			Type:     schema.TypeString,
			Computed: true,
		},
		"target_iscsi_addresses": {
			Type:     schema.TypeSet,
			Computed: true,
			Elem: &schema.Schema{
				Type: schema.TypeString,
			},
		},
	}

	return volumeResourceFunctions.Resource
}

// Implements ResourceProvider
type volumeProvider struct {
	BaseResourceProvider
}

func (vp *volumeProvider) PrepareCreate(ctx context.Context, d *schema.ResourceData) (InvokeWriteAPI, ResourcePost, error) {
	tenantName := rdString(ctx, d, "tenant_name")
	tenantSpaceName := rdString(ctx, d, "tenant_space_name")
	name := rdString(ctx, d, "name")
	displayName := rdStringDefault(ctx, d, "display_name", name)

	body := hmrest.VolumePost{
		Name:             name,
		DisplayName:      displayName,
		Size:             int64(rdInt(d, "size")),
		StorageClass:     rdString(ctx, d, "storage_class_name"),
		PlacementGroup:   rdString(ctx, d, "placement_group_name"),
		ProtectionPolicy: rdString(ctx, d, "protection_policy_name"),
	}

	fn := func(ctx context.Context, client *hmrest.APIClient, body RequestSpec) (*hmrest.Operation, error) {
		op, _, err := client.VolumesApi.CreateVolume(ctx, *body.(*hmrest.VolumePost), tenantName, tenantSpaceName, nil)
		return &op, err
	}
	return fn, &body, nil
}

func (vp *volumeProvider) ReadResource(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) error {
	vol, _, err := client.VolumesApi.GetVolumeById(ctx, d.Id(), nil)
	if err != nil {
		return err
	}

	hostNames := []string{}
	for _, hap := range vol.HostAccessPolicies {
		hostNames = append(hostNames, hap.Name)
	}
	err = d.Set("host_names", hostNames)
	if err != nil {
		return err
	}

	d.Set("tenant_name", vol.Tenant.Name)
	d.Set("tenant_space_name", vol.TenantSpace.Name)
	d.Set("storage_class_name", vol.StorageClass.Name)
	d.Set("placement_group_name", vol.PlacementGroup.Name)
	d.Set("name", vol.Name)
	d.Set("display_name", vol.DisplayName)
	d.Set("size", vol.Size)
	d.Set("serial_number", vol.SerialNumber)
	d.Set("created_at", vol.CreatedAt)
	if vol.ProtectionPolicy != nil {
		d.Set("protection_policy_name", vol.ProtectionPolicy.Name)
	}
	if vol.Target != nil {
		if vol.Target.Iscsi != nil {
			d.Set("target_iscsi_iqn", vol.Target.Iscsi.Iqn)
			d.Set("target_iscsi_addresses", vol.Target.Iscsi.Addresses)
		}
	}
	return nil
}

// ColumeProvider.PrepareUpdate will update the attributes of the volume.
//
// If a new size is provided, it must be larger than the current size.  Only
// extending volumes is supported at this time, since truncating volumes can
// lead to data loss.
func (vp *volumeProvider) PrepareUpdate(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) (InvokeWriteAPI, []ResourcePatch, error) {
	volumeName := d.Get("name").(string)
	tenantSpaceName := d.Get("tenant_space_name").(string)
	tenantName := d.Get("tenant_name").(string)

	var patches []ResourcePatch // []*hmrest.VolumePatch

	if d.HasChange("display_name") {
		displayName := d.Get("display_name").(string)
		tflog.Trace(ctx, "update",
			"resource", "volume",
			"parameter", "display_name",
			"to", displayName,
			"patch_idx", len(patches),
		)
		patches = append(patches, &hmrest.VolumePatch{
			DisplayName: &hmrest.NullableString{Value: displayName},
		})
	}

	if d.HasChange("protection_policy_name") {
		protectionPolicyName := d.Get("protection_policy_name").(string)
		tflog.Trace(ctx, "update",
			"resource", "volume",
			"parameter", "protection_policy_name",
			"to", protectionPolicyName,
			"patch_idx", len(patches),
		)
		patches = append(patches, &hmrest.VolumePatch{
			ProtectionPolicy: &hmrest.NullableString{Value: protectionPolicyName},
		})
	}

	// if there is a change to placement groups, then we need to remove the hosts and then re-add them
	reAddHosts := false
	if d.HasChange("placement_group_name") {
		reAddHosts = true
		tflog.Trace(ctx, "update",
			"resource", "volume",
			"parameter", "host_names",
			"to", "",
			"patch_idx", len(patches),
			"message", "temporary removal of hosts for placement_groups_name change",
		)
		patches = append(patches, &hmrest.VolumePatch{
			HostAccessPolicies: &hmrest.NullableString{Value: ""},
		})
	}

	if d.HasChange("storage_class_name") || d.HasChange("placement_group_name") {
		patch := &hmrest.VolumePatch{}
		if d.HasChange("storage_class_name") {
			storageClassName := d.Get("storage_class_name").(string)
			tflog.Trace(ctx, "update",
				"resource", "volume",
				"parameter", "storage_class_name",
				"to", storageClassName,
				"patch_idx", len(patches),
			)
			patch.StorageClass = &hmrest.NullableString{Value: storageClassName}
		}
		if d.HasChange("placement_group_name") {
			placementGroupName := d.Get("placement_group_name").(string)
			tflog.Trace(ctx, "update",
				"resource", "volume",
				"parameter", "placement_group_name",
				"to", placementGroupName,
				"patch_idx", len(patches),
			)
			patch.PlacementGroup = &hmrest.NullableString{Value: placementGroupName}
		}
		patches = append(patches, patch)
	}

	if d.HasChange("host_names") || reAddHosts {
		s := ""
		for idx, item := range d.Get("host_names").(*schema.Set).List() {
			if idx != 0 {
				s += ","
			}
			s += item.(string)
		}
		tflog.Trace(ctx, "update",
			"resource", "volume",
			"parameter", "host_names",
			"to", s,
			"patch_idx", len(patches),
			"readded", reAddHosts,
		)
		patches = append(patches, &hmrest.VolumePatch{
			HostAccessPolicies: &hmrest.NullableString{Value: s},
		})
	}

	if d.HasChange("size") {
		size := d.Get("size").(int)
		tflog.Trace(ctx, "update",
			"resource", "volume",
			"parameter", "size",
			"to", size,
			"patch_idx", len(patches),
		)

		patches = append(patches, &hmrest.VolumePatch{
			Size: &hmrest.NullableSize{Value: int64(size)},
		})
	}
	fn := func(ctx context.Context, client *hmrest.APIClient, body RequestSpec) (*hmrest.Operation, error) {
		op, _, err := client.VolumesApi.UpdateVolume(ctx, *body.(*hmrest.VolumePatch), tenantName, tenantSpaceName, volumeName, nil)
		return &op, err
	}

	return fn, patches, nil
}

func (vp *volumeProvider) PrepareDelete(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) (InvokeWriteAPI, error) {
	volumeName := d.Get("name").(string)
	tenantSpaceName := d.Get("tenant_space_name").(string)
	tenantName := d.Get("tenant_name").(string)

	fn := func(ctx context.Context, client *hmrest.APIClient, body RequestSpec) (*hmrest.Operation, error) {
		// TODO: HM-2437 this should be a patch
		tflog.Trace(ctx, "removing host assignments before deleting volume")
		op, _, err := client.VolumesApi.UpdateVolume(ctx, hmrest.VolumePatch{
			HostAccessPolicies: &hmrest.NullableString{Value: ""},
		}, tenantName, tenantSpaceName, volumeName, nil)
		utilities.TraceError(ctx, err)
		if err != nil {
			return &op, err
		}
		succeeded, err := WaitOnOperation(ctx, &op, client)
		if err != nil {
			return &op, err
		}
		if !succeeded {
			tflog.Error(ctx, "failed removing host assignments")
			return &op, fmt.Errorf("failed to clear out host assignments as part of deleting volume")
		}
		tflog.Trace(ctx, "done removing host assignments")

		op, _, err = client.VolumesApi.DeleteVolume(ctx, tenantName, tenantSpaceName, volumeName, nil)
		return &op, err
	}
	return fn, nil
}
