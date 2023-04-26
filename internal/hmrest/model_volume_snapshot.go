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

type VolumeSnapshot struct {
	// An immutable, globally unique, system generated identifier.
	Id string `json:"id"`
	// The name of the resource, supplied by the user at creation, and used in the URI path of a resource.
	Name string `json:"name"`
	// The URI of the resource.
	SelfLink string `json:"self_link"`
	// The display name of the resource.
	DisplayName string `json:"display_name,omitempty"`
	// A serial number generated by the system when the volume snapshot is created. The serial number is unique across all arrays.
	SerialNumber string `json:"serial_number"`
	// The serial number of the volume this volume snapshot is created from.
	VolumeSerialNumber string `json:"volume_serial_number"`
	// The volume snapshot creation time. Measured in milliseconds since the UNIX epoch.
	CreatedAt int64 `json:"created_at"`
	// Volume snapshots with the same consistency_id are crash consistency.
	ConsistencyId string `json:"consistency_id"`
	// True if the volume snapshot has been destroyed and is pending eradication. The time_remaining value displays the amount of time left until the destroyed volume snapshot is permanently eradicated.
	Destroyed bool `json:"destroyed,omitempty"`
	// The amount of time left until the destroyed volume snapshot is permanently eradicated. Only valid when destroyed is true. Measured in milliseconds. An expired but not yet eradicated volume snapshot has destroyed=true and time_remaining=0.
	TimeRemaining int64 `json:"time_remaining,omitempty"`
	// The virtual size of the volume snapshot. Measured in bytes.
	Size             int64                `json:"size"`
	Tenant           *TenantRef           `json:"tenant"`
	TenantSpace      *TenantSpaceRef      `json:"tenant_space"`
	Snapshot         *SnapshotRef         `json:"snapshot"`
	Volume           *VolumeRef           `json:"volume,omitempty"`
	ProtectionPolicy *ProtectionPolicyRef `json:"protection_policy,omitempty"`
	PlacementGroup   *PlacementGroupRef   `json:"placement_group"`
}
