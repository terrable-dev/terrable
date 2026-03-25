module "offline_core" {
  environment_variables = {
    GLOBAL_ENV = "global-env-var"
  }

  timeout = 3

  handlers = {
    EchoHandler = {
      source = "./src/Echo.ts"
      http = {
        GET  = "/"
        POST = "/"
        PUT  = "/"
      }
    }

    EchoCallback = {
      source = "./src/EchoCallback.ts"
      http = {
        GET = "/echo-callback"
      }
    }

    EchoEnvTest = {
      source = "./src/Echo.ts"
      http = {
        GET = "/echo-env-test"
      }
    }

    DelayedHandler = {
      source = "./src/Delayed.ts"
      http = {
        GET = "/delayed"
      }
    }

    TimeoutDelay = {
      timeout = 1
      source  = "./src/TimeoutDelay.ts"
      http = {
        GET = "/timeout"
      }
    }

    GlobalTimeoutDelay = {
      source = "./src/GlobalTimeoutDelay.ts"
      http = {
        GET = "/timeout-global"
      }
    }

    SqsHandler = {
      source = "./src/Sqs.ts"
      sqs = {
        queue = "arn:aws:sqs:eu-west-1:000000000000:test-queue"
      }
    }

    CollisionOne = {
      source = "./src/Collision1/Collision.ts"
      http = {
        GET = "/collision1"
      }
    }

    CollisionTwo = {
      source = "./src/Collision2/Collision.ts"
      http = {
        GET = "/collision2"
      }
    }
  }
}
