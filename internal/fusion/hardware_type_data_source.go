/*
Copyright 2023 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package fusion

import (
	"context"

	"github.com/antihax/optional"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"

	"github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/utilities"
)

// Implements DataSource
type hardwareTypeDataSource struct{}

// This is our entry point for the hardware type data source
func dataSourceHardwareType() *schema.Resource {
	ds := &hardwareTypeDataSource{}

	dsSchema := map[string]*schema.Schema{
		optionArrayType: {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The array type of the Hardware Type.",
		},
		optionMediaType: {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "The media type of the Hardware Type",
		},
		optionItems: {
			Type:     schema.TypeList,
			Computed: true,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					optionName: {
						Type:         schema.TypeString,
						Required:     true,
						ValidateFunc: validation.StringIsNotEmpty,
						Description:  "The name of the hardware type.",
					},
					optionDisplayName: {
						Type:         schema.TypeString,
						Optional:     true,
						ValidateFunc: validation.StringIsNotEmpty,
						Description:  "The human-readable name of the Hardware Type. If not provided, defaults to I(name).",
					},
					optionArrayType: {
						Type:        schema.TypeString,
						Required:    true,
						Description: "The array type of the Hardware Type.",
					},
					optionMediaType: {
						Type:        schema.TypeString,
						Required:    true,
						Description: "The media type of the Hardware Type",
					},
				},
			},
			Description: "List of matching Hardware Types.",
		},
	}

	hardwareTypeDataSourceFunctions := NewBaseDataSourceFunctions(resourceKindHardwareType, ds, dsSchema)
	// Override default description as there's no resource for this data source.
	hardwareTypeDataSourceFunctions.Resource.Description = "Provides details about any Hardware Type matching the given parameters."
	return hardwareTypeDataSourceFunctions.Resource
}

func (ds *hardwareTypeDataSource) ReadDataSource(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) error {
	options := hmrest.HardwareTypesApiListHardwareTypesOpts{}

	options.MediaType = optional.NewString(rdString(ctx, d, optionMediaType))
	options.ArrayType = optional.NewString(rdString(ctx, d, optionArrayType))

	resp, _, err := client.HardwareTypesApi.ListHardwareTypes(ctx, &options)
	if err != nil {
		return err
	}

	hwTypesList := make([]map[string]interface{}, 0, resp.Count)

	for _, hwType := range resp.Items {
		hwTypesList = append(hwTypesList, map[string]interface{}{
			optionName:        hwType.Name,
			optionDisplayName: hwType.DisplayName,
			optionArrayType:   hwType.ArrayType,
			optionMediaType:   hwType.MediaType,
		})
	}

	if err := d.Set(optionItems, hwTypesList); err != nil {
		return err
	}

	d.SetId(utilities.GetIdForDataSource())

	return nil
}
