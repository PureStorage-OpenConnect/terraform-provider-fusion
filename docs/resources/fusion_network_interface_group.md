---
# generated by https://github.com/hashicorp/terraform-plugin-docs
page_title: "fusion_network_interface_group Resource - public"
subcategory: ""
description: |-
  
---

# fusion_network_interface_group (Resource)





<!-- schema generated by tfplugindocs -->
## Schema

### Required

- `availability_zone` (String) The name of the availability zone for the network interface group.
- `eth` (Block List, Min: 1, Max: 1) (see [below for nested schema](#nestedblock--eth))
- `name` (String) The name of the network interface group.
- `region` (String) Region for the network interface group.

### Optional

- `display_name` (String) The human name of the network interface group. If not provided, defaults to I(name).
- `group_type` (String) The type of network interface group.

### Read-Only

- `id` (String) The ID of this resource.

<a id="nestedblock--eth"></a>
### Nested Schema for `eth`

Required:

- `gateway` (String) Address of the subnet gateway. Currently must be a valid IPv4 address.
- `prefix` (String) Network prefix in CIDR notation. Required to create a new network interface group. Currently only IPv4 addresses with subnet mask are supported.

Optional:

- `mtu` (Number) MTU setting for the subnet.

