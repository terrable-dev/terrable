module "invalid_handler" {
  handlers = {
    MissingHandler = {
      source = "./src/MissingHandler.ts"
      http = {
        GET = "/"
      }
    }
  }
}
