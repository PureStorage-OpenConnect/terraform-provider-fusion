/*
Copyright 2022 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package fusion

import (
	"context"
	"fmt"

	"github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/utilities"
	hmrest "github.com/PureStorage-OpenConnect/terraform-provider-fusion/internal/hmrest"

	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/hashicorp/terraform-plugin-sdk/v2/diag"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
)

//
// ResourceProvider is implemented for each resource.
//

type ResourcePatch interface {
}
type ResourcePost interface { // could have: id, name
}
type RequestSpec interface{}

// type InvokeReadMultiAPI func(ctx context.Context, client *hmrest.APIClient) (resource []interface{}, err error)
// type InvokeReadSingleAPI func(ctx context.Context, client *hmrest.APIClient) (resource interface{}, err error)
type InvokeWriteAPI func(ctx context.Context, client *hmrest.APIClient, body RequestSpec) (operation *hmrest.Operation, err error)

// This is what you need to implement as the owner of a resource. Use the BaseResourceFunctions to build a schema.
type ResourceProvider interface {
	// PrepareCreate returns a function which will call the Create REST API on this object and return an operation;
	// also returns the post body to pass to that function. Invoke the function using the post body.
	PrepareCreate(ctx context.Context, d *schema.ResourceData) (fn InvokeWriteAPI, post ResourcePost, err error)

	// ReadResource synchronously reads the resource via its REST API.
	ReadResource(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) (err error)

	// PrepareUpdate returns a function which will call the Update REST API on this object and return an operation.
	// Invoke it with each of the patches.
	PrepareUpdate(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) (fn InvokeWriteAPI, patches []ResourcePatch, err error)

	// PrepareDelete returns a function which will call the Delete REST API on this object and return an operation. Invoke it.
	PrepareDelete(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) (fn InvokeWriteAPI, err error)

	// ResourceImporter is a function which is called when Terraform is importing a resource.
	ImportResource(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) (ds []*schema.ResourceData, err error)
}

// Actually, an empty implementation which returns "not implemented" errors. :-)
type BaseResourceProvider struct {
	ResourceKind string
}

func (p *BaseResourceProvider) PrepareCreate(ctx context.Context, d *schema.ResourceData) (fn InvokeWriteAPI, post ResourcePost, err error) {
	return nil, nil, fmt.Errorf("unsupported operation: create %s", p.ResourceKind)
}

func (p *BaseResourceProvider) ReadResource(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) (err error) {
	return fmt.Errorf("unsupported operation: read %s", p.ResourceKind)
}

func (p *BaseResourceProvider) PrepareUpdate(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) (fn InvokeWriteAPI, patches []ResourcePatch, err error) {
	return nil, nil, fmt.Errorf("unsupported operation: update %s", p.ResourceKind)
}

func (p *BaseResourceProvider) PrepareDelete(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) (fn InvokeWriteAPI, err error) {
	return nil, fmt.Errorf("unsupported operation: delete %s", p.ResourceKind)
}

func (p *BaseResourceProvider) ImportResource(ctx context.Context, client *hmrest.APIClient, d *schema.ResourceData) (ds []*schema.ResourceData, err error) {
	return nil, fmt.Errorf("unsupported operation: import %s", p.ResourceKind)
}

//
// Resource functions internally implement the interface defined by Terraform.
//

// Implements interface to Terraform: resource-CRUD
type BaseResourceFunctions struct {
	*schema.Resource
	ResourceKind string // We're for volume, tenant space, storage class, etc. More likely to come.
	Provider     ResourceProvider
}

func NewBaseResourceFunctions(resourceKind string, provider ResourceProvider) *BaseResourceFunctions {
	result := &BaseResourceFunctions{&schema.Resource{}, resourceKind, provider}
	result.Resource.CreateContext = result.resourceCreate
	result.Resource.ReadContext = result.resourceRead
	result.Resource.UpdateContext = result.resourceUpdate
	result.Resource.DeleteContext = result.resourceDelete
	result.Resource.Importer = &schema.ResourceImporter{
		StateContext: result.resourceImport,
	}
	return result
}

// resourceCreate creates a HM Resource, generically, relying on ResourceProvider.ComputePost
func (f *BaseResourceFunctions) resourceCreate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client, ctx := f.resourceBoilerplate(ctx, "Create", d, m)

	// invoke the interface
	callAPI, body, err := f.Provider.PrepareCreate(ctx, d)
	if err != nil {
		tflog.Error(ctx, "in preparing post", "error_message", err)
		return diag.FromErr(err)
	}
	tflog.Debug(ctx, "Post", "body", body)
	op, err := callAPI(ctx, client, body)
	if err != nil {
		utilities.TraceError(ctx, err)
		return utilities.ProcessClientError(ctx, "create", err)
	}

	// Wait on Operation
	succeeded, err := utilities.WaitOnOperation(ctx, op, client) // updates op with latest
	if err != nil {
		utilities.TraceError(ctx, err)
		return utilities.ProcessClientError(ctx, "get wait status", err)
	}

	if !succeeded {
		tflog.Error(ctx, "REST create failed", "error_message", op.Error_.Message,
			"PureCode", op.Error_.PureCode, "HttpCode", op.Error_.HttpCode)
		return diag.Errorf(op.Error_.Message)
	}

	// succeeded!
	tflog.Debug(ctx, "created successfully", "operation_result", op.Result)
	d.SetId(op.Result.Resource.Id)

	return f.resourceRead(ctx, d, m)
}

func (f *BaseResourceFunctions) resourceRead(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client, _ := f.resourceBoilerplate(ctx, "Read", d, m)
	err := f.Provider.ReadResource(ctx, client, d)
	return utilities.ProcessClientError(ctx, "read", err)
}

func (f *BaseResourceFunctions) resourceUpdate(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client, ctx := f.resourceBoilerplate(ctx, "Update", d, m)

	callAPI, patches, err := f.Provider.PrepareUpdate(ctx, client, d)
	if err != nil {
		d.Partial(true)
		return diag.FromErr(err)
	}

	err = executePatches(ctx, callAPI, patches, client, "resourceUpdate")
	if err != nil {
		d.Partial(true)
		return utilities.ProcessClientError(ctx, "resourceUpdate", err)
	}

	return f.resourceRead(ctx, d, m)
}

func (f *BaseResourceFunctions) resourceDelete(ctx context.Context, d *schema.ResourceData, m interface{}) diag.Diagnostics {
	client, ctx := f.resourceBoilerplate(ctx, "Delete", d, m)

	callAPI, err := f.Provider.PrepareDelete(ctx, client, d)
	if err != nil {
		tflog.Error(ctx, "in compute delete or volume: REST DELETE volume failed", "error_message", err)
		return diag.FromErr(err)
	}

	op, err := callAPI(ctx, client, nil) // no body for delete
	if err != nil {
		return utilities.ProcessClientError(ctx, "delete", err)
	}

	succeeded, err := utilities.WaitOnOperation(ctx, op, client)
	if err != nil {
		return utilities.ProcessClientError(ctx, "get wait status", err)
	}

	if !succeeded {
		tflog.Error(ctx, "REST delete failed", "error_message", op.Error_.Message,
			"PureCode", op.Error_.PureCode, "HttpCode", op.Error_.HttpCode)
		return diag.Errorf(op.Error_.Message)
	}

	return nil
}

func (f *BaseResourceFunctions) resourceImport(ctx context.Context, d *schema.ResourceData, m interface{}) ([]*schema.ResourceData, error) {
	client, ctx := f.resourceBoilerplate(ctx, "Import", d, m)
	return f.Provider.ImportResource(ctx, client, d)
}

func executePatches(ctx context.Context, fn InvokeWriteAPI, patches []ResourcePatch, client *hmrest.APIClient, opSource string) error {
	// Start operations for each update
	for i, p := range patches {
		ctx := tflog.With(ctx, "patch_idx", i)
		tflog.Debug(ctx, "Starting operation to apply a patch", "patch_op", opSource, "patch_num", i, "patch", p)
		op, err := fn(ctx, client, p)
		utilities.TraceOperation(ctx, op, "Applying Patch")
		if err != nil {
			return err
		}

		// Right now we are forcing all the operations to complete serially
		// because there are certain patch operations that need to happen
		// in order.  Later on we can get more clever and try to come up
		// with patch groups that can be done in parallel together
		succeeded, err := utilities.WaitOnOperation(ctx, op, client)
		if err != nil {
			return err
		}
		if !succeeded {
			return fmt.Errorf("operation failed Message:%s ID:%s", op.Error_.Message, op.Id)
		}
	}
	return nil
}

// A function used at the top of each CRUD function to grab stuff we need. Belongs in resource_functions.
func (f *BaseResourceFunctions) resourceBoilerplate(ctx context.Context, action string, d *schema.ResourceData, m interface{}) (*hmrest.APIClient, context.Context) {
	ctx = tflog.With(ctx, "resource_kind", f.ResourceKind)
	tflog.Debug(ctx, "resource", "action", action, "state", d.State())

	client := m.(*hmrest.APIClient)

	return client, ctx
}

// These RD wrappers don't do much yet, just ensure we get good logging on errors.
// But for future's sake ... good fences make good neighbors.
func rdString(ctx context.Context, d *schema.ResourceData, key string) string {
	value := d.Get(key)
	if value == nil {
		return ""
	}
	s, ok := value.(string) // If not set, provides empty string.
	if !ok {
		tflog.Error(ctx, "Got unexpected type value", "key", key, "type", fmt.Sprintf("%T", value), "value", value)
		return value.(string) // Force the runtime error if not ok.
	}
	return s
}

func rdStringDefault(ctx context.Context, d *schema.ResourceData, key string, defaultValue string) string {
	value := rdString(ctx, d, key)
	if value == "" {
		return defaultValue
	}
	return value
}

func rdInt(d *schema.ResourceData, key string) int { return d.Get(key).(int) }

func rdStringSet(ctx context.Context, d *schema.ResourceData, key string) []string {
	value := d.Get(key)
	set, ok := value.(*schema.Set)
	if !ok {
		tflog.Error(ctx, "Got unexpected type value", "key", key, "type", fmt.Sprintf("%T", value), "value", value)
		set = value.(*schema.Set)
	} else if set == nil {
		tflog.Error(ctx, "Got unexpected value", "key", key, "type", fmt.Sprintf("%T", value), "value", value)
	}

	interfaceList := set.List()
	stringList := make([]string, len(interfaceList))

	for i, val := range interfaceList {
		stringList[i] = val.(string)
	}

	return stringList
}

func getFirstError(errs ...error) error {
	for _, err := range errs {
		if err != nil {
			return err
		}
	}
	return nil
}
