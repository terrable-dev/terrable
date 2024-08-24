provider "aws" {
  region              = "eu-west-1"
}

terraform {
  required_version = ">= 1.9.2"
}

module "simple_api" {
  source = "terrable-dev/terrable-api/aws"
  version = "0.0.1"
  api_name = "simple-api"
  
  handlers = {
    HelloWorld: {
        source = "./src/HelloWorld.ts"
        http = {
          method = "GET"
          path = "/"
        }
    },

    HelloPost: {
        source = "./src/HelloPost.ts"
        http = {
          method = "POST"
          path = "/hello-post"
        }
    }
  }
}
