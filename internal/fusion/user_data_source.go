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
type userDataSource struct{}

// This is our entry point for the User data source
func dataSourceUser() *schema.Resource {
	ds := &userDataSource{}

	dsSchema := map[string]*schema.Schema{
		optionItems: {
			Type:     schema.TypeList,
			Computed: true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					optionId: {
						Type:        schema.TypeString,
						Required:    true,
						Description: "An immutable, globally unique, system generated identifier.",
					},
					optionName: {
						Type:        schema.TypeString,
						Required:    true,
						Description: "The User's name.",
					},
					optionDisplayName: {
						Type:        schema.TypeString,
						Required:    true,
						Description: "The human-readable name of the User.",
					},
					optionEmail: {
						Type:        schema.TypeString,
						Required:    true,
						Description: "The email address associated with the User.",
					},
				},
			},
			Description: "List of matching Users.",
		},
		optionEmail: {
			Type:         schema.TypeString,
			Optional:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "The email address associated with the User.",
		},
		optionName: {
			Type:         schema.TypeString,
			Optional:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "The User's name.",
		},
	}

	userDataSourceFunctions := NewBaseDataSourceFunctions(resourceKindUser, ds, dsSchema)
	// Override default description as there's no resource for this data source.
	userDataSourceFunctions.Resource.Description = "Provides details about any User matching the given parameters."

	return userDataSourceFunctions.Resource
}

func (ds *userDataSource) ReadDataSource(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) error {
	var opts hmrest.IdentityManagerApiListUsersOpts

	if email, ok := d.GetOk(optionEmail); ok {
		opts.Email = optional.NewString(email.(string))
	}

	if name, ok := d.GetOk(optionName); ok {
		opts.Name = optional.NewString(name.(string))
	}

	resp, _, err := client.IdentityManagerApi.ListUsers(ctx, &opts)
	if err != nil {
		return err
	}

	userList := make([]map[string]interface{}, 0, len(resp))

	for _, user := range resp {
		userList = append(userList, map[string]interface{}{
			optionId:          user.Id,
			optionName:        user.Name,
			optionDisplayName: user.DisplayName,
			optionEmail:       user.Email,
		})
	}

	if err := d.Set(optionItems, userList); err != nil {
		return err
	}

	d.SetId(utilities.GetIdForDataSource())

	return nil
}
