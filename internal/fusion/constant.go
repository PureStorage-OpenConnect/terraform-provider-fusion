/*
Copyright 2023 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package fusion

const (
	optionIqn                               = "iqn"
	optionPersonality                       = "personality"
	optionName                              = "name"
	optionDisplayName                       = "display_name"
	optionAvailabilityZone                  = "availability_zone"
	optionRegion                            = "region"
	optionGroupType                         = "group_type"
	optionGroupTypeEth                      = "eth"
	optionGateway                           = "gateway"
	optionPrefix                            = "prefix"
	optionMtu                               = "mtu"
	optionItems                             = "items"
	optionStorageService                    = "storage_service"
	optionSizeLimit                         = "size_limit"
	optionIopsLimit                         = "iops_limit"
	optionBandwidthLimit                    = "bandwidth_limit"
	optionSourceLink                        = "source_link"
	optionSize                              = "size"
	optionTenant                            = "tenant"
	optionTenantSpace                       = "tenant_space"
	optionSnapshot                          = "snapshot"
	optionVolumeSnapshot                    = "volume_snapshot"
	optionVolume                            = "volume"
	optionIscsi                             = "iscsi"
	optionAddress                           = "address"
	optionNetworkInterfaceGroups            = "network_interface_groups"
	optionApplianceId                       = "appliance_id"
	optionHostName                          = "host_name"
	optionHardwareType                      = "hardware_type"
	optionApartmentId                       = "apartment_id"
	optionMaintenanceMode                   = "maintenance_mode"
	optionUnavailableMode                   = "unavailable_mode"
	optionPreexistingRegion                 = "pure-us-west"
	optionPreexistingAvailabilityZone       = "az1"
	optionLocalRPO                          = "local_rpo"
	optionLocalRetention                    = "local_retention"
	maxDisplayName                          = 256
	storageClassIopsMin               int64 = 100
	storageClassIopsMax               int64 = 100 * 1000 * 1000
	storageClassBandwidthMin          int64 = 1048576              // 2^20
	storageClassBandwidthMax          int64 = 549755813888         // 2^39
	storageClassSizeMin               int64 = 1048576              // 1 MiB
	storageClassSizeMax               int64 = 4 * 1125899906842624 // 4 Pib
	optionArrayType                         = "array_type"
	optionMediaType                         = "media_type"
	optionPublicKey                         = "public_key"
)

var hapPersonalities = []string{
	"windows", "linux", "esxi", "oracle-vm-server", "aix", "hitachi-vsp", "hpux", "solaris", "vms",
}
