/*
Copyright 2022 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package auth

import (
	"context"
	"crypto/rsa"
	"fmt"
	"io/ioutil"
	"time"

	"github.com/golang-jwt/jwt"
	"golang.org/x/oauth2"
)

const DefaultAuthNEndpoint = "https://api.pure1.purestorage.com/oauth2/1.0/token"
const AuthNEndpointOverrideEnvVarName = "FUSION_TOKEN_ENDPOINT"

func ReadPrivateKeyFile(privateKeyPath string) (string, error) {
	privateKey, err := ioutil.ReadFile(privateKeyPath)
	if err != nil {
		return "", fmt.Errorf("failed to read private key file path:%s err:%w", privateKeyPath, err)
	}

	return string(privateKey), nil
}

func DecryptPrivateKeyWithPassword(privateKeyString, privateKeyPassword string) (*rsa.PrivateKey, error) {
	privateKey, err := jwt.ParseRSAPrivateKeyFromPEMWithPassword([]byte(privateKeyString), privateKeyPassword)
	if err != nil {
		return privateKey, fmt.Errorf("failed to parse private key with password %w", err)
	}

	return privateKey, nil
}

func StringToPrivateKey(privateKeyString, privateKeyPassword string) (*rsa.PrivateKey, error) {
	var privateKey *rsa.PrivateKey
	if privateKeyPassword != "" {
		return DecryptPrivateKeyWithPassword(privateKeyString, privateKeyPassword)
	}
	privateKey, err := jwt.ParseRSAPrivateKeyFromPEM([]byte(privateKeyString))
	if err != nil {
		return privateKey, fmt.Errorf("failed to parse private key %w", err)
	}

	return privateKey, nil
}

// Connects to Pure1 Authentication endpoint with issuerID signed with private key specified by given path
// This returns an access token that is good for one hour, in any exceptional cases it returns an empty string
// privateKeyPassword is not a mandatory, it can be empty if private key doesn't encrypted
func GetPure1SelfSignedAccessTokenGoodForOneHour(ctx context.Context, issuerId, privateKeyString, authNEndpoint, privateKeyPassword string) (string, error) {
	privateKey, err := StringToPrivateKey(privateKeyString, privateKeyPassword)
	if err != nil {
		return "", err
	}

	signedIdentityToken, err := jwt.NewWithClaims(jwt.SigningMethodRS256, jwt.StandardClaims{
		Issuer:    issuerId,
		IssuedAt:  time.Now().Unix(),
		ExpiresAt: time.Now().Add(3600 * time.Second).Unix(),
	}).SignedString(privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign identity token err:%w", err)
	}

	config := oauth2.Config{Endpoint: oauth2.Endpoint{TokenURL: authNEndpoint}}
	exchangedToken, err := config.Exchange(ctx, "",
		oauth2.SetAuthURLParam("grant_type", "urn:ietf:params:oauth:grant-type:token-exchange"),
		oauth2.SetAuthURLParam("subject_token", signedIdentityToken),
		oauth2.SetAuthURLParam("subject_token_type", "urn:ietf:params:oauth:token-type:jwt"),
	)
	if err != nil {
		return "", fmt.Errorf("failed to exchange token endpoint:%s err:%w", authNEndpoint, err)
	}
	return exchangedToken.AccessToken, nil
}
