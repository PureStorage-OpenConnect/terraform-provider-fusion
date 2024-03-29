---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "fusion_availability_zone Data Source - public"
subcategory: ""
description: |-
  Provides details about any AvailabilityZone matching the given parameters. For more info about the AvailabilityZone type, see its documentation.
---

# fusion_availability_zone (Data Source)

Provides details about any `AvailabilityZone` matching the given parameters. For more info about the `AvailabilityZone` type, see its documentation.

## Example Usage

```terraform
data "fusion_availability_zone" "availability_zone_list" {
    region = "us-west"
}
```

<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `region` (String) Region name.

### Read-Only

- `id` (String) The ID of this resource.
- `items` (List of Object) List matching Availability Zones. (see [below for nested schema](#nestedatt--items))

<a id="nestedatt--items"></a>
### Nested Schema for `items`

Read-Only:

- `display_name` (String)
- `name` (String)
- `region` (String)


