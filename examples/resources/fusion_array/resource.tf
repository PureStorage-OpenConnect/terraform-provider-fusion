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
