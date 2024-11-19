provider "aws" {
  region              = "eu-west-2"
}

terraform {
  required_version = ">= 1.9.2"
}

resource "aws_ssm_parameter" "my-param" {
  name = "example-ssm"
  type = "String"
  value = "#"

  lifecycle {
    ignore_changes = [ value ]
  }
}

resource "aws_ssm_parameter" "local-param" {
  name = "local-ssm"
  type = "String"
  value = "local-ssmval"
}

module "simple_api" {
  source = "terrable-dev/terrable-api/aws"
  api_name = "simple-api"
  
  global_environment_variables = {
    TEST_ENV: {
      value: "my-flat-var"
    }
  }
  
  handlers = {
    EchoHandler: {
        source = "./src/Echo.ts"
        http = {
          GET = "/",
          POST = "/"
        }
    }
    TestHandler: {
        environment_variables = {
          LOCAL_ENV = {
            value: "local-env"
          }
        }
        source = "./src/Echo.ts"
        http = {
          GET = "/get",
        }
    }
  }

  depends_on = [ aws_ssm_parameter.my-param, aws_ssm_parameter.local-param ]
}
