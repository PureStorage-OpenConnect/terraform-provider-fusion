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
	"os"
	"path"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	"golang.org/x/oauth2"

	"github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/auth"
	"github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/utilities"
	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"
)

const (
	accessTokenVar               = "FUSION_ACCESS_TOKEN"
	privateKeyPasswordVar        = "FUSION_PRIVATE_KEY_PASSWORD"
	hostVar                      = "FUSION_API_HOST"
	issuerIdVar                  = "FUSION_ISSUER_ID"
	privateKeyPathVar            = "FUSION_PRIVATE_KEY_FILE"
	privateKeyVar                = "FUSION_PRIVATE_KEY"
	fusionConfigVar              = "FUSION_CONFIG"
	fusionConfigProfileVar       = "FUSION_CONFIG_PROFILE"
	defaultHost                  = "https://api.pure1.purestorage.com/fusion"
	bothOptionsNotProvidedString = "neither %[1]s nor %[2]s specified. Must be provided at least in one of the places: configuration block, enviromental variable or Fusion config file"
)

const basePath = "api/v1"

var providerVersion = "dev"

// Provider is the terraform resource provider called by main.go
func Provider() *schema.Provider {
	return &schema.Provider{
		Schema: map[string]*schema.Schema{
			optionHost: {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringIsNotEmpty,
				Description:  "The URL of Fusion API host.",
			},
			optionIssuerId: {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{optionAccessToken},
				ValidateFunc:  validation.StringIsNotEmpty,
				Description:   "The Issuer ID, used together with private key to authenticate the client.",
			},
			// TODO add to documentation that PRIVATE_KEY env variable has higher priority than PRIVATE_KEY_FILE
			optionPrivateKeyFile: {
				Type:          schema.TypeString,
				Optional:      true,
				ConflictsWith: []string{optionPrivateKey, optionAccessToken},
				ValidateFunc:  validation.StringIsNotEmpty,
				Description:   "The Path to the Private Key File to be used for the authentication.",
			},
			optionPrivateKey: {
				Type:          schema.TypeString,
				Optional:      true,
				Sensitive:     true,
				ConflictsWith: []string{optionPrivateKeyFile, optionAccessToken},
				ValidateFunc:  validation.StringIsNotEmpty,
				Description: "Raw string with Private Key to be used for the authentication. Accepts PKCS#1 format. " +
					"Include the `-----BEGIN RSA PRIVATE KEY-----` and `-----END RSA PRIVATE KEY-----` lines.",
			},
			optionAccessToken: {
				Type:          schema.TypeString,
				Optional:      true,
				Sensitive:     true,
				ValidateFunc:  validation.StringIsNotEmpty,
				ConflictsWith: []string{optionIssuerId, optionPrivateKeyFile, optionPrivateKey},
				Description:   "The Access Token for the Fusion API.",
			},
			optionFusionConfig: {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringIsNotEmpty,
				Description:  "The Path to the Fusion Config File containing authentication profiles.",
			},
			optionTokenEndpoint: {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringIsNotEmpty,
				Description:  "The URL of the Fusion authentication token endpoint.",
			},
			optionFusionConfigProfile: {
				Type:         schema.TypeString,
				Optional:     true,
				ValidateFunc: validation.StringIsNotEmpty,
				Description:  "The name of the profile in the Fusion configuration file to use.",
			},
			optionPrivateKeyPassword: {
				Type:         schema.TypeString,
				Optional:     true,
				Sensitive:    true,
				ValidateFunc: validation.StringIsNotEmpty,
				Description:  "The password of encrypted RSA private key.",
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
			"fusion_role_assignment":         resourceRoleAssignment(),
			"fusion_network_interface":       resourceNetworkInterface(),
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
			"fusion_availability_zone":       dataSourceAvailabilityZone(),
			"fusion_protection_policy":       dataSourceProtectionPolicy(),
			"fusion_snapshot":                dataSourceSnapshot(),
			"fusion_volume":                  dataSourceVolume(),
			"fusion_role":                    dataSourceRole(),
			"fusion_user":                    dataSourceUser(),
			"fusion_volume_snapshot":         dataSourceVolumeSnapshot(),
			"fusion_placement_group":         dataSourcePlacementGroup(),
			"fusion_network_interface":       dataSourceNetworkInterface(),
		},

		ConfigureContextFunc: configureProvider,
	}
}

func configureProvider(ctx context.Context, d *schema.ResourceData) (interface{}, diag.Diagnostics) {
	hostField := d.Get(optionHost).(string)
	issuerIdField := d.Get(optionIssuerId).(string)
	privateKeyField := d.Get(optionPrivateKey).(string)
	privateKeyFileField := d.Get(optionPrivateKeyFile).(string)
	tokenEndpointField := d.Get(optionTokenEndpoint).(string)
	configPathField := d.Get(optionFusionConfig).(string)
	accessTokenField := d.Get(optionAccessToken).(string)
	configProfileField := d.Get(optionFusionConfigProfile).(string)
	privateKeyPasswordKeyField := d.Get(optionPrivateKeyPassword).(string)

	configPathEnv := os.Getenv(fusionConfigVar)
	tokenEndpointEnv := os.Getenv(auth.AuthNEndpointOverrideEnvVarName)
	hostEnv := os.Getenv(hostVar)
	issuerIdEnv := os.Getenv(issuerIdVar)
	privateKeyEnv := os.Getenv(privateKeyVar)
	privateKeyFileEnv := os.Getenv(privateKeyPathVar)
	accessTokenEnv := os.Getenv(accessTokenVar)
	configProfileEnv := os.Getenv(fusionConfigProfileVar)
	privateKeyPasswordKeyEnv := os.Getenv(privateKeyPasswordVar)

	var fusionProfileConfig ProfileConfig
	var err error

	configProfile := configProfileField
	if configProfile == "" {
		configProfile = configProfileEnv
	}

	if configPathField != "" {
		tflog.Debug(ctx, "reading config from TF field")
		fusionProfileConfig, err = GetProfileConfig(configPathField, configProfile)
		if err != nil {
			return nil, diag.Errorf("error reading config from TF block: %s", err)
		}
	} else if configPathEnv != "" {
		tflog.Debug(ctx, "reading config from environment variable")
		fusionProfileConfig, err = GetProfileConfig(configPathEnv, configProfile)
		if err != nil {
			return nil, diag.Errorf("error reading config from environment variable: %s", err)
		}
	} else {
		tflog.Debug(ctx, "trying to read config from default path $HOME/.pure/fusion.json")
		homeConfigPath, _ := GetHomeConfigPath()
		fusionProfileConfig, err = GetProfileConfig(homeConfigPath, configProfile)
		if configProfile != "" && err != nil {
			return nil, diag.Errorf("error reading config from default path $HOME/.pure/fusion.json: %s", err)
		} else if err != nil {
			fusionProfileConfig = ProfileConfig{}
		}
	}

	tokenEndpoint := getOption(ctx, optionTokenEndpoint, []string{tokenEndpointField, tokenEndpointEnv,
		fusionProfileConfig.TokenEndpoint, auth.DefaultAuthNEndpoint})
	host := getOption(ctx, optionHost, []string{hostField, hostEnv, fusionProfileConfig.ApiHost, defaultHost})

	if host == "" {
		return nil, diag.Errorf("no Fusion host specified")
	}

	accessToken, issuerId, _ := getMostPrioritisedParameter(ctx,
		[]string{accessTokenField, accessTokenEnv, fusionProfileConfig.AccessToken},
		[]string{issuerIdField, issuerIdEnv, fusionProfileConfig.IssuerId},
		optionAccessToken, optionIssuerId)
	if accessToken != "" {
		client, err := NewHMClientWithAccessToken(ctx, host, accessToken)
		if err != nil {
			return nil, diag.FromErr(err)
		}
		return client, nil
	}
	if issuerId == "" {
		return nil, diag.Errorf(
			bothOptionsNotProvidedString,
			optionAccessToken, optionIssuerId)
	}

	privateKey, privateKeyFile, _ := getMostPrioritisedParameter(ctx,
		[]string{privateKeyField, privateKeyEnv, fusionProfileConfig.PrivateKey},
		[]string{privateKeyFileField, privateKeyFileEnv, fusionProfileConfig.PrivateKeyFile},
		optionPrivateKey, optionPrivateKeyFile)
	if privateKey == "" && privateKeyFile == "" {
		return nil, diag.Errorf(
			bothOptionsNotProvidedString,
			optionPrivateKey, optionPrivateKeyFile)
	}
	var privateKeyPassword string
	if privateKey == "" {
		privateKey, err = auth.ReadPrivateKeyFile(privateKeyFile)
		if err != nil {
			tflog.Error(ctx, "cannot read private key")
			return nil, diag.FromErr(fmt.Errorf("cannot read private key file. err: %s", err))
		}
		privateKeyPassword = getOption(ctx, optionPrivateKeyPassword, []string{privateKeyPasswordKeyField, privateKeyPasswordKeyEnv, fusionProfileConfig.PrivateKeyPassword})
	}

	client, err := NewHMClient(ctx, host, issuerId, privateKey, tokenEndpoint, privateKeyPassword)
	if err != nil {
		return nil, diag.FromErr(err)
	}
	return client, nil
}

func logOptionUsage(ctx context.Context, position int, optionName string) {
	switch position {
	case 0:
		tflog.Debug(ctx, "using TF field", "for option", optionName)
	case 1:
		tflog.Debug(ctx, "using environment variable", "for option", optionName)
	case 2:
		tflog.Debug(ctx, "using Fusion config file for", " option", optionName)
	case 3:
		tflog.Debug(ctx, "using default value", "for option", optionName)
	}
}

// Returns the first parameter as first argument if it has more priority
// Otherwise returns second parameter as second argument
// The first parameter has more priority then the second of the same index
// Expects array with same sizes
func getMostPrioritisedParameter(ctx context.Context, firstParameters, secondParameters []string, firstParameterName, secondParameterName string) (string, string, error) {
	if len(firstParameters) != len(secondParameters) {
		return "", "", fmt.Errorf("expected arrays with same length")
	}

	for i, firstParameterOption := range firstParameters {
		if firstParameterOption != "" {
			logOptionUsage(ctx, i, firstParameterName)
			return firstParameterOption, "", nil
		}
		if secondParameters[i] != "" {
			logOptionUsage(ctx, i, secondParameterName)
			return "", secondParameters[i], nil
		}
	}

	return "", "", nil
}

func getOption(ctx context.Context, optionName string, priorities []string) string {
	for i, option := range priorities {
		if option != "" {
			logOptionUsage(ctx, i, optionName)
			return option
		}
	}
	return ""
}

func getAccessToken(ctx context.Context, issuerId, privateKey, tokenEndpoint, privateKeyPassword string) (string, error) {
	var accessToken string

	err := utilities.Retry(ctx, time.Millisecond*100, 0.7, 13, "pure1_token", func() (bool, error) {
		t, err := auth.GetPure1SelfSignedAccessTokenGoodForOneHour(ctx, issuerId, privateKey, tokenEndpoint, privateKeyPassword)
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
		return "", err
	}

	return accessToken, nil
}

func NewHMClient(ctx context.Context, host, issuerId, privateKey, tokenEndpoint, privateKeyPassword string) (*hmrest.APIClient, error) {
	tflog.Debug(ctx, "Using Fusion", optionHost, host)
	accessToken, err := getAccessToken(ctx, issuerId, privateKey, tokenEndpoint, privateKeyPassword)
	if err != nil {
		return nil, err
	}

	tflog.Debug(ctx, "API token has been successfully retrieved")
	return NewHMClientWithAccessToken(ctx, host, accessToken)
}

func NewHMClientWithAccessToken(ctx context.Context, host, accessToken string) (*hmrest.APIClient, error) {
	url, err := url.Parse(host)
	if err != nil {
		return nil, err
	}
	url.Path = path.Join(url.Path, basePath)

	return hmrest.NewAPIClient(&hmrest.Configuration{
		BasePath:      url.String(),
		DefaultHeader: map[string]string{"Authorization": "Bearer " + accessToken},
		UserAgent:     fmt.Sprintf("terraform-provider-fusion/%s", providerVersion),
	}), nil
}
