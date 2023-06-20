package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccReleaseChannelDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testAccReleaseChannelDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.prodvana_release_channel.test", "name", "staging"),
					resource.TestCheckResourceAttr("data.prodvana_release_channel.test", "application", dataSourceAppName),
					resource.TestCheckResourceAttrSet("data.prodvana_release_channel.test", "version"),
					resource.TestCheckResourceAttr("data.prodvana_release_channel.test", "runtimes.0.runtime", "default"),
				),
			},
		},
	})
}

var testAccReleaseChannelDataSourceConfig = fmt.Sprintf(`
data "prodvana_release_channel" "test" {
  name = "staging"
  application = %[1]q
}
`, dataSourceAppName)
