# API Gateway
module "api_gateway" {
  source  = "terraform-aws-modules/apigateway-v2/aws"
  version = "~> 5.0"

  name          = "${var.name_prefix}-api"
  description   = "Clickup API Gateway"
  protocol_type = "HTTP"

  # Only create domain name if provided
  create_domain_name = var.domain_name != "" ? true : false
  domain_name        = var.domain_name != "" ? var.domain_name : null
  hosted_zone_name   = var.hosted_zone_name != "" ? var.hosted_zone_name : null

  stage_default_route_settings = {
    detailed_metrics_enabled = false
    throttling_burst_limit   = 2
    throttling_rate_limit    = 1
  }

  # Configure CORS if needed
  cors_configuration = var.enable_cors ? {
    allow_headers = [
      "content-type", "x-amz-date", "authorization", "x-api-key", "x-amz-security-token", "x-amz-user-agent"
    ]
    allow_methods = ["*"]
    allow_origins = ["*"]
  } : null

  # Routes and integrations
  routes = {
    "$default" = {
      integration = {
        uri                    = module.lambda_http.lambda_function_arn
        payload_format_version = "1.0"
        timeout_milliseconds   = 10000
      }
    }

    "GET /health" = {
      detailed_metrics_enabled = false
      throttling_rate_limit    = 2
      throttling_burst_limit   = 1

      integration = {
        uri                    = module.lambda_http.lambda_function_arn
        payload_format_version = "1.0"
        timeout_milliseconds   = 5000
      }
    },
  }
}
