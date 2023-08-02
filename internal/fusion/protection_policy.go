/*
Copyright 2023 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package fusion

import (
	"context"
	"fmt"
	"strconv"

	"github.com/antihax/optional"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"

	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"

	"github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/utilities"
)

// Implements ProtectionPolicy
type protectionPolicyProvider struct {
	BaseResourceProvider
}

func schemaProtectionPolicy() map[string]*schema.Schema {
	return map[string]*schema.Schema{
		optionName: {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringIsNotEmpty,
			Description:  "The name of the Protection Policy.",
		},
		optionDisplayName: {
			Type:         schema.TypeString,
			Optional:     true,
			Computed:     true,
			ValidateFunc: validation.StringLenBetween(1, maxDisplayName),
			Description:  "The human-readable name of the Protection Policy. If not provided, defaults to I(name).",
		},
		optionLocalRPO: {
			Type:             schema.TypeInt,
			Required:         true,
			ValidateFunc:     validation.IntAtLeast(localRpoMin),
			DiffSuppressFunc: DiffSuppressForHumanReadableTimePeriod,
			Description:      "The Recovery Point Objective for Snapshots. Value should be specified in minutes. Minimum value is 10 minutes.",
		},
		optionLocalRetention: {
			Type:             schema.TypeString,
			Required:         true,
			ValidateDiagFunc: IsValidLocalRetention,
			DiffSuppressFunc: DiffSuppressForHumanReadableTimePeriod,
			Description:      "The Retention Duration for periodic snapshots. Minimum value is 10 minutes. Value can be provided as (m|M)inutes, (h|H)ours, (d|D)ays, (w|W)eeks, or (y|Y)ears. If no unit is provided, minutes are assumed.",
		},
		optionDestroySnapshotsOnDelete: {
			Type:     schema.TypeBool,
			Optional: true,
			Default:  false,
			Description: "Before deleting Protection Policy, Snapshots within it will be deleted. " +
				"If `false` then any Snapshots will need to be deleted as a separate step before removing the Protection Policy.",
		},
	}
}

func resourceProtectionPolicy() *schema.Resource {
	p := &protectionPolicyProvider{BaseResourceProvider{ResourceKind: resourceKindProtectionPolicy}}
	protectionPolicyFunctions := NewBaseResourceFunctions(resourceKindProtectionPolicy, p)
	protectionPolicyFunctions.Resource.Description = `A Protection Policy (e.g. "Hourly") is published by the AZ Admin. It specifies how often the recovery of snapshots are made.`
	protectionPolicyFunctions.Resource.Schema = schemaProtectionPolicy()

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
	pp, _, err := client.ProtectionPoliciesApi.GetProtectionPolicyById(ctx, d.Id(), nil)
	if err != nil {
		return err
	}

	return p.loadProtectionPolicy(pp, d)
}

func (p *protectionPolicyProvider) PrepareDelete(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) (InvokeWriteAPI, error) {
	name := rdString(ctx, d, optionName)
	destroySnaps := d.Get(optionDestroySnapshotsOnDelete).(bool)

	fn := func(ctx context.Context, client *hmrest.APIClient, body RequestSpec) (*hmrest.Operation, error) {
		if destroySnaps {
			snapshots, _, err := client.SnapshotsApi.QuerySnapshots(ctx, &hmrest.SnapshotsApiQuerySnapshotsOpts{
				ProtectionPolicyId: optional.NewString(d.Id()),
			})

			if err != nil {
				tflog.Error(ctx, "Failed listing snapshots", "protection_policy_id", d.Id())
				utilities.TraceError(ctx, err)
				return nil, err
			}

			if len(snapshots.Items) > 0 {
				tflog.Info(ctx, "Deleting Snapshots in order to delete Protection Policy", "protection_policy", name)
				deleteSnapshots(ctx, &snapshots, client)
			} else {
				tflog.Debug(ctx, "No snapshots found", "protection_policy_id", d.Id())
			}
		}

		op, _, err := client.ProtectionPoliciesApi.DeleteProtectionPolicy(ctx, name, nil)
		return &op, err
	}
	return fn, nil
}

func (p *protectionPolicyProvider) PrepareUpdate(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) (InvokeWriteAPI, []ResourcePatch, error) {
	if err := utilities.CheckImmutableFieldsExcept(ctx, d, optionDestroySnapshotsOnDelete); err != nil {
		return nil, nil, err
	}

	return DummyInvokeWriteAPI, []ResourcePatch{}, nil
}

func (p *protectionPolicyProvider) ImportResource(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) ([]*schema.ResourceData, error) {
	var orderedRequiredGroupNames = []string{resourceGroupNameProtectionPolicy}
	// The ID is user provided value - we expect self link
	selfLinkFieldsWithValues, err := utilities.ParseSelfLink(d.Id(), orderedRequiredGroupNames)
	if err != nil {
		return nil, fmt.Errorf("invalid protection_policy import path. Expected path in format '/protection-policies/<protection-policy>'")
	}

	protectionPolicy, _, err := client.ProtectionPoliciesApi.GetProtectionPolicy(ctx, selfLinkFieldsWithValues[resourceGroupNameProtectionPolicy], nil)
	if err != nil {
		utilities.TraceError(ctx, err)
		return nil, err
	}

	err = p.loadProtectionPolicy(protectionPolicy, d)
	if err != nil {
		return nil, err
	}

	d.SetId(protectionPolicy.Id)

	return []*schema.ResourceData{d}, nil
}

func (p *protectionPolicyProvider) loadProtectionPolicy(pp hmrest.ProtectionPolicy, d *schema.ResourceData) error {
	err := getFirstError(
		d.Set(optionName, pp.Name),
		d.Set(optionDisplayName, pp.DisplayName),
	)

	for _, obj := range pp.Objectives {
		if rpo, ok := obj.(*hmrest.Rpo); ok {
			rpoValue, _ := utilities.StringISO8601MinutesToInt(rpo.Rpo)
			err = getFirstError(err, d.Set(optionLocalRPO, rpoValue))
			continue
		}

		if retention, ok := obj.(*hmrest.Retention); ok {
			retentionValue, _ := utilities.StringISO8601MinutesToInt(retention.After)
			err = getFirstError(err, d.Set(optionLocalRetention, strconv.Itoa(retentionValue)))
			continue
		}
	}
	return err
}
