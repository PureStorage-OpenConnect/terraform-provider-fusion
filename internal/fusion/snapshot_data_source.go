/*
Copyright 2023 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package fusion

import (
	"context"
	"strconv"

	"github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/utilities"
	"github.com/antihax/optional"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"
)

// Implements DataSource
type snapshotDataSource struct{}

// This is our entry point for the Snapshot data source
func dataSourceSnapshot() *schema.Resource {
	ds := &snapshotDataSource{}

	dsSchema := map[string]*schema.Schema{
		optionVolume: {
			Type:          schema.TypeString,
			Optional:      true,
			ConflictsWith: []string{optionPlacementGroup},
			ValidateFunc:  validation.StringIsNotEmpty,
			Description:   "The name of the Volume for Snapshot creation.",
		},
		optionPlacementGroup: {
			Type:          schema.TypeString,
			Optional:      true,
			ConflictsWith: []string{optionVolume},
			ValidateFunc:  validation.StringIsNotEmpty,
			Description:   "The name of the Placement Group for Snapshot creation.",
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
		optionProtectionPolicyId: {
			Type:         schema.TypeString,
			Optional:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "ID of the Protection Policy.",
		},
		optionItems: {
			Type:     schema.TypeList,
			Computed: true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					optionName: {
						Type:        schema.TypeString,
						Required:    true,
						Description: "The name of the Snapshot.",
					},
					optionDisplayName: {
						Type:        schema.TypeString,
						Optional:    true,
						Description: "The human-readable name of the Snapshot.",
					},
					optionTenant: {
						Type:        schema.TypeString,
						Required:    true,
						Description: "The name of the Tenant.",
					},
					optionTenantSpace: {
						Type:        schema.TypeString,
						Required:    true,
						Description: "The name of the Tenant Space.",
					},
					optionProtectionPolicy: {
						Type:        schema.TypeString,
						Optional:    true,
						Description: "The name of the Protection Policy.",
					},
					optionTimeRemaining: {
						Type:        schema.TypeString,
						Optional:    true,
						Description: "Remaining time of Snapshot. Only relevant if destroyed == true.",
					},
					optionDestroyed: {
						Type:        schema.TypeBool,
						Required:    true,
						Description: "Whether the Snapshot is destroyed.",
					},
				},
			},
			Description: "List of matching Snapshots.",
		},
	}

	snapshotDataSourceFunctions := NewBaseDataSourceFunctions(resourceKindSnapshot, ds, dsSchema)
	// Override default description as there's no resource for this data source.
	snapshotDataSourceFunctions.Resource.Description = "Provides details about any Snapshot matching the given parameters."

	return snapshotDataSourceFunctions.Resource
}

func (ds *snapshotDataSource) ReadDataSource(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) error {
	tenant, _ := d.Get(optionTenant).(string)
	tenantSpace, _ := d.Get(optionTenantSpace).(string)
	volume, _ := d.Get(optionVolume).(string)
	placementGroup, _ := d.Get(optionPlacementGroup).(string)
	protectionPolicyId, _ := d.Get(optionProtectionPolicyId).(string)

	opts := hmrest.SnapshotsApiListSnapshotsOpts{}
	if volume != "" {
		opts.Volume = optional.NewString(volume)
	}
	if placementGroup != "" {
		opts.PlacementGroup = optional.NewString(placementGroup)
	}
	if protectionPolicyId != "" {
		opts.ProtectionPolicyId = optional.NewString(protectionPolicyId)
	}

	resp, _, err := client.SnapshotsApi.ListSnapshots(ctx, tenant, tenantSpace, &opts)
	if err != nil {
		return err
	}

	snapshotList := make([]map[string]interface{}, resp.Count)

	for i, snapshot := range resp.Items {
		snapshotList[i] = map[string]interface{}{
			optionName:        snapshot.Name,
			optionDisplayName: snapshot.DisplayName,
			optionTenant:      snapshot.Tenant.Name,
			optionTenantSpace: snapshot.TenantSpace.Name,
			optionDestroyed:   snapshot.Destroyed,
		}
		if snapshot.Destroyed {
			snapshotList[i][optionTimeRemaining] = strconv.FormatInt(snapshot.TimeRemaining, 10)
		}
		if snapshot.ProtectionPolicy != nil {
			snapshotList[i][optionProtectionPolicy] = snapshot.ProtectionPolicy.Name
		}
	}

	if err := d.Set(optionItems, snapshotList); err != nil {
		return err
	}

	d.SetId(utilities.GetIdForDataSource())

	return nil
}
