/*
Copyright 2022 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package utilities

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"reflect"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"
)

func TraceOperation(ctx context.Context, op *hmrest.Operation, userMessage string) {
	tflog.Trace(ctx, "trace_operation",
		"user_message", userMessage,
		"op_id", op.Id,
		"op_request_type", op.RequestType,
		"op_error_dump", fmt.Sprintf("%#v", op.Error_),
		"op_status", op.Status,
		"op_retry_in", op.RetryIn,
	)
}

func TracePatch(ctx context.Context, resourceType string, resourceName string, parameterName string, value interface{}, patchIdx int) {
	tflog.Trace(
		ctx,
		"update",
		"resource_type", resourceType,
		"resource_name", resourceName,
		"parameter", parameterName,
		"to", value,
		"patch_idx", patchIdx)
}

// Recursively print error details
func TraceError(ctx context.Context, err error) {
	for err != nil {
		tflog.Trace(ctx, "trace_error",
			"error_message", err.Error(),
			"error_type", fmt.Sprintf("%T", err),
			"error_dump", fmt.Sprintf("%#v", err),
			"error_package_path", reflect.TypeOf(err).PkgPath(),
		)

		// Check for common fields that match given types, and print extra info
		if e, ok := err.(hmrest.GenericSwaggerError); ok {
			tflog.Trace(ctx, "trace_error",
				"generic_swagger_body", string(e.Body()),
				"generic_swagger_model_dump", fmt.Sprintf("%#v", e.Model()),
			)
		}

		if response, ok := reflectFieldAsInterface(err, "Response").(*http.Response); ok {
			if response == nil {
				tflog.Trace(ctx, "trace_error",
					"response", nil,
				)
			} else {
				tflog.Trace(ctx, "trace_error",
					"response_status_code", response.StatusCode,
					"response_status", response.Status,
				)
				if response.Request != nil {
					tflog.Trace(ctx, "trace_error",
						"request_uri", response.Request.RequestURI,
						"request_host", response.Request.Host,
					)
				}
			}
		}

		if body, ok := reflectFieldAsInterface(err, "Body").([]uint8); ok {
			tflog.Trace(ctx, "trace_error",
				"body", string(body),
			)
		}

		err = errors.Unwrap(err)
	}
}
