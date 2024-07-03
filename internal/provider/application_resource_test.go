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
					resource.TestCheckNoResourceAttr("prodvana_application.app", "description"),
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
					resource.TestCheckNoResourceAttr("prodvana_application.app", "description"),
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

func TestAccApplicationResourceWithDescription(t *testing.T) {
	appName := uniqueTestName("app-tests")
	appDesc := "This is a test description"
	emptyDesc := "Another test description"

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccApplicationResourceConfigWithDescription(appName, appDesc),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("prodvana_application.app", "name", appName),
					resource.TestCheckResourceAttr("prodvana_application.app", "description", appDesc),
					resource.TestCheckResourceAttrSet("prodvana_application.app", "version"),
					resource.TestCheckResourceAttrSet("prodvana_application.app", "id"),
					resource.TestCheckResourceAttr("prodvana_application.app", "no_cleanup_on_delete", "false"),
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
				Config: testAccApplicationResourceConfigWithDescription(appName, emptyDesc),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("prodvana_application.app", "name", appName),
					resource.TestCheckResourceAttr("prodvana_application.app", "description", emptyDesc),
					resource.TestCheckResourceAttrSet("prodvana_application.app", "version"),
					resource.TestCheckResourceAttrSet("prodvana_application.app", "id"),
					resource.TestCheckResourceAttr("prodvana_application.app", "no_cleanup_on_delete", "false"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccApplicationResourceNoCleanupOnDelete(t *testing.T) {
	appName := uniqueTestName("app-tests")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccApplicationResourceConfigNoCleanupOnDelete(appName, true),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("prodvana_application.app", "name", appName),
					resource.TestCheckNoResourceAttr("prodvana_application.app", "description"),
					resource.TestCheckResourceAttrSet("prodvana_application.app", "version"),
					resource.TestCheckResourceAttrSet("prodvana_application.app", "id"),
					resource.TestCheckResourceAttr("prodvana_application.app", "no_cleanup_on_delete", "true"),
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
				Config: testAccApplicationResourceConfigNoCleanupOnDelete(appName, false),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("prodvana_application.app", "name", appName),
					resource.TestCheckNoResourceAttr("prodvana_application.app", "description"),
					resource.TestCheckResourceAttrSet("prodvana_application.app", "version"),
					resource.TestCheckResourceAttrSet("prodvana_application.app", "id"),
					resource.TestCheckResourceAttr("prodvana_application.app", "no_cleanup_on_delete", "false"),
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

func testAccApplicationResourceConfigWithDescription(name, desc string) string {
	return fmt.Sprintf(`
resource "prodvana_application" "app" {
  name = %[1]q
  description = %[2]q
}
`, name, desc)
}

func testAccApplicationResourceConfigNoCleanupOnDelete(name string, noCleanup bool) string {
	return fmt.Sprintf(`
resource "prodvana_application" "app" {
  name = %[1]q
  no_cleanup_on_delete = %[2]t
}
`, name, noCleanup)
}
