---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "fusion_array Data Source - public"
subcategory: ""
description: |-
  
---

# fusion_array (Data Source)





<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `availability_zone` (String)
- `region` (String)

### Read-Only

- `id` (String) The ID of this resource.
- `items` (List of Object) (see [below for nested schema](#nestedatt--items))

<a id="nestedatt--items"></a>
### Nested Schema for `items`

Read-Only:

- `apartment_id` (String)
- `appliance_id` (String)
- `availability_zone` (String)
- `display_name` (String)
- `hardware_type` (String)
- `host_name` (String)
- `maintenance_mode` (Boolean)
- `name` (String)
- `region` (String)
- `unavailable_mode` (Boolean)

