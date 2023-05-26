/*
Copyright 2023 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package fusion

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	"github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/utilities"
	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"
)

// Implements DataSource
type tenantSpaceDataSource struct{}

// This is our entry point for the Tenant Space data source
func dataSourceTenantSpace() *schema.Resource {
	ds := &tenantSpaceDataSource{}

	dsSchema := map[string]*schema.Schema{
		optionTenant: {
			Type:     schema.TypeString,
			Required: true,
		},
		optionItems: {
			Type:     schema.TypeList,
			Computed: true,
			Elem: &schema.Resource{
				Schema: schemaTenantSpace(),
			},
		},
	}

	tenantSpaceDataSourceFunctions := NewBaseDataSourceFunctions("TenantSpace", ds, dsSchema)

	return tenantSpaceDataSourceFunctions.Resource
}

func (ds *tenantSpaceDataSource) ReadDataSource(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) error {
	resp, _, err := client.TenantSpacesApi.ListTenantSpaces(ctx, rdString(ctx, d, optionTenant), nil)
	if err != nil {
		return err
	}

	tenantSpacesList := make([]map[string]interface{}, resp.Count)

	for _, ts := range resp.Items {
		tenantSpacesList = append(tenantSpacesList, map[string]interface{}{
			optionName:        ts.Name,
			optionDisplayName: ts.DisplayName,
			optionTenant:      ts.Tenant.Name,
		})
	}

	if err := d.Set(optionItems, tenantSpacesList); err != nil {
		return err
	}

	d.SetId(utilities.GetIdForDataSource())

	return nil
}
