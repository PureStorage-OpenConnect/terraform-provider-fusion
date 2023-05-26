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
			Description: "Hardware type array type",
		},
		optionMediaType: {
			Type:        schema.TypeString,
			Optional:    true,
			Description: "Hardware type media type",
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
						Description:  "The human name of the hardware type. If not provided, defaults to I(name).",
					},
					optionArrayType: {
						Type:        schema.TypeString,
						Required:    true,
						Description: "Hardware type array type",
					},
					optionMediaType: {
						Type:        schema.TypeString,
						Required:    true,
						Description: "Hardware type media type",
					},
				},
			},
		},
	}

	hardwareTypeDataSourceFunctions := NewBaseDataSourceFunctions("HardwareType", ds, dsSchema)
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
