package provider

import (
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccECRRegistryResource(t *testing.T) {
	name := uniqueTestName("tf-ecr")
	keyId := os.Getenv("ECR_ACCESS_KEY_ID")
	accessKey := os.Getenv("ECR_SECRET_ACCESS_KEY")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testinga
			{
				Config: testAccK8sECRRegistryResource(name, "us-west-2", keyId, accessKey),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("prodvana_ecr_registry.test", "id"),
					resource.TestCheckResourceAttr("prodvana_ecr_registry.test", "region", "us-west-2"),
					resource.TestCheckResourceAttr("prodvana_ecr_registry.test", "name", name),
					resource.TestCheckResourceAttr("prodvana_ecr_registry.test", "credentials_auth.access_key_id", keyId),
					resource.TestCheckResourceAttr("prodvana_ecr_registry.test", "credentials_auth.secret_access_key", accessKey),
				),
			},
			// Update and Read test
			{
				Config: testAccK8sECRRegistryResource(name, "us-west-2", keyId, accessKey),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("prodvana_ecr_registry.test", "id"),
					resource.TestCheckResourceAttr("prodvana_ecr_registry.test", "region", "us-west-2"),
					resource.TestCheckResourceAttr("prodvana_ecr_registry.test", "name", name),
					resource.TestCheckResourceAttr("prodvana_ecr_registry.test", "credentials_auth.access_key_id", keyId),
					resource.TestCheckResourceAttr("prodvana_ecr_registry.test", "credentials_auth.secret_access_key", accessKey),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccK8sECRRegistryResource(name, region, accessKeyId, secretAccessKey string) string {
	return fmt.Sprintf(`
resource "prodvana_ecr_registry" "test" {
  name = %[1]q
  region = %[2]q
  credentials_auth = {
	access_key_id = %[3]q
	secret_access_key = %[4]q
  }
}
`, name, region, accessKeyId, secretAccessKey)
}
