variable "instance_type" {
  default = "t2.micro"
}

variable "root_block_device_volume_size" {
  default = 50
}

variable "block_device_volume_size" {
  default = 100
}

variable "block_device_iops" {
  default = 400
}

variable "hello_world_function_memory_size" {
  default = 512
}

provider "aws" {
  region                      = "us-east-1"
  skip_credentials_validation = true
  skip_requesting_account_id  = true
}

resource "aws_instance" "web_app" {
  ami           = "ami-674cbc1e"
  instance_type = var.instance_type

  root_block_device {
    volume_size = var.root_block_device_volume_size
  }

  ebs_block_device {
    device_name = "my_data"
    volume_type = "io1"
    volume_size = var.block_device_volume_size
    iops        = var.block_device_iops
  }
}

resource "aws_lambda_function" "hello_world" {
  function_name = "hello_world"
  role          = "arn:aws:lambda:us-east-1:account-id:resource-id"
  handler       = "exports.test"
  runtime       = "nodejs12.x"
  memory_size   = var.hello_world_function_memory_size
}
