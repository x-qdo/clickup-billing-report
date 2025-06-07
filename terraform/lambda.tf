# HTTP Lambda Function (API Gateway target)
module "lambda_http" {
  source  = "terraform-aws-modules/lambda/aws"
  version = "~> 6.0"

  function_name = "${var.name_prefix}-http"
  description   = "Bot HTTP handler for API Gateway"

  create_package = false
  package_type   = "Image"
  image_uri      = "${aws_ecr_repository.clickup-report.repository_url}:${var.image_tag}-http"

  memory_size = 512
  timeout     = 60

  environment_variables = local.lambda_env

  vpc_subnet_ids                = []
  vpc_security_group_ids        = []
  attach_network_policy         = length([]) > 0 ? true : false
  attach_cloudwatch_logs_policy = true

  cloudwatch_logs_retention_in_days = 5
  architectures                     = ["arm64"]

  # Add permissions to access DynamoDB and SQS
  attach_policy_statements = true
  policy_statements = {
    dynamodb = {
      effect = "Allow",
      actions = [
        "dynamodb:GetItem",
        "dynamodb:PutItem",
        "dynamodb:UpdateItem",
        "dynamodb:DeleteItem",
        "dynamodb:Query",
        "dynamodb:Scan",
        "dynamodb:BatchWriteItem",
        "dynamodb:BatchGetItem"
      ],
      resources = [
        aws_dynamodb_table.clients.arn,
        aws_dynamodb_table.developers.arn,
        aws_dynamodb_table.settings.arn,
        aws_dynamodb_table.sessions.arn,
      ]
    },
  }

  publish = true

  allowed_triggers = {
    APIGateway = {
      service    = "apigateway"
      source_arn = "${module.api_gateway.api_execution_arn}/*/*"
    }
  }
}
