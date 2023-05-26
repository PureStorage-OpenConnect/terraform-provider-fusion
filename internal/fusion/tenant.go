/*
Copyright 2023 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package fusion

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"
)

// Implements ResourceProvider
type tenantProvider struct {
	BaseResourceProvider
}

func schemaTenant() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		optionName: {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "The name of the tenant.",
		},
		optionDisplayName: {
			Type:         schema.TypeString,
			Optional:     true,
			Computed:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "The human name of the tenant. If not provided, defaults to I(name).",
		},
	}
}

func resourceTenant() *schema.Resource {
	p := &tenantProvider{BaseResourceProvider{ResourceKind: "Tenant"}}

	tenantResourceFunctions := NewBaseResourceFunctions("Tenant", p)
	tenantResourceFunctions.Resource.Schema = schemaTenant()

	return tenantResourceFunctions.Resource
}

func (p *tenantProvider) PrepareCreate(ctx context.Context, d *schema.ResourceData) (InvokeWriteAPI, ResourcePost, error) {
	name := rdString(ctx, d, optionName)
	displayName := rdStringDefault(ctx, d, optionDisplayName, name)

	body := hmrest.TenantPost{
		Name:        name,
		DisplayName: displayName,
	}

	fn := func(ctx context.Context, client *hmrest.APIClient, body RequestSpec) (*hmrest.Operation, error) {
		op, _, err := client.TenantsApi.CreateTenant(ctx, *body.(*hmrest.TenantPost), nil)
		return &op, err
	}

	return fn, &body, nil
}

func (p *tenantProvider) ReadResource(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) error {
	t, _, err := client.TenantsApi.GetTenantById(ctx, d.Id(), nil)
	if err != nil {
		return err
	}

	d.Set(optionName, t.Name)
	d.Set(optionDisplayName, t.DisplayName)

	return nil
}

func (p *tenantProvider) PrepareDelete(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) (InvokeWriteAPI, error) {
	name := rdString(ctx, d, optionName)

	fn := func(ctx context.Context, client *hmrest.APIClient, body RequestSpec) (*hmrest.Operation, error) {
		op, _, err := client.TenantsApi.DeleteTenant(ctx, name, nil)
		return &op, err
	}
	return fn, nil
}

func (p *tenantProvider) PrepareUpdate(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) (InvokeWriteAPI, []ResourcePatch, error) {
	var patches []ResourcePatch
	name := rdString(ctx, d, optionName)

	if d.HasChangeExcept(optionDisplayName) {
		d.Partial(true)
		return nil, nil, fmt.Errorf("attempting to update an immutable field")
	}

	displayName := rdStringDefault(ctx, d, optionDisplayName, name)
	tflog.Info(ctx, "Updating", optionDisplayName, displayName)
	patches = append(patches, &hmrest.TenantPatch{
		DisplayName: &hmrest.NullableString{Value: displayName},
	})

	fn := func(ctx context.Context, client *hmrest.APIClient, body RequestSpec) (*hmrest.Operation, error) {
		op, _, err := client.TenantsApi.UpdateTenant(ctx, *body.(*hmrest.TenantPatch), name, nil)
		return &op, err
	}

	return fn, patches, nil
}
