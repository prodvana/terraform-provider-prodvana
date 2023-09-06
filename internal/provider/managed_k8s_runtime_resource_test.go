package provider

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

func TestAccManagedK8sRuntimeResource(t *testing.T) {
	runtimeName := uniqueTestName("managed-k8s-tests")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing
			{
				Config: testAccManagedK8sRuntimeResourceConfig(runtimeName, "foo", map[string]string{
					"foo": "bar",
					"baz": "qux",
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("prodvana_managed_k8s_runtime."+runtimeName, "name", runtimeName),
					resource.TestCheckResourceAttrSet("prodvana_managed_k8s_runtime."+runtimeName, "id"),
					resource.TestCheckResourceAttr("prodvana_managed_k8s_runtime."+runtimeName, "agent_env.PROXY", "foo"),
					resource.TestCheckResourceAttr("prodvana_managed_k8s_runtime."+runtimeName, "labels.0.label", "foo"),
					resource.TestCheckResourceAttr("prodvana_managed_k8s_runtime."+runtimeName, "labels.0.value", "bar"),
					resource.TestCheckResourceAttr("prodvana_managed_k8s_runtime."+runtimeName, "labels.1.label", "baz"),
					resource.TestCheckResourceAttr("prodvana_managed_k8s_runtime."+runtimeName, "labels.1.value", "qux"),
				),
			},
			// Update and Read testing
			{
				Config: testAccManagedK8sRuntimeResourceConfig(runtimeName, "bar", map[string]string{
					"foo": "notbar",
				}),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("prodvana_managed_k8s_runtime."+runtimeName, "name", runtimeName),
					resource.TestCheckResourceAttrSet("prodvana_managed_k8s_runtime."+runtimeName, "id"),
					resource.TestCheckResourceAttr("prodvana_managed_k8s_runtime."+runtimeName, "agent_env.PROXY", "bar"),
					resource.TestCheckResourceAttr("prodvana_managed_k8s_runtime."+runtimeName, "labels.0.label", "foo"),
					resource.TestCheckResourceAttr("prodvana_managed_k8s_runtime."+runtimeName, "labels.0.value", "notbar"),
					resource.TestCheckNoResourceAttr("prodvana_managed_k8s_runtime."+runtimeName, "labels.1.label"),
					resource.TestCheckNoResourceAttr("prodvana_managed_k8s_runtime."+runtimeName, "labels.1.value"),
				),
			},
			// Delete testing automatically occurs in TestCase
		},
	})
}

func TestAccManagedK8sRuntimeResourceK8sAuth(t *testing.T) {
	if os.Getenv(resource.EnvTfAcc) != "1" {
		t.Skipf("Skipping acceptance test due to %s", resource.EnvTfAcc)
	}
	ctx := context.Background()
	cfgPath := os.ExpandEnv("${HOME}/.kube/config")
	cfg, err := clientcmd.LoadFromFile(cfgPath)
	if err != nil {
		t.Fatal(err)
	}
	context := cfg.CurrentContext
	if context != "kind-kind" {
		t.Fatal("This test requires a kind cluster to be running")
	}
	host := cfg.Clusters[cfg.Contexts[context].Cluster].Server
	caCert := string(cfg.Clusters[cfg.Contexts[context].Cluster].CertificateAuthorityData)
	clientCert := string(cfg.AuthInfos[cfg.Contexts[context].AuthInfo].ClientCertificateData)
	clientKey := string(cfg.AuthInfos[cfg.Contexts[context].AuthInfo].ClientKeyData)

	clientConfig, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: cfgPath},
		&clientcmd.ConfigOverrides{CurrentContext: "kind-kind"},
	).ClientConfig()
	if err != nil {
		t.Fatal(err)
	}
	clientSet, err := kubernetes.NewForConfig(clientConfig)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		err := clientSet.CoreV1().ServiceAccounts("default").Delete(ctx, "tf-provider-test", metav1.DeleteOptions{})
		if err != nil && !k8s_errors.IsNotFound(err) {
			t.Fatal(err)
		}
		err = clientSet.RbacV1().ClusterRoleBindings().Delete(ctx, "tf-provider-test", metav1.DeleteOptions{})
		if err != nil && !k8s_errors.IsNotFound(err) {
			t.Fatal(err)
		}
		err = clientSet.CoreV1().Secrets("default").Delete(ctx, "tf-provider-test", metav1.DeleteOptions{})
		if err != nil && !k8s_errors.IsNotFound(err) {
			t.Fatal(err)
		}
	}()

	saSpec := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "tf-provider-test",
			Namespace: "default",
		},
	}
	_, err = clientSet.CoreV1().ServiceAccounts("default").Create(ctx, saSpec, metav1.CreateOptions{})
	if err != nil && !k8s_errors.IsAlreadyExists(err) {
		t.Fatal(err)
	}

	roleBindingSpec := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: "tf-provider-test",
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      saSpec.Name,
				Namespace: saSpec.Namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "cluster-admin",
		},
	}

	_, err = clientSet.RbacV1().ClusterRoleBindings().Create(ctx, roleBindingSpec, metav1.CreateOptions{})
	if err != nil && !k8s_errors.IsAlreadyExists(err) {
		t.Fatal(err)
	}

	secretSpec := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: "tf-provider-test",
			Annotations: map[string]string{
				"kubernetes.io/service-account.name": saSpec.Name,
			},
		},
		Type: corev1.SecretTypeServiceAccountToken,
	}
	_, err = clientSet.CoreV1().Secrets("default").Create(ctx, secretSpec, metav1.CreateOptions{})
	if err != nil && !k8s_errors.IsAlreadyExists(err) {
		t.Fatal(err)
	}

	var token string
	for token == "" {
		secret, err := clientSet.CoreV1().Secrets("default").Get(ctx, "tf-provider-test", metav1.GetOptions{})
		if err != nil {
			t.Fatal(err)
		}
		token = string(secret.Data["token"])
	}

	runtimeName := uniqueTestName("managed-k8s-tests")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { testAccPreCheck(t) },
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// configpath + context
			{
				Config: testAccManagedK8sRuntimeResourceConfigPath(runtimeName, "~/.kube/config", "kind-kind"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("prodvana_managed_k8s_runtime."+runtimeName, "name", runtimeName),
					resource.TestCheckResourceAttrSet("prodvana_managed_k8s_runtime."+runtimeName, "id"),
				),
			},
			{
				Config:  testAccManagedK8sRuntimeResourceConfigPath(runtimeName, "~/.kube/config", "kind-kind"),
				Destroy: true,
			},
			// configpaths
			{
				Config: testAccManagedK8sRuntimeResourceConfigPaths(runtimeName, `["~/.kube/config"]`, "kind-kind"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("prodvana_managed_k8s_runtime."+runtimeName, "name", runtimeName),
					resource.TestCheckResourceAttrSet("prodvana_managed_k8s_runtime."+runtimeName, "id"),
				),
			},
			{
				Config:  testAccManagedK8sRuntimeResourceConfigPaths(runtimeName, `["~/.kube/config"]`, "kind-kind"),
				Destroy: true,
			},
			// ServiceAccount token
			{
				Config: testAccManagedK8sRuntimeResourceToken(runtimeName, host, caCert, token),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("prodvana_managed_k8s_runtime."+runtimeName, "name", runtimeName),
					resource.TestCheckResourceAttrSet("prodvana_managed_k8s_runtime."+runtimeName, "id"),
				),
			},
			{
				Config:  testAccManagedK8sRuntimeResourceConfig(runtimeName, "foo", nil),
				Destroy: true,
			},
			// client cert + key
			{
				Config: testAccManagedK8sRuntimeResourceClientCerts(runtimeName, host, caCert, clientCert, clientKey),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("prodvana_managed_k8s_runtime."+runtimeName, "name", runtimeName),
					resource.TestCheckResourceAttrSet("prodvana_managed_k8s_runtime."+runtimeName, "id"),
				),
			},
			{
				Config:  testAccManagedK8sRuntimeResourceClientCerts(runtimeName, host, caCert, clientCert, clientKey),
				Destroy: true,
			},

			// Delete testing automatically occurs in TestCase
		},
	})
}

func testAccManagedK8sRuntimeResourceConfig(name string, proxy string, labels map[string]string) string {
	labelStr := ""
	for k, v := range labels {
		labelStr += fmt.Sprintf(`
		{
			label = %[1]q
			value = %[2]q
		},
		`, k, v)
	}
	return fmt.Sprintf(`
resource "prodvana_managed_k8s_runtime" "%[1]s" {
  name = %[1]q
  // TODO: kube config options
  config_path = "~/.kube/config"
  config_context = "kind-kind"

  agent_env = {
	"PROXY" = %[2]q 
  }
 labels = [
	%[3]s
 ]
}
`, name, proxy, labelStr)
}

func testAccManagedK8sRuntimeResourceConfigPath(name string, configPath, context string) string {
	return fmt.Sprintf(`
resource "prodvana_managed_k8s_runtime" "%[1]s" {
  name = %[1]q
  config_path = %[2]q 
  config_context = %[3]q
}
`, name, configPath, context)
}

func testAccManagedK8sRuntimeResourceConfigPaths(name string, configPaths, context string) string {
	return fmt.Sprintf(`
resource "prodvana_managed_k8s_runtime" "%[1]s" {
  name = %[1]q
  config_paths = %[2]s
  config_context = %[3]q
}
`, name, configPaths, context)
}

func testAccManagedK8sRuntimeResourceToken(name string, host, caCert, token string) string {
	return fmt.Sprintf(`
resource "prodvana_managed_k8s_runtime" "%[1]s" {
  name = %[1]q
  host = %[2]q
  cluster_ca_certificate = %[3]q
  token = %[4]q
}
`, name, host, caCert, token)
}

func testAccManagedK8sRuntimeResourceClientCerts(name string, host, caCert, clientCert, clientKey string) string {
	return fmt.Sprintf(`
resource "prodvana_managed_k8s_runtime" "%[1]s" {
  name = %[1]q
  host = %[2]q
  cluster_ca_certificate = %[3]q
  client_certificate = %[4]q
  client_key = %[5]q 
}
`, name, host, caCert, clientCert, clientKey)
}
