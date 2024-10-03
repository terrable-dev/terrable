provider "aws" {
  region              = "eu-west-2"
}

terraform {
  required_version = ">= 1.9.2"
}

module "simple_api" {
  source = "terrable-dev/terrable-api/aws"
  version = "0.0.4"
  api_name = "simple-api"
  
  handlers = {
    EchoHandler: {
        source = "./src/Echo.ts"
        http = {
          GET = "/",
          POST = "/"
        }
    },
    DelayedHandler: {
      source = "./src/Delayed.ts"
        http = {
          GET = "/delayed",
        }
    }
  }
}
