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

type Operation struct {
	// The UUID of the operation.
	Id string `json:"id"`
	// The URI of the operation, e.g., /tenants/<t>/tenant-spaces/<ts>/volumes/<v>.
	SelfLink string            `json:"self_link"`
	Request  *OperationRequest `json:"request,omitempty"`
	// Request type is a combination of action and resource kind, e.g., \"CreateVolume\".
	RequestType string `json:"request_type"`
	// The request ID specified with the REST call (or system generated) used for idempotence when making API calls. Any name is valid.
	RequestId string `json:"request_id"`
	// The URI of the request collection in which this operation was created. Valid values are \"/\", \"/<tenants>/<t>\" or \"/<tenants>/<t>/tenant-spaces<ts>\".
	RequestCollection string           `json:"request_collection,omitempty"`
	State             *OperationState  `json:"state,omitempty"`
	Result            *OperationResult `json:"result,omitempty"`
	// The latest status of the operation indicates if it is waiting (Pending), active (Running, Aborting) or complete (Succeeded, Failed).
	Status string `json:"status"`
	// Recommended time to wait before getting the operation again to observe status change (polling interval). Unit is milliseconds, e.g., 100.
	RetryIn int32       `json:"retry_in"`
	Error_  *ModelError `json:"error,omitempty"`
	// The time the operation was created, in milliseconds since the Unix.
	CreatedAt int64 `json:"created_at"`
}
