/*
Copyright 2023 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package fusion

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"
)

// Implements ResourceProvider
type availabilityZoneProvider struct {
	BaseResourceProvider
}

// This is our entry point for the Availability Zone resource.
func resourceAvailabilityZone() *schema.Resource {
	p := &availabilityZoneProvider{BaseResourceProvider{ResourceKind: "AvailabilityZone"}}
	availabilityZoneResourceFunctions := NewBaseResourceFunctions("AvailabilityZone", p)

	availabilityZoneResourceFunctions.Resource.Schema = map[string]*schema.Schema{
		"name": {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "The name of the Availability Zone.",
		},
		"display_name": {
			Type:         schema.TypeString,
			Optional:     true,
			Computed:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "The human name of the Availability Zone. If not provided, defaults to I(name).",
		},
		"region": {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "Region within which the AZ is created.",
		},
	}

	return availabilityZoneResourceFunctions.Resource
}

func (p *availabilityZoneProvider) PrepareCreate(ctx context.Context, d *schema.ResourceData) (InvokeWriteAPI, ResourcePost, error) {
	region := rdString(ctx, d, "region")
	name := rdString(ctx, d, "name")
	displayName := rdStringDefault(ctx, d, "display_name", name)

	body := hmrest.AvailabilityZonePost{
		Name:        name,
		DisplayName: displayName,
	}

	fn := func(ctx context.Context, client *hmrest.APIClient, body RequestSpec) (*hmrest.Operation, error) {
		op, _, err := client.AvailabilityZonesApi.CreateAvailabilityZone(ctx, *body.(*hmrest.AvailabilityZonePost), region, nil)
		return &op, err
	}
	return fn, &body, nil
}

func (p *availabilityZoneProvider) ReadResource(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) error {
	az, _, err := client.AvailabilityZonesApi.GetAvailabilityZoneById(ctx, d.Id(), nil)
	if err != nil {
		return err
	}

	d.Set("name", az.Name)
	d.Set("display_name", az.DisplayName)
	d.Set("region", az.Region.Name)

	return nil
}

func (p *availabilityZoneProvider) PrepareDelete(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) (InvokeWriteAPI, error) {
	availabilityZoneName := rdString(ctx, d, "name")
	region := rdString(ctx, d, "region")

	fn := func(ctx context.Context, client *hmrest.APIClient, body RequestSpec) (*hmrest.Operation, error) {
		op, _, err := client.AvailabilityZonesApi.DeleteAvailabilityZone(ctx, region, availabilityZoneName, nil)
		return &op, err
	}
	return fn, nil
}
