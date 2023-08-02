/*
Copyright 2023 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package fusion

import (
	context "context"

	"github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/utilities"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"
)

type arrayDataSource struct{}

func dataSourceArray() *schema.Resource {
	array := NewBaseDataSourceFunctions(resourceKindArray, &arrayDataSource{},
		map[string]*schema.Schema{
			optionItems: {
				Type:     schema.TypeList,
				Computed: true,
				Elem: &schema.Resource{
					Schema: createArraySchema(),
				},
				Description: "List of matching Arrays.",
			},
			optionAvailabilityZone: {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
				Description:  "The name of Availability Zone within which the Array is created.",
			},
			optionRegion: {
				Type:         schema.TypeString,
				Required:     true,
				ValidateFunc: validation.StringIsNotEmpty,
				Description:  "The name of Region within which the Array is created.",
			},
		})
	return array.Resource
}

func (ds *arrayDataSource) ReadDataSource(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) error {
	region := d.Get(optionRegion).(string)
	availabilityZone := d.Get(optionAvailabilityZone).(string)
	listing, _, err := client.ArraysApi.ListArrays(ctx, region, availabilityZone, nil)
	if err != nil {
		return err
	}

	arraysList := make([]map[string]interface{}, 0, listing.Count)

	for _, array := range listing.Items {
		arraysList = append(arraysList, map[string]interface{}{
			optionName:             array.Name,
			optionDisplayName:      array.DisplayName,
			optionAvailabilityZone: array.AvailabilityZone.Name,
			optionRegion:           array.Region.Name,
			optionApplianceId:      array.ApplianceId,
			optionHostName:         array.HostName,
			optionHardwareType:     array.HardwareType.Name,
			optionApartmentId:      array.ApartmentId,
			optionMaintenanceMode:  array.MaintenanceMode,
			optionUnavailableMode:  array.UnavailableMode,
		})
	}

	if err := d.Set("items", arraysList); err != nil {
		return err
	}

	d.SetId(utilities.GetIdForDataSource())

	return nil
}
