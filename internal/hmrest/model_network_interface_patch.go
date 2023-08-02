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
 * API version: 1.2
 * Generated by: Swagger Codegen (https://github.com/swagger-api/swagger-codegen.git)
 */
package fusion

// (Provider)
type NetworkInterfacePatch struct {
	DisplayName           *NullableString           `json:"display_name,omitempty"`
	Eth                   *NetworkInterfacePatchEth `json:"eth,omitempty"`
	Enabled               *NullableBoolean          `json:"enabled,omitempty"`
	NetworkInterfaceGroup *NullableString           `json:"network_interface_group,omitempty"`
}
