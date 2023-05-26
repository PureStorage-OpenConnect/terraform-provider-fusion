/*
Copyright 2023 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package fusion

import (
	"context"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"
)

// Implements ResourceProvider
type apiClientProvider struct {
	BaseResourceProvider
}

func resourceApiClient() *schema.Resource {
	p := &apiClientProvider{BaseResourceProvider{ResourceKind: "ApiClient"}}
	apiClientResourceFunctions := NewBaseResourceFunctions("ApiClient", p)

	apiClientResourceFunctions.Resource.Schema = map[string]*schema.Schema{
		optionDisplayName: {
			Type:        schema.TypeString,
			Required:    true,
			Description: "The human name of the API client.",
		},
		optionPublicKey: {
			Type:     schema.TypeString,
			Required: true,
			Description: "The API clients PEM formatted (Base64 encoded) RSA public key." +
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

	d.Set(optionDisplayName, ac.DisplayName)
	d.Set(optionPublicKey, ac.PublicKey)

	return nil
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
