provider "aws" {
  region                      = "us-east-1"
  skip_credentials_validation = true
  skip_requesting_account_id  = true
}

module "base" {
  source                           = "../modules/example"

  instance_type                    = "t2.medium"
  root_block_device_volume_size    = 50
  block_device_volume_size         = 100
  block_device_iops                = 400
  
  hello_world_function_memory_size = 512
}
