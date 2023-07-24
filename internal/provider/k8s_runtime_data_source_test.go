package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccK8sRuntimeDataSource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Read testing
			{
				Config: testAccRuntimeDataSourceConfig,
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.prodvana_k8s_runtime.test", "name", "default"),
					resource.TestCheckResourceAttrSet("data.prodvana_k8s_runtime.test", "agent_api_token"),
				),
			},
		},
	})
}

var testAccRuntimeDataSourceConfig = `
data "prodvana_k8s_runtime" "test" {
  name = "default"
}
`
