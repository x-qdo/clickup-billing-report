# Create ECR repository
resource "aws_ecr_repository" "clickup-report" {
  name                 = "clickup-report"
  image_tag_mutability = "MUTABLE"

  image_scanning_configuration {
    scan_on_push = false
  }
}
