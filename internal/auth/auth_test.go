/*
Copyright 2022 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/
package auth_test

import (
	"context"
	"os"
	"testing"

	"github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/auth"
)

// Performs a test against production pure1 auth endpoint
func TestProduction(t *testing.T) {
	t.Setenv("FUSION_TOKEN_ENDPOINT", "")

	if os.Getenv("TEST_PURE1_PROD_ISSUERID") == "" {
		t.Skip("TEST_PURE1_PROD_ISSUERID not set")
	}

	privateKey, err := auth.ReadPrivateKeyFile(os.Getenv("TEST_PURE1_PROD_PRIVATE_KEY_PATH"))
	if err != nil {
		t.Errorf("main: %s", err)
	}

	_, err = auth.GetPure1SelfSignedAccessTokenGoodForOneHour(
		context.Background(),
		os.Getenv("TEST_PURE1_PROD_ISSUERID"),
		privateKey,
		auth.DefaultAuthNEndpoint,
		"",
	)
	if err != nil {
		t.Errorf("main: %s", err)
	}
}

// Performs a test against endpoint specified by PURE1_AUTHENTICATION_ENDPOINT if its set...
func TestStaging(t *testing.T) {
	if os.Getenv("FUSION_TOKEN_ENDPOINT") == "" {
		t.Skip("FUSION_TOKEN_ENDPOINT not set")
	}
	if os.Getenv("TEST_PURE1_STAGING_ISSUERID") == "" {
		t.Skip("TEST_PURE1_STAGING_ISSUERID not set")
	}
	if os.Getenv("TEST_PURE1_STAGING_PRIVATE_KEY_PATH") == "" {
		t.Skip("TEST_PURE1_STAGING_PRIVATE_KEY_PATH not set")
	}

	privateKey, err := auth.ReadPrivateKeyFile(os.Getenv("TEST_PURE1_STAGING_PRIVATE_KEY_PATH"))
	if err != nil {
		t.Errorf("main: %s", err)
	}

	_, err = auth.GetPure1SelfSignedAccessTokenGoodForOneHour(
		context.Background(),
		os.Getenv("TEST_PURE1_STAGING_ISSUERID"),
		privateKey,
		os.Getenv("FUSION_TOKEN_ENDPOINT"),
		"",
	)
	if err != nil {
		t.Errorf("main: %s", err)
	}
}

func TestStagingEncrypted(t *testing.T) {
	if os.Getenv("FUSION_TOKEN_ENDPOINT") == "" {
		t.Skip("FUSION_TOKEN_ENDPOINT not set")
	}
	if os.Getenv("TEST_PURE1_STAGING_ISSUERID") == "" {
		t.Skip("TEST_PURE1_STAGING_ISSUERID not set")
	}
	if os.Getenv("TEST_PURE1_STAGING_ENCRYPTED_PRIVATE_KEY_PATH") == "" {
		t.Skip("TEST_PURE1_STAGING_ENCRYPTED_PRIVATE_KEY_PATH not set")
	}
	if os.Getenv("STAGING_PRIVATE_KEY_PASSWORD") == "" {
		t.Skip("STAGING_PRIVATE_KEY_PASSWORD not set")
	}

	privateKey, err := auth.ReadPrivateKeyFile(os.Getenv("TEST_PURE1_STAGING_ENCRYPTED_PRIVATE_KEY_PATH"))
	if err != nil {
		t.Errorf("main: %s", err)
	}

	_, err = auth.GetPure1SelfSignedAccessTokenGoodForOneHour(
		context.Background(),
		os.Getenv("TEST_PURE1_STAGING_ISSUERID"),
		privateKey,
		os.Getenv("FUSION_TOKEN_ENDPOINT"),
		os.Getenv("STAGING_PRIVATE_KEY_PASSWORD"),
	)
	if err != nil {
		t.Errorf("main: %s", err)
	}
}
