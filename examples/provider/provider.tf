# org_slug and api_token can also be passed as environment variables:
# export PVN_ORG_SLUG=my-org
# export PVN_API_TOKEN=<api-token>

variable "api_token" {
  type = string
}

provider "prodvana" {
  org_slug  = "my-org"
  api_token = var.api_token
}
