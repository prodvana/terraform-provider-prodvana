package provider

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
)

func TestAccReleaseChannelResource(t *testing.T) {
	appName := uniqueTestName("rc-tests")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccReleaseChannelResourceConfig("staging", appName, nil),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("prodvana_release_channel.staging", "name", "staging"),
					resource.TestCheckResourceAttrSet("prodvana_release_channel.staging", "version"),
					resource.TestCheckResourceAttrSet("prodvana_release_channel.staging", "id"),
					resource.TestCheckResourceAttr("prodvana_release_channel.staging", "runtimes.0.runtime", "default"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "prodvana_release_channel.staging",
				ImportStateId:     appName + "/staging",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: testAccReleaseChannelResourceConfig("staging", appName, map[string]string{
					"TEST_VAR_ONE": "test value one",
					"TEST_VAR_TWO": "test value two",
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("prodvana_release_channel.staging", "name", "staging"),
					resource.TestCheckResourceAttrSet("prodvana_release_channel.staging", "version"),
					resource.TestCheckResourceAttrSet("prodvana_release_channel.staging", "id"),
					resource.TestCheckResourceAttr("prodvana_release_channel.staging", "runtimes.0.runtime", "default"),
					resource.TestCheckResourceAttr("prodvana_release_channel.staging", "policy.default_env.TEST_VAR_ONE.value", "test value one"),
					resource.TestCheckResourceAttr("prodvana_release_channel.staging", "policy.default_env.TEST_VAR_TWO.value", "test value two"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccReleaseChannelResourceWithContainerOrchestration(t *testing.T) {
	appName := uniqueTestName("rc-tests")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccReleaseChannelResourceWithK8sNamespace(appName, "test-namespace"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "name", "test"),
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "runtimes.0.k8s_namespace", "test-namespace"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "prodvana_release_channel.test",
				ImportStateId:     appName + "/test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: testAccReleaseChannelResourceWithK8sNamespace(appName, "test-namespace"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "name", "test"),
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "runtimes.0.k8s_namespace", "test-namespace"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccReleaseChannelResourceWithRuntimeType(t *testing.T) {
	appName := uniqueTestName("rc-tests")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccReleaseChannelResourceWithRuntimeType(appName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "name", "test"),
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "runtimes.0.type", "LONG_LIVED_COMPUTE"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "prodvana_release_channel.test",
				ImportStateId:     appName + "/test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: testAccReleaseChannelResourceWithRuntimeType(appName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "name", "test"),
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "runtimes.0.type", "LONG_LIVED_COMPUTE"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccReleaseChannelResourceWithStablePrecondition(t *testing.T) {
	appName := uniqueTestName("rc-tests")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccReleaseChannelResourceWithPreconditions(appName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "name", "test"),

					resource.TestCheckResourceAttr("prodvana_release_channel.test", "release_channel_stable_preconditions.0.release_channel", "pre"),
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "release_channel_stable_preconditions.1.release_channel", "pre2"),

					resource.TestCheckResourceAttr("prodvana_release_channel.test", "manual_approval_preconditions.0.name", "approval1"),
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "manual_approval_preconditions.1.name", "approval2"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "prodvana_release_channel.test",
				ImportStateId:     appName + "/test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: testAccReleaseChannelResourceWithPreconditions(appName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "name", "test"),

					resource.TestCheckResourceAttr("prodvana_release_channel.test", "release_channel_stable_preconditions.0.release_channel", "pre"),
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "release_channel_stable_preconditions.1.release_channel", "pre2"),

					resource.TestCheckResourceAttr("prodvana_release_channel.test", "manual_approval_preconditions.0.name", "approval1"),
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "manual_approval_preconditions.1.name", "approval2"),
				),
			},
			{
				Config: testAccReleaseChannelResourceWithoutPreconditions(appName),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "name", "test"),

					resource.TestCheckNoResourceAttr("prodvana_release_channel.test", "release_channel_stable_preconditions"),
					resource.TestCheckNoResourceAttr("prodvana_release_channel.test", "manual_approval_preconditions"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccReleaseChannelResourceWithProtections(t *testing.T) {
	appName := uniqueTestName("rc-tests")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccReleaseChannelResourceWithProtections(appName, "foo", 10),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "name", "test"),

					resource.TestCheckResourceAttr("prodvana_release_channel.test", "protections.0.name", "param-test"),
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "protections.0.ref.parameters.0.name", "paramA"),
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "protections.0.ref.parameters.0.string_value", "foo"),
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "protections.0.ref.parameters.1.name", "paramB"),
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "protections.0.ref.parameters.1.int_value", "10"),
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "protections.0.ref.parameters.2.name", "paramC"),
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "protections.0.ref.parameters.2.secret_value.key", "tf-testing-secret"),
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "protections.0.ref.parameters.2.secret_value.version", "tf-testing-secret-0"),

					resource.TestCheckResourceAttrSet("prodvana_release_channel.test", "convergence_protections.0.pre_approval.%"),
					resource.TestCheckResourceAttrSet("prodvana_release_channel.test", "convergence_protections.0.post_approval.%"),
					resource.TestCheckResourceAttrSet("prodvana_release_channel.test", "convergence_protections.0.deployment.%"),
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "convergence_protections.0.post_deployment.delay_check_duration", "10s"),
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "convergence_protections.0.post_deployment.check_duration", "30s"),

					resource.TestCheckResourceAttr("prodvana_release_channel.test", "convergence_protections.0.name", "param-test"),
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "convergence_protections.0.ref.parameters.0.name", "paramA"),
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "convergence_protections.0.ref.parameters.0.string_value", "foo"),
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "convergence_protections.0.ref.parameters.1.name", "paramB"),
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "convergence_protections.0.ref.parameters.1.int_value", "10"),
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "convergence_protections.0.ref.parameters.2.name", "paramC"),
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "convergence_protections.0.ref.parameters.2.secret_value.key", "tf-testing-secret"),
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "convergence_protections.0.ref.parameters.2.secret_value.version", "tf-testing-secret-0"),

					resource.TestCheckResourceAttrSet("prodvana_release_channel.test", "convergence_protections.0.pre_approval.%"),
					resource.TestCheckResourceAttrSet("prodvana_release_channel.test", "convergence_protections.0.post_approval.%"),
					resource.TestCheckResourceAttrSet("prodvana_release_channel.test", "convergence_protections.0.deployment.%"),
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "convergence_protections.0.post_deployment.delay_check_duration", "10s"),
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "convergence_protections.0.post_deployment.check_duration", "30s"),

					resource.TestCheckResourceAttr("prodvana_release_channel.test", "service_instance_protections.0.name", "param-test"),
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "service_instance_protections.0.ref.parameters.0.name", "paramA"),
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "service_instance_protections.0.ref.parameters.0.string_value", "foo"),
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "service_instance_protections.0.ref.parameters.1.name", "paramB"),
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "service_instance_protections.0.ref.parameters.1.int_value", "10"),
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "service_instance_protections.0.ref.parameters.2.name", "paramC"),
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "service_instance_protections.0.ref.parameters.2.secret_value.key", "tf-testing-secret"),
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "service_instance_protections.0.ref.parameters.2.secret_value.version", "tf-testing-secret-0"),

					resource.TestCheckResourceAttrSet("prodvana_release_channel.test", "service_instance_protections.0.pre_approval.%"),
					resource.TestCheckResourceAttrSet("prodvana_release_channel.test", "service_instance_protections.0.post_approval.%"),
					resource.TestCheckResourceAttrSet("prodvana_release_channel.test", "service_instance_protections.0.deployment.%"),
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "service_instance_protections.0.post_deployment.delay_check_duration", "10s"),
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "service_instance_protections.0.post_deployment.check_duration", "30s"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "prodvana_release_channel.test",
				ImportStateId:     appName + "/test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: testAccReleaseChannelResourceWithProtections(appName, "bar", 20),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "name", "test"),

					resource.TestCheckResourceAttr("prodvana_release_channel.test", "protections.0.name", "param-test"),
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "protections.0.ref.parameters.0.name", "paramA"),
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "protections.0.ref.parameters.0.string_value", "bar"),
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "protections.0.ref.parameters.1.name", "paramB"),
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "protections.0.ref.parameters.1.int_value", "20"),
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "protections.0.ref.parameters.2.name", "paramC"),
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "protections.0.ref.parameters.2.secret_value.key", "tf-testing-secret"),
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "protections.0.ref.parameters.2.secret_value.version", "tf-testing-secret-0"),

					resource.TestCheckResourceAttrSet("prodvana_release_channel.test", "protections.0.pre_approval.%"),
					resource.TestCheckResourceAttrSet("prodvana_release_channel.test", "protections.0.post_approval.%"),
					resource.TestCheckResourceAttrSet("prodvana_release_channel.test", "protections.0.deployment.%"),
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "protections.0.post_deployment.delay_check_duration", "10s"),
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "protections.0.post_deployment.check_duration", "30s"),

					resource.TestCheckResourceAttr("prodvana_release_channel.test", "convergence_protections.0.name", "param-test"),
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "convergence_protections.0.ref.parameters.0.name", "paramA"),
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "convergence_protections.0.ref.parameters.0.string_value", "bar"),
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "convergence_protections.0.ref.parameters.1.name", "paramB"),
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "convergence_protections.0.ref.parameters.1.int_value", "20"),
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "convergence_protections.0.ref.parameters.2.name", "paramC"),
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "convergence_protections.0.ref.parameters.2.secret_value.key", "tf-testing-secret"),
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "convergence_protections.0.ref.parameters.2.secret_value.version", "tf-testing-secret-0"),

					resource.TestCheckResourceAttrSet("prodvana_release_channel.test", "convergence_protections.0.pre_approval.%"),
					resource.TestCheckResourceAttrSet("prodvana_release_channel.test", "convergence_protections.0.post_approval.%"),
					resource.TestCheckResourceAttrSet("prodvana_release_channel.test", "convergence_protections.0.deployment.%"),
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "convergence_protections.0.post_deployment.delay_check_duration", "10s"),
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "convergence_protections.0.post_deployment.check_duration", "30s"),

					resource.TestCheckResourceAttr("prodvana_release_channel.test", "service_instance_protections.0.name", "param-test"),
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "service_instance_protections.0.ref.parameters.0.name", "paramA"),
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "service_instance_protections.0.ref.parameters.0.string_value", "bar"),
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "service_instance_protections.0.ref.parameters.1.name", "paramB"),
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "service_instance_protections.0.ref.parameters.1.int_value", "20"),
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "service_instance_protections.0.ref.parameters.2.name", "paramC"),
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "service_instance_protections.0.ref.parameters.2.secret_value.key", "tf-testing-secret"),
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "service_instance_protections.0.ref.parameters.2.secret_value.version", "tf-testing-secret-0"),

					resource.TestCheckResourceAttrSet("prodvana_release_channel.test", "service_instance_protections.0.pre_approval.%"),
					resource.TestCheckResourceAttrSet("prodvana_release_channel.test", "service_instance_protections.0.post_approval.%"),
					resource.TestCheckResourceAttrSet("prodvana_release_channel.test", "service_instance_protections.0.deployment.%"),
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "service_instance_protections.0.post_deployment.delay_check_duration", "10s"),
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "service_instance_protections.0.post_deployment.check_duration", "30s"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccReleaseChannelResourceWithConstant(t *testing.T) {
	appName := uniqueTestName("rc-tests")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccReleaseChannelResourceWithConstant(appName, "foo", "bar"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "name", "test"),
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "constants.0.name", "foo"),
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "constants.0.string_value", "bar"),
				),
			},
			// ImportState testing
			{
				ResourceName:      "prodvana_release_channel.test",
				ImportStateId:     appName + "/test",
				ImportState:       true,
				ImportStateVerify: true,
			},
			// Update and Read testing
			{
				Config: testAccReleaseChannelResourceWithConstant(appName, "foo", "baz"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "name", "test"),
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "constants.0.name", "foo"),
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "constants.0.string_value", "baz"),
				),
			},
			{
				Config: testAccReleaseChannelResourceWithConstant(appName, "bar", "bee"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "name", "test"),
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "constants.0.name", "bar"),
					resource.TestCheckResourceAttr("prodvana_release_channel.test", "constants.0.string_value", "bee"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccReleaseChannelResourceWithRuntimeType(app string) string {
	return fmt.Sprintf(`
%[1]s

resource "prodvana_release_channel" "test" {
  name = "test"
  application = prodvana_application.%[2]s.name
  runtimes = [
	{
		runtime = "default"
		type = "LONG_LIVED_COMPUTE"
	},
  ]
}
`, testAccApplicationResourceConfig(app), app)
}

func testAccReleaseChannelResourceWithPreconditions(app string) string {
	return fmt.Sprintf(`
%[1]s

resource "prodvana_release_channel" "pre" {
  name = "pre"
  application = prodvana_application.%[2]s.name
  runtimes = [
	{
		runtime = "default"
	},
  ]
}

resource "prodvana_release_channel" "pre2" {
  name = "pre2"
  application = prodvana_application.%[2]s.name
  runtimes = [
	{
		runtime = "default"
	},
  ]
}

resource "prodvana_release_channel" "test" {
  name = "test"
  application = prodvana_application.%[2]s.name
  runtimes = [
	{
		runtime = "default"
	},
  ]
  release_channel_stable_preconditions = [
	{
		  release_channel = prodvana_release_channel.pre.name
		  duration = "2s"
	},
	{
		  release_channel = prodvana_release_channel.pre2.name
		  duration = "2s"
	},
  ]
  manual_approval_preconditions = [
	{
		name = "approval1"
	},
	{
		name = "approval2"
	},
  ]
}
`, testAccApplicationResourceConfig(app), app)
}

func testAccReleaseChannelResourceWithoutPreconditions(app string) string {
	return fmt.Sprintf(`
%[1]s

resource "prodvana_release_channel" "pre" {
  name = "pre"
  application = prodvana_application.%[2]s.name
  runtimes = [
	{
		runtime = "default"
	},
  ]
}

resource "prodvana_release_channel" "pre2" {
  name = "pre2"
  application = prodvana_application.%[2]s.name
  runtimes = [
	{
		runtime = "default"
	},
  ]
}

resource "prodvana_release_channel" "test" {
  name = "test"
  application = prodvana_application.%[2]s.name
  runtimes = [
	{
		runtime = "default"
	},
  ]
}
`, testAccApplicationResourceConfig(app), app)
}

func testAccReleaseChannelResourceWithK8sNamespace(app string, namespace string) string {
	return fmt.Sprintf(`
%[1]s

resource "prodvana_release_channel" "test" {
  name = "test"
  application = prodvana_application.%[2]s.name
  runtimes = [
	{
		runtime = "default"
		k8s_namespace = %[3]q
	},
  ]
}
`, testAccApplicationResourceConfig(app), app, namespace)
}

func testAccReleaseChannelResourceConfig(name, app string, env map[string]string) string {
	policy := ""
	if env != nil {
		policy = "  policy = {\n    default_env = {\n"
		for key, value := range env {
			policy += fmt.Sprintf("    %[1]q = { value = %[2]q }\n", key, value)
		}
		policy += "    }\n  }"
	}

	return fmt.Sprintf(`
%[1]s

resource "prodvana_release_channel" "%[2]s" {
  name = %[2]q
  application = prodvana_application.%[3]s.name
  runtimes = [
	{
		runtime = "default"
	},
  ]
  %[4]s
}
`, testAccApplicationResourceConfig(app), name, app, policy)
}

func testAccReleaseChannelResourceWithProtections(app string, paramA string, paramB int64) string {
	return fmt.Sprintf(`
%[1]s

resource "prodvana_release_channel" "test" {
  name = "test"
  application = prodvana_application.%[2]s.name
  runtimes = [
	{
		runtime = "default"
	},
  ]
  protections = [
    {
		ref = {
		  name = "param-test"
		  parameters = [
		  	{
		  		name = "paramA"
		  		string_value = %[3]q
		  	},
		  	{
		  		name = "paramB"
		  		int_value = %[4]d
		  	},
		  	{
		  		name = "paramC"
		  		secret_value = {
					key = "tf-testing-secret"
					version = "tf-testing-secret-0"
				}
		  	},
		  ]
		}
		pre_approval = {
			enabled = true
		}
		post_approval = {
			enabled = true
		}
		deployment = {
			enabled = true
		}
		post_deployment = {
			enabled = true
			delay_check_duration = "10s"
			check_duration = "30s"

		}
	}
  ]
  convergence_protections = [
    {
		ref = {
		  name = "param-test"
		  parameters = [
		  	{
		  		name = "paramA"
		  		string_value = %[3]q
		  	},
		  	{
		  		name = "paramB"
		  		int_value = %[4]d
		  	},
		  	{
		  		name = "paramC"
		  		secret_value = {
					key = "tf-testing-secret"
					version = "tf-testing-secret-0"
				}
		  	},
		  ]
		}
		pre_approval = {
			enabled = true
		}
		post_approval = {
			enabled = true
		}
		deployment = {
			enabled = true
		}
		post_deployment = {
			enabled = true
			delay_check_duration = "10s"
			check_duration = "30s"

		}
	}
  ]
  service_instance_protections = [
    {
		ref = {
		  name = "param-test"
		  parameters = [
		  	{
		  		name = "paramA"
		  		string_value = %[3]q
		  	},
		  	{
		  		name = "paramB"
		  		int_value = %[4]d
		  	},
		  	{
		  		name = "paramC"
		  		secret_value = {
					key = "tf-testing-secret"
					version = "tf-testing-secret-0"
				}
		  	},
		  ]
		}
		pre_approval = {
			enabled = true
		}
		post_approval = {
			enabled = true
		}
		deployment = {
			enabled = true
		}
		post_deployment = {
			enabled = true
			delay_check_duration = "10s"
			check_duration = "30s"

		}
	}
  ]
}
`, testAccApplicationResourceConfig(app), app, paramA, paramB)
}

func testAccReleaseChannelResourceWithConstant(app string, key, value string) string {
	return fmt.Sprintf(`
%[1]s

resource "prodvana_release_channel" "test" {
  name = "test"
  application = prodvana_application.%[2]s.name
  runtimes = [
	{
		runtime = "default"
	},
  ]
  constants = [
    {
		name = %[3]q
		string_value = %[4]q
	}
  ]
}
`, testAccApplicationResourceConfig(app), app, key, value)
}
