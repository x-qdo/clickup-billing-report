locals {
  lambda_env = {
    DYNAMODB_CLIENTS_TABLE    = var.clients_table_name
    DYNAMODB_DEVELOPERS_TABLE = var.developers_table_name
    DYNAMODB_SETTINGS_TABLE   = var.settings_table_name
    DYNAMODB_SESSIONS_TABLE   = var.sessions_table_name
    DYNAMODB_JOBS_TABLE       = var.jobs_table_name
    SQS_JOBS_QUEUE_URL        = aws_sqs_queue.jobs.url
    S3_JOBS_BUCKET            = aws_s3_bucket.jobs.id
  }
}

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
        aws_dynamodb_table.jobs.arn,
      ]
    },
    sqs = {
      effect = "Allow",
      actions = [
        "sqs:SendMessage",
      ],
      resources = [
        aws_sqs_queue.jobs.arn,
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

# Worker Lambda Function (SQS triggered)
module "lambda_worker" {
  source  = "terraform-aws-modules/lambda/aws"
  version = "~> 6.0"

  function_name = "${var.name_prefix}-worker"
  description   = "Job worker for async report generation"

  create_package = false
  package_type   = "Image"
  image_uri      = "${aws_ecr_repository.clickup-report.repository_url}:${var.image_tag}-worker"

  memory_size = 512
  timeout     = 900 # 15 minutes

  environment_variables = local.lambda_env

  vpc_subnet_ids                = []
  vpc_security_group_ids        = []
  attach_network_policy         = length([]) > 0 ? true : false
  attach_cloudwatch_logs_policy = true

  cloudwatch_logs_retention_in_days = 5
  architectures                     = ["arm64"]

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
        aws_dynamodb_table.jobs.arn,
      ]
    },
    sqs = {
      effect = "Allow",
      actions = [
        "sqs:ReceiveMessage",
        "sqs:DeleteMessage",
        "sqs:GetQueueAttributes",
      ],
      resources = [
        aws_sqs_queue.jobs.arn,
      ]
    },
    s3 = {
      effect = "Allow",
      actions = [
        "s3:PutObject",
        "s3:GetObject",
      ],
      resources = [
        "${aws_s3_bucket.jobs.arn}/*",
      ]
    },
  }

  publish = true

  event_source_mapping = {
    sqs = {
      event_source_arn = aws_sqs_queue.jobs.arn
      batch_size       = 1
    }
  }
}
