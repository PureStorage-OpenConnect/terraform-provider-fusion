/*
Copyright 2022 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package fusion

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"testing"

	"github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/utilities"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-log/tfsdklog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"

	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"
)

type fusionAuth struct {
	IssuerID       string `json:"issuer_id"`
	PrivatePEMFile string `json:"private_pem_file"`
}

type fusionProfile struct {
	Env      string
	Endpoint string
	Auth     fusionAuth
}

type fusionConfig struct {
	DefaultProfile string `json:"default_profile"`
	Profiles       map[string]fusionProfile
}

var testAccProvider *schema.Provider
var testAccProvidersFactory map[string]func() (*schema.Provider, error)

var testAccProfile fusionProfile
var testAccProfileConfigure sync.Once

var testAccConfigure sync.Once
var testURL, testIssuer, testPrivKey string

const testAccTenant = "acc-tenant"
const testAccStorageService = "acc-storageservice"

func init() {
	testAccProvider = Provider()

	testAccProvidersFactory = map[string]func() (*schema.Provider, error){
		"fusion": func() (*schema.Provider, error) { return testAccProvider, nil },
	}
}

func TestProvider(t *testing.T) {
	if err := Provider().InternalValidate(); err != nil {
		t.Fatalf("err: %s", err)
	}
}

func TestProvider_impl(t *testing.T) {
	var _ *schema.Provider = Provider()
}

func TestAccProvider_invalidConfigs(t *testing.T) {
	// We want to test missing parameter values in the provider config, so we need to temporarily
	// unset the parameter environment variables to prevent the provider from using them when the
	// parameter is missing from the template
	testUnsetProviderEnvVars(t)

	resource.Test(t, resource.TestCase{
		ProviderFactories: testAccProvidersFactory,
		Steps: []resource.TestStep{
			{
				Config:      testNoURLConfig(),
				ExpectError: regexp.MustCompile("No Fusion host specified"),
			},
			{
				Config:      testNoIssuerConfig(),
				ExpectError: regexp.MustCompile("No issuer ID specified"),
			},
			{
				Config:      testNoPrivateKeyConfig(),
				ExpectError: regexp.MustCompile("No private key file specified."),
			},
		},
	})
}

// We want to test the provider config but Terraform will only initialize the provider if we are creating
// resources with that provider, so we need to append this resource to test the provider config. We are
// only testing invalid provider configurations here, so this resource won't actually be created
func testTSConfig() string {
	return `
	resource "fusion_tenant_space" "ts" {
		name          = "sales"
		display_name  = "Sales"
		tenant		  = "acc-tenant"
	}
	`
}

func testNoURLConfig() string {
	return `
	provider "fusion" {
		issuer_id = "pure1:apikey:0000000000000000"
		private_key_file = "path/to/nowhere"
	}
	` + testTSConfig()
}

func testNoIssuerConfig() string {
	return `
	provider "fusion" {
		host = "http://localhost:8080"
		private_key_file = "path/to/nowhere"
	}
	` + testTSConfig()
}

func testNoPrivateKeyConfig() string {
	return `
	provider "fusion" {
		host = "http://localhost:8080"
		issuer_id = "pure1:apikey:0000000000000000"
	}
	` + testTSConfig()
}

func testUnsetProviderEnvVars(t *testing.T) {
	t.Setenv(hostVar, "")
	t.Setenv(issuerIdVar, "")
	t.Setenv(privateKeyPathVar, "")
}

func testGetFusionProfile(t *testing.T) fusionProfile {

	testAccProfileConfigure.Do(func() {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			t.Fatalf("error reading home directory: %s", err)
		}
		configPath := filepath.Join(homeDir, ".pure", "fusion.json")

		jsonData, err := os.ReadFile(configPath)
		if err != nil {
			t.Fatalf("unable to read config file at %s: %s", configPath, err)
		}

		var config fusionConfig
		err = json.Unmarshal(jsonData, &config)
		if err != nil {
			t.Fatalf("unable to deserialize config: %s", err)
		}

		profile, ok := config.Profiles[config.DefaultProfile]
		if !ok {
			t.Fatalf("unable to find profile for default profile %s", config.DefaultProfile)
		}
		testAccProfile = profile
	})

	return testAccProfile
}

func testCreateTenant(ctx context.Context, client *hmrest.APIClient, t *testing.T) {

	_, resp, err := client.TenantsApi.GetTenant(ctx, testAccTenant, nil)
	if err == nil {
		return // Tenant already exists, nothing to do
	} else {
		if resp != nil {
			if resp.StatusCode != http.StatusNotFound {
				t.Fatalf("Failed to create test tenant %s, error: %s resp-status: %d: %s",
					testAccTenant, err, resp.StatusCode, resp.Status)
			}
		} else {
			t.Fatalf("Failed to create test tenant %s, error: %s resp == nil", testAccTenant, err)
		}
	}

	// Create tenant
	body := hmrest.TenantPost{
		Name: testAccTenant,
	}
	op, _, err := client.TenantsApi.CreateTenant(ctx, body, nil)
	if err != nil {
		t.Fatalf("Failed to create test tenant %s, error: %s", testAccTenant, err)
	}

	succeeded, err := utilities.WaitOnOperation(ctx, &op, client)
	if err != nil {
		t.Fatalf("Failed to create test tenant %s, error: %s", testAccTenant, err)
	} else if !succeeded {
		t.Fatalf("Failed to create test tenant %s", testAccTenant)
	}

}

func testCreateStorageService(ctx context.Context, client *hmrest.APIClient, t *testing.T) {
	_, resp, err := client.StorageServicesApi.GetStorageService(ctx, testAccStorageService, nil)
	if err == nil {
		return // Storage serivce already exists, nothing to do
	} else {
		if resp != nil {
			if resp.StatusCode != http.StatusNotFound {
				t.Fatalf("Failed to create test storage service %s, error: %s resp-status: %d: %s",
					testAccStorageService, err, resp.StatusCode, resp.Status)
			}
		} else {
			t.Fatalf("Failed to create test storage service %s, error: %s resp == nil", testAccStorageService, err)
		}
	}

	// Create storage serivce
	body := hmrest.StorageServicePost{
		Name:          testAccStorageService,
		HardwareTypes: []string{"flash-array-x"},
	}
	op, _, err := client.StorageServicesApi.CreateStorageService(ctx, body, nil)
	if err != nil {
		t.Fatalf("Failed to create test storage service %s, error: %s", testAccStorageService, err)
	}

	succeeded, err := utilities.WaitOnOperation(ctx, &op, client)
	if err != nil {
		t.Fatalf("Failed to create test storage service %s, error: %s", testAccStorageService, err)
	} else if !succeeded {
		t.Fatalf("Failed to create test storage service %s", testAccStorageService)
	}
}

// Creates provider objects that can be used by test resources.
func testCreateProviderObjects(t *testing.T) {
	ctx := setupTestCtx(t)

	tflog.Trace(ctx, "testCreateProviderObjects")

	client, err := NewHMClient(ctx, testURL, testIssuer, testPrivKey)
	if err != nil {
		t.Fatalf("Failed to create test client, err: %s", err)
	}

	testCreateTenant(ctx, client, t)
	testCreateStorageService(ctx, client, t)
}

// sets the provider config values in the environment
func ConfigureApiClientForTests(t *testing.T) {

	logFmt := "Required env var %s not set, searching for value in fusion config file"

	if os.Getenv(hostVar) == "" {
		t.Logf(logFmt, hostVar) // TODO HM-2140 move this to the terraform logs?
		profile := testGetFusionProfile(t)
		os.Setenv(hostVar, profile.Endpoint)
	}
	if os.Getenv(issuerIdVar) == "" {
		t.Logf(logFmt, issuerIdVar)
		profile := testGetFusionProfile(t)
		os.Setenv(issuerIdVar, profile.Auth.IssuerID)
	}
	if os.Getenv(privateKeyPathVar) == "" {
		t.Logf(logFmt, privateKeyPathVar)
		profile := testGetFusionProfile(t)
		os.Setenv(privateKeyPathVar, profile.Auth.PrivatePEMFile)
	}

	// save the values here so we can use them in test setup
	testURL = os.Getenv(hostVar)
	testIssuer = os.Getenv(issuerIdVar)
	testPrivKey = os.Getenv(privateKeyPathVar)
}

func testAccPreCheck(t *testing.T) {
	testAccConfigure.Do(func() {
		ConfigureApiClientForTests(t)
		testCreateProviderObjects(t)
	})
}

func setupTestCtx(t *testing.T) context.Context {
	ctx := context.Background()

	// This is needed to make tflog work at early stages of unit tests
	ctx = tfsdklog.RegisterTestSink(ctx, t)
	ctx = tfsdklog.NewRootProviderLogger(ctx)
	return ctx
}
