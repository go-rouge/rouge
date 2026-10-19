# a comment
terraform {
  required_version = ">= 1.5.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.0"
    }
  }
}

provider "aws" {
  region = var.region
}

variable "region" {
  type    = string
  default = "eu-west-3"
}

variable "count_of" {
  type    = number
  default = 3
}

locals {
  name    = "app-${var.region}"
  escaped = "a literal $${not_interpolated} brace"
  tags    = { Owner = "team", Env = "prod" }
  slug    = "${replace(local.name, "/[^a-z0-9]+/", "-")}"
  from    = "${data.aws_ami.latest.id}"
  always  = "${true}"
  mixed   = "${var.count_of > 1 && false ? "many" : "one"}"
}

resource "aws_instance" "web" {
  count         = var.count_of
  ami           = data.aws_ami.latest.id
  instance_type = "t3.micro"

  tags = merge(local.tags, {
    Name = "${local.name}-${count.index + 1}"
  })

  user_data = <<-EOT
    #!/bin/sh
    echo "${local.name}" > /tmp/name
  EOT
}

data "aws_ami" "latest" {
  most_recent = true
  owners      = ["amazon"]

  filter {
    name   = "name"
    values = ["al2023-ami-*"]
  }
}

output "public_ip" {
  value     = aws_instance.web[0].public_ip
  sensitive = false
}
