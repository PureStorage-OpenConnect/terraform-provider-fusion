/*
Copyright 2023 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package utilities

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

var ErrImmutableFieldChanged error = errors.New("attempt to update an immutable field")

func GetIdForDataSource() string {
	return strconv.FormatInt(time.Now().Unix(), 10)
}

func CheckImmutableFieldsExcept(ctx context.Context, d *schema.ResourceData, fieldNames ...string) error {
	if d.HasChangesExcept(fieldNames...) {
		d.Partial(true)
		tflog.Error(
			ctx,
			"attempt to update an immutable field",
			"resource_id", d.Id())
		return ErrImmutableFieldChanged
	}
	return nil
}
