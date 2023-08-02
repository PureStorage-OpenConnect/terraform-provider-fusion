/*
Copyright 2023 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package fusion

import (
	"context"

	"github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/utilities"
	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

type DataSource interface {
	// Synchronously reads the data source via its REST API.
	ReadDataSource(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) (err error)
}

type BaseDataSourceFunctions struct {
	*schema.Resource
	DataSourceKind string
	DataSource     DataSource
}

func genericDataSourceDescription(dataSourceName string) string {
	return "Provides details about any `" + dataSourceName + "` matching the given parameters. For more info about the `" +
		dataSourceName + "` type, see its documentation."
}

func NewBaseDataSourceFunctions(resourceKind string, dataSource DataSource, dsSchema map[string]*schema.Schema) *BaseDataSourceFunctions {
	result := &BaseDataSourceFunctions{&schema.Resource{}, resourceKind, dataSource}
	result.Resource.ReadContext = result.dataSourceRead
	result.Resource.Schema = dsSchema
	result.Resource.Description = genericDataSourceDescription(resourceKind)
	return result
}

func (f *BaseDataSourceFunctions) dataSourceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client, _ := f.dataSourceBoilerplate(ctx, "Read", d, m)
	err := f.DataSource.ReadDataSource(ctx, client, d)
	return utilities.ProcessClientError(ctx, "read", err)
}

// A function used at the top of the datasource READ function to grab stuff we need.
func (f *BaseDataSourceFunctions) dataSourceBoilerplate(ctx context.Context, action string, d *schema.ResourceData, m interface{}) (*hmrest.APIClient, context.Context) {
	ctx = tflog.With(ctx, "datasource_kind", f.DataSourceKind)
	tflog.Debug(ctx, "datasource", "action", action, "state", d.State())

	client := m.(*hmrest.APIClient)

	return client, ctx
}
