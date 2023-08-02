/*
Copyright 2023 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package fusion

import (
	"context"
	"fmt"
	"strings"

	"github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/utilities"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"
)

// Implements ResourceProvider
type roleAssignmentProvider struct {
	BaseResourceProvider
}

func schemaRoleAssignment() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		optionName: {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The name of the Role Assignment.",
		},
		optionRoleName: {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "The name of the Role to be assigned.",
		},
		optionPrincipal: {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "The unique ID of the principal (User or API Client) to assign to the Role.",
		},
		optionScope: {
			Type:     schema.TypeList,
			Required: true,
			MaxItems: 1,
			Elem: &schema.Resource{
				Schema: map[string]*schema.Schema{
					optionTenant: {
						Type:         schema.TypeString,
						Optional:     true,
						ValidateFunc: validation.StringIsNotEmpty,
						Description:  "The name of the Tenant the user has the Role applied to.",
					},
					optionTenantSpace: {
						Type:         schema.TypeString,
						Optional:     true,
						RequiredWith: []string{optionScope + ".0." + optionTenant},
						ValidateFunc: validation.StringIsNotEmpty,
						Description:  "The name of the Tenant Space the user has the Role applied to.",
					},
				},
			},
			Description: "The level to which the Role is assigned. Empty scope sets the scope to the whole organization.",
		},
	}
}

// This is our entry point for the Role Assignment resource
func resourceRoleAssignment() *schema.Resource {
	p := &roleAssignmentProvider{BaseResourceProvider{ResourceKind: resourceKindRoleAssignment}}
	roleAssignmentResourceFunctions := NewBaseResourceFunctions(resourceKindRoleAssignment, p)

	roleAssignmentResourceFunctions.Description = "A role assignment records that a principal (User or API Client)" +
		" is assigned to a role, scoped to a particular resource and its chidren."
	roleAssignmentResourceFunctions.Resource.Schema = schemaRoleAssignment()

	return roleAssignmentResourceFunctions.Resource
}

func (p *roleAssignmentProvider) PrepareCreate(ctx context.Context, d *schema.ResourceData) (InvokeWriteAPI, ResourcePost, error) {
	roleName := rdString(ctx, d, optionRoleName)

	body := hmrest.RoleAssignmentPost{
		Principal: rdString(ctx, d, optionPrincipal),
		Scope:     p.getScope(ctx, d),
	}

	fn := func(ctx context.Context, client *hmrest.APIClient, body RequestSpec) (*hmrest.Operation, error) {
		op, _, err := client.RoleAssignmentsApi.CreateRoleAssignment(ctx, *body.(*hmrest.RoleAssignmentPost), roleName, nil)
		return &op, err
	}

	return fn, &body, nil
}

func (p *roleAssignmentProvider) ReadResource(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) error {
	roleAssignment, _, err := client.RoleAssignmentsApi.GetRoleAssignmentById(ctx, d.Id(), nil)
	if err != nil {
		return err
	}

	return p.loadRoleAssignment(roleAssignment, d)
}

func (p *roleAssignmentProvider) PrepareDelete(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) (InvokeWriteAPI, error) {
	name := rdString(ctx, d, optionName)
	roleName := rdString(ctx, d, optionRoleName)

	fn := func(ctx context.Context, client *hmrest.APIClient, body RequestSpec) (*hmrest.Operation, error) {
		op, _, err := client.RoleAssignmentsApi.DeleteRoleAssignment(ctx, roleName, name, nil)
		return &op, err
	}
	return fn, nil
}

func (p *roleAssignmentProvider) ImportResource(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) ([]*schema.ResourceData, error) {
	var orderedRequiredGroupNames = []string{
		resourceGroupNameRole,
		resourceGroupNameRoleAssignment,
	}
	// The ID is user provided value - we expect self link
	selfLinkFieldsWithValues, err := utilities.ParseSelfLink(d.Id(), orderedRequiredGroupNames)
	if err != nil {
		return nil, fmt.Errorf("invalid role_assignment import path. Expected path in format '/roles/<role>/role-assignments/<role-assignment>'")
	}

	roleAssignment, _, err := client.RoleAssignmentsApi.GetRoleAssignment(ctx, selfLinkFieldsWithValues[resourceGroupNameRole], selfLinkFieldsWithValues[resourceGroupNameRoleAssignment], nil)
	if err != nil {
		utilities.TraceError(ctx, err)
		return nil, err
	}

	err = p.loadRoleAssignment(roleAssignment, d)
	if err != nil {
		return nil, err
	}

	d.SetId(roleAssignment.Id)

	return []*schema.ResourceData{d}, nil
}

func (p *roleAssignmentProvider) loadRoleAssignment(roleAssignment hmrest.RoleAssignment, d *schema.ResourceData) error {
	return getFirstError(
		d.Set(optionName, roleAssignment.Name),
		d.Set(optionRoleName, roleAssignment.Role.Name),
		d.Set(optionPrincipal, roleAssignment.Principal),
		d.Set(optionScope, []interface{}{p.readScope(roleAssignment.Scope.SelfLink)}),
	)
}

func (p *roleAssignmentProvider) readScope(scopeStr string) map[string]interface{} {
	scope := make(map[string]interface{})

	if scopeStr == "/" { // Org
		return scope
	}

	scopeParts := strings.Split(scopeStr, "/")
	if len(scopeParts) >= 3 { // Tenant
		scope[optionTenant] = scopeParts[2]
	}

	if len(scopeParts) == 5 {
		scope[optionTenantSpace] = scopeParts[4]
	}

	return scope
}

func (p *roleAssignmentProvider) getScope(ctx context.Context, d *schema.ResourceData) string {
	tenant, tenantOk := d.GetOk(optionScope + ".0." + optionTenant)
	tenantSpace, tenantSpaceOk := d.GetOk(optionScope + ".0." + optionTenantSpace)

	if !tenantOk && !tenantSpaceOk { // Organization scope
		return "/"
	}

	if !tenantSpaceOk { // Tenant scope
		return "/tenants/" + tenant.(string)
	}

	// Tenant Space scope
	return fmt.Sprintf("/tenants/%s/tenant-spaces/%s", tenant.(string), tenantSpace.(string))
}
