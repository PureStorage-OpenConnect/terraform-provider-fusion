// A storage service that uses only FlashArray//X
resource "fusion_storage_service" "storage_service_x" {
  name           = "storage-service-x"
  display_name   = "Storage X"
  hardware_types = ["flash-array-x"]
}

// A high performance storage class using X
resource "fusion_storage_class" "storage_class_performance" {
  name            = "storage-class-performance"
  display_name    = "performance"
  storage_service = fusion_storage_service.storage_service_x.name
  size_limit      = "1T"
  iops_limit      = 10000
  bandwidth_limit = "1G"
}

// A generic storage service
resource "fusion_storage_service" "storage_service_generic" {
  name           = "storage-service-generic"
  display_name   = "Storage Generic"
  hardware_types = ["flash-array-x", "flash-array-c"]
}

// A standard DB storage class using the generic storage service
resource "fusion_storage_class" "storage_class_db_standard" {
  name            = "storage-class-db-standard"
  display_name    = "DB standard"
  storage_service = fusion_storage_service.storage_service_generic.name
  size_limit      = "100T"
  iops_limit      = 5000
  bandwidth_limit = "250M"
}
