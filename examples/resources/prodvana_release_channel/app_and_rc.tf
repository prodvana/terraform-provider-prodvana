resource "prodvana_application" "app" {
  name = "my-app"
}

resource "prodvana_release_channel" "staging" {
  name        = "staging"
  application = prodvana_application.app.name
}

resource "prodvana_release_channel" "prod" {
  name        = "prod"
  application = prodvana_application.app.name
}
