data "local_file" "john_doe_pubkey" {
  filename = "/path/to/john_doe_rsa.pub"
}

resource "fusion_api_client" "john_doe" {
  display_name = "John Doe"
  public_key   = data.local_file.john_doe_pubkey.content
}
