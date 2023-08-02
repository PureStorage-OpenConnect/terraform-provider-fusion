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

type NetworkInterfaceGroupEth struct {
	// The IPv4 prefix for Network Interfaces in this Network Interface Group.
	Prefix string `json:"prefix"`
	// The IPv4 address of the gateway for Network Interfaces in this Network Interface Group.
	Gateway string `json:"gateway,omitempty"`
	// The VLAN ID for this Network Interface Group.
	Vlan int32 `json:"vlan,omitempty"`
	// The MTU for Network Interfaces in this Network Interface Group.
	Mtu int32 `json:"mtu"`
}
