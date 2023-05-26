package utilities

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"
)

// Wait on an operation until its status reaches Succeeded (or Completed) or Failed.
// Return succeeded = true if status reaches Succeeded (or Completed), Failed if status reached Failed, and err otherwise.
// On return,
//
//	 op will be up to date with the most recent GET of the operation, EVEN when we're returning an error.
//		if err != nil, then we have an error. Ignore succeeded (it will be false, but it doesn't mean the operation failed.)
//	 If err == nil, then check succeeded. It is true iff (op.Status == "Succeeded" || op.Status == "Completed") && op.Status != "Failed"
func WaitOnOperation(ctx context.Context, op *hmrest.Operation, client *hmrest.APIClient) (succeeded bool, err error) {
	TraceOperation(ctx, op, "waitOnOperation")
	tflog.Debug(ctx, "Waiting for operation",
		"op_type", op.RequestType,
		"op_id", op.Id,
		"op_status", op.Status,
		"op_retry_in", op.RetryIn)
	if op.Status == "" && op.Id == "" && op.RetryIn == 0 {
		tflog.Error(ctx, "waitOnOperation with null op")
		return false, fmt.Errorf("waitOnOperation with null op")
	}
	for op.Status != "Succeeded" && op.Status != "Completed" && op.Status != "Failed" {
		time.Sleep(time.Duration(op.RetryIn) * time.Millisecond)
		opNew, _, err := client.OperationsApi.GetOperation(ctx, op.Id, nil)
		TraceOperation(ctx, &opNew, "waitOnOperation")
		TraceError(ctx, err)
		if err != nil {
			return false, err
		}
		*op = opNew
	}

	// Now op.Status must be Succeeded or Completed or Failed.
	if op.Status == "Failed" {
		tflog.Error(ctx, "waitOnOperation FAILED with Error",
			"error_message", op.Error_,
			"operation", op)
		return false, nil
	}

	// op.Status must be Succeeded or Completed.
	TraceOperation(ctx, op, "waitOnOperation Succeeded")
	return true, nil
}

func ProcessClientError(ctx context.Context, op string, err error) diag.Diagnostics {
	TraceError(ctx, err)
	modelError, convError := hmrest.ToModelError(err)
	if convError != nil {
		tflog.Warn(ctx, "Error while converting error",
			"error_message", convError.Error(),
			"unconverted error", err,
			"operation", op)
		return diag.FromErr(err)
	} else {
		tflog.Error(ctx, "REST ",
			"operation", op,
			"error_message", modelError.Message)
		return diag.Errorf(modelError.Message)
	}
}
