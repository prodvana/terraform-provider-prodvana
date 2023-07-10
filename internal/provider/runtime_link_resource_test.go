package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRuntimeLinkResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccK8sRuntimeLinkResourceConfig("default", "20s"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("prodvana_runtime_link.test", "id"),
					resource.TestCheckResourceAttr("prodvana_runtime_link.test", "timeout", "20s"),
				),
			},
			// Update and Read test
			{
				Config: testAccK8sRuntimeLinkResourceConfig("default", "30s"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("prodvana_runtime_link.test", "id"),
					resource.TestCheckResourceAttr("prodvana_runtime_link.test", "timeout", "30s"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccK8sRuntimeLinkResourceConfig(name string, timeout string) string {
	return fmt.Sprintf(`
resource "prodvana_runtime_link" "test" {
  name = "default"
  timeout = %[2]q
}
`, name, timeout)
}
