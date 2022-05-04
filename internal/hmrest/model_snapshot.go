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

type Snapshot struct {
	// An immutable, globally unique, system generated identifier.
	Id string `json:"id"`
	// The name of the resource, supplied by the user at creation, and used in the URI path of a resource.
	Name string `json:"name"`
	// The URI of the resource.
	SelfLink string `json:"self_link"`
	// The display name of the resource.
	DisplayName string          `json:"display_name,omitempty"`
	Tenant      *TenantRef      `json:"tenant"`
	TenantSpace *TenantSpaceRef `json:"tenant_space"`
	// The URI of volume snapshots in the snapshot.
	VolumeSnapshotsLink string               `json:"volume_snapshots_link"`
	ProtectionPolicy    *ProtectionPolicyRef `json:"protection_policy,omitempty"`
	// Unimplemented - The amount of time left until the destroyed snapshot is permanently eradicated. Measured in milliseconds. Before the time_remaining period has elapsed, the destroyed snapshot can be recovered by setting destroyed=false.
	TimeRemaining int64 `json:"time_remaining,omitempty"`
}