module "http_api_cors" {
  http_api = {
    cors_configuration = {
      allow_origins     = ["https://app.example.com"]
      allow_methods     = ["GET", "POST", "PUT", "OPTIONS"]
      allow_headers     = ["content-type", "authorization"]
      expose_headers    = ["x-terrable-request-id"]
      allow_credentials = true
      max_age           = 600
    }
  }

  handlers = {
    EchoHandler = {
      source = "./src/Echo.ts"
      http = {
        GET  = "/"
        POST = "/"
        PUT  = "/"
      }
    }
  }
}
