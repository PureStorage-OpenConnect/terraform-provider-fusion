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
	"github.com/antihax/optional"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"
)

// Implements DataSource
type volumeSnapshotDataSource struct{}

// This is our entry point for the VolumeSnapshot data source
func dataSourceVolumeSnapshot() *schema.Resource {
	ds := &volumeSnapshotDataSource{}

	dsSchema := map[string]*schema.Schema{
		optionSnapshot: {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "The name of Snapshot.",
		},
		optionTenant: {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "The name of Tenant.",
		},
		optionTenantSpace: {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "The name of Tenant Space.",
		},
		optionCreatedAt: {
			Type:             schema.TypeString,
			Optional:         true,
			ValidateDiagFunc: utilities.StringIsInt64,
			Description:      "The Volume Snapshot creation time. Measured in milliseconds since the UNIX epoch.",
		},
		optionVolumeId: {
			Type:         schema.TypeString,
			Optional:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "ID of the Volume.",
		},
		optionProtectionPolicyId: {
			Type:         schema.TypeString,
			Optional:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "ID of the Protection Policy.",
		},
		optionPlacementGroupId: {
			Type:         schema.TypeString,
			Optional:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "ID of the Placement Group.",
		},
		optionItems: {
			Type:     schema.TypeList,
			Computed: true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					optionName: {
						Type:        schema.TypeString,
						Required:    true,
						Description: "The name of Volume Snapshot.",
					},
					optionDisplayName: {
						Type:        schema.TypeString,
						Optional:    true,
						Description: "The human-readable name of the Volume Snapshot.",
					},
					optionSize: {
						Type:        schema.TypeString,
						Required:    true,
						Description: "Size of the Volume Snapshot in bytes.",
					},
					optionTenant: {
						Type:        schema.TypeString,
						Required:    true,
						Description: "The name of Tenant.",
					},
					optionTenantSpace: {
						Type:        schema.TypeString,
						Required:    true,
						Description: "The name of Tenant Space.",
					},
					optionSnapshot: {
						Type:        schema.TypeString,
						Required:    true,
						Description: "The name of the Snapshot.",
					},
					optionVolume: {
						Type:        schema.TypeString,
						Optional:    true,
						Description: "The Volume of Volume Snapshot.",
					},
					optionProtectionPolicy: {
						Type:        schema.TypeString,
						Optional:    true,
						Description: "The name of Protection Policy.",
					},
					optionPlacementGroup: {
						Type:        schema.TypeString,
						Required:    true,
						Description: "The name of Placement Group.",
					},
					optionVolumeSerialNumber: {
						Type:         schema.TypeString,
						Required:     true,
						ValidateFunc: validation.StringIsNotEmpty,
						Description:  "The serial number of Volume.",
					},
					optionSerialNumber: {
						Type:         schema.TypeString,
						Required:     true,
						ValidateFunc: validation.StringIsNotEmpty,
						Description:  "The serial number of Snapshot.",
					},
					optionTimeRemaining: {
						Type:        schema.TypeString,
						Required:    true,
						Description: "Remaining time of Volume Snapshot.",
					},
					optionConsistencyId: {
						Type:        schema.TypeString,
						Required:    true,
						Description: "Consistency ID.",
					},
					optionCreatedAt: {
						Type:             schema.TypeString,
						Required:         true,
						ValidateDiagFunc: utilities.StringIsInt64,
						Description:      "The Volume Snapshot creation time measured in milliseconds.",
					},
					optionDestroyed: {
						Type:        schema.TypeBool,
						Optional:    true,
						Description: "Whether the Volume Snapshot is destroyed.",
					},
				},
			},
			Description: "List of matching Volume Snapshots.",
		},
	}

	volumeSnapshotDataSourceFunctions := NewBaseDataSourceFunctions(resourceKindVolumeSnapshot, ds, dsSchema)
	// Override default description as there's no resource for this data source.
	volumeSnapshotDataSourceFunctions.Resource.Description = "Provides details about any Volume Snapshot matching the given parameters."

	return volumeSnapshotDataSourceFunctions.Resource
}

func (ds *volumeSnapshotDataSource) ReadDataSource(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) error {
	tenant, _ := d.Get(optionTenant).(string)
	tenantSpace, _ := d.Get(optionTenantSpace).(string)
	snapshot, _ := d.Get(optionSnapshot).(string)
	createdAtString, _ := d.Get(optionCreatedAt).(string)
	protectionPolicyId, _ := d.Get(optionProtectionPolicyId).(string)
	placementGroupId, _ := d.Get(optionPlacementGroupId).(string)
	volumeId, _ := d.Get(optionVolumeId).(string)

	var localOpts hmrest.VolumeSnapshotsApiListVolumeSnapshotsOpts
	if protectionPolicyId != "" {
		localOpts.ProtectionPolicyId = optional.NewString(protectionPolicyId)
	}
	if placementGroupId != "" {
		localOpts.PlacementGroupId = optional.NewString(placementGroupId)
	}
	if volumeId != "" {
		localOpts.VolumeId = optional.NewString(volumeId)
	}
	if createdAtString != "" {
		createdAt, err := strconv.ParseInt(createdAtString, 10, 64)
		if err != nil {
			return fmt.Errorf("%s must be a number", optionCreatedAt)
		}
		localOpts.CreatedAt = optional.NewInt64(createdAt)
	}

	resp, _, err := client.VolumeSnapshotsApi.ListVolumeSnapshots(ctx, tenant, tenantSpace, snapshot, &localOpts)
	if err != nil {
		return err
	}
	volumeSnapshotList := make([]map[string]interface{}, resp.Count)

	for i, volumeSnapshot := range resp.Items {
		volumeSnapshotList[i] = map[string]interface{}{
			optionName:               volumeSnapshot.Name,
			optionDisplayName:        volumeSnapshot.DisplayName,
			optionTenant:             volumeSnapshot.Tenant.Name,
			optionTenantSpace:        volumeSnapshot.TenantSpace.Name,
			optionCreatedAt:          strconv.FormatInt(volumeSnapshot.CreatedAt, 10),
			optionSize:               strconv.FormatInt(volumeSnapshot.Size, 10),
			optionSerialNumber:       volumeSnapshot.SerialNumber,
			optionSnapshot:           volumeSnapshot.Snapshot.Name,
			optionVolumeSerialNumber: volumeSnapshot.VolumeSerialNumber,
			optionConsistencyId:      volumeSnapshot.ConsistencyId,
			optionTimeRemaining:      strconv.FormatInt(volumeSnapshot.TimeRemaining, 10),
			optionDestroyed:          volumeSnapshot.Destroyed,
			optionPlacementGroup:     volumeSnapshot.PlacementGroup.Name,
		}
		if volumeSnapshot.ProtectionPolicy != nil {
			volumeSnapshotList[i][optionProtectionPolicy] = volumeSnapshot.ProtectionPolicy.Name
		}
		if volumeSnapshot.Volume != nil {
			volumeSnapshotList[i][optionVolume] = volumeSnapshot.Volume.Name
		}
	}

	if err := d.Set(optionItems, volumeSnapshotList); err != nil {
		return err
	}

	d.SetId(utilities.GetIdForDataSource())

	return nil
}
