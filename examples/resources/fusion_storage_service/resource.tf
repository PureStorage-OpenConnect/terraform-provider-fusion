resource "fusion_storage_service" "storage_service_generic" {
  name           = "storage-service-generic"
  display_name   = "Storage Generic"
  hardware_types = ["flash-array-x", "flash-array-c"]
}
