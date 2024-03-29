---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "fusion_storage_service Data Source - public"
subcategory: ""
description: |-
  Provides details about any StorageService matching the given parameters. For more info about the StorageService type, see its documentation.
---

# fusion_storage_service (Data Source)

Provides details about any `StorageService` matching the given parameters. For more info about the `StorageService` type, see its documentation.

## Example Usage

```terraform
data "fusion_storage_service" "storage_service_list" {
    availability_zone = "west-dc-1"
    region = "us-west"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Read-Only

- `id` (String) The ID of this resource.
- `items` (List of Object) List of matching Storage Services. (see [below for nested schema](#nestedatt--items))

<a id="nestedatt--items"></a>
### Nested Schema for `items`

Read-Only:

- `display_name` (String)
- `hardware_types` (Set of String)
- `name` (String)


