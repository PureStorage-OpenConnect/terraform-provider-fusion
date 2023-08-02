/*
Copyright 2023 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package fusion

import (
	"context"
	"fmt"

	"github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/utilities"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"
)

// Implements ResourceProvider
type tenantSpaceProvider struct {
	BaseResourceProvider
}

func schemaTenantSpace() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		optionName: {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "The name of the Tenant Space.",
		},
		optionDisplayName: {
			Type:         schema.TypeString,
			Optional:     true,
			Computed:     true,
			ValidateFunc: validation.StringLenBetween(1, maxDisplayName),
			Description:  "The human-readable name of the Tenant Space. If not provided, defaults to I(name).",
		},
		optionTenant: {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "The name of the Tenant.",
		},
	}
}

// This is our entry point for the Tenant Space resource
func resourceTenantSpace() *schema.Resource {
	p := &tenantSpaceProvider{BaseResourceProvider{ResourceKind: resourceKindTenantSpace}}

	tenantSpaceResourceFunctions := NewBaseResourceFunctions(resourceKindTenantSpace, p)
	tenantSpaceResourceFunctions.Resource.Description = `A Tenant Space (e.g. "Wiki") contains resources ` +
		`(Volumes, Volume Snapshots, etc.) and exists inside a Tenant.`
	tenantSpaceResourceFunctions.Resource.Schema = schemaTenantSpace()

	return tenantSpaceResourceFunctions.Resource
}

func (p *tenantSpaceProvider) PrepareCreate(ctx context.Context, d *schema.ResourceData) (InvokeWriteAPI, ResourcePost, error) {
	name := rdString(ctx, d, optionName)
	displayName := rdStringDefault(ctx, d, optionDisplayName, name)
	tenant := rdString(ctx, d, optionTenant)

	body := hmrest.TenantSpacePost{
		Name:        name,
		DisplayName: displayName,
	}

	// REVIEW: Should we return an interface instead? What does that look like? The closure lets us use variables above.
	fn := func(ctx context.Context, client *hmrest.APIClient, body RequestSpec) (*hmrest.Operation, error) {
		op, _, err := client.TenantSpacesApi.CreateTenantSpace(ctx, *body.(*hmrest.TenantSpacePost), tenant, nil)
		return &op, err
	}
	return fn, &body, nil
}

func (p *tenantSpaceProvider) ReadResource(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) error {
	ts, _, err := client.TenantSpacesApi.GetTenantSpaceById(ctx, d.Id(), nil)
	if err != nil {
		return err
	}

	return p.loadTenantSpace(ts, d)
}

func (p *tenantSpaceProvider) PrepareDelete(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) (InvokeWriteAPI, error) {
	name := rdString(ctx, d, optionName)
	tenant := rdString(ctx, d, optionTenant)

	fn := func(ctx context.Context, client *hmrest.APIClient, body RequestSpec) (*hmrest.Operation, error) {
		op, _, err := client.TenantSpacesApi.DeleteTenantSpace(ctx, tenant, name, nil)
		return &op, err
	}
	return fn, nil
}

func (p *tenantSpaceProvider) PrepareUpdate(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) (InvokeWriteAPI, []ResourcePatch, error) {
	if d.HasChangeExcept(optionDisplayName) {
		d.Partial(true)
		return nil, nil, fmt.Errorf("attempting to update an immutable field")
	}

	name := rdString(ctx, d, optionName)
	displayName := rdStringDefault(ctx, d, optionDisplayName, name)
	tenant := rdString(ctx, d, optionTenant)

	tflog.Info(ctx, "Updating", optionDisplayName, displayName)
	patches := []ResourcePatch{
		&hmrest.TenantSpacePatch{
			DisplayName: &hmrest.NullableString{Value: displayName},
		},
	}

	fn := func(ctx context.Context, client *hmrest.APIClient, body RequestSpec) (*hmrest.Operation, error) {
		op, _, err := client.TenantSpacesApi.UpdateTenantSpace(ctx, *body.(*hmrest.TenantSpacePatch), tenant, name, nil)
		return &op, err
	}

	return fn, patches, nil
}

func (p *tenantSpaceProvider) ImportResource(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) ([]*schema.ResourceData, error) {
	var orderedRequiredGroupNames = []string{
		resourceGroupNameTenant,
		resourceGroupNameTenantSpace,
	}
	// The ID is user provided value - we expect self link
	selfLinkFieldsWithValues, err := utilities.ParseSelfLink(d.Id(), orderedRequiredGroupNames)
	if err != nil {
		return nil, fmt.Errorf("invalid tenant_space import path. Expected path in format '/tenants/<tenant>/tenant-spaces/<tenant-space>'")
	}

	tenantSpace, _, err := client.TenantSpacesApi.GetTenantSpace(ctx, selfLinkFieldsWithValues[resourceGroupNameTenant], selfLinkFieldsWithValues[resourceGroupNameTenantSpace], nil)
	if err != nil {
		utilities.TraceError(ctx, err)
		return nil, err
	}

	err = p.loadTenantSpace(tenantSpace, d)
	if err != nil {
		return nil, err
	}

	d.SetId(tenantSpace.Id)

	return []*schema.ResourceData{d}, nil
}

func (p *tenantSpaceProvider) loadTenantSpace(ts hmrest.TenantSpace, d *schema.ResourceData) error {
	return getFirstError(
		d.Set(optionName, ts.Name),
		d.Set(optionDisplayName, ts.DisplayName),
		d.Set(optionTenant, ts.Tenant.Name),
	)
}
