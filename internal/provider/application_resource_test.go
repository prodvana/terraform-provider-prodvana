package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccApplicationResource(t *testing.T) {
	appName := uniqueTestName("app-tests")
	appName2 := uniqueTestName("app-tests")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccApplicationResourceConfig(appName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("prodvana_application.app", "name", appName),
					resource.TestCheckResourceAttrSet("prodvana_application.app", "version"),
					resource.TestCheckResourceAttrSet("prodvana_application.app", "id"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "prodvana_application.app",
				ImportStateId:     appName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: testAccApplicationResourceConfig(appName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("prodvana_application.app", "name", appName),
					resource.TestCheckResourceAttrSet("prodvana_application.app", "version"),
					resource.TestCheckResourceAttrSet("prodvana_application.app", "id"),
				),
			},
			// application name change forces recreate test
			{
				Config: testAccApplicationResourceConfig(appName2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("prodvana_application.app", "name", appName2),
					resource.TestCheckResourceAttrSet("prodvana_application.app", "version"),
					resource.TestCheckResourceAttrSet("prodvana_application.app", "id"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func uniqueTestName(name string) string {
	prefix := fmt.Sprintf("tf-provider-test-%s-", name)
	return prefix + acctest.RandStringFromCharSet(40-len(prefix), acctest.CharSetAlphaNum)
}

func testAccApplicationResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "prodvana_application" "app" {
  name = %[1]q
}
`, name)
}
