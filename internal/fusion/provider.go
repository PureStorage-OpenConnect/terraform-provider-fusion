/*
Copyright 2022 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package fusion

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"path"
	"time"

	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"golang.org/x/oauth2"

	"github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/auth"
	"github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/utilities"
	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"
)

const (
	hostVar           = "FUSION_HOST"
	issuerIdVar       = "FUSION_ISSUER_ID"
	privateKeyPathVar = "FUSION_PRIVATE_KEY_FILE"
)

const basePath = "api/1.0"

var providerVersion = "dev"

// Provider is the terraform resource provider called by main.go
func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			"host": {
				Type:             schema.TypeString,
				Optional:         true,
				DefaultFunc:      schema.EnvDefaultFunc(hostVar, ""),
				ValidateDiagFunc: validateProviderParam("Fusion host", hostVar),
			},
			"issuer_id": {
				Type:             schema.TypeString,
				Optional:         true,
				DefaultFunc:      schema.EnvDefaultFunc(issuerIdVar, ""),
				ValidateDiagFunc: validateProviderParam("issuer ID", issuerIdVar),
			},
			"private_key_file": {
				Type:             schema.TypeString,
				Optional:         true,
				DefaultFunc:      schema.EnvDefaultFunc(privateKeyPathVar, ""),
				ValidateDiagFunc: validateProviderParam("private key file", privateKeyPathVar),
			},
		},

		ResourcesMap: map[string]*schema.Resource{
			"fusion_api_client":              resourceApiClient(),
			"fusion_host_access_policy":      resourceHostAccessPolicy(),
			"fusion_placement_group":         resourcePlacementGroup(),
			"fusion_tenant_space":            resourceTenantSpace(),
			"fusion_volume":                  resourceVolume(),
			"fusion_storage_service":         resourceStorageService(),
			"fusion_storage_class":           resourceStorageClass(),
			"fusion_region":                  resourceRegion(),
			"fusion_availability_zone":       resourceAvailabilityZone(),
			"fusion_network_interface_group": resourceNetworkInterfaceGroup(),
			"fusion_tenant":                  resourceTenant(),
			"fusion_storage_endpoint":        resourceStorageEndpoint(),
			"fusion_array":                   resourceArray(),
			"fusion_protection_policy":       resourceProtectionPolicy(),
		},

		DataSourcesMap: map[string]*schema.Resource{
			"fusion_storage_service":         dataSourceStorageService(),
			"fusion_region":                  dataSourceRegion(),
			"fusion_tenant":                  dataSourceTenant(),
			"fusion_tenant_space":            dataSourceTenantSpace(),
			"fusion_array":                   dataSourceArray(),
			"fusion_storage_class":           dataSourceStorageClass(),
			"fusion_network_interface_group": dataSourceNetworkInterfaceGroup(),
			"fusion_hardware_type":           dataSourceHardwareType(),
			"fusion_storage_endpoint":        dataSourceStorageEndpoint(),
			"fusion_host_access_policy":      dataSourceHostAccessPolicy(),
		},

		ConfigureContextFunc: configureProvider,
	}
}

func validateProviderParam(param, paramEnvVar string) schema.SchemaValidateDiagFunc {
	return func(val interface{}, p cty.Path) diag.Diagnostics {
		if val.(string) == "" {
			return diag.Errorf("No %[1]s specified. The %[1]s must be provided either in the provider "+
				"configuration block or with the %[2]s environment variable.", param, paramEnvVar)
		}
		return nil
	}
}

func configureProvider(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	url := d.Get("host").(string)
	issuerId := d.Get("issuer_id").(string)
	privateKeyPath := d.Get("private_key_file").(string)

	client, err := NewHMClient(ctx, url, issuerId, privateKeyPath)
	if err != nil {
		return nil, diag.FromErr(err)
	}
	return client, nil
}

func NewHMClient(ctx context.Context, host, issuerId, privateKeyPath string) (*hmrest.APIClient, error) {
	tflog.Debug(ctx, "Using Fusion", "host", host)

	url, err := url.Parse(host)
	if err != nil {
		return nil, err
	}
	url.Path = path.Join(url.Path, basePath)

	var accessToken string
	err = utilities.Retry(ctx, time.Millisecond*100, 0.7, 13, "pure1_token", func() (bool, error) {
		t, err := auth.GetPure1SelfSignedAccessTokenGoodForOneHour(ctx, issuerId, privateKeyPath)
		accessToken = t
		var oauthErr *oauth2.RetrieveError
		if errors.As(err, &oauthErr) {
			c := oauthErr.Response.StatusCode
			if !(c >= 500 && c < 600) {
				// If it isn't a 500 error, then we don't retry anymore
				return true, err
			}
		} else {
			// If it isn't an RetrieveError then we also dont retry
			return true, err
		}
		return false, err
	})
	if err != nil {
		utilities.TraceError(ctx, err)
		tflog.Error(ctx, "Error getting API token", "error", err)
		return nil, err
	}
	tflog.Debug(ctx, "API token has been successfully retrieved")
	return hmrest.NewAPIClient(&hmrest.Configuration{
		BasePath:      url.String(), // Client only works if we set the BasePath to the scheme + host + actual base path
		DefaultHeader: map[string]string{"Authorization": "Bearer " + accessToken},
		UserAgent:     fmt.Sprintf("terraform-provider-fusion/%s", providerVersion),
	}), nil
}
