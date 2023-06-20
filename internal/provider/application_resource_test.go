package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccApplicationResource(t *testing.T) {
	appName := testAppName("app-tests")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccApplicationResourceConfig(appName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("prodvana_application."+appName, "name", appName),
					resource.TestCheckResourceAttrSet("prodvana_application."+appName, "version"),
					resource.TestCheckResourceAttrSet("prodvana_application."+appName, "id"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "prodvana_application." + appName,
				ImportStateId:     appName,
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: testAccApplicationResourceConfig(appName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("prodvana_application."+appName, "name", appName),
					resource.TestCheckResourceAttrSet("prodvana_application."+appName, "version"),
					resource.TestCheckResourceAttrSet("prodvana_application."+appName, "id"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAppName(name string) string {
	prefix := fmt.Sprintf("tf-provider-test-%s-", name)
	return prefix + acctest.RandStringFromCharSet(40-len(prefix), acctest.CharSetAlphaNum)
}

func testAccApplicationResourceConfig(name string) string {
	return fmt.Sprintf(`
resource "prodvana_application" "%[1]s" {
  name = %[1]q
}
`, name)
}
