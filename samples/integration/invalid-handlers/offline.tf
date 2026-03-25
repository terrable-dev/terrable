module "invalid_handlers" {
  handlers = {
    MissingHandler = {
      source = "./src/MissingHandler.ts"
      http = {
        GET = "/missing"
      }
    }

    BrokenHandler = {
      source = "./src/BrokenHandler.ts"
      http = {
        GET = "/broken"
      }
    }
  }
}
