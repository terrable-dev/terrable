provider "aws" {
  region              = "eu-west-2"
}

terraform {
  required_version = ">= 1.9.2"
}

module "simple_api" {
  source = "../../../terraform-terrable-api"
  api_name = "simple-api"
  
  handlers = {
    Echo: {
        source = "./src/Echo.ts"
        http = {
          GET = "/",
          POST = "/"
        }
    },
    Delayed: {
      source = "./src/Delayed.ts"
        http = {
          GET = "/delayed",
        }
    }
  }
}
