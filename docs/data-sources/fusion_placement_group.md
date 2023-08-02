---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "fusion_placement_group Data Source - public"
subcategory: ""
description: |-
  Provides details about any PlacementGroup matching the given parameters. For more info about the PlacementGroup type, see its documentation.
---

# fusion_placement_group (Data Source)

Provides details about any `PlacementGroup` matching the given parameters. For more info about the `PlacementGroup` type, see its documentation.

## Example Usage

```terraform
data "fusion_placement_group" "placement_group_list" {
    tenant = "database-team"
    tenant_space = "mongodb"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `tenant` (String) The name of the Tenant.
- `tenant_space` (String) The name of the Tenant Space.

### Optional

- `iqn` (String) The iSCSI qualified name (IQN) associated with the Placement Group.

### Read-Only

- `id` (String) The ID of this resource.
- `items` (List of Object) List of matching Placement Groups. (see [below for nested schema](#nestedatt--items))

<a id="nestedatt--items"></a>
### Nested Schema for `items`

Read-Only:

- `array` (String)
- `availability_zone` (String)
- `destroy_snapshots_on_delete` (Boolean)
- `display_name` (String)
- `name` (String)
- `region` (String)
- `storage_service` (String)
- `tenant` (String)
- `tenant_space` (String)

