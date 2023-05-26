/*
Copyright 2023 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package fusion

import (
	"context"
	"strconv"

	"github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/utilities"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"
)

// Implements DataSource
type storageClassDataSource struct{}

// This is our entry point for the Storage Class data source
func dataSourceStorageClass() *schema.Resource {
	ds := &storageClassDataSource{}

	dsSchema := map[string]*schema.Schema{
		optionItems: {
			Type:     schema.TypeList,
			Computed: true,
			Elem: &schema.Resource{
				Schema: schemaStorageClass(),
			},
		},
		optionStorageService: {
			Type:         schema.TypeString,
			Required:     true,
			ValidateFunc: validation.StringIsNotEmpty,
		},
	}

	storageClassDataSourceFunctions := NewBaseDataSourceFunctions("StorageClass", ds, dsSchema)

	return storageClassDataSourceFunctions.Resource
}

func (ds *storageClassDataSource) ReadDataSource(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) error {
	resp, _, err := client.StorageClassesApi.ListStorageClasses(ctx, rdString(ctx, d, optionStorageService), nil)
	if err != nil {
		return err
	}

	storageClassesList := make([]map[string]interface{}, resp.Count)

	for i, sc := range resp.Items {
		storageClassesList[i] = map[string]interface{}{
			optionName:           sc.Name,
			optionDisplayName:    sc.DisplayName,
			optionStorageService: sc.StorageService.Name,
			optionBandwidthLimit: strconv.FormatInt(sc.BandwidthLimit, 10),
			optionSizeLimit:      strconv.FormatInt(sc.SizeLimit, 10),
			optionIopsLimit:      strconv.FormatInt(sc.IopsLimit, 10),
		}
	}

	if err := d.Set(optionItems, storageClassesList); err != nil {
		return err
	}

	d.SetId(utilities.GetIdForDataSource())

	return nil
}
