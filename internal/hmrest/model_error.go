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

type ModelError struct {
	// The error message, e.g., \"Cannot delete a volume while it is connected to hosts\".
	Message string `json:"message"`
	// Key-value pairs containing details about the error.
	Details map[string]string `json:"details"`
	// Pure Code describing the error. May be more specific than the HTTP code. The code may be one of the following. INTERNAL - An internal error occurred. NOT_FOUND - A resource was not found. ALREADY_EXISTS - A resource cannot be created because one already exists by this name. INVALID_ARGUMENT - An argument value is incorrect. NOT_AUTHENTICATED - The client is not authenticated. PERMISSION_DENIED - The client is authenticated, but lacks a specific permission required for this action. NOT_IMPLEMENTED - The functionality has not been implemented yet. FAILED_PRECONDITION - A precondition of the action is not fulfilled. For instance, trying to use a resource that is being deleted. CONFLICT - This action came into conflict with another. FAILED_TRANSACTION - An action could not be performed due to conflict with another transaction. CANCELED - The action was canceled. DEADLINE_EXCEEDED - The action could not be completed in the time allotted. UNAVAILABLE - A required service is unavailable. EXHAUSTED - A required resource is not available. For instance, there is no storage space in the Placement Group.
	PureCode string `json:"pure_code"`
	// The HTTP code returned by the request. It will be the same as the header response status code. The code may be one of the following. 400 Bad Request - The request payload is malformed; e.g. incorrection JSON. 401 Unauthorized - The client is not authenticated. 403 Forbidden - The client is authenticated, but lacks a specific permission required for this action. 404 Not Found - A resource was not found. 408 Request Timeout - The action could not be completed in the time allotted. 409 Conflict - This action came into conflict with another. 412 Precondition Failed - A precondition of the action is not fulfilled. For instance, trying to use a resource that is being deleted. 422 Unprocessable Entity - An argument value is incorrect. The request was well-formed but was unable to be followed due to semantic errors; e.g. an incorrect enum value. 429 Too Many Requests - The user has sent too many requests in a given amount of time (\"rate limiting\"). 500 Internal Server Error - An internal error occurred. 501 Not Implemented - The functionality has not been implemented yet. 503 Service Unavailable - A required service is unavailable.
	HttpCode int32 `json:"http_code"`
}
