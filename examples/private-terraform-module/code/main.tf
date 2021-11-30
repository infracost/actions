provider "aws" {
  region                      = "us-east-1"
  skip_credentials_validation = true
  skip_requesting_account_id  = true
  access_key                  = "mock_access_key"
  secret_key                  = "mock_secret_key"
}

# This is just a private clone of https://github.com/terraform-aws-modules/terraform-aws-ec2-instance
module "ec2_cluster" {
  source                 = "git@github.com:infracost/terraform-private-module-example.git"

  name                   = "my-instance"
  ami                    = "ami-ebd02392"
  instance_type          = "t2.micro"
  key_name               = "user1"
  monitoring             = true
  vpc_security_group_ids = ["sg-12345678"]
  subnet_id              = "subnet-eddcdzz4"
}
