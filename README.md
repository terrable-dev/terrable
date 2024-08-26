<h1 style="text-align: center">
    Terrable
</h1>

<p align="center">
    <em>"What if there was something that helped me write and deploy terraformed API Gateways and run them locally?"</em>
</p>
<p align="center">
    <strong>"That sounds like a <em>terrable</em> idea"</strong>
</p>

---

## Installation

Using Go

```
go install github.com/terrable-dev/terrable@latest
```

## Usage

The terrable CLI works with a companion Terraform module, found on the Terraform Registry at https://registry.terraform.io/modules/terrable-dev/terrable-api/aws/latest.

You can configure your API in Terraform as follows:

```terraform
module "example_api" {
  source = "terrable-dev/terrable-api/aws"
  version = "0.0.1"
  api_name = "example-api"
  
  handlers = {
    ExampleHandler: {
        source = "./ExampleHandler.ts"
        http = {
          method = "GET"
          path = "/"
        }
    },
  }
}
```

And then, using the terrable CLI, run this configuration locally:

```
terrable -f terraform_file.tf -m example_api
```
