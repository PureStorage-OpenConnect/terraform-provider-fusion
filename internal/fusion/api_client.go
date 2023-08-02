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
	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"
)

// Implements ResourceProvider
type apiClientProvider struct {
	BaseResourceProvider
}

func resourceApiClient() *schema.Resource {
	p := &apiClientProvider{BaseResourceProvider{ResourceKind: resourceKindApiClient}}
	apiClientResourceFunctions := NewBaseResourceFunctions(resourceKindApiClient, p)

	apiClientResourceFunctions.Resource.Description = "API clients are used to authenticate with the Pure Fusion API."
	apiClientResourceFunctions.Resource.Schema = map[string]*schema.Schema{
		optionName: {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The name of API Client.",
		},
		optionCreatorId: {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The ID of Principal that created the API Client.",
		},
		optionIssuer: {
			Type:        schema.TypeString,
			Computed:    true,
			Description: "The name of API client.",
		},
		optionLastKeyUpdate: {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "The last time API client was updated.",
		},
		optionLastUsed: {
			Type:        schema.TypeInt,
			Computed:    true,
			Description: "The last time API client was used.",
		},
		optionDisplayName: {
			Type:        schema.TypeString,
			Required:    true,
			Description: "The human-readable name of the API client.",
		},
		optionPublicKey: {
			Type:     schema.TypeString,
			Required: true,
			Description: "The API client's PEM formatted (Base64 encoded) RSA public key. " +
				"Include the --BEGIN PUBLIC KEY-- and --END PUBLIC KEY-- lines.",
		},
	}

	return apiClientResourceFunctions.Resource
}

func (p *apiClientProvider) PrepareCreate(ctx context.Context, d *schema.ResourceData) (InvokeWriteAPI, ResourcePost, error) {
	displayName := rdString(ctx, d, optionDisplayName)
	publicKey := rdString(ctx, d, optionPublicKey)
	body := hmrest.ApiClientPost{
		PublicKey:   publicKey,
		DisplayName: displayName,
	}
	fn := func(ctx context.Context, client *hmrest.APIClient, body RequestSpec) (*hmrest.Operation, error) {
		cl, _, err := client.IdentityManagerApi.CreateApiClient(ctx, *body.(*hmrest.ApiClientPost), nil)

		// TODO: BaseResourceOperation expects operation. ApiClient endpoit does not return operations.
		// Let's create a fake operation for now. This will be removed when BaseResourceOperation is refactored in HM-5543.
		op := &hmrest.Operation{Status: "Succeeded", Result: &hmrest.OperationResult{Resource: &hmrest.ResourceReference{Id: cl.Id}}}
		return op, err
	}
	return fn, &body, nil
}

func (p *apiClientProvider) ReadResource(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) error {
	ac, _, err := client.IdentityManagerApi.GetApiClientById(ctx, d.Id(), nil)
	if err != nil {
		return err
	}

	return p.loadApiClient(ac, d)
}

func (p *apiClientProvider) PrepareDelete(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) (InvokeWriteAPI, error) {
	fn := func(ctx context.Context, client *hmrest.APIClient, body RequestSpec) (*hmrest.Operation, error) {
		_, _, err := client.IdentityManagerApi.DeleteApiClient(ctx, d.Id(), nil)

		// TODO: BaseResourceOperation expects operation. ApiClient endpoit does not return operations.
		// Let's create a fake operation for now. This will be removed when BaseResourceOperation is refactored in HM-5543.
		op := &hmrest.Operation{Status: "Succeeded", Result: &hmrest.OperationResult{Resource: &hmrest.ResourceReference{Id: d.Id()}}}
		return op, err
	}
	return fn, nil
}

func (p *apiClientProvider) ImportResource(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) ([]*schema.ResourceData, error) {
	var orderedRequiredGroupNames = []string{resourceGroupNameApiClient}
	// The ID is user provided value - we expect self link
	selfLinkFieldsWithValues, err := utilities.ParseSelfLink(d.Id(), orderedRequiredGroupNames)
	if err != nil {
		return nil, fmt.Errorf("invalid api_client import path. Expected path in format '/api-clients/<api-client-id>'")
	}

	apiClient, _, err := client.IdentityManagerApi.GetApiClientById(ctx, selfLinkFieldsWithValues[resourceGroupNameApiClient], nil)
	if err != nil {
		utilities.TraceError(ctx, err)
		return nil, err
	}

	err = p.loadApiClient(apiClient, d)

	if err != nil {
		return nil, err
	}
	d.SetId(apiClient.Id)

	return []*schema.ResourceData{d}, nil
}

func (p *apiClientProvider) loadApiClient(ac hmrest.ApiClient, d *schema.ResourceData) error {
	return getFirstError(
		d.Set(optionDisplayName, ac.DisplayName),
		d.Set(optionPublicKey, ac.PublicKey),
		d.Set(optionName, ac.Name),
		d.Set(optionIssuer, ac.Issuer),
		d.Set(optionCreatorId, ac.CreatorId),
		d.Set(optionLastKeyUpdate, ac.LastKeyUpdate),
		d.Set(optionLastUsed, ac.LastUsed),
	)
}
