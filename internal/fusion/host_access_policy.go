/*
Copyright 2023 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package fusion

import (
	"context"

	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

func schemaHostAccessPolicy() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		optionName: {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "The name of the Host access policy.",
		},
		optionDisplayName: {
			Type:         schema.TypeString,
			Optional:     true,
			Computed:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "The human name of the Host access policy. If not provided, defaults to I(name).",
		},
		optionIqn: {
			Type:             schema.TypeString,
			Required:         true,
			Description:      "The iSCSI qualified name (IQN) associated with the host.",
			ValidateDiagFunc: IsValidIQN,
		},
		optionPersonality: {
			Type:         schema.TypeString,
			Optional:     true,
			Default:      "linux",
			Description:  "The Personality of the Host machine.",
			ValidateFunc: validation.StringInSlice(hapPersonalities, false),
		},
	}
}

func resourceHostAccessPolicy() *schema.Resource {
	p := &hostAccessPolicyProvider{BaseResourceProvider{ResourceKind: "HostAccessPolicy"}}
	hostAccessPolicyResourceFunctions := NewBaseResourceFunctions("HostAccessPolicy", p)

	hostAccessPolicyResourceFunctions.Resource.Schema = schemaHostAccessPolicy()
	return hostAccessPolicyResourceFunctions.Resource
}

// Implements ResourceProvider
type hostAccessPolicyProvider struct {
	BaseResourceProvider
}

func (p *hostAccessPolicyProvider) PrepareCreate(ctx context.Context, d *schema.ResourceData) (InvokeWriteAPI, ResourcePost, error) {
	hostAccessPolicyName := rdString(ctx, d, optionName)
	displayName := rdStringDefault(ctx, d, optionDisplayName, hostAccessPolicyName)
	iqn := rdString(ctx, d, optionIqn)
	personality := rdString(ctx, d, optionPersonality)

	body := hmrest.HostAccessPoliciesPost{
		Name:        hostAccessPolicyName,
		DisplayName: displayName,
		Iqn:         iqn,
		Personality: personality,
	}

	fn := func(ctx context.Context, client *hmrest.APIClient, body RequestSpec) (*hmrest.Operation, error) {
		op, _, err := client.HostAccessPoliciesApi.CreateHostAccessPolicy(ctx, *body.(*hmrest.HostAccessPoliciesPost), nil)
		return &op, err
	}
	return fn, &body, nil
}

func (p *hostAccessPolicyProvider) ReadResource(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) error {
	hap, _, err := client.HostAccessPoliciesApi.GetHostAccessPolicyById(ctx, d.Id(), nil)
	if err != nil {
		return err
	}

	d.Set(optionName, hap.Name)
	d.Set(optionDisplayName, hap.DisplayName)
	d.Set(optionIqn, hap.Iqn)
	d.Set(optionPersonality, hap.Personality)

	return nil
}

func (p *hostAccessPolicyProvider) PrepareDelete(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) (InvokeWriteAPI, error) {
	hostAccessPolicyName := rdString(ctx, d, optionName)

	fn := func(ctx context.Context, client *hmrest.APIClient, body RequestSpec) (*hmrest.Operation, error) {
		op, _, err := client.HostAccessPoliciesApi.DeleteHostAccessPolicy(ctx, hostAccessPolicyName, nil)
		return &op, err
	}
	return fn, nil
}
