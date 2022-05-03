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
 * API version: 1.0
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */
package fusion

type TenantSpaceList struct {
	// count of items in the list
	Count int32 `json:"count"`
	// True if not all items in the search were returned in the provided array.
	MoreItemsRemaining bool `json:"more_items_remaining,omitempty"`
	// A JSON array of Tenant Spaces
	Items []TenantSpace `json:"items"`
}
