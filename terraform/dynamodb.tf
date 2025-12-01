resource "aws_dynamodb_table" "clients" {
  name         = var.clients_table_name
  billing_mode = "PAY_PER_REQUEST"

  hash_key = "Name"

  attribute {
    name = "Name"
    type = "S"
  }

  tags = var.common_tags
}

resource "aws_dynamodb_table" "developers" {
  name         = var.developers_table_name
  billing_mode = "PAY_PER_REQUEST"

  hash_key = "Name"

  attribute {
    name = "Name"
    type = "S"
  }

  tags = var.common_tags
}

resource "aws_dynamodb_table" "settings" {
  name         = var.settings_table_name
  billing_mode = "PAY_PER_REQUEST"

  hash_key = "SettingKey"

  attribute {
    name = "SettingKey"
    type = "S"
  }

  tags = var.common_tags
}

resource "aws_dynamodb_table" "sessions" {
  name         = var.sessions_table_name
  billing_mode = "PAY_PER_REQUEST"

  hash_key = "SessionID"

  attribute {
    name = "SessionID"
    type = "S"
  }

  ttl {
    enabled        = true
    attribute_name = "TTL"
  }

  tags = var.common_tags
}

resource "aws_dynamodb_table" "jobs" {
  name         = var.jobs_table_name
  billing_mode = "PAY_PER_REQUEST"

  hash_key = "JobID"

  attribute {
    name = "JobID"
    type = "S"
  }

  ttl {
    enabled        = true
    attribute_name = "TTL"
  }

  tags = var.common_tags
}
