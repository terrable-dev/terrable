provider "aws" {
  region              = "eu-west-2"
}

terraform {
  required_version = ">= 1.9.2"
}

module "simple_api" {
  source = "../../../terraform-aws-terrable-api"
  api_name = "simple-api"

  global_environment_variables = {
    GLOBAL_ENV = "global-env-var"
  }

  runtime = "nodejs20.x"
  
  rest_api = {
    endpoint_type = "REGIONAL"
  }

  handlers = {
    EchoHandler: {
        source = "./src/Echo.ts"
        environment_variables = {
          ECHO_ENV = "echo-env"
        }
        http = {
          GET = "/",
          POST = "/"
        }
    },

    # Echo Handler configured with a callback-style instead of async / await
      EchoCallback: {
        source = "./src/EchoCallback.ts"
        http = {
          GET = "/echo-callback"
        }
    },

    # Echo Handler with no local environment variables
    EchoHandlerNoLocalEnv: {
        source = "./src/Echo.ts"
        http = {
          GET = "/echo-no-env",
        }
    },
    DelayedHandler: {
      source = "./src/Delayed.ts"
        http = {
          GET = "/delayed",
        }
    },

    # These two handlers deliberately share a source file with the same name to verify
    # they do not collide when transpiled into a "Collision.js" file

    CollisionOne: {
      source = "./src/Collision1/Collision.ts"
        http = {
          GET = "/collision1",
        }
    },
    CollisionTwo: {
      source = "./src/Collision2/Collision.ts"
        http = {
          GET = "/collision2",
        }
    }
  }
}
