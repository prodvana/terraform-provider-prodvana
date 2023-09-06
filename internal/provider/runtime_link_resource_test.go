package provider

import (
	"fmt"
	"os"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/prodvana/terraform-provider-prodvana/internal/provider/labels"
	"k8s.io/client-go/tools/clientcmd"
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

func TestAccRuntimeLinkResourceTimeout(t *testing.T) {
	runtimeName := uniqueTestName("runtime-link-tests")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config:      testAccK8sRuntimeLinkResourceConfigTimeout(runtimeName),
				ExpectError: regexp.MustCompile(".*Timeout waiting for runtime link status.*"),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccRuntimeLinkResourceFullFlow(t *testing.T) {
	if os.Getenv(resource.EnvTfAcc) != "1" {
		t.Skipf("Skipping acceptance test due to %s", resource.EnvTfAcc)
	}
	cfgPath := os.ExpandEnv("${HOME}/.kube/config")
	cfg, err := clientcmd.LoadFromFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	context := cfg.CurrentContext
	if context != "kind-kind" {
		t.Fatal("This test requires a kind cluster to be running")
	}

	runtimeName := uniqueTestName("runtime-link-tests")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		ExternalProviders: map[string]resource.ExternalProvider{
			"kubernetes": {
				Source:            "hashicorp/kubernetes",
				VersionConstraint: "2.23.0",
			},
		},
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccK8sRuntimeLinkResourceConfigFullFlow(runtimeName, []labels.LabelDefinition{
					{
						Label: "foo",
						Value: "bar",
					},
					{
						Label: "baz",
						Value: "qux",
					},
				}, cfgPath, context),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("prodvana_runtime_link.test", "id"),
					resource.TestCheckResourceAttr("prodvana_runtime_link.test", "timeout", "10m"),
					resource.TestCheckResourceAttr("prodvana_runtime_link.test", "labels.0.label", "foo"),
					resource.TestCheckResourceAttr("prodvana_runtime_link.test", "labels.0.value", "bar"),
					resource.TestCheckResourceAttr("prodvana_runtime_link.test", "labels.1.label", "baz"),
					resource.TestCheckResourceAttr("prodvana_runtime_link.test", "labels.1.value", "qux"),
				),
			},
			// Update and Read test
			{
				Config: testAccK8sRuntimeLinkResourceConfigFullFlow(runtimeName, []labels.LabelDefinition{
					{
						Label: "foo",
						Value: "notbar",
					},
				}, cfgPath, context),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrSet("prodvana_runtime_link.test", "id"),
					resource.TestCheckResourceAttr("prodvana_runtime_link.test", "timeout", "10m"),
					resource.TestCheckResourceAttr("prodvana_runtime_link.test", "labels.0.label", "foo"),
					resource.TestCheckResourceAttr("prodvana_runtime_link.test", "labels.0.value", "notbar"),
					resource.TestCheckNoResourceAttr("prodvana_runtime_link.test", "labels.1.label"),
					resource.TestCheckNoResourceAttr("prodvana_runtime_link.test", "labels.1.value"),
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

func testAccK8sRuntimeLinkResourceConfigTimeout(name string) string {
	return fmt.Sprintf(`
resource "prodvana_k8s_runtime" "test" {
  name = %[1]q
}

resource "prodvana_runtime_link" "test" {
  name = prodvana_k8s_runtime.test.name
  timeout = "1s"
}
`, name)
}

func testAccK8sRuntimeLinkResourceConfigFullFlow(name string, labels []labels.LabelDefinition, configPath, context string) string {
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
provider "kubernetes" {
	config_path    = "%[3]s"
	config_context = "%[4]s"
	}
resource "prodvana_k8s_runtime" "test" {
  name = %[1]q
}
resource "kubernetes_namespace_v1" "agent" {
  metadata {
    name = "prodvana"
  }
}

resource "kubernetes_deployment_v1" "agent" {
  metadata {
    name      = "agent"
    namespace = kubernetes_namespace_v1.agent.metadata.0.name
  }

  spec {
    replicas = 1
	selector {
	  match_labels = {
	    app = "prodvana-agent"
	  }
	}
    template {
      metadata {
        labels = {
          app = "prodvana-agent"
        }
      }
      spec {
        container {
          name  = "prodvana-agent"
          // image = "us-docker.pkg.dev/pvn-infra/pvn-public/agent:4b2b408950898f23ab1082e93d2afa890261a898"
		  image = prodvana_k8s_runtime.test.agent_image
		  args = prodvana_k8s_runtime.test.agent_args
          // args = [
          //   "/agent",
          //   "--clusterid",
          //   prodvana_k8s_runtime.test.id,
          //   "--auth",
          //   prodvana_k8s_runtime.test.agent_api_token,
          //   "--server-addr",
          //   "api.prodvana-cont-testing-staging.staging.prodvana.io",
          // ]
        }
      }
    }
  }
}

resource "prodvana_runtime_link" "test" {
  name = prodvana_k8s_runtime.test.name
  labels = [
	%[2]s
  ]
  depends_on = [
	kubernetes_deployment_v1.agent,
  ]
}
`, name, labelStr, configPath, context)
}
