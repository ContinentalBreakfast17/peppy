terraform {
  required_version = "1.3.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "= 4.24.0"
    }
    archive = {
      source  = "hashicorp/archive"
      version = "= 2.2.0"
    }
  }

  backend "s3" {}
}

provider "aws" {
  region = "us-east-1"
}

provider "aws" {
  region = "us-east-2"
  alias  = "us_east_2"
}

# if terraform ever supports dynamic providers this can be used, until then :(
# locals {
#   regional_providers = {
#     us-east-2 = aws.us_east_2,
#   }

#   replica_regions = {
#     for region, provider in local.regional_providers :
#     region => provider
#     if contains(var.replica_regions, region)
#   }

#   regions = merge(replica_regions, { us-east-1 = aws })
# }