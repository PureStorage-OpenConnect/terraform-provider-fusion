/*
Copyright 2023 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package fusion

import (
	"context"
	"strconv"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"

	"github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/utilities"
)

// Implements ProtectionPolicy
type protectionPolicyProvider struct {
	BaseResourceProvider
}

func resourceProtectionPolicy() *schema.Resource {
	p := &protectionPolicyProvider{BaseResourceProvider{ResourceKind: "ProtectionPolicy"}}
	protectionPolicyFunctions := NewBaseResourceFunctions("ProtectionPolicy", p)

	protectionPolicyFunctions.Resource.Schema = map[string]*schema.Schema{
		optionName: {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "The name of the protection policy.",
		},
		optionDisplayName: {
			Type:         schema.TypeString,
			Optional:     true,
			Computed:     true,
			ValidateFunc: validation.StringLenBetween(1, maxDisplayName),
			Description:  "The human name of the protection policy. If not provided, defaults to I(name).",
		},
		optionLocalRPO: {
			Type:             schema.TypeInt,
			Required:         true,
			ValidateFunc:     validation.IntAtLeast(localRpoMin),
			DiffSuppressFunc: DiffSuppressForHumanReadableTimePeriod,
			Description:      "Recovery Point Objective for snapshots. Value should be specified in minutes. Minimum value is 10 minutes.",
		},
		optionLocalRetention: {
			Type:             schema.TypeString,
			Required:         true,
			ValidateDiagFunc: IsValidLocalRetention,
			DiffSuppressFunc: DiffSuppressForHumanReadableTimePeriod,
			Description:      "Retention Duration for periodic snapshots. Minimum value is 10 minutes. Value can be provided as m(inutes), h(ours), d(ays), w(eeks), or y(ears). If no unit is provided, minutes are assumed.",
		},
	}

	return protectionPolicyFunctions.Resource
}

func (p *protectionPolicyProvider) PrepareCreate(ctx context.Context, d *schema.ResourceData) (InvokeWriteAPI, ResourcePost, error) {
	name := rdString(ctx, d, optionName)
	displayName := rdStringDefault(ctx, d, optionDisplayName, name)
	localRPO := rdInt(d, optionLocalRPO)
	localRetention, _ := utilities.ParseHumanReadableTimePeriodIntoMinutes(rdString(ctx, d, optionLocalRetention))

	body := hmrest.ProtectionPolicyPost{
		Name:        name,
		DisplayName: displayName,
		Objectives: []hmrest.OneOfProtectionPolicyPostObjectivesItems{
			&hmrest.Rpo{
				Type_: "RPO",
				Rpo:   utilities.MinutesToStringISO8601(localRPO),
			},
			&hmrest.Retention{
				Type_: "Retention",
				After: utilities.MinutesToStringISO8601(localRetention),
			},
		},
	}

	fn := func(ctx context.Context, client *hmrest.APIClient, body RequestSpec) (*hmrest.Operation, error) {
		op, _, err := client.ProtectionPoliciesApi.CreateProtectionPolicy(ctx, *body.(*hmrest.ProtectionPolicyPost), nil)
		return &op, err
	}

	return fn, &body, nil
}

func (p *protectionPolicyProvider) ReadResource(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) error {
	t, _, err := client.ProtectionPoliciesApi.GetProtectionPolicyById(ctx, d.Id(), nil)
	if err != nil {
		return err
	}

	d.Set(optionName, t.Name)
	d.Set(optionDisplayName, t.DisplayName)

	for _, obj := range t.Objectives {
		if rpo, ok := obj.(*hmrest.Rpo); ok {
			rpoValue, _ := utilities.StringISO8601MinutesToInt(rpo.Rpo)
			d.Set(optionLocalRPO, rpoValue)
			continue
		}

		if retention, ok := obj.(*hmrest.Retention); ok {
			retentionValue, _ := utilities.StringISO8601MinutesToInt(retention.After)
			d.Set(optionLocalRetention, strconv.Itoa(retentionValue))
			continue
		}
	}

	return nil
}

func (p *protectionPolicyProvider) PrepareDelete(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) (InvokeWriteAPI, error) {
	name := rdString(ctx, d, optionName)

	fn := func(ctx context.Context, client *hmrest.APIClient, body RequestSpec) (*hmrest.Operation, error) {
		op, _, err := client.ProtectionPoliciesApi.DeleteProtectionPolicy(ctx, name, nil)
		return &op, err
	}
	return fn, nil
}
