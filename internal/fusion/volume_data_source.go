/*
Copyright 2023 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package fusion

import (
	context "context"
	"strconv"

	"github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/utilities"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"
)

type volumeDataSource struct{}

func dataSourceVolume() *schema.Resource {
	volumeSchema := schemaVolume()

	// self-references in a schema break when nested into a data source
	// data sourced volumes are not direct user input anyways so these
	// checks can be silently dropped
	volumeSchema[optionSize].ConflictsWith = nil
	volumeSchema[optionSourceLink].ConflictsWith = nil
	linkSchema := volumeSchema[optionSourceLink].Elem.(*schema.Resource).Schema
	linkSchema[optionSnapshot].RequiredWith = nil
	linkSchema[optionSnapshot].ConflictsWith = nil
	linkSchema[optionVolumeSnapshot].RequiredWith = nil
	linkSchema[optionVolumeSnapshot].ConflictsWith = nil
	linkSchema[optionVolume].ConflictsWith = nil

	ds := &volumeDataSource{}

	dsSchema := map[string]*schema.Schema{
		optionTenant: {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "Tenant to list Volumes from.",
		},
		optionTenantSpace: {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "Tenant space to list Volumes from.",
		},
		optionItems: {
			Type:     schema.TypeList,
			Computed: true,
			Elem: &schema.Resource{
				Schema: volumeSchema,
			},
			Description: "List of matching Volumes.",
		},
	}

	volumeDataSourceFunctions := NewBaseDataSourceFunctions(resourceKindVolume, ds, dsSchema)

	return volumeDataSourceFunctions.Resource
}

func (ds *volumeDataSource) ReadDataSource(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) error {
	tenant := d.Get(optionTenant).(string)
	tenantSpace := d.Get(optionTenantSpace).(string)
	resp, _, err := client.VolumesApi.ListVolumes(ctx, tenant, tenantSpace, nil)
	if err != nil {
		return err
	}

	volumesList := make([]map[string]interface{}, 0, resp.Count)

	for _, vol := range resp.Items {
		volInfo := map[string]interface{}{
			optionName:           vol.Name,
			optionTenant:         vol.Tenant.Name,
			optionTenantSpace:    vol.TenantSpace.Name,
			optionStorageClass:   vol.StorageClass.Name,
			optionPlacementGroup: vol.PlacementGroup.Name,
			optionDisplayName:    vol.DisplayName,
			optionSize:           strconv.FormatInt(vol.Size, 10),
			optionSerialNumber:   vol.SerialNumber,
			optionCreatedAt:      vol.CreatedAt,
		}
		hostNames := []string{}
		for _, hap := range vol.HostAccessPolicies {
			hostNames = append(hostNames, hap.Name)
		}
		volInfo[optionHostAccessPolicies] = hostNames

		if vol.ProtectionPolicy != nil {
			volInfo[optionProtectionPolicy] = vol.ProtectionPolicy.Name
		}
		if vol.Target != nil {
			if vol.Target.Iscsi != nil {
				volInfo[optionTargetIscsiIqn] = vol.Target.Iscsi.Iqn
				volInfo[optionTargetIscsiAddresses] = vol.Target.Iscsi.Addresses
			}
		}

		volumesList = append(volumesList, volInfo)
	}

	if err := d.Set(optionItems, volumesList); err != nil {
		return err
	}

	d.SetId(utilities.GetIdForDataSource())

	return nil
}
