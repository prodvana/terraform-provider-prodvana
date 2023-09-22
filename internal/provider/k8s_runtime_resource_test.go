package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/prodvana/terraform-provider-prodvana/internal/provider/labels"
)

func TestAccK8sRuntimeResource(t *testing.T) {
	runtimeName := uniqueTestName("runtime-tests")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccK8sRuntimeResourceConfig(runtimeName, nil),
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
				Config: testAccK8sRuntimeResourceConfig(runtimeName, nil),
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
func TestAccK8sRuntimeResourceLabels(t *testing.T) {
	runtimeName := uniqueTestName("runtime-tests")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccK8sRuntimeResourceConfig(runtimeName, []labels.LabelDefinition{
					{
						Label: "foo",
						Value: "bar",
					},
					{
						Label: "baz",
						Value: "qux@",
					},
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("prodvana_k8s_runtime.test", "name", runtimeName),
					resource.TestCheckResourceAttrSet("prodvana_k8s_runtime.test", "id"),
					resource.TestCheckResourceAttrSet("prodvana_k8s_runtime.test", "agent_api_token"),
					resource.TestCheckResourceAttr("prodvana_k8s_runtime.test", "labels.0.label", "foo"),
					resource.TestCheckResourceAttr("prodvana_k8s_runtime.test", "labels.0.value", "bar"),
					resource.TestCheckResourceAttr("prodvana_k8s_runtime.test", "labels.1.label", "baz"),
					resource.TestCheckResourceAttr("prodvana_k8s_runtime.test", "labels.1.value", "qux@"),
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
				Config: testAccK8sRuntimeResourceConfig(runtimeName, []labels.LabelDefinition{
					{
						Label: "foo",
						Value: "not-bar",
					},
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("prodvana_k8s_runtime.test", "name", runtimeName),
					resource.TestCheckResourceAttrSet("prodvana_k8s_runtime.test", "id"),
					resource.TestCheckResourceAttrSet("prodvana_k8s_runtime.test", "agent_api_token"),
					resource.TestCheckResourceAttr("prodvana_k8s_runtime.test", "labels.0.label", "foo"),
					resource.TestCheckResourceAttr("prodvana_k8s_runtime.test", "labels.0.value", "notbar"),
					resource.TestCheckNoResourceAttr("prodvana_k8s_runtime.test", "labels.1.label"),
					resource.TestCheckNoResourceAttr("prodvana_k8s_runtime.test", "labels.1.value"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccK8sRuntimeResourceConfig(name string, labels []labels.LabelDefinition) string {
	labelStr := ""
	for _, label := range labels {
		labelStr += fmt.Sprintf(`
		{
			label = %[1]q
			value = %[2]q
		},
		`, label.Label, label.Value)
	}
	return fmt.Sprintf(`
resource "prodvana_k8s_runtime" "test" {
  name = %[1]q
  labels = [
	%[2]s
  ]
}
`, name, labelStr)
}
