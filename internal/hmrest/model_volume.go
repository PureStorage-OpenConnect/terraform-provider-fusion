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

type Volume struct {
	// An immutable, globally unique, system generated identifier.
	Id string `json:"id"`
	// The name of the resource, supplied by the user at creation, and used in the URI path of a resource.
	Name string `json:"name"`
	// The URI of the resource.
	SelfLink string `json:"self_link"`
	// The display name of the resource.
	DisplayName string `json:"display_name,omitempty"`
	// The size of the volume
	Size                 int64                 `json:"size,omitempty"`
	Tenant               *TenantRef            `json:"tenant"`
	TenantSpace          *TenantSpaceRef       `json:"tenant_space"`
	StorageClass         *StorageClassRef      `json:"storage_class"`
	ProtectionPolicy     *ProtectionPolicyRef  `json:"protection_policy,omitempty"`
	PlacementGroup       *PlacementGroupRef    `json:"placement_group,omitempty"`
	Array                *ArrayRef             `json:"array,omitempty"`
	CreatedAt            int64                 `json:"created_at,omitempty"`
	SourceVolumeSnapshot *VolumeSnapshotRef    `json:"source_volume_snapshot,omitempty"`
	HostAccessPolicies   []HostAccessPolicyRef `json:"host_access_policies,omitempty"`
	// Volume Serial Numbers, aka LUN Serial Numbers. This will be visible to initiators that connect to the volume.
	SerialNumber string  `json:"serial_number"`
	Target       *Target `json:"target,omitempty"`
}
