/*
Copyright 2023 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package fusion

import (
	"context"
	"strconv"

	"github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/utilities"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"
)

// Implements DataSource
type protectionPolicyDataSource struct{}

// This is our entry point for the Protection Policy data source
func dataSourceProtectionPolicy() *schema.Resource {
	ds := &protectionPolicyDataSource{}

	dsSchema := map[string]*schema.Schema{
		optionItems: {
			Type:     schema.TypeList,
			Computed: true,
			Elem: &schema.Resource{
				Schema: schemaProtectionPolicy(),
			},
			Description: "List of matching Protection Policies.",
		},
	}

	protectionPolicyDataSourceFunctions := NewBaseDataSourceFunctions(resourceKindProtectionPolicy, ds, dsSchema)

	return protectionPolicyDataSourceFunctions.Resource
}

func (ds *protectionPolicyDataSource) ReadDataSource(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) error {
	resp, _, err := client.ProtectionPoliciesApi.ListProtectionPolicies(ctx, nil)
	if err != nil {
		return err
	}

	protectionPolicyesList := make([]map[string]interface{}, resp.Count)

	for i, pp := range resp.Items {
		var rpoValue, retentionValue int = 0, 0

		for _, obj := range pp.Objectives {
			if rpo, ok := obj.(*hmrest.Rpo); ok {
				rpoValue, _ = utilities.StringISO8601MinutesToInt(rpo.Rpo)
				continue
			}

			if retention, ok := obj.(*hmrest.Retention); ok {
				retentionValue, _ = utilities.StringISO8601MinutesToInt(retention.After)
				continue
			}
		}

		protectionPolicyesList[i] = map[string]interface{}{
			optionName:           pp.Name,
			optionDisplayName:    pp.DisplayName,
			optionLocalRPO:       rpoValue,
			optionLocalRetention: strconv.Itoa(retentionValue),
		}
	}

	if err := d.Set(optionItems, protectionPolicyesList); err != nil {
		return err
	}

	d.SetId(utilities.GetIdForDataSource())

	return nil
}
