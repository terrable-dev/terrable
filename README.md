<h1 align="center">
    Terrable
</h1>

<p align="center">
    <em>"What if there was something that helped me write and deploy terraformed API Gateways and run them locally?"</em>
</p>
<p align="center">
    <strong>"That sounds like a <em>terrable</em> idea"</strong>
</p>

---

Terrable is a CLI tool that works seamlessly with its companion Terraform module to simplify building, testing
and deploying AWS API Gateways.

## Features

- Easy configuration of API Gateways using Terraform
- Local development and testing of API endpoints
- Seamless deployment to AWS
- TypeScript support for handler functions

## Installation

Install Terrable using Go:

```bash
go install github.com/terrable-dev/terrable@latest
```

## Quick Start

1. Use the Terrable Terraform module in your configuration:

```terraform
module "example_api" {
  source    = "terrable-dev/terrable-api/aws"
  api_name  = "example-api"
  runtime   = "nodejs20.x"

  handlers = {
    ExampleHandler: {
        source = "./ExampleHandler.ts"
        http = {
          GET = "/"
        }
    }
  }
}
```

2. Run your API locally using the Terrable CLI:

```bash
terrable -file terraform_file.tf -module example_api
```
