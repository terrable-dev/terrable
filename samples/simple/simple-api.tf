provider "aws" {
  region              = "eu-west-2"
}

terraform {
  required_version = ">= 1.9.0"
}

resource "aws_sqs_queue" "test_queue" {
  name = "test-queue"
}

module "simple_api" {
  source = "terrable-dev/terrable-api/aws"
  api_name = "simple-api"

  environment_variables = {
    GLOBAL_ENV = "global-env-var"
  }

  timeout = 3
  runtime = "nodejs20.x"
  
  rest_api = {
    endpoint_type = "REGIONAL"
  }

  handlers = {
    EchoHandler: {
        source = "./src/Echo.ts"
        http = {
          GET = "/",
          POST = "/",
          PUT = "/",
        }
    },

    # Echo Handler configured with a callback-style instead of async / await
    EchoCallback: {
        source = "./src/EchoCallback.ts"
        http = {
          GET = "/echo-callback"
        }
    },

    # Echo Handler with some variables that should be overwritten by the .env file
    EchoEnvTest: {
        source = "./src/Echo.ts"
        http = {
          GET = "/echo-env-test"
        }
    },
    
    DelayedHandler: {
      source = "./src/Delayed.ts"
        http = {
          GET = "/delayed"
        }
    },

    TimeoutDelay: {
      timeout = 1
      source = "./src/TimeoutDelay.ts"
        http = {
          GET = "/timeout"
        }
    },

    SqsHandler: {
      source = "./src/Sqs.ts"
        sqs = {
          queue = aws_sqs_queue.test_queue.arn
        }
    },

    # These two handlers deliberately share a source file with the same name to verify
    # they do not collide when transpiled into a "Collision.js" file

    CollisionOne: {
      source = "./src/Collision1/Collision.ts"
        http = {
          GET = "/collision1"
        }
    },
    CollisionTwo: {
      source = "./src/Collision2/Collision.ts"
        http = {
          GET = "/collision2"
        }
    }
  }
}
