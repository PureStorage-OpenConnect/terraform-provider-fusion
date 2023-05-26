/*
Copyright 2023 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package fusion

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/acctest"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/v2/terraform"
	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"
)

// GeneratePublicKey generates a new 2048bit public key in PEM format.
func generatePublicKey() (string, error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", fmt.Errorf("error generating keypair: %w", err)
	}

	publicKey := &privateKey.PublicKey
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return "", fmt.Errorf("error marshaling public key bytes: %w", err)
	}
	publicKeyBlock := pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	}
	publicKeyPEM := pem.EncodeToMemory(&publicKeyBlock)
	return string(publicKeyPEM), nil
}

// Creates and destroys
func TestAccApiClient_basic(t *testing.T) {
	rNameConfig := acctest.RandomWithPrefix("ac_test")
	rName := "fusion_api_client." + rNameConfig
	displayName := acctest.RandomWithPrefix("ac-display-name")
	publicKey, err := generatePublicKey()
	if err != nil {
		t.FailNow()
	}

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckApiClientDestroy,
		Steps: []resource.TestStep{
			// Create Api Client and validate its fields
			{
				Config: testApiClientConfig(rNameConfig, displayName, publicKey),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(rName, "display_name", displayName),
					resource.TestCheckResourceAttr(rName, "public_key", publicKey),
					testApiClientExists(rName),
				),
			},
		},
	})
}

func TestAccApiClient_update(t *testing.T) {
	rNameConfig := acctest.RandomWithPrefix("ac_test")
	rName := "fusion_api_client." + rNameConfig
	displayName := acctest.RandomWithPrefix("ac-display-name")
	publicKey, err := generatePublicKey()
	if err != nil {
		t.FailNow()
	}

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckAvailabilityZoneDestroy,
		Steps: []resource.TestStep{
			// Create Api Client and validate its fields
			{
				Config: testApiClientConfig(rNameConfig, displayName, publicKey),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr(rName, "display_name", displayName),
					resource.TestCheckResourceAttr(rName, "public_key", publicKey),
					testApiClientExists(rName),
				),
			},
			// Api Client does not support update
			{
				Config:      testApiClientConfig(rNameConfig, "immutable", publicKey),
				ExpectError: regexp.MustCompile("unsupported operation: update"),
			},
			// TODO: Remove this step once the HM-5438 bug is resolved
			{
				Config: testApiClientConfig(rNameConfig, displayName, publicKey),
			},
		},
	})
}

func TestAccApiClient_multiple(t *testing.T) {
	rNameConfig1 := acctest.RandomWithPrefix("ac_test")
	rName1 := "fusion_api_client." + rNameConfig1
	displayName1 := acctest.RandomWithPrefix("ac-display-name")
	publicKey1, err := generatePublicKey()
	if err != nil {
		t.FailNow()
	}

	rNameConfig2 := acctest.RandomWithPrefix("ac_test2")
	rName2 := "fusion_api_client." + rNameConfig2
	displayName2 := acctest.RandomWithPrefix("ac-display-name2")
	publicKey2, err := generatePublicKey()
	if err != nil {
		t.FailNow()
	}

	resource.ParallelTest(t, resource.TestCase{
		PreCheck:          func() { testAccPreCheck(t) },
		ProviderFactories: testAccProvidersFactory,
		CheckDestroy:      testCheckAvailabilityZoneDestroy,
		Steps: []resource.TestStep{
			// Sanity check two can be created at once
			{
				Config: testApiClientConfig(rNameConfig1, displayName1, publicKey1) + "\n" +
					testApiClientConfig(rNameConfig2, displayName2, publicKey2),
				Check: resource.ComposeTestCheckFunc(
					testApiClientExists(rName1),
					testApiClientExists(rName2),
				),
			},
			// Create two with same name
			{
				Config: testApiClientConfig(rNameConfig1, displayName1, publicKey1) + "\n" +
					testApiClientConfig(rNameConfig2, displayName2, publicKey2) + "\n" +
					testApiClientConfig("conflictRN", displayName1, publicKey1),
				ExpectError: regexp.MustCompile("already exists"),
			},
		},
	})
}

func testApiClientExists(rName string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		tfApiClient, ok := s.RootModule().Resources[rName]
		if !ok {
			return fmt.Errorf("resource not found: %s", rName)
		}
		if tfApiClient.Type != "fusion_api_client" {
			return fmt.Errorf("expected type: fusion_api_client. Found: %s", tfApiClient.Type)
		}
		attrs := tfApiClient.Primary.Attributes

		goclientApiClient, _, err := testAccProvider.Meta().(*hmrest.APIClient).IdentityManagerApi.GetApiClientById(context.Background(), attrs["id"], nil)
		if err != nil {
			return fmt.Errorf("go client returned error while searching for %s. Error: %s", attrs["name"], err)
		}
		if strings.Compare(goclientApiClient.DisplayName, attrs["display_name"]) != 0 ||
			strings.Compare(goclientApiClient.PublicKey, attrs["public_key"]) != 0 {
			return fmt.Errorf("terraform api client doesn't match goclients api client")
		}
		return nil
	}
}

func testCheckApiClientDestroy(s *terraform.State) error {
	client := testAccProvider.Meta().(*hmrest.APIClient)

	for _, rs := range s.RootModule().Resources {
		if rs.Type != "fusion_api_client" {
			continue
		}
		attrs := rs.Primary.Attributes
		id := attrs["id"]
		_, _, err := client.IdentityManagerApi.GetApiClient(context.Background(), id, nil)

		if err != nil {
			continue
		}

		return fmt.Errorf("api client may still exist")
	}
	return nil
}

func testApiClientConfig(rName, displayName, publicKey string) string {
	publicKey = strings.ReplaceAll(publicKey, "\n", "\\n")
	text := fmt.Sprintf(`
	resource "fusion_api_client" "%[1]s" {
		display_name	= "%[2]s"
		public_key		= "%[3]s"
	}
	`, rName, displayName, publicKey)
	tflog.Info(context.Background(), "testApiClientConfig: ", "config", text)
	return text
}
