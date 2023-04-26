/*
Copyright 2022 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

// Code generated DO NOT EDIT.
/*
 * Pure Fusion API
 *
 * Pure Fusion is fully API-driven. Most APIs which change the system (POST, PATCH, DELETE) return an Operation in status \"Pending\" or \"Running\". You can poll (GET) the operation to check its status, waiting for it to change to \"Succeeded\" or \"Failed\".
 *
 * API version: 1.1
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */
package fusion

type ApiClient struct {
	// An immutable, globally unique, system generated identifier.
	Id string `json:"id"`
	// The name of the resource, supplied by the user at creation, and used in the URI path of a resource.
	Name string `json:"name"`
	// The URI of the resource.
	SelfLink string `json:"self_link"`
	// The display name of the resource.
	DisplayName string `json:"display_name,omitempty"`
	// The name of API client
	Issuer string `json:"issuer"`
	// Public key in PEM format associated with the API Client
	PublicKey string `json:"public_key"`
	// The last time API client was updated
	LastKeyUpdate float64 `json:"last_key_update"`
	// The last time API client was used
	LastUsed float64 `json:"last_used"`
	// The Id of Principal that created the API Client
	CreatorId string `json:"creator_id"`
}
