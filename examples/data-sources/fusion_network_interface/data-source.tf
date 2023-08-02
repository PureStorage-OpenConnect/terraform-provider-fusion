data "fusion_network_interface" "network_interface_list" {
    availability_zone = "west-dc-1"
    region = "us-west"
    array = "flasharray1"
}
