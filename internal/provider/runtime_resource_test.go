package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccRuntimeResource(t *testing.T) {
	runtimeName := uniqueTestName("runtime-tests")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccK8sRuntimeResourceConfig(runtimeName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("prodvana_runtime.test", "name", runtimeName),
					resource.TestCheckResourceAttrSet("prodvana_runtime.test", "id"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "prodvana_runtime.test",
				ImportStateId:     runtimeName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: testAccK8sRuntimeResourceConfig(runtimeName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("prodvana_runtime.test", "name", runtimeName),
					resource.TestCheckResourceAttrSet("prodvana_runtime.test", "id"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccK8sRuntimeResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "prodvana_runtime" "test" {
  name = %[1]q
  type = "K8S"
}
`, name)
}
