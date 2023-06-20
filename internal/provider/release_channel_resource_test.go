package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccReleaseChannelResource(t *testing.T) {
	appName := testAppName("rc-tests")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccReleaseChannelResourceConfig("staging", appName, nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("prodvana_release_channel.staging", "name", "staging"),
					resource.TestCheckResourceAttrSet("prodvana_release_channel.staging", "version"),
					resource.TestCheckResourceAttrSet("prodvana_release_channel.staging", "id"),
					resource.TestCheckResourceAttr("prodvana_release_channel.staging", "runtimes.0.runtime", "default"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "prodvana_release_channel.staging",
				ImportStateId:     appName + "/staging",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: testAccReleaseChannelResourceConfig("staging", appName, map[string]string{
					"TEST_VAR_ONE": "test value one",
					"TEST_VAR_TWO": "test value two",
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("prodvana_release_channel.staging", "name", "staging"),
					resource.TestCheckResourceAttrSet("prodvana_release_channel.staging", "version"),
					resource.TestCheckResourceAttrSet("prodvana_release_channel.staging", "id"),
					resource.TestCheckResourceAttr("prodvana_release_channel.staging", "runtimes.0.runtime", "default"),
					resource.TestCheckResourceAttr("prodvana_release_channel.staging", "policy.default_env.TEST_VAR_ONE.value", "test value one"),
					resource.TestCheckResourceAttr("prodvana_release_channel.staging", "policy.default_env.TEST_VAR_TWO.value", "test value two"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccReleaseChannelResourceConfig(name, app string, env map[string]string) string {
	policy := ""
	if env != nil {
		policy = "  policy = {\n    default_env = {\n"
		for key, value := range env {
			policy += fmt.Sprintf("    %[1]q = { value = %[2]q }\n", key, value)
		}
		policy += "    }\n  }"
	}

	return fmt.Sprintf(`
%[1]s

resource "prodvana_release_channel" "%[2]s" {
  name = %[2]q
  application = prodvana_application.%[3]s.name
  runtimes = [
	{
		runtime = "default"
	},
  ]
  %[4]s
}
`, testAccApplicationResourceConfig(app), name, app, policy)
}
