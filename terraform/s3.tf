# S3 Bucket for Job Results (Excel files)
resource "aws_s3_bucket" "jobs" {
  bucket = var.jobs_bucket_name

  tags = var.common_tags
}

resource "aws_s3_bucket_lifecycle_configuration" "jobs" {
  bucket = aws_s3_bucket.jobs.id

  rule {
    id     = "delete-old-files"
    status = "Enabled"

    expiration {
      days = 7
    }
  }
}

resource "aws_s3_bucket_public_access_block" "jobs" {
  bucket = aws_s3_bucket.jobs.id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_server_side_encryption_configuration" "jobs" {
  bucket = aws_s3_bucket.jobs.id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

resource "aws_s3_bucket_cors_configuration" "jobs" {
  bucket = aws_s3_bucket.jobs.id

  cors_rule {
    allowed_headers = ["*"]
    allowed_methods = ["GET", "HEAD"]
    allowed_origins = ["*"]
    expose_headers  = ["Content-Disposition", "Content-Type", "Content-Length"]
    max_age_seconds = 3600
  }
}
