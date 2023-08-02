/*
Copyright 2023 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package fusion

import (
	"context"

	"github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/utilities"
	"github.com/antihax/optional"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"
)

// Implements DataSource
type roleDataSource struct{}

// This is our entry point for the Role data source
func dataSourceRole() *schema.Resource {
	ds := &roleDataSource{}

	dsSchema := map[string]*schema.Schema{
		optionItems: {
			Type:     schema.TypeList,
			Computed: true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					optionName: {
						Type:        schema.TypeString,
						Required:    true,
						Description: "The Role name.",
					},
					optionDisplayName: {
						Type:        schema.TypeString,
						Required:    true,
						Description: "The human-readable name of the Role.",
					},
					optionDescription: {
						Type:        schema.TypeString,
						Required:    true,
						Description: "A description of the Role's capabilities.",
					},
					optionAssignableScopes: {
						Type:     schema.TypeList,
						Required: true,
						Elem: &schema.Schema{
							Type:         schema.TypeString,
							ValidateFunc: validation.StringIsNotEmpty,
						},
						Description: "A list of resource kinds the Role can be scoped to.",
					},
				},
			},
			Description: "List of matching Roles.",
		},
		optionAssignableScope: {
			Type:         schema.TypeString,
			Optional:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "Resource kind the Role is scoped to.",
		},
	}

	roleDataSourceFunctions := NewBaseDataSourceFunctions(resourceKindRole, ds, dsSchema)
	// Override default description as there's no resource for this data source.
	roleDataSourceFunctions.Resource.Description = "Provides details about any Role matching the given parameters."
	return roleDataSourceFunctions.Resource
}

func (ds *roleDataSource) ReadDataSource(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) error {
	var opts hmrest.RolesApiListRolesOpts

	if scope, ok := d.GetOk(optionAssignableScope); ok {
		opts = hmrest.RolesApiListRolesOpts{
			AssignableScope: optional.NewString(scope.(string)),
		}
	}

	resp, _, err := client.RolesApi.ListRoles(ctx, &opts)
	if err != nil {
		return err
	}

	rolesList := make([]map[string]interface{}, 0, len(resp))

	for _, role := range resp {
		rolesList = append(rolesList, map[string]interface{}{
			optionName:             role.Name,
			optionDisplayName:      role.DisplayName,
			optionDescription:      role.Description,
			optionAssignableScopes: role.AssignableScopes,
		})
	}

	if err := d.Set(optionItems, rolesList); err != nil {
		return err
	}

	d.SetId(utilities.GetIdForDataSource())

	return nil
}
