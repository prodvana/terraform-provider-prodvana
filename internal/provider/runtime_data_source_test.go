package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRuntimeDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testAccRuntimeDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.prodvana_runtime.test", "name", "default"),
					resource.TestCheckResourceAttr("data.prodvana_runtime.test", "type", "K8S"),
				),
			},
		},
	})
}

var testAccRuntimeDataSourceConfig = `
data "prodvana_runtime" "test" {
  name = "default"
}
`
