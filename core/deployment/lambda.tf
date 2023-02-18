locals {
  lambda = "core${local.suffix}"
  lambda_secret = {
    production = "arn:aws:secretsmanager:us-east-2:027889758114:secret:neon/main/core/lambda-C335bP"
    staging    = "arn:aws:secretsmanager:us-east-2:027889758114:secret:neon/main/core/lambda-C335bP"
  }
  neon_db = {
    production = {
      endpoint = "ep-wild-wind-389577.us-east-2.aws.neon.tech"
    }
  }
}

resource "aws_iam_role" "lambda_core" {
  name = "Lambda${local.suffix}"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = "sts:AssumeRole"
        Principal = {
          Service = "lambda.amazonaws.com"
        }
      },
    ]
  })
}

data "aws_iam_policy_document" "lambda_core" {
  statement {
    effect = "Allow"
    actions = [
      "logs:CreateLogGroup",
      "logs:CreateLogStream",
      "logs:PutLogEvents",
    ]
    resources = ["arn:aws:logs:*:*:*"]
  }

  statement {
    effect    = "Allow"
    actions   = ["secretsmanager:ListSecrets"]
    resources = ["*"]
  }

  statement {
    effect = "Allow"
    actions = [
      "secretsmanager:GetResourcePolicy",
      "secretsmanager:GetSecretValue",
      "secretsmanager:DescribeSecret",
      "secretsmanager:ListSecretVersionIds",
    ]
    resources = [
      local.lambda_secret[var.environment]
    ]
  }
}

resource "aws_iam_policy" "lambda_core" {
  name   = "LambdaCore${local.suffix}"
  policy = data.aws_iam_policy_document.lambda_core.json
}

resource "aws_iam_role_policy_attachment" "lambda_core" {
  policy_arn = aws_iam_policy.lambda_core.arn
  role       = aws_iam_role.lambda_core.name
}

resource "aws_cloudwatch_log_group" "lambda_core" {
  name              = "/aws/lambda/${local.lambda}"
  retention_in_days = 7
}

resource "null_resource" "lambda_core" {
  triggers = {
    md5 = base64sha256(join(",", [
      for file in concat(
        [for f in fileset("${path.module}/../", "{*.go,go.mod,go.sum}") : "${path.module}/../${f}"],
        [for f in fileset("${path.module}/../compression", "*.go") : "${path.module}/../compression/${f}"],
        [for f in fileset("${path.module}/../handler", "*.go") : "${path.module}/../handler/${f}"],
        [for f in fileset("${path.module}/../storage", "{*.go,go.mod,go.sum}") : "${path.module}/../storage/${f}"],
        [for f in fileset("${path.module}/../cmd/lambda", "{*.go,go.mod,go.sum}") : "${path.module}/../cmd/lambda/${f}"],
      ) : filemd5(file)
    ]))
  }

  provisioner "local-exec" {
    command = "cd ${path.module}/.. && make build"
  }
}

data "local_file" "lambda_core" {
  filename   = "${path.module}/../bin/lambda.zip"
  depends_on = [null_resource.lambda_core]
}

resource "aws_lambda_function" "core" {
  function_name = local.lambda
  role          = aws_iam_role.lambda_core.arn

  filename         = data.local_file.lambda_core.filename
  source_code_hash = null_resource.lambda_core.triggers.md5
  runtime          = "go1.x"
  handler          = "lambda"
  memory_size      = 256
  timeout          = 120

  environment {
    variables = {
      OPENAI_API_KEY     = var.openai_api_key
      OPENAI_MAX_TOKENS  = var.openai_max_tokens
      OPENAI_TEMPERATURE = var.openai_temperature
      CORS_HEADERS       = jsonencode(local.cors_headers)
      NEON_DBNAME        = "core"
      NEON_USER          = "lambda"
      NEON_HOST          = local.neon_db[var.environment]["endpoint"]
      NEON_PASSWORD      = var.neon_password
    }
  }

  depends_on = [null_resource.lambda_core]
}
