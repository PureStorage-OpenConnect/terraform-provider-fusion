resource "fusion_region" "us_east" {
  name         = "us-east"
  display_name = "US East"
}

resource "fusion_availability_zone" "east_dc_1" {
  name         = "east-dc-1"
  display_name = "East DC 1"
  region       = fusion_region.us_east.name
}

resource "fusion_array" "flasharray1" {
  name              = "flasharray1"
  display_name      = "Flasharray 1"
  region            = fusion_region.us_east.name
  availability_zone = fusion_availability_zone.east_dc_1.name
  hardware_type     = "flash-array-x"
  appliance_id      = "1187351-242133817-5976825671211737521"
  host_name         = "flasharray1"
}

resource "fusion_network_interface_group" "east_dc_1_primary" {
  name              = "${fusion_availability_zone.east_dc_1.name}-primary"
  display_name      = "${fusion_availability_zone.east_dc_1.display_name} Primary NIG"
  availability_zone = fusion_availability_zone.east_dc_1.name
  region            = fusion_availability_zone.east_dc_1.region
  group_type        = "eth"
  eth {
    prefix  = "172.17.1.1/16"
    gateway = "172.17.1.1"
  }
}

resource "fusion_network_interface" "flasharray1_eth1" {
  name                    = "${fusion_array.flasharray1.name}-eth1"
  display_name            = "${fusion_array.flasharray1.display_name} Ethernet 1"
  region                  = fusion_array.flasharray1.region
  availability_zone       = fusion_array.flasharray1.availability_zone
  array                   = fusion_array.flasharray1.name
  enabled                 = true
  network_interface_group = fusion_network_interface_group.east_dc_1_primary.name
  interface_type          = "eth"
  eth {
    address = "172.17.1.10/32"
    gateway = "172.17.1.1"
  }
}
