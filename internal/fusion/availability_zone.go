/*
Copyright 2023 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package fusion

import (
	"context"
	"fmt"

	"github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/utilities"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"
)

// Implements ResourceProvider
type availabilityZoneProvider struct {
	BaseResourceProvider
}

func schemaAvailabilityZone() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		optionName: {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "The name of the Availability Zone.",
		},
		optionDisplayName: {
			Type:         schema.TypeString,
			Optional:     true,
			Computed:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "The human-readable name of the Availability Zone. If not provided, defaults to I(name).",
		},
		optionRegion: {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "The name of Region within which the Availability Zone is created.",
		},
	}
}

// This is our entry point for the Availability Zone resource.
func resourceAvailabilityZone() *schema.Resource {
	p := &availabilityZoneProvider{BaseResourceProvider{ResourceKind: resourceKindAvailabilityZone}}
	availabilityZoneResourceFunctions := NewBaseResourceFunctions(resourceKindAvailabilityZone, p)
	availabilityZoneResourceFunctions.Resource.Description = `An Availability Zone (AZ) (e.g. "NYC DC-1")"` +
		` is a fault domain within a Region. It contains Arrays.`
	availabilityZoneResourceFunctions.Resource.Schema = schemaAvailabilityZone()

	return availabilityZoneResourceFunctions.Resource
}

func (p *availabilityZoneProvider) PrepareCreate(ctx context.Context, d *schema.ResourceData) (InvokeWriteAPI, ResourcePost, error) {
	region := rdString(ctx, d, optionRegion)
	name := rdString(ctx, d, optionName)
	displayName := rdStringDefault(ctx, d, optionDisplayName, name)

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

	return p.loadAZ(az, d)
}

func (p *availabilityZoneProvider) PrepareDelete(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) (InvokeWriteAPI, error) {
	availabilityZoneName := rdString(ctx, d, optionName)
	region := rdString(ctx, d, optionRegion)

	fn := func(ctx context.Context, client *hmrest.APIClient, body RequestSpec) (*hmrest.Operation, error) {
		op, _, err := client.AvailabilityZonesApi.DeleteAvailabilityZone(ctx, region, availabilityZoneName, nil)
		return &op, err
	}
	return fn, nil
}

func (p *availabilityZoneProvider) ImportResource(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) ([]*schema.ResourceData, error) {
	var orderedRequiredGroupNames = []string{
		resourceGroupNameRegion,
		resourceGroupNameAvailabilityZone,
	}
	// The ID is user provided value - we expect self link
	selfLinkFieldsWithValues, err := utilities.ParseSelfLink(d.Id(), orderedRequiredGroupNames)
	if err != nil {
		return nil, fmt.Errorf("invalid availability_zone import path. Expected path in format '/regions/<region>/availability-zones/<availability-zone>'")
	}

	az, _, err := client.AvailabilityZonesApi.GetAvailabilityZone(ctx, selfLinkFieldsWithValues[resourceGroupNameRegion], selfLinkFieldsWithValues[resourceGroupNameAvailabilityZone], nil)
	if err != nil {
		utilities.TraceError(ctx, err)
		return nil, err
	}

	err = p.loadAZ(az, d)
	if err != nil {
		return nil, err
	}

	d.SetId(az.Id)

	return []*schema.ResourceData{d}, nil
}

func (p *availabilityZoneProvider) loadAZ(az hmrest.AvailabilityZone, d *schema.ResourceData) error {
	return getFirstError(
		d.Set(optionName, az.Name),
		d.Set(optionDisplayName, az.DisplayName),
		d.Set(optionRegion, az.Region.Name),
	)
}
