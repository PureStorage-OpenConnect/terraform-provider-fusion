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

// (Provider)
type NetworkInterface struct {
	// An immutable, globally unique, system generated identifier.
	Id string `json:"id"`
	// The name of the resource, supplied by the user at creation, and used in the URI path of a resource.
	Name string `json:"name"`
	// The URI of the resource.
	SelfLink string `json:"self_link"`
	// The display name of the resource.
	DisplayName      string               `json:"display_name,omitempty"`
	Region           *RegionRef           `json:"region,omitempty"`
	AvailabilityZone *AvailabilityZoneRef `json:"availability_zone,omitempty"`
	Array            *ArrayRef            `json:"array,omitempty"`
	// The interface type.
	InterfaceType string               `json:"interface_type"`
	Eth           *NetworkInterfaceEth `json:"eth,omitempty"`
	// The services provided by this Network Interface.
	Services []string `json:"services,omitempty"`
	// True if interface is in use.
	Enabled               bool                      `json:"enabled"`
	NetworkInterfaceGroup *NetworkInterfaceGroupRef `json:"network_interface_group,omitempty"`
	// Configured speed of this Network Interface. Typically this is the maximum speed of the port or bond represented by the Network Interface.
	MaxSpeed int64 `json:"max_speed"`
}