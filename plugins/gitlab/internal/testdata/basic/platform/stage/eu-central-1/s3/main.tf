# S3 module - no dependencies (root level)

terraform {
  backend "s3" {
    bucket = "terraform-state"
    key    = "platform/stage/eu-central-1/s3/terraform.tfstate"
    region = "eu-central-1"
  }
}

resource "aws_s3_bucket" "data" {
  bucket = "stage-data-bucket"

  tags = {
    Name        = "stage-data"
    Environment = "stage"
  }
}

output "bucket_name" {
  value = aws_s3_bucket.data.bucket
}

output "bucket_arn" {
  value = aws_s3_bucket.data.arn
}
