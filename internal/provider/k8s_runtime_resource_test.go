package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccK8sRuntimeResource(t *testing.T) {
	runtimeName := uniqueTestName("runtime-tests")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccK8sRuntimeResourceConfig(runtimeName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("prodvana_k8s_runtime.test", "name", runtimeName),
					resource.TestCheckResourceAttrSet("prodvana_k8s_runtime.test", "id"),
					resource.TestCheckResourceAttrSet("prodvana_k8s_runtime.test", "agent_api_token"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "prodvana_k8s_runtime.test",
				ImportStateId:     runtimeName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: testAccK8sRuntimeResourceConfig(runtimeName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("prodvana_k8s_runtime.test", "name", runtimeName),
					resource.TestCheckResourceAttrSet("prodvana_k8s_runtime.test", "id"),
					resource.TestCheckResourceAttrSet("prodvana_k8s_runtime.test", "agent_api_token"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccK8sRuntimeResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "prodvana_k8s_runtime" "test" {
  name = %[1]q
}
`, name)
}
