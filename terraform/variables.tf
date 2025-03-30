variable "clients_table_name" {
  description = "Name for the DynamoDB table storing client information."
  type        = string
  default     = "ClickUpReporter-Clients"
}

variable "developers_table_name" {
  description = "Name for the DynamoDB table storing developer information."
  type        = string
  default     = "ClickUpReporter-Developers"
}

variable "settings_table_name" {
  description = "Name for the DynamoDB table storing global settings."
  type        = string
  default     = "ClickUpReporter-Settings"
}

variable "sessions_table_name" {
  description = "Name for the DynamoDB table storing user session state."
  type        = string
  default     = "ClickUpReporter-Sessions"
}

variable "environment" {
  type    = string
  default = "prod"
}

variable "common_tags" {
  description = "Common tags to apply to all resources."
  type        = map(string)
  default     = {}
}
