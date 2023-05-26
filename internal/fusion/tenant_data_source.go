/*
Copyright 2023 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package fusion

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"

	"github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/utilities"
)

// Implements DataSource
type tenantDataSource struct{}

// This is our entry point for the Tenant data source
func dataSourceTenant() *schema.Resource {
	ds := &tenantDataSource{}

	dsSchema := map[string]*schema.Schema{
		optionItems: {
			Type:     schema.TypeList,
			Computed: true,
			Elem: &schema.Resource{
				Schema: schemaTenant(),
			},
		},
	}

	tenantDataSourceFunctions := NewBaseDataSourceFunctions("Tenant", ds, dsSchema)
	return tenantDataSourceFunctions.Resource
}

func (ds *tenantDataSource) ReadDataSource(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) error {
	resp, _, err := client.TenantsApi.ListTenants(ctx, nil)
	if err != nil {
		return err
	}

	tenantList := make([]map[string]interface{}, 0, resp.Count)

	for _, tenant := range resp.Items {
		tenantList = append(tenantList, map[string]interface{}{
			optionName:        tenant.Name,
			optionDisplayName: tenant.DisplayName,
		})
	}

	if err := d.Set(optionItems, tenantList); err != nil {
		return err
	}

	d.SetId(utilities.GetIdForDataSource())

	return nil
}
