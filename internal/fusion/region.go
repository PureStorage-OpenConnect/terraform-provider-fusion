/*
Copyright 2023 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package fusion

import (
	context "context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"
)

// Implements ResourceProvider
type regionProvider struct {
	BaseResourceProvider
}

func schemaRegion() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		"name": {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "The name of the Region.",
		},
		"display_name": {
			Type:         schema.TypeString,
			Optional:     true,
			Computed:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "The human name of the Region. If not provided, defaults to I(name).",
		},
	}
}

// This is our entry point for the Region resource
func resourceRegion() *schema.Resource {
	vp := &regionProvider{BaseResourceProvider{ResourceKind: "Region"}}
	regionResourceFunctions := NewBaseResourceFunctions("Region", vp)

	regionResourceFunctions.Resource.Schema = schemaRegion()

	return regionResourceFunctions.Resource
}

func (vp *regionProvider) PrepareCreate(ctx context.Context, d *schema.ResourceData) (InvokeWriteAPI, ResourcePost, error) {
	name := rdString(ctx, d, "name")
	displayName := rdStringDefault(ctx, d, "display_name", name)

	body := hmrest.RegionPost{
		Name:        name,
		DisplayName: displayName,
	}

	fn := func(ctx context.Context, client *hmrest.APIClient, body RequestSpec) (*hmrest.Operation, error) {
		op, _, err := client.RegionsApi.CreateRegion(ctx, *body.(*hmrest.RegionPost), nil)
		return &op, err
	}
	return fn, &body, nil
}

func (vp *regionProvider) ReadResource(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) error {
	region, _, err := client.RegionsApi.GetRegionById(ctx, d.Id(), nil)
	if err != nil {
		return err
	}

	d.Set("name", region.Name)
	d.Set("display_name", region.DisplayName)
	return nil
}

func (vp *regionProvider) PrepareDelete(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) (InvokeWriteAPI, error) {
	name := rdString(ctx, d, "name")

	fn := func(ctx context.Context, client *hmrest.APIClient, body RequestSpec) (*hmrest.Operation, error) {
		op, _, err := client.RegionsApi.DeleteRegion(ctx, name, nil)
		return &op, err
	}
	return fn, nil
}

func (vp *regionProvider) PrepareUpdate(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) (InvokeWriteAPI, []ResourcePatch, error) {
	var patches []ResourcePatch

	regionName := rdString(ctx, d, "name")
	if d.HasChangeExcept("display_name") {
		d.Partial(true)
		return nil, nil, fmt.Errorf("attempting to update an immutable field")
	} else {
		displayName := rdStringDefault(ctx, d, "display_name", regionName)
		tflog.Info(ctx, "Updating", "display_name", displayName)
		patches = append(patches, &hmrest.RegionPatch{
			DisplayName: &hmrest.NullableString{Value: displayName},
		})
	}

	fn := func(ctx context.Context, client *hmrest.APIClient, body RequestSpec) (*hmrest.Operation, error) {
		op, _, err := client.RegionsApi.UpdateRegion(ctx, *body.(*hmrest.RegionPatch), regionName, nil)
		return &op, err
	}

	return fn, patches, nil
}
