/*
Copyright 2023 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package fusion

import (
	"context"
	"errors"

	"github.com/antihax/optional"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	"github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/utilities"
	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"
)

// Implements DataSource
type placementGroupDataSource struct{}

// This is our entry point for the Placement Group data source
func dataSourcePlacementGroup() *schema.Resource {
	ds := &placementGroupDataSource{}

	dsSchema := map[string]*schema.Schema{
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
		optionIqn: {
			Type:             schema.TypeString,
			Optional:         true,
			ValidateDiagFunc: IsValidIQN,
			Description:      "The iSCSI qualified name (IQN) associated with the Placement Group.",
		},
		optionItems: {
			Type:     schema.TypeList,
			Computed: true,
			Elem: &schema.Resource{
				Schema: schemaPlacementGroup(),
			},
			Description: "List of matching Placement Groups.",
		},
	}

	placementGroupDataSourceFunctions := NewBaseDataSourceFunctions(resourceKindPlacementGroup, ds, dsSchema)

	return placementGroupDataSourceFunctions.Resource
}

func (ds *placementGroupDataSource) ReadDataSource(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) error {
	var opts hmrest.PlacementGroupsApiListPlacementGroupsOpts
	var requiredAZResourceGroupNames = []string{
		resourceGroupNameRegion,
		resourceGroupNameAvailabilityZone,
	}

	tenant := rdString(ctx, d, optionTenant)
	tenantSpace := rdString(ctx, d, optionTenantSpace)

	if iqn, ok := d.GetOk(optionIqn); ok {
		opts = hmrest.PlacementGroupsApiListPlacementGroupsOpts{
			Iqn: optional.NewString(iqn.(string)),
		}
	}

	resp, _, err := client.PlacementGroupsApi.ListPlacementGroups(ctx, tenant, tenantSpace, &opts)
	if err != nil {
		return err
	}

	pgList := make([]map[string]interface{}, 0, resp.Count)

	for _, pg := range resp.Items {
		parsedSelfLink, err := utilities.ParseSelfLink(pg.AvailabilityZone.SelfLink, requiredAZResourceGroupNames)
		if err != nil {
			err := errors.New("invalid AZ self link, expected format: '/regions/<region>/availability-zones/<availability-zone>'")
			tflog.Error(ctx, "Skipping placement group during listing it's data source", "error", err, "name", pg.Name)
			continue
		}

		pgList = append(pgList, map[string]interface{}{
			optionName:             pg.Name,
			optionDisplayName:      pg.DisplayName,
			optionTenant:           pg.Tenant.Name,
			optionTenantSpace:      pg.TenantSpace.Name,
			optionRegion:           parsedSelfLink[resourceGroupNameRegion],
			optionAvailabilityZone: pg.AvailabilityZone.Name,
			optionStorageService:   pg.StorageService.Name,
			optionArray:            pg.Array.Name,
		})
	}

	if err := d.Set(optionItems, pgList); err != nil {
		return err
	}

	d.SetId(utilities.GetIdForDataSource())

	return nil
}
