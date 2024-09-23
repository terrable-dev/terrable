provider "aws" {
  region              = "eu-west-2"
}

terraform {
  required_version = ">= 1.9.2"
}

module "simple_api" {
  source = "terrable-dev/terrable-api/aws"
  version = "0.0.3"
  api_name = "simple-api"
  
  handlers = {
    Echo: {
        source = "./src/Echo.ts"
        http = {
          method = "GET"
          path = "/"
        }
    },
    Echo: {
        source = "./src/Echo.ts"
        http = {
          method = "GET"
          path = "/"
        }
    },
    Async: {
      source = "./src/Async.ts"
      http = {
        method = "GET",
        path = "/async"
      }
    }
  }
}
