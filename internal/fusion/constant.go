/*
Copyright 2023 Pure Storage Inc
SPDX-License-Identifier: Apache-2.0
*/

package fusion

const (
	optionId                                = "id"
	optionIqn                               = "iqn"
	optionPersonality                       = "personality"
	optionName                              = "name"
	optionDisplayName                       = "display_name"
	optionAvailabilityZone                  = "availability_zone"
	optionRegion                            = "region"
	optionGroupType                         = "group_type"
	optionEth                               = "eth"
	optionFc                                = "fc"
	optionGateway                           = "gateway"
	optionPrefix                            = "prefix"
	optionMtu                               = "mtu"
	optionItems                             = "items"
	optionStorageService                    = "storage_service"
	optionStorageClass                      = "storage_class"
	optionPlacementGroup                    = "placement_group"
	optionProtectionPolicy                  = "protection_policy"
	optionEradicateOnDelete                 = "eradicate_on_delete"
	optionCreatedAt                         = "created_at"
	optionVolumeId                          = "volume_id"
	optionProtectionPolicyId                = "protection_policy_id"
	optionPlacementGroupId                  = "placement_group_id"
	optionSerialNumber                      = "serial_number"
	optionTargetIscsiIqn                    = "target_iscsi_iqn"
	optionTargetIscsiAddresses              = "target_iscsi_addresses"
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
	optionDiscoveryInterfaces               = "discovery_interfaces"
	optionCbsAzureIscsi                     = "cbs_azure_iscsi"
	optionAddress                           = "address"
	optionNetworkInterfaceGroups            = "network_interface_groups"
	optionApplianceId                       = "appliance_id"
	optionHostName                          = "host_name"
	optionHostAccessPolicies                = "host_access_policies"
	optionHardwareType                      = "hardware_type"
	optionApartmentId                       = "apartment_id"
	optionMaintenanceMode                   = "maintenance_mode"
	optionUnavailableMode                   = "unavailable_mode"
	optionLocalRPO                          = "local_rpo"
	optionLocalRetention                    = "local_retention"
	optionArray                             = "array"
	optionDestroySnapshotsOnDelete          = "destroy_snapshots_on_delete"
	optionServices                          = "services"
	optionEnabled                           = "enabled"
	optionNetworkInterfaceGroup             = "network_interface_group"
	optionMaxSpeed                          = "max_speed"
	optionInterfaceType                     = "interface_type"
	optionVlan                              = "vlan"
	optionMac                               = "mac"
	optionWwn                               = "wwn"
	optionRoleName                          = "role_name"
	optionPrincipal                         = "principal"
	optionScope                             = "scope"
	optionAssignableScope                   = "assignable_scope"
	optionAssignableScopes                  = "assignable_scopes"
	optionDescription                       = "description"
	optionEmail                             = "email"
	optionCreatorId                         = "creator_id"
	optionIssuer                            = "issuer"
	optionLastKeyUpdate                     = "last_key_update"
	optionLastUsed                          = "last_used"
	optionArrayType                         = "array_type"
	optionMediaType                         = "media_type"
	optionPublicKey                         = "public_key"
	optionHost                              = "api_host"
	optionIssuerId                          = "issuer_id"
	optionPrivateKeyFile                    = "private_key_file"
	optionPrivateKey                        = "private_key"
	optionTimeRemaining                     = "time_remaining"
	optionDestroyed                         = "destroyed"
	optionVolumeSerialNumber                = "volume_serial_number"
	optionConsistencyId                     = "consistency_id"
	optionAccessToken                       = "access_token"
	optionFusionConfig                      = "fusion_config"
	optionTokenEndpoint                     = "token_endpoint"
	optionStorageEndpointCollectionIdentity = "storage_endpoint_collection_identity"
	optionLoadBalancer                      = "load_balancer"
	optionLoadBalancerAddresses             = "load_balancer_addresses"
	optionFusionConfigProfile               = "fusion_config_profile"
	optionPrivateKeyPassword                = "private_key_password"
	optionHardwareTypes                     = "hardware_types"
)

const (
	maxDisplayName                 = 256
	storageClassIopsMin      int64 = 100
	storageClassIopsMax      int64 = 100 * 1000 * 1000
	storageClassBandwidthMin int64 = 1048576              // 2^20
	storageClassBandwidthMax int64 = 549755813888         // 2^39
	storageClassSizeMin      int64 = 1048576              // 1 MiB
	storageClassSizeMax      int64 = 4 * 1125899906842624 // 4 Pib
	volumeSizeMin            int64 = 1 << 20              // 1 MiB
	volumeSizeMax            int64 = 4 * (1 << 50)        // 4 Pib

	cbsAzureIscsiLoadBalancerAddressesAmount = 2

	endpointTypeIscsi         = "iscsi"
	endpointTypeCbsAzureIscsi = "cbs-azure-iscsi"
)

const (
	resourceKindApiClient             = "ApiClient"
	resourceKindArray                 = "Array"
	resourceKindAvailabilityZone      = "AvailabilityZone"
	resourceKindHardwareType          = "HardwareType"
	resourceKindHostAccessPolicy      = "HostAccessPolicy"
	resourceKindNetworkInterface      = "NetworkInterface"
	resourceKindNetworkInterfaceGroup = "NetworkInterfaceGroup"
	resourceKindPlacementGroup        = "PlacementGroup"
	resourceKindProtectionPolicy      = "ProtectionPolicy"
	resourceKindRegion                = "Region"
	resourceKindRoleAssignment        = "RoleAssignment"
	resourceKindRole                  = "Role"
	resourceKindSnapshot              = "Snapshot"
	resourceKindStorageClass          = "StorageClass"
	resourceKindStorageEndpoint       = "StorageEndpoint"
	resourceKindStorageService        = "StorageService"
	resourceKindTenant                = "Tenant"
	resourceKindTenantSpace           = "TenantSpace"
	resourceKindUser                  = "User"
	resourceKindVolume                = "Volume"
	resourceKindVolumeSnapshot        = "VolumeSnapshot"
)

const (
	resourceGroupNameRegion                = "regions"
	resourceGroupNameApiClient             = "api-clients"
	resourceGroupNameAvailabilityZone      = "availability-zones"
	resourceGroupNameArray                 = "arrays"
	resourceGroupNameHostAccessPolicy      = "host-access-policies"
	resourceGroupNameNetworkInterfaceGroup = "network-interface-groups"
	resourceGroupNameNetworkInterface      = "network-interfaces"
	resourceGroupNameTenant                = "tenants"
	resourceGroupNameTenantSpace           = "tenant-spaces"
	resourceGroupNamePlacementGroup        = "placement-groups"
	resourceGroupNameProtectionPolicy      = "protection-policies"
	resourceGroupNameRole                  = "roles"
	resourceGroupNameRoleAssignment        = "role-assignments"
	resourceGroupNameStorageService        = "storage-services"
	resourceGroupNameStorageClass          = "storage-classes"
	resourceGroupNameStorageEndpoint       = "storage-endpoints"
	resourceGroupNameVolume                = "volumes"
)

var hapPersonalities = []string{
	"windows", "linux", "esxi", "oracle-vm-server", "aix", "hitachi-vsp", "hpux", "solaris", "vms",
}

var interfaceTypes = []string{
	optionEth, optionFc,
}

const (
	hwTypeArrayX       = "flash-array-x"
	hwTypeArrayC       = "flash-array-c"
	hwTypeArrayXOptane = "flash-array-x-optane"
	hwTypeArrayXL      = "flash-array-xl"
)

var hwTypes = []string{
	hwTypeArrayX,
	hwTypeArrayC,
	hwTypeArrayXOptane,
	hwTypeArrayXL,
}
