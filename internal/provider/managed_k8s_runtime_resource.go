package provider

import (
	"context"
	"fmt"

	"github.com/mitchellh/go-homedir"
	"github.com/pkg/errors"
	env_pb "github.com/prodvana/prodvana-public/go/prodvana-sdk/proto/prodvana/environment"
	"github.com/prodvana/prodvana-public/go/prodvana-sdk/proto/prodvana/version"
	"github.com/prodvana/terraform-provider-prodvana/internal/provider/defaults"
	"github.com/prodvana/terraform-provider-prodvana/internal/provider/labels"
	"github.com/prodvana/terraform-provider-prodvana/internal/provider/validators"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	apimachineryschema "k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const (
	defaultAgentNamespace    = "prodvana"
	clusterRoleBindingName   = "prodvana-access"
	serviceAccountName       = "prodvana"
	agentDeploymentName      = "prodvana-agent"
	agentRuntimeIdAnnotation = "prodvana.io/runtime-id"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ManagedK8sRuntimeResource{}

func NewManagedK8sRuntimeResource() resource.Resource {
	return &ManagedK8sRuntimeResource{}
}

// ManagedK8sRuntimeResource defines the resource implementation.
type ManagedK8sRuntimeResource struct {
	client    env_pb.EnvironmentManagerClient
	clientset *kubernetes.Clientset
}

// ManagedK8sRuntimeResourceModel describes the resource data model.
type ManagedK8sRuntimeResourceModel struct {
	Name types.String `tfsdk:"name"`
	Id   types.String `tfsdk:"id"`

	AgentEnv types.Map `tfsdk:"agent_env"`

	Labels types.List `tfsdk:"labels"`

	// Matches the authentication options provided by terraform-provider-kubernetes
	Host                  types.String `tfsdk:"host"`
	Username              types.String `tfsdk:"username"`
	Password              types.String `tfsdk:"password"`
	Insecure              types.Bool   `tfsdk:"insecure"`
	TlsServerName         types.String `tfsdk:"tls_server_name"`
	ClientCertificate     types.String `tfsdk:"client_certificate"`
	ClientKey             types.String `tfsdk:"client_key"`
	ClusterCaCertificate  types.String `tfsdk:"cluster_ca_certificate"`
	ConfigPaths           types.List   `tfsdk:"config_paths"`
	ConfigPath            types.String `tfsdk:"config_path"`
	ConfigContext         types.String `tfsdk:"config_context"`
	ConfigContextAuthInfo types.String `tfsdk:"config_context_auth_info"`
	ConfigContextCluster  types.String `tfsdk:"config_context_cluster"`
	Token                 types.String `tfsdk:"token"`
	ProxyUrl              types.String `tfsdk:"proxy_url"`
	Exec                  *execModel   `tfsdk:"exec"`

	Timeout types.String `tfsdk:"timeout"`

	// TODO: annotation / label passthrough

	//  read-only computed attributes
	// the runtime_id as read from the agent annotation,
	// used by the resource to detect if the underlying k8s
	// cluster changed / the runtime was renamed
	AgentRuntimeId types.String `tfsdk:"agent_runtime_id"`
	// the k8s namespace the agent is running in
	AgentNamespace types.String `tfsdk:"agent_namespace"`
}

type execModel struct {
	ApiVersion types.String `tfsdk:"api_version"`
	Command    types.String `tfsdk:"command"`
	Env        types.Map    `tfsdk:"env"`
	Args       types.List   `tfsdk:"args"`
}

// borrowed heavily from https://github.com/hashicorp/terraform-provider-kubernetes/blob/7c0a6540aa99288200075999b0b1edad12dcfc83/kubernetes/provider.go#L491
func (r *ManagedK8sRuntimeResource) initializeConfiguration(ctx context.Context, diags diag.Diagnostics, planData *ManagedK8sRuntimeResourceModel) (*rest.Config, error) {
	overrides := &clientcmd.ConfigOverrides{}
	loader := &clientcmd.ClientConfigLoadingRules{}

	configPaths := []string{}

	if !planData.ConfigPath.IsNull() {
		configPaths = []string{planData.ConfigPath.ValueString()}
	} else if !planData.ConfigPaths.IsNull() {
		convDiags := planData.ConfigPaths.ElementsAs(ctx, &configPaths, false)
		diags.Append(convDiags...)
		if diags.HasError() {
			return nil, errors.Errorf("failed to convert config_paths to []string: %v", diags.Errors())
		}
	}

	if len(configPaths) > 0 {
		expandedPaths := []string{}
		for _, p := range configPaths {
			path, err := homedir.Expand(p)
			if err != nil {
				return nil, err
			}
			tflog.Debug(ctx, fmt.Sprintf("Using kubeconfig: %s", path))
			expandedPaths = append(expandedPaths, path)
		}

		if len(expandedPaths) == 1 {
			loader.ExplicitPath = expandedPaths[0]
		} else {
			loader.Precedence = expandedPaths
		}

		ctxOk := !planData.ConfigContext.IsNull()
		authInfoOk := !planData.ConfigContextAuthInfo.IsNull()
		clusterOk := !planData.ConfigContextCluster.IsNull()
		if ctxOk || authInfoOk || clusterOk {
			if ctxOk {
				overrides.CurrentContext = planData.ConfigContext.ValueString()
				tflog.Debug(ctx, fmt.Sprintf("Using custom current context: %q", overrides.CurrentContext))
			}

			overrides.Context = clientcmdapi.Context{}
			if authInfoOk {
				overrides.Context.AuthInfo = planData.ConfigContextAuthInfo.ValueString()
			}
			if clusterOk {
				overrides.Context.Cluster = planData.ConfigContextCluster.ValueString()
			}
			tflog.Debug(ctx, fmt.Sprintf("Using overridden context: %#v", overrides.Context))
		}
	}

	// Overriding with static configuration
	if !planData.Insecure.IsNull() {
		overrides.ClusterInfo.InsecureSkipTLSVerify = planData.Insecure.ValueBool()
	}
	if !planData.TlsServerName.IsNull() {
		overrides.ClusterInfo.TLSServerName = planData.TlsServerName.ValueString()
	}
	if !planData.ClusterCaCertificate.IsNull() {
		overrides.ClusterInfo.CertificateAuthorityData = []byte(planData.ClusterCaCertificate.ValueString())
	}
	if !planData.ClientCertificate.IsNull() {
		overrides.AuthInfo.ClientCertificateData = []byte(planData.ClientCertificate.ValueString())
	}
	if !planData.Host.IsNull() {
		// Server has to be the complete address of the kubernetes cluster (scheme://hostname:port), not just the hostname,
		// because `overrides` are processed too late to be taken into account by `defaultServerUrlFor()`.
		// This basically replicates what defaultServerUrlFor() does with config but for overrides,
		// see https://github.com/kubernetes/client-go/blob/v12.0.0/rest/url_utils.go#L85-L87
		hasCA := len(overrides.ClusterInfo.CertificateAuthorityData) != 0
		hasCert := len(overrides.AuthInfo.ClientCertificateData) != 0
		defaultTLS := hasCA || hasCert || overrides.ClusterInfo.InsecureSkipTLSVerify
		host, _, err := rest.DefaultServerURL(
			planData.Host.ValueString(), "", apimachineryschema.GroupVersion{}, defaultTLS,
		)
		if err != nil {
			return nil, errors.Wrap(err, "failed to parse host")
		}

		overrides.ClusterInfo.Server = host.String()
	}

	if !planData.Username.IsNull() {
		overrides.AuthInfo.Username = planData.Username.ValueString()
	}
	if !planData.Password.IsNull() {
		overrides.AuthInfo.Password = planData.Password.ValueString()
	}
	if !planData.ClientKey.IsNull() {
		overrides.AuthInfo.ClientKeyData = []byte(planData.ClientKey.ValueString())
	}
	if !planData.Token.IsNull() {
		overrides.AuthInfo.Token = planData.Token.ValueString()
	}
	if planData.Exec != nil {
		exec := &clientcmdapi.ExecConfig{}
		exec.InteractiveMode = clientcmdapi.IfAvailableExecInteractiveMode
		exec.APIVersion = planData.Exec.ApiVersion.ValueString()
		exec.Command = planData.Exec.Command.ValueString()
		if !planData.Exec.Args.IsNull() {
			argDiags := planData.Exec.Args.ElementsAs(ctx, &exec.Args, false)
			if argDiags.HasError() {
				diags.Append(argDiags...)
				return nil, errors.Errorf("Failed to parse exec args: %v", argDiags.Errors())
			}
		}
		if !planData.Exec.Env.IsNull() {
			envs := map[string]string{}
			envDiags := planData.Exec.Env.ElementsAs(ctx, &envs, false)
			if envDiags.HasError() {
				diags.Append(envDiags...)
				return nil, errors.Errorf("Failed to parse exec env: %v", envDiags.Errors())
			}
			exec.Env = []clientcmdapi.ExecEnvVar{}
			for kk, vv := range envs {
				exec.Env = append(exec.Env, clientcmdapi.ExecEnvVar{Name: kk, Value: vv})
			}
		}
		overrides.AuthInfo.Exec = exec
	}
	tflog.Debug(ctx, fmt.Sprintf("exec: %#v", overrides.AuthInfo.Exec))

	if !planData.ProxyUrl.IsNull() {
		overrides.ClusterDefaults.ProxyURL = planData.ProxyUrl.ValueString()
	}

	cc := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loader, overrides)
	cfg, err := cc.ClientConfig()
	if err != nil {
		return nil, errors.Wrapf(err, "Invalid kubernetes configuration was supplied")
	}
	return cfg, nil
}

func (r *ManagedK8sRuntimeResource) clientSet(ctx context.Context, diags diag.Diagnostics, planData *ManagedK8sRuntimeResourceModel) (*kubernetes.Clientset, error) {
	if r.clientset != nil {
		return r.clientset, nil
	}
	cfg, err := r.initializeConfiguration(ctx, diags, planData)
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to create kubernetes client")
	}
	r.clientset = clientset

	return r.clientset, nil
}

func (r *ManagedK8sRuntimeResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_managed_k8s_runtime"
}

func (r *ManagedK8sRuntimeResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `(Alpha! This feature is still in progress.) Manages a Kubernetes [Runtime](https://docs.prodvana.io/docs/prodvana-concepts#runtime).
This resource links a Kubernetes runtime with Prodvana and fully manages the agent lifecycle.

The agent will be installed as a Kubernetes deployment in the specified namespace, *by this resource*. After the initial agent install, Prodvana will manage the agent lifecycle, including upgrades, outside of Terraform.
`,
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Runtime name",
				Required:            true,
				Validators:          validators.DefaultNameValidators(),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Runtime identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"labels": schema.ListNestedAttribute{
				MarkdownDescription: "List of labels to apply to the runtime",
				Computed:            true,
				Optional:            true,
				NestedObject:        labels.LabelDefinitionNestedObjectResourceSchema(),
			},
			"timeout": schema.StringAttribute{
				MarkdownDescription: "How long to wait for the runtime linking to complete. A valid Go duration string, e.g. `10m` or `1h`. Defaults to `10m`",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("10m"),
			},
			"agent_runtime_id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The runtime identifier of the agent",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"agent_namespace": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "The namespace of the agent",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"agent_env": schema.MapAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Environment variables to pass to the agent. Useful for cases like passing proxy configuration to the agent if needed.",
				Default:             mapdefault.StaticValue(types.MapNull(types.StringType)),
			},
			"host": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "The address of the Kubernetes cluster (scheme://hostname:port)",
				Default:             defaults.EnvStringValue("KUBE_HOST"),
			},
			"username": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Username for basic authentication to the Kubernetes cluster",
				Default:             defaults.EnvStringValue("KUBE_USER"),
			},
			"password": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Password for basic authentication to the Kubernetes cluster",
				Default:             defaults.EnvStringValue("KUBE_PASSWORD"),
			},
			"insecure": schema.BoolAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Whether server should be accessed without verifying the TLS certificate",
				Default:             defaults.EnvBoolValue("KUBE_INSECURE", false),
			},
			"tls_server_name": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Server name passed to the server for SNI and is used in the client to check server certificates against",
				Default:             defaults.EnvStringValue("KUBE_TLS_SERVER_NAME"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},

			"client_certificate": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "PEM-encoded client certificate for TLS authentication.",
				Default:             defaults.EnvStringValue("KUBE_CLIENT_CERT_DATA"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"client_key": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "PEM-encoded client certificate key for TLS authentication.",
				Default:             defaults.EnvStringValue("KUBE_CLIENT_KEY_DATA"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"cluster_ca_certificate": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "PEM-encoded root certificates bundle for TLS authentication.",
				Default:             defaults.EnvStringValue("KUBE_CLUSTER_CA_CERT_DATA"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"config_paths": schema.ListAttribute{
				ElementType:         types.StringType,
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "A list of paths to kube config files.",
				Default:             defaults.EnvPathListValue("KUBE_CONFIG_PATHS"),
			},
			"config_path": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Path to the kube config file.",
				Default:             defaults.EnvStringValue("KUBE_CONFIG_PATH"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"config_context": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Context to use from the kube config file.",
				Default:             defaults.EnvStringValue("KUBE_CTX"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"config_context_auth_info": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Authentication info context of the kube config (name of the kubeconfig user, `--user` flag in `kubectl`).",
				Default:             defaults.EnvStringValue("KUBE_CTX_AUTH_INFO"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"config_context_cluster": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Cluster context of the kube config (name of the kubeconfig cluster, `--cluster` flag in `kubectl`).",
				Default:             defaults.EnvStringValue("KUBE_CTX_CLUSTER"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"token": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Token to authenticate an service account",
				Default:             defaults.EnvStringValue("KUBE_TOKEN"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"exec": schema.SingleNestedAttribute{
				Optional:            true,
				MarkdownDescription: "Exec configuration for authentication to the Kubernetes cluster",
				Attributes: map[string]schema.Attribute{
					"api_version": schema.StringAttribute{
						Required:            true,
						MarkdownDescription: "API version of the exec credential plugin",
					},
					"command": schema.StringAttribute{
						Required:            true,
						MarkdownDescription: "Command to execute",
					},
					"args": schema.ListAttribute{
						ElementType:         types.StringType,
						Optional:            true,
						MarkdownDescription: "Arguments to pass when executing the command",
					},
					"env": schema.MapAttribute{
						ElementType:         types.StringType,
						Optional:            true,
						MarkdownDescription: "Environment variables to set when executing the command",
					},
				},
			},
			"proxy_url": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Proxy URL to use when accessing the Kubernetes cluster",
				Default:             defaults.EnvStringValue("KUBE_PROXY_URL"),
			},
		},
	}
}

func (r *ManagedK8sRuntimeResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	conn, ok := req.ProviderData.(*grpc.ClientConn)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *grpc.ClientConn, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	r.client = env_pb.NewEnvironmentManagerClient(conn)
}

func getDeploymentRuntimeId(ctx context.Context, clientSet *kubernetes.Clientset, data *ManagedK8sRuntimeResourceModel) (bool, string, error) {
	agentDeploy, err := clientSet.AppsV1().Deployments(data.AgentNamespace.ValueString()).Get(ctx, agentDeploymentName, metav1.GetOptions{})
	if err != nil {
		if k8s_errors.IsNotFound(err) {
			return false, "", nil
		}
		return false, "", errors.Wrapf(err, "Unable to read agent deployment for %s", data.Name.ValueString())
	}
	return true, agentDeploy.Annotations[agentRuntimeIdAnnotation], nil
}

func readManagedK8sRuntimeData(ctx context.Context, diags diag.Diagnostics, client env_pb.EnvironmentManagerClient, clientSet *kubernetes.Clientset, data *ManagedK8sRuntimeResourceModel) error {
	resp, err := client.GetCluster(ctx, &env_pb.GetClusterReq{
		Runtime:     data.Name.ValueString(),
		IncludeAuth: true,
	})
	if err != nil {
		return errors.Wrapf(err, "Unable to read runtime state for %s", data.Name.ValueString())
	}

	data.Name = types.StringValue(resp.Cluster.Name)
	data.Id = types.StringValue(resp.Cluster.Id)
	tfLabels := types.ListNull(labels.LabelDefinitionObjectType)
	if data.Labels.IsUnknown() || data.Labels.IsNull() {
		tfLabels = labels.LabelDefinitionsToTerraformList(ctx, resp.Cluster.Config.Labels, diags)
		if diags.HasError() {
			return errors.Errorf("Failed to convert labels: %v", diags.Errors())
		}
	} else if !data.Labels.IsNull() {
		userProvidedLabels := labels.LabelDefinitionsFromTerraformList(ctx, data.Labels, diags)
		if diags.HasError() {
			return errors.Errorf("Failed to convert labels: %v", diags.Errors())
		}
		tfLabels = labels.LabelDefinitionsToTerraformListWithValidation(ctx, resp.Cluster.Config.Labels, userProvidedLabels, diags)
	}
	data.Labels = tfLabels

	if resp.Cluster.Type != env_pb.ClusterType_K8S {
		return errors.Errorf("Unexpected non-Kubernetes runtime type: %s. Did the runtime change outside Terraform?", resp.Cluster.Type.String())
	}

	found, runtimeId, err := getDeploymentRuntimeId(ctx, clientSet, data)
	if err != nil {
		return errors.Wrapf(err, "Unable to read agent deployment for %s", data.Name.ValueString())
	}
	if found {
		data.AgentRuntimeId = types.StringValue(runtimeId)
	} else {
		data.AgentRuntimeId = types.StringNull()
	}
	return nil
}

func (r *ManagedK8sRuntimeResource) refresh(ctx context.Context, diags diag.Diagnostics, clientset *kubernetes.Clientset, data *ManagedK8sRuntimeResourceModel) error {
	return readManagedK8sRuntimeData(ctx, diags, r.client, clientset, data)
}

func deleteKubernetesObjects(ctx context.Context, namespace string, clientSet *kubernetes.Clientset) error {
	tflog.Trace(ctx, "Deleting agent k8s objects")
	err := clientSet.AppsV1().Deployments(namespace).Delete(ctx, agentDeploymentName, metav1.DeleteOptions{})
	if err != nil && !k8s_errors.IsNotFound(err) {
		return errors.Wrapf(err, "Failed to delete agent deployment")
	}
	err = clientSet.RbacV1().ClusterRoleBindings().Delete(ctx, clusterRoleBindingName, metav1.DeleteOptions{})
	if err != nil && !k8s_errors.IsNotFound(err) {
		return errors.Wrapf(err, "Failed to delete agent cluster role binding")
	}
	err = clientSet.CoreV1().ServiceAccounts(namespace).Delete(ctx, serviceAccountName, metav1.DeleteOptions{})
	if err != nil && !k8s_errors.IsNotFound(err) {
		return errors.Wrapf(err, "Failed to delete agent service account")
	}
	deletePropagation := metav1.DeletePropagationForeground
	graceSeconds := int64(0)
	err = clientSet.CoreV1().Namespaces().Delete(ctx, namespace, metav1.DeleteOptions{
		GracePeriodSeconds: &graceSeconds,
		PropagationPolicy:  &deletePropagation,
	})
	if err != nil && !k8s_errors.IsNotFound(err) {
		return errors.Wrapf(err, "Failed to delete agent namespace")
	}
	// wait for namespace to be fully deleted
	for {
		_, err := clientSet.CoreV1().Namespaces().Get(ctx, namespace, metav1.GetOptions{})
		if err != nil && k8s_errors.IsNotFound(err) {
			break
		}
	}

	return nil
}

func (r *ManagedK8sRuntimeResource) createOrUpdate(ctx context.Context, diags diag.Diagnostics, planData, stateData *ManagedK8sRuntimeResourceModel) error {
	var req *env_pb.LinkClusterReq = &env_pb.LinkClusterReq{
		Name:   planData.Name.ValueString(),
		Source: version.Source_IAC,
	}

	agentEnvValue, valueDiags := planData.AgentEnv.ToMapValue(ctx)
	diags.Append(valueDiags...)
	if diags.HasError() {
		return errors.Errorf("Failed to convert agent_env to map: %v", diags.Errors())
	}

	var agentEnv map[string]string = nil
	if !agentEnvValue.IsNull() {
		unpackEnv := map[string]string{}
		valueDiags = agentEnvValue.ElementsAs(ctx, &unpackEnv, false)
		diags.Append(valueDiags...)
		if diags.HasError() {
			return errors.Errorf("Failed to convert agent_env to map: %v", diags.Errors())
		}
		agentEnv = unpackEnv
	}

	req.Type = env_pb.ClusterType_K8S
	req.Auth = &env_pb.ClusterAuth{
		AuthOneof: &env_pb.ClusterAuth_K8S{
			K8S: &env_pb.ClusterAuth_K8SAuth{
				AgentExternallyManaged: true,
				AgentEnv:               agentEnv,
			},
		},
	}
	linkResp, err := r.client.LinkCluster(ctx, req)
	if err != nil {
		return err
	}
	planData.Id = types.StringValue(linkResp.ClusterId)

	clientSet, err := r.clientSet(ctx, diags, planData)
	if err != nil {
		return err
	}

	// TODO(mike): support supplying custom namespace name
	namespace := defaultAgentNamespace
	planData.AgentNamespace = types.StringValue(namespace)

	create := stateData == nil
	if create {
		found, runtimeId, err := getDeploymentRuntimeId(ctx, clientSet, planData)
		if err != nil {
			return errors.Wrap(err, "unable to verify kubernetes agent state of new cluster")
		}
		if found && runtimeId != linkResp.ClusterId {
			return errors.Errorf("found existing agent deployment in cluster with a different runtime id: %s", runtimeId)
		}
	} else {
		// the only changes we must handle are labels, or the agent_env attribute as
		// this needs to be passed on to the apiserver so it can update the agent
		// properly, and then here we should recreate the deployment with the new env vars
		// Why recreate here instead of letting apiserver handle it in its own update loop?
		// The env may contain proxy information, and if the proxy is changed, the agent
		// may no longer be able to talk with apiserver and so cannot be updated FROM apiserver.

		if agentEnvValue.Equal(stateData.AgentEnv) && planData.Labels.Equal(stateData.Labels) {
			// nothing to do
			return nil
		}
		tflog.Trace(ctx, "agent_env changed, must recreate the agent deployment")
		err = deleteKubernetesObjects(ctx, namespace, clientSet)
		if err != nil {
			return err
		}
	}

	// namespace
	namespaceSpec := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}
	_, err = clientSet.CoreV1().Namespaces().Create(ctx, namespaceSpec, metav1.CreateOptions{})
	if err != nil && !k8s_errors.IsAlreadyExists(err) {
		return errors.Wrapf(err, "Failed to create agent namespace")
	}

	// service account
	saSpec := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      serviceAccountName,
			Namespace: namespaceSpec.Name,
		},
	}
	_, err = clientSet.CoreV1().ServiceAccounts(saSpec.Namespace).Create(ctx, saSpec, metav1.CreateOptions{})
	if err != nil && !k8s_errors.IsAlreadyExists(err) {
		return errors.Wrapf(err, "Failed to create agent service account")
	}

	// service account cluster role binding
	roleBindingSpec := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: clusterRoleBindingName,
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
		return errors.Wrapf(err, "Failed to create agent cluster role binding")
	}

	env := []corev1.EnvVar{
		{
			Name:  "PVN_NAMESPACE",
			Value: namespaceSpec.Name,
		},
		{
			Name:  "PVN_RELEASE_CHANNEL",
			Value: "prodvana",
		},
	}
	for k, v := range agentEnv {
		env = append(env, corev1.EnvVar{
			Name:  k,
			Value: v,
		})
	}

	// deployment
	var replicas int32 = 1
	deploymentSpec := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      agentDeploymentName,
			Namespace: namespaceSpec.Name,
			Annotations: map[string]string{
				agentRuntimeIdAnnotation: linkResp.ClusterId,
			},
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": agentDeploymentName,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app":                 agentDeploymentName,
						"prodvana.io/service": agentDeploymentName,
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: saSpec.Name,
					Containers: []corev1.Container{
						{
							Name:            "default",
							Args:            linkResp.K8SAgentArgs,
							Env:             env,
							Image:           linkResp.K8SAgentImage,
							ImagePullPolicy: corev1.PullAlways,
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 5100,
									Protocol:      corev1.ProtocolTCP,
								},
							},
						},
					},
				},
			},
		},
	}

	tflog.Info(ctx, fmt.Sprintf("Creating new deployment: %#v", deploymentSpec))
	_, err = clientSet.AppsV1().Deployments(namespaceSpec.Name).Create(ctx, deploymentSpec, metav1.CreateOptions{})
	if err != nil && !k8s_errors.IsAlreadyExists(err) {
		return errors.Wrapf(err, "Failed to create agent deployment")
	}

	err = WaitForClusterWithTimeout(ctx, r.client, linkResp.ClusterId, planData.Name.ValueString(), planData.Timeout.ValueString())
	if err != nil && !k8s_errors.IsAlreadyExists(err) {
		return errors.Wrapf(err, "Runtime linking failed")
	}

	getResp, err := r.client.GetCluster(ctx, &env_pb.GetClusterReq{
		Runtime: planData.Name.ValueString(),
	})
	if err != nil {
		return err
	}

	config := getResp.Cluster.Config
	labelProtos := labels.LabelDefinitionProtosFromTerraformList(ctx, planData.Labels, diags)
	if diags.HasError() {
		return errors.Errorf("Failed to convert labels: %v", diags.Errors())
	}
	config.Labels = labelProtos

	_, err = r.client.ConfigureCluster(ctx, &env_pb.ConfigureClusterReq{
		RuntimeName: planData.Name.ValueString(),
		Config:      config,
	})
	if err != nil {
		return err
	}

	return r.refresh(ctx, diags, clientSet, planData)
}

func (r *ManagedK8sRuntimeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *ManagedK8sRuntimeResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Trace(ctx, "creating runtime resource")
	err := r.createOrUpdate(ctx, resp.Diagnostics, data, nil)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create runtime, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "created runtime resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ManagedK8sRuntimeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *ManagedK8sRuntimeResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	clientSet, err := r.clientSet(ctx, resp.Diagnostics, data)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create client set, got error: %s", err))
		return
	}
	err = r.refresh(ctx, resp.Diagnostics, clientSet, data)
	if err != nil {
		// if the runtime does not exist, remove the resource
		if status.Code(err) == codes.NotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read runtime state for %s, got error: %s", data.Name.ValueString(), err))
		return
	}
	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ManagedK8sRuntimeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData *ManagedK8sRuntimeResourceModel
	var stateData *ManagedK8sRuntimeResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)

	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Trace(ctx, "updating runtime resource")
	err := r.createOrUpdate(ctx, resp.Diagnostics, planData, stateData)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update runtime, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "updated runtime resource")

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &planData)...)
}

func (r *ManagedK8sRuntimeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *ManagedK8sRuntimeResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
	_, err := r.client.RemoveCluster(ctx, &env_pb.RemoveClusterReq{
		Name: data.Name.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete runtime, got error: %s", err))
		return
	}
	clientSet, err := r.clientSet(ctx, resp.Diagnostics, data)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to kubernetes create client set, got error: %s", err))
		return
	}
	err = deleteKubernetesObjects(ctx, data.AgentNamespace.ValueString(), clientSet)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete runtime, got error: %s", err))
		return
	}
	tflog.Trace(ctx, "deleted runtime resource")
}
