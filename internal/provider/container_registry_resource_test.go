package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccContainerRegistryResource(t *testing.T) {
	name := uniqueTestName("tf-dockerhub-authed")
	password := os.Getenv("DOCKERHUB_PASSWORD")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccK8sContainerRegistryResource(name, "https://index.docker.io", "prodvana", password),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("prodvana_container_registry.test", "id"),
					resource.TestCheckResourceAttr("prodvana_container_registry.test", "url", "https://index.docker.io"),
					resource.TestCheckResourceAttr("prodvana_container_registry.test", "name", name),
					resource.TestCheckResourceAttr("prodvana_container_registry.test", "public", "false"),
					resource.TestCheckResourceAttr("prodvana_container_registry.test", "username", "prodvana"),
					resource.TestCheckResourceAttr("prodvana_container_registry.test", "password", password),
				),
			},
			// Update and Read test
			{
				Config: testAccK8sContainerRegistryResource(name, "https://index.docker.io", "prodvana", password),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("prodvana_container_registry.test", "id"),
					resource.TestCheckResourceAttr("prodvana_container_registry.test", "url", "https://index.docker.io"),
					resource.TestCheckResourceAttr("prodvana_container_registry.test", "name", name),
					resource.TestCheckResourceAttr("prodvana_container_registry.test", "public", "false"),
					resource.TestCheckResourceAttr("prodvana_container_registry.test", "username", "prodvana"),
					resource.TestCheckResourceAttr("prodvana_container_registry.test", "password", password),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccContainerRegistryResourcePublic(t *testing.T) {
	name := uniqueTestName("tf-dockerhub-pub")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccK8sContainerRegistryResourcePublic(name, "https://index.docker.io"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("prodvana_container_registry.test", "id"),
					resource.TestCheckResourceAttr("prodvana_container_registry.test", "url", "https://index.docker.io"),
					resource.TestCheckResourceAttr("prodvana_container_registry.test", "name", name),
					resource.TestCheckResourceAttr("prodvana_container_registry.test", "public", "true"),
					resource.TestCheckNoResourceAttr("prodvana_container_registry.test", "username"),
					resource.TestCheckNoResourceAttr("prodvana_container_registry.test", "password"),
				),
			},
			// Update and Read test
			{
				Config: testAccK8sContainerRegistryResourcePublic(name, "https://index.docker.io"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("prodvana_container_registry.test", "id"),
					resource.TestCheckResourceAttr("prodvana_container_registry.test", "url", "https://index.docker.io"),
					resource.TestCheckResourceAttr("prodvana_container_registry.test", "name", name),
					resource.TestCheckResourceAttr("prodvana_container_registry.test", "public", "true"),
					resource.TestCheckNoResourceAttr("prodvana_container_registry.test", "username"),
					resource.TestCheckNoResourceAttr("prodvana_container_registry.test", "password"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccK8sContainerRegistryResource(name, url, username, password string) string {
	return fmt.Sprintf(`
resource "prodvana_container_registry" "test" {
  name = %[1]q
  url = %[2]q
  username = %[3]q
  password = %[4]q
}
`, name, url, username, password)
}

func testAccK8sContainerRegistryResourcePublic(name, url string) string {
	return fmt.Sprintf(`
resource "prodvana_container_registry" "test" {
  name = %[1]q
  url = %[2]q
  public = true
}
`, name, url)
}
