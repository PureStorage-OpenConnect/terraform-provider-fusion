package fusion

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGetProfileConfig_invalidConfigs(t *testing.T) {
	tests := []struct {
		name, config, profileName string
		expected                  string
	}{
		{"no default field",
			`{
				"profiles": {
			  		"test-profile": {
						"env": "test-env",
						"endpoint": "test-endpoint",
						"auth": {
							"issuer_id": "test-issuer",
							"private_pem_file": "test-key-path"
						}
					}
				}
			}`,
			"",
			"config does not have required field `default_profile`"},
		{"no profiles field",
			`{
				"default_profile": "test-profile"
			}`,
			"",
			"config does not have required field `profiles`"},
		{"no profile in profiles",
			`{
				"default_profile": "test-profile",
				"profiles": {
					"test2-profile": {
						"env": "test-env",
						"endpoint": "test-endpoint",
						"auth": {
							"issuer_id": "test-issuer",
							"private_pem_file": "test-key-path"
						}
					}
				}
			}`,
			"",
			"profile does not exist",
		},
		{"no issuer id",
			`{
				"default_profile": "test-profile",
				"profiles": {
					"test-profile": {
						"env": "test-env",
						"endpoint": "test-endpoint",
						"auth": {
							"private_pem_file": "test-key-path"
						}
					}
				}
			}`,
			"",
			"profile does not have required auth fields",
		},
		{"no private key file",
			`{
				"default_profile": "test-profile",
				"profiles": {
					"test-profile": {
						"env": "test-env",
						"endpoint": "test-endpoint",
						"auth": {
							"issuer_id": "test-issuer"
						}
					}
				}
			}`,
			"",
			"profile does not have required auth fields",
		},
		{"no auth field ",
			`{
				"default_profile": "test-profile",
				"profiles": {
					"test-profile": {
						"env": "test-env",
						"endpoint": "test-endpoint"
					}
				}
			}`,
			"",
			"profile does not have required auth fields"},
		{"specified profile does not exist",
			`{
				"default_profile": "test-profile",
				"profiles": {
					"test-profile": {
						"env": "test-env",
						"endpoint": "test-endpoint",
						"auth": {
							"issuer_id": "test-issuer",
							"private_pem_file": "test-key-path"
						}
					}
				}
			}`,
			"not_exists",
			"profile does not exist"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pathToConfig := filepath.Join(t.TempDir(), "fusion.json")
			err := os.WriteFile(pathToConfig, []byte(tt.config), 0644)
			if err != nil {
				t.Errorf("Cannot create config file for testing")
			}

			_, err = GetProfileConfig(pathToConfig, tt.profileName)
			if !strings.Contains(err.Error(), tt.expected) {
				t.Errorf("expected err: %s actual: %v", tt.expected, err)
			}
		})
	}
}

func TestGetProfileConfig_validConfigs(t *testing.T) {
	tests := []struct {
		name, config, profileName string
		expected                  ProfileConfig
	}{
		{"one config without token_endpoint and access_token",
			`{
				"default_profile": "test-profile",
				"profiles": {
					"test-profile": {
						"env": "test-env",
						"endpoint": "test-endpoint",
						"auth": {
							"issuer_id": "test-issuer",
							"private_pem_file": "test-key-path"
						}
					}
				}
			}`,
			"",
			ProfileConfig{
				ApiHost: "test-endpoint",
				Auth: Auth{
					IssuerId:       "test-issuer",
					PrivateKeyFile: "test-key-path",
				},
			}},
		{"specified profile without token_endpoint and access_token",
			`{
				"default_profile": "test-profile1",
				"profiles": {
					"test-profile1": {
						"env": "test-env1",
						"endpoint": "test-endpoint1",
						"auth": {
							"issuer_id": "test-issuer1",
							"private_pem_file": "test-key-path1"
						}
					},
					"test-profile": {
						"env": "test-env",
						"endpoint": "test-endpoint",
						"auth": {
							"issuer_id": "test-issuer",
							"private_pem_file": "test-key-path"
						}
					}
				}
			}`,
			"test-profile",
			ProfileConfig{
				ApiHost: "test-endpoint",
				Auth: Auth{
					IssuerId:       "test-issuer",
					PrivateKeyFile: "test-key-path",
				},
			}},
		{"one config without access_token",
			`{
				"default_profile": "test-profile",
				"profiles": {
					"test-profile": {
						"env": "test-env",
						"endpoint": "test-endpoint",
						"auth": {
							"issuer_id": "test-issuer",
							"private_pem_file": "test-key-path",
							"token_endpoint": "test-token-endpoint"
						}
					}
				}
			}`,
			"",
			ProfileConfig{
				ApiHost: "test-endpoint",
				Auth: Auth{
					IssuerId:       "test-issuer",
					PrivateKeyFile: "test-key-path",
					TokenEndpoint:  "test-token-endpoint",
				},
			}},
		{"one config with access_token",
			`{
				"default_profile": "test-profile",
				"profiles": {
					"test-profile": {
						"env": "test-env",
						"endpoint": "test-endpoint",
						"auth": {
							"access_token": "test-token"
						}
					}
				}
			}`,
			"",
			ProfileConfig{
				ApiHost: "test-endpoint",
				Auth: Auth{
					AccessToken: "test-token",
				},
			}},
		{"one config",
			`{
				"default_profile": "test-profile",
				"profiles": {
					"test-profile": {
						"env": "test-env",
						"endpoint": "test-endpoint",
						"auth": {
							"issuer_id": "test-issuer",
							"private_pem_file": "test-key-path",
							"token_endpoint": "test-token-endpoint",
							"access_token": "test-access-token"
						}
					}
				}
			}`,
			"",
			ProfileConfig{
				ApiHost: "test-endpoint",
				Auth: Auth{
					IssuerId:       "test-issuer",
					PrivateKeyFile: "test-key-path",
					TokenEndpoint:  "test-token-endpoint",
					AccessToken:    "test-access-token",
				},
			}},
		{"multiple configs",
			`{
				"default_profile": "test-profile",
				"profiles": {
				"test-profile2": {
					"env": "test-env",
					"endpoint": "test-endpoint",
					"auth": {
						"issuer_id": "test-issuer",
						"private_pem_file": "test-key-path",
						"token_endpoint": "test-token-endpoint",
						"access_token": "test-access-token"
					}
				},
				"test-profile": {
					"env": "test-env2",
					"endpoint": "test-endpoint2",
					"auth": {
						"issuer_id": "test-issuer2",
						"private_pem_file": "test-key-path2",
						"token_endpoint": "test-token-endpoint2",
						"access_token": "test-access-token2"
					}
				}
				}
			}`,
			"",
			ProfileConfig{
				ApiHost: "test-endpoint2",
				Auth: Auth{
					IssuerId:       "test-issuer2",
					PrivateKeyFile: "test-key-path2",
					TokenEndpoint:  "test-token-endpoint2",
					AccessToken:    "test-access-token2",
				},
			}},
		{"multiple configs and not standard profile",
			`{
				"default_profile": "test-profile2",
				"profiles": {
					"test-profile": {
						"env": "test-env",
						"endpoint": "test-endpoint",
						"auth": {
							"issuer_id": "test-issuer",
							"private_pem_file": "test-key-path",
							"token_endpoint": "test-token-endpoint",
							"access_token": "test-access-token"
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
			}`,
			"test-profile",
			ProfileConfig{
				ApiHost: "test-endpoint",
				Auth: Auth{
					IssuerId:       "test-issuer",
					PrivateKeyFile: "test-key-path",
					TokenEndpoint:  "test-token-endpoint",
					AccessToken:    "test-access-token",
				},
			}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pathToConfig := filepath.Join(t.TempDir(), "fusion.json")
			err := os.WriteFile(pathToConfig, []byte(tt.config), 0644)
			if err != nil {
				t.Errorf("Cannot create config file for testing")
			}

			profile, err := GetProfileConfig(pathToConfig, tt.profileName)
			if err != nil {
				t.Errorf("%s", err)
			}

			if tt.expected != profile {
				t.Errorf("expected: %s actual: %s", tt.expected, profile)
			}
		})
	}
}
