resource "prodvana_release_channel" "example" {
  name        = "my-rc"
  application = "my-app"
  runtimes = [
    {
      runtime = "default"
    }
  ]
  policy = {
    default_env = {
      "MY_ENV_VAR" = {
        value = "my value"
      }
    }
  }
}
