# SQS Queue for Job Processing
resource "aws_sqs_queue" "jobs_dlq" {
  name = "${var.name_prefix}-jobs-dlq"

  message_retention_seconds = 1209600 # 14 days

  tags = var.common_tags
}

resource "aws_sqs_queue" "jobs" {
  name = "${var.name_prefix}-jobs"

  visibility_timeout_seconds = 900  # 15 minutes (match Lambda timeout)
  message_retention_seconds  = 86400 # 1 day
  receive_wait_time_seconds  = 20    # Long polling

  redrive_policy = jsonencode({
    deadLetterTargetArn = aws_sqs_queue.jobs_dlq.arn
    maxReceiveCount     = 3
  })

  tags = var.common_tags
}
