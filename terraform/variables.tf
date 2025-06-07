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

variable "domain_name" {
  description = "ClickUp API key for authentication."
  type        = string
  default     = "clickup.k9s.qdo.ee"
}

variable "hosted_zone_name" {
  type    = string
  default = "k9s.qdo.ee"
}
variable "name_prefix" {
  type    = string
  default = "clickup-report"
}
variable "image_tag" {
  type    = string
  default = "20250607-9"
}

variable "enable_cors" {
  type    = bool
  default = true
}
