provider "aws" {
  region = "eu-west-1"
  default_tags {
    tags = {
      Environment = "prod"
      Managed-By  = "Terraform"
      Project     = "ClickUpBillingReport"
    }
  }
}

terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "5.96.0"
    }
  }

  backend "s3" {
    bucket  = "qdo-terraform-state-prod"
    key     = "prod/clickup.tfstate"
    region  = "eu-west-1"
    encrypt = true
  }
}

