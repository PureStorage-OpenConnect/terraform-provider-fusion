/*
Copyright 2023 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package fusion

import (
	"context"

	"github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/utilities"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"
)

type hostAccessPolicyDataSource struct{}

// This is our entry point for the Host Access Policy data source
func dataSourceHostAccessPolicy() *schema.Resource {
	ds := &hostAccessPolicyDataSource{}

	dsSchema := map[string]*schema.Schema{
		optionItems: {
			Type:     schema.TypeList,
			Computed: true,
			Elem: &schema.Resource{
				Schema: schemaHostAccessPolicy(),
			},
			Description: "List of matching Host Access Policies.",
		},
	}

	hostAccessPolicyDataSourceFunctions := NewBaseDataSourceFunctions(resourceKindHostAccessPolicy, ds, dsSchema)

	return hostAccessPolicyDataSourceFunctions.Resource
}

func (ds *hostAccessPolicyDataSource) ReadDataSource(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) (err error) {
	resp, _, err := client.HostAccessPoliciesApi.ListHostAccessPolicies(ctx, nil)
	if err != nil {
		return err
	}

	hostAccessPolicyList := make([]map[string]interface{}, 0, resp.Count)

	for _, hap := range resp.Items {
		hostAccessPolicyList = append(hostAccessPolicyList, map[string]interface{}{
			optionName:        hap.Name,
			optionDisplayName: hap.DisplayName,
			optionIqn:         hap.Iqn,
			optionPersonality: hap.Personality,
		})
	}

	if err := d.Set(optionItems, hostAccessPolicyList); err != nil {
		return err
	}

	d.SetId(utilities.GetIdForDataSource())

	return nil
}
