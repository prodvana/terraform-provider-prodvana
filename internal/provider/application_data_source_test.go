package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

var (
	dataSourceAppName = "terraform-provider-testing-data-sources"
)

func TestAccApplicationDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testAccApplicationDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.prodvana_application.test", "name", dataSourceAppName),
					resource.TestCheckResourceAttrSet("data.prodvana_application.test", "id"),
					resource.TestCheckResourceAttrSet("data.prodvana_application.test", "version"),
				),
			},
		},
	})
}

var testAccApplicationDataSourceConfig = fmt.Sprintf(`
data "prodvana_application" "test" {
  name = %[1]q
}
`, dataSourceAppName)
