/*
Copyright 2022 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package fusion

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"testing"

	"github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/auth"
	"github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/utilities"

	"github.com/hashicorp/terraform-plugin-log/tfsdklog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"
)

var testAccProvider *schema.Provider
var testAccProvidersFactory map[string]func() (*schema.Provider, error)

var testAccProfile ProfileConfig
var testAccProfileConfigure sync.Once

var testAccConfigure sync.Once
var testURL, testIssuer, testPrivKey, testPrivKeyPassword string

var preexistingRegion = os.Getenv(varPreexistingRegion)
var preexistingAvailabilityZone = os.Getenv(varPreexistingAvailabilityZone)

func init() {
	testAccProvider = Provider()

	if preexistingAvailabilityZone == "" {
		preexistingAvailabilityZone = "az1"
	}
	if preexistingRegion == "" {
		preexistingRegion = "pure-us-west"
	}

	testAccProvidersFactory = map[string]func() (*schema.Provider, error){
		"fusion": func() (*schema.Provider, error) { return testAccProvider, nil },
	}
}

func TestProvider(t *testing.T) {
	utilities.CheckTestSkip(t)

	if err := Provider().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestProvider_impl(t *testing.T) {
	utilities.CheckTestSkip(t)

	var _ *schema.Provider = Provider()
}

func TestAccProvider_privateKeyConfig(t *testing.T) {
	utilities.CheckTestSkip(t)

	testGetFusionProfile(t)
	t.Setenv(privateKeyPathVar, "")
	key, err := auth.ReadPrivateKeyFile(testAccProfile.PrivateKeyFile)
	if err != nil {
		t.Errorf("cannot get private key err: %s", err)
	}

	config := testPrivateKeyStringConfig(key)

	tenantName := acctest.RandomWithPrefix("tenant")

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProvidersFactory,
		Steps: []resource.TestStep{
			{
				Config: config + testTenantConfig(tenantName, tenantName, tenantName),
			},
		},
	})

	// Set env variable for private key
	key, err = auth.ReadPrivateKeyFile(testAccProfile.PrivateKeyFile)
	if err != nil {
		t.Errorf("cannot get private key err: %s", err)
	}
	t.Setenv(privateKeyVar, key)

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProvidersFactory,
		Steps: []resource.TestStep{
			{
				Config: testProviderEmptyConfig() + testTenantConfig(tenantName, tenantName, tenantName),
			},
		},
	})
	t.Setenv(privateKeyVar, "")

}

func TestAccProvider_accessTokenKeyConfig(t *testing.T) {
	utilities.CheckTestSkip(t)

	ctx := setupTestCtx(t)
	testGetFusionProfile(t)
	t.Setenv(hostVar, testAccProfile.ApiHost)

	tenantName := acctest.RandomWithPrefix("tenant")

	key, err := auth.ReadPrivateKeyFile(testAccProfile.PrivateKeyFile)
	if err != nil {
		t.Errorf("cannot get private key err: %s", err)
	}

	tokenEndpoint := os.Getenv(auth.AuthNEndpointOverrideEnvVarName)
	if tokenEndpoint == "" {
		tokenEndpoint = auth.DefaultAuthNEndpoint
	}

	accessToken, err := getAccessToken(ctx, testAccProfile.IssuerId, key, tokenEndpoint, "")
	if err != nil {
		t.Errorf("cannot get access token err: %s", err)
	}
	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProvidersFactory,
		Steps: []resource.TestStep{
			{
				Config: testProviderAccessTokenConfig(accessToken) + testTenantConfig(tenantName, tenantName, tenantName),
			},
		},
	})

	testUnsetProviderEnvVars(t)
	t.Setenv(hostVar, testAccProfile.ApiHost)
	t.Setenv(accessTokenVar, accessToken)

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProvidersFactory,
		Steps: []resource.TestStep{
			{
				Config: testProviderEmptyConfig() + testTenantConfig(tenantName, tenantName, tenantName),
			},
		},
	})

	t.Setenv(accessTokenVar, "")

	ConfigureApiClientForTests(t)
	testUnsetProviderEnvVars(t)
	profile := testAccProfile
	profile.IssuerId = ""
	profile.PrivateKey = ""
	profile.PrivateKeyFile = ""
	profile.AccessToken = accessToken

	fusionConfigPath := filepath.Join(t.TempDir(), "fusion.json")
	os.WriteFile(fusionConfigPath, []byte(testFusionConfigWithDefaultProfile(profile)), 0777)

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProvidersFactory,
		Steps: []resource.TestStep{
			{
				Config: testProviderConfigWithFusionConfig(fusionConfigPath) + testTenantConfig(tenantName, tenantName, tenantName),
			},
		},
	})
}

func TestAccProvider_fusionConfig(t *testing.T) {
	utilities.CheckTestSkip(t)

	testGetFusionProfile(t)
	testUnsetProviderEnvVars(t)

	fusionConfigPath := filepath.Join(t.TempDir(), "fusion.json")
	pathNotExists := filepath.Join(t.TempDir(), "fusion.json")
	os.WriteFile(fusionConfigPath, []byte(testFusionConfigWithDefaultProfile(testAccProfile)), 0777)
	defer os.Remove(fusionConfigPath)

	tenantName := acctest.RandomWithPrefix("tenant")
	t.Setenv(fusionConfigVar, fusionConfigPath)

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProvidersFactory,
		Steps: []resource.TestStep{
			{
				Config: testProviderConfigWithFusionConfig(fusionConfigPath) + testTenantConfig(tenantName, tenantName, tenantName),
			},
		},
	})

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProvidersFactory,
		Steps: []resource.TestStep{
			{
				Config:      testProviderConfigWithFusionConfig(pathNotExists) + testTenantConfig(tenantName, tenantName, tenantName),
				ExpectError: regexp.MustCompile("cannot read fusion config"),
			},
		},
	})

}

func TestAccProvider_fusionConfigEnv(t *testing.T) {
	utilities.CheckTestSkip(t)

	testGetFusionProfile(t)
	testUnsetProviderEnvVars(t)

	fusionConfigPath := filepath.Join(t.TempDir(), "fusion.json")
	pathNotExists := filepath.Join(t.TempDir(), "fusion.json")
	os.WriteFile(fusionConfigPath, []byte(testFusionConfigWithDefaultProfile(testAccProfile)), 0777)
	defer os.Remove(fusionConfigPath)

	tenantName := acctest.RandomWithPrefix("tenant")
	t.Setenv(fusionConfigVar, fusionConfigPath)

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProvidersFactory,
		Steps: []resource.TestStep{
			{
				Config: testProviderEmptyConfig() + testTenantConfig(tenantName, tenantName, tenantName),
			},
		},
	})

	t.Setenv(fusionConfigVar, pathNotExists)

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProvidersFactory,
		Steps: []resource.TestStep{
			{
				Config:      testProviderEmptyConfig() + testTenantConfig(tenantName, tenantName, tenantName),
				ExpectError: regexp.MustCompile("cannot read fusion config"),
			},
		},
	})
}

func TestAccProvider_fusionConfigProfile(t *testing.T) {
	utilities.CheckTestSkip(t)

	testAccProfile = testGetFusionProfile(t)
	testUnsetProviderEnvVars(t)

	fusionConfigPath := filepath.Join(t.TempDir(), "fusion.json")
	profileName := "test-profile"
	os.WriteFile(fusionConfigPath, []byte(testFusionConfigWithoutDefaultProfile(profileName, testAccProfile)), 0777)
	defer os.Remove(fusionConfigPath)

	tenantName := acctest.RandomWithPrefix("tenant")

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProvidersFactory,
		Steps: []resource.TestStep{
			{
				Config: testProviderConfigWithFusionProfile(fusionConfigPath, profileName) + testTenantConfig(tenantName, tenantName, tenantName),
			},
		},
	})

	t.Setenv(fusionConfigProfileVar, profileName)

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProvidersFactory,
		Steps: []resource.TestStep{
			{
				Config: testProviderConfigWithFusionConfig(fusionConfigPath) + testTenantConfig(tenantName, tenantName, tenantName),
			},
		},
	})
}

func testFusionConfigWithoutDefaultProfile(profileName string, profile ProfileConfig) string {
	return fmt.Sprintf(
		`{
			"default_profile": "%[1]s",
			"profiles": {
				"%[2]s": {
					"env": "test-env",
					"endpoint": "%[3]s",
					"auth": {
						"issuer_id": "%[4]s",
						"private_pem_file": "%[5]s",
						"private_key": "%[6]s",
						"token_endpoint": "%[7]s",
						"access_token": "%[8]s"
					}
				},
				"%[1]s": {
					"env": "test-env2",
					"endpoint": "test-endpoint2",
					"auth": {
						"issuer_id": "test-issuer2",
						"private_pem_file": "test-key-path2",
						"token_endpoint": "test-token-endpoint2",
						"access_token": "test-acces-token2"
					}
				}
			}
		}`, acctest.RandomWithPrefix("fake-profile"), profileName, profile.ApiHost, profile.IssuerId, profile.PrivateKeyFile,
		profile.PrivateKey, profile.TokenEndpoint, profile.AccessToken)
}

func testFusionConfigWithDefaultProfile(profile ProfileConfig) string {
	return fmt.Sprintf(
		`{
			"default_profile": "test-profile",
			"profiles": {
				"test-profile": {
					"env": "test-env",
					"endpoint": "%[1]s",
					"auth": {
						"issuer_id": "%[2]s",
						"private_pem_file": "%[3]s",
						"private_key": "%[4]s",
						"token_endpoint": "%[5]s",
						"access_token": "%[6]s"
					}
				},
				"test-profile2": {
					"env": "test-env2",
					"endpoint": "test-endpoint2",
					"auth": {
						"issuer_id": "test-issuer2",
						"private_pem_file": "test-key-path2",
						"token_endpoint": "test-token-endpoint2",
						"access_token": "test-acces-token2"
					}
				}
			}
		}`, profile.ApiHost, profile.IssuerId, profile.PrivateKeyFile, profile.PrivateKey,
		profile.TokenEndpoint, profile.AccessToken)
}

func testProviderConfigWithFusionProfile(path string, profile string) string {
	return fmt.Sprintf(`provider "fusion" {
		fusion_config = "%s"
		fusion_config_profile = "%s"
	}`, path, profile)
}

func testProviderConfigWithFusionConfig(path string) string {
	return fmt.Sprintf(`provider "fusion" {
		fusion_config = "%s"
	}`, path)
}

func testProviderEmptyConfig() string {
	return `provider "fusion" {}`
}

func testProviderAccessTokenConfig(accessToken string) string {
	return fmt.Sprintf(`provider "fusion" {
		access_token = "%[1]s"
	}`, accessToken)
}

func testPrivateKeyStringConfig(key string) string {
	return fmt.Sprintf(`
	provider "fusion" {
		private_key =<<EOF
%s
EOF
	}`, key)
}

func testUnsetProviderEnvVars(t *testing.T) {
	t.Setenv(hostVar, "")
	t.Setenv(issuerIdVar, "")
	t.Setenv(privateKeyPathVar, "")
	t.Setenv(privateKeyVar, "")
	t.Setenv(accessTokenVar, "")
	t.Setenv(fusionConfigVar, "")
	t.Setenv(fusionConfigProfileVar, "")

}

func testGetFusionProfile(t *testing.T) ProfileConfig {
	testAccProfileConfigure.Do(func() {
		configPath := os.Getenv(fusionConfigVar)
		if configPath == "" {
			var err error
			configPath, err = GetHomeConfigPath()
			if err != nil {
				t.Fatalf("error reading home directory: %s", err)
			}
		}

		var err error
		testAccProfile, err = GetProfileConfig(configPath, "")
		if err != nil {
			t.Fatalf("unable to get config profile at %s: %s", configPath, err)
		}
	})

	return testAccProfile
}

func newTestHMClient(ctx context.Context, host, issuerId, privateKey, privateKeyPassword string) (*hmrest.APIClient, error) {
	tokenEndpoint := os.Getenv(auth.AuthNEndpointOverrideEnvVarName)
	if tokenEndpoint == "" {
		tokenEndpoint = auth.DefaultAuthNEndpoint
	}
	return NewHMClient(ctx, host, issuerId, privateKey, tokenEndpoint, privateKeyPassword)
}

// sets the provider config values in the environment
func ConfigureApiClientForTests(t *testing.T) {

	logFmt := "Required env var %s not set, searching for value in fusion config file"
	if os.Getenv(fusionConfigVar) == "" {
		configPath, err := GetHomeConfigPath()
		if err != nil {
			t.Fatalf("error reading home directory: %s", err)
		}
		os.Setenv(fusionConfigVar, configPath)
	}
	if os.Getenv(hostVar) == "" {
		t.Logf(logFmt, hostVar) // TODO HM-2140 move this to the terraform logs?
		profile := testGetFusionProfile(t)
		os.Setenv(hostVar, profile.ApiHost)
	}
	if os.Getenv(issuerIdVar) == "" {
		t.Logf(logFmt, issuerIdVar)
		profile := testGetFusionProfile(t)
		os.Setenv(issuerIdVar, profile.IssuerId)
	}
	if os.Getenv(privateKeyPathVar) == "" {
		t.Logf(logFmt, privateKeyPathVar)
		profile := testGetFusionProfile(t)
		os.Setenv(privateKeyPathVar, profile.PrivateKeyFile)
	}

	// save the values here so we can use them in test setup
	testURL = os.Getenv(hostVar)
	testIssuer = os.Getenv(issuerIdVar)
	privKey, err := auth.ReadPrivateKeyFile(os.Getenv(privateKeyPathVar))
	if err != nil {
		t.Errorf("cannot configure API client for testing err: %s", err)
	}
	testPrivKey = privKey
}

func testAccPreCheck(t *testing.T) {
	testAccConfigure.Do(func() {
		ConfigureApiClientForTests(t)
	})
}

func testAccPreCheckWithReturningClient(ctx context.Context, t *testing.T) *hmrest.APIClient {
	testAccPreCheck(t)
	// Setup HM client
	client, err := newTestHMClient(ctx, testURL, testIssuer, testPrivKey, testPrivKeyPassword)
	if err != nil {
		t.Fatal("Cannot setup api client for testing", err)
	}
	return client
}

func setupTestCtx(t *testing.T) context.Context {
	ctx := context.Background()

	// This is needed to make tflog work at early stages of unit tests
	ctx = tfsdklog.RegisterTestSink(ctx, t)
	ctx = tfsdklog.NewRootProviderLogger(ctx)
	return ctx
}
