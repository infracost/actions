provider "aws" {
  region                      = "us-east-1"
  skip_credentials_validation = true
  skip_requesting_account_id  = true
}

module "base" {
  source                           = "../modules/example"

  instance_type                    = "m5.4xlarge"
  root_block_device_volume_size    = 100
  block_device_volume_size         = 1000
  block_device_iops                = 800
  
  hello_world_function_memory_size = 1024
}
