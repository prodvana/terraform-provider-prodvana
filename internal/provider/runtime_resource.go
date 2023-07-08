package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	env_pb "github.com/prodvana/prodvana-public/go/prodvana-sdk/proto/prodvana/environment"
	"github.com/prodvana/prodvana-public/go/prodvana-sdk/proto/prodvana/version"
	"github.com/prodvana/terraform-provider-prodvana/internal/provider/validators"
	"google.golang.org/grpc"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &RuntimeResource{}
var _ resource.ResourceWithImportState = &RuntimeResource{}

func NewRuntimeResource() resource.Resource {
	return &RuntimeResource{}
}

// RuntimeResource defines the resource implementation.
type RuntimeResource struct {
	client env_pb.EnvironmentManagerClient
}

// RuntimeResouceModel describes the resource data model.
type RuntimeResourceModel struct {
	Name types.String `tfsdk:"name"`
	Id   types.String `tfsdk:"id"`
	Type types.String `tfsdk:"type"`

	K8S *k8sRuntimeOptions `tfsdk:"k8s"`
	// ECS *ecsRuntimeOptions `tfsdk:"ecs"`
}

type k8sRuntimeOptions struct {
	AgentEnv               map[string]string `tfsdk:"agent_env"`
	AgentExternallyManaged types.Bool        `tfsdk:"agent_externally_managed"`
	ApiToken               types.String      `tfsdk:"api_token"`
}

// type ecsRuntimeOptions struct {
// 	AccessKey     types.String `tfsdk:"access_key"`
// 	SecretKey     types.String `tfsdk:"secret_key"`
// 	Region        types.String `tfsdk:"region"`
// 	AssumeRoleArn types.String `tfsdk:"assume_role_arn"`
// 	ClusterArn    types.String `tfsdk:"cluster_arn"`
// }

func (r *RuntimeResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_runtime"
}

func (r *RuntimeResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	// TODO(mike): add support for other cluster types as needed
	clusterTypes := []string{
		// env_pb.ClusterType_ECS.String(),
		env_pb.ClusterType_K8S.String(),
	}

	resp.Schema = schema.Schema{
		MarkdownDescription: "(Alpha! This feature is still in progress.) This resource allows you to manage a Prodvana [Runtime](https://docs.prodvana.io/docs/prodvana-concepts#runtime).",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Runtime name",
				Required:            true,
				Validators:          validators.DefaultNameValidators(),
			},
			"type": schema.StringAttribute{
				MarkdownDescription: fmt.Sprintf("Type of the runtime, one of (%s)", strings.Join(clusterTypes, ", ")),
				Required:            true,
				Validators: []validator.String{
					stringvalidator.OneOf(clusterTypes...),
				},
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Runtime identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"k8s": schema.SingleNestedAttribute{
				MarkdownDescription: "K8S Runtime Configuration Options. These are only valid when `type` is set to `K8S`",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"agent_env": schema.MapAttribute{
						ElementType:         types.StringType,
						MarkdownDescription: "Environment variables to pass to the agent configuration. Useful for things like proxy configuration. Only useful when `agent_externally_managed` is false.",
						Optional:            true,
					},
					"agent_externally_managed": schema.BoolAttribute{
						MarkdownDescription: "Whether the agent lifecycle is handled externally by the runtime owner. When true, Prodvana will not update the agent. Default false.",
						Optional:            true,
						Computed:            true,
						Default:             booldefault.StaticBool(false),
					},
					"api_token": schema.StringAttribute{
						Optional:            true,
						Computed:            true,
						MarkdownDescription: "API Token used for linking the Kubernetes Prodvana agent",
						Sensitive:           true,
						PlanModifiers: []planmodifier.String{
							stringplanmodifier.UseStateForUnknown(),
						},
					},
				},
				Validators: []validator.Object{
					// objectvalidator.ConflictsWith(path.Expressions{
					// 	path.MatchRoot("ecs"),
					// }...),
					validators.CheckAttributeAtPath(path.MatchRoot("type"), types.StringValue(env_pb.ClusterType_K8S.String())),
				},
			},
			// TODO(mike): enable support for ECS -- need a good testing story
			// "ecs": schema.SingleNestedAttribute{
			// 	MarkdownDescription: "ECS Runtime Configuration Options. These are only valid when `type` is set to `ECS`",
			// 	Optional:            true,
			// 	Attributes: map[string]schema.Attribute{
			// 		"access_key": schema.StringAttribute{
			// 			MarkdownDescription: "AWS Access Key ID with permissions to the ECS cluster",
			// 			Required:            true,
			// 		},
			// 		"secret_key": schema.StringAttribute{
			// 			MarkdownDescription: "AWS Secret Key with permissions to the ECS cluster",
			// 			Required:            true,
			// 		},
			// 		"region": schema.StringAttribute{
			// 			MarkdownDescription: "AWS Region of the ECS cluster",
			// 			Required:            true,
			// 		},
			// 		"assume_role_arn": schema.StringAttribute{
			// 			MarkdownDescription: "AWS role to assume when accessing the ECS cluster",
			// 			Optional:            true,
			// 		},
			// 		"cluster_arn": schema.StringAttribute{
			// 			MarkdownDescription: "ARN of the ECS cluster",
			// 			Required:            true,
			// 		},
			// 	},
			// 	Validators: []validator.Object{
			// 		objectvalidator.ConflictsWith(path.Expressions{
			// 			path.MatchRoot("k8s"),
			// 		}...),
			// 		validators.CheckAttributeAtPath(path.MatchRoot("type"), types.StringValue(env_pb.ClusterType_ECS.String())),
			// 	},
			// },
		},
	}
}

func (r *RuntimeResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func readRuntimeData(ctx context.Context, client env_pb.EnvironmentManagerClient, data *RuntimeResourceModel) error {
	resp, err := client.GetCluster(ctx, &env_pb.GetClusterReq{
		Runtime:     data.Name.ValueString(),
		IncludeAuth: true,
	})
	if err != nil {
		return errors.Wrapf(err, "Unable to read runtime state for %s", data.Name.ValueString())
	}

	data.Name = types.StringValue(resp.Cluster.Name)
	data.Id = types.StringValue(resp.Cluster.Id)
	data.Type = types.StringValue(resp.Cluster.Type.String())

	if resp.Cluster.Type == env_pb.ClusterType_K8S {
		tokenResp, err := client.GetClusterAgentApiToken(ctx, &env_pb.GetClusterAgentApiTokenReq{
			Name: data.Name.ValueString(),
		})
		if err != nil {
			return errors.Wrapf(err, "Unable to read runtime state for %s", data.Name.ValueString())
		}

		if data.K8S == nil {
			data.K8S = &k8sRuntimeOptions{}
		}

		data.K8S.ApiToken = types.StringValue(tokenResp.ApiToken)
		if resp.Cluster.Auth != nil && resp.Cluster.Auth.GetK8S() != nil {
			k8sAuth := resp.Cluster.Auth.GetK8S()

			data.K8S.AgentEnv = k8sAuth.AgentEnv
			data.K8S.AgentExternallyManaged = types.BoolValue(k8sAuth.AgentExternallyManaged)
		} else {
			data.K8S.AgentExternallyManaged = types.BoolValue(false)
		}
	}
	return nil
}

func (r *RuntimeResource) refresh(ctx context.Context, data *RuntimeResourceModel) error {
	return readRuntimeData(ctx, r.client, data)
}

func (r *RuntimeResource) createOrUpdate(ctx context.Context, planData *RuntimeResourceModel) error {
	var req *env_pb.LinkClusterReq = &env_pb.LinkClusterReq{
		Name: planData.Name.ValueString(),

		Source: version.Source_IAC,
	}

	switch planData.Type.ValueString() {
	case env_pb.ClusterType_K8S.String():
		req.Type = env_pb.ClusterType_K8S
		req.Auth = &env_pb.ClusterAuth{
			K8SAgentAuth: true,
			AuthOneof: &env_pb.ClusterAuth_K8S{
				K8S: &env_pb.ClusterAuth_K8SAuth{
					AgentEnv:               planData.K8S.AgentEnv,
					AgentExternallyManaged: planData.K8S.AgentExternallyManaged.ValueBool(),
				},
			},
		}

	// case env_pb.ClusterType_ECS.String():
	// 	req.Type = env_pb.ClusterType_ECS
	// 	req.Auth = &env_pb.ClusterAuth{
	// 		AuthOneof: &env_pb.ClusterAuth_Ecs{
	// 			Ecs: &env_pb.ClusterAuth_ECSAuth{
	// 				AccessKey:     planData.ECS.AccessKey.ValueString(),
	// 				SecretKey:     planData.ECS.SecretKey.ValueString(),
	// 				Region:        planData.ECS.Region.ValueString(),
	// 				AssumeRoleArn: planData.ECS.AssumeRoleArn.ValueString(),
	// 				ClusterArn:    planData.ECS.ClusterArn.ValueString(),
	// 			},
	// 		},
	// 	}
	default:
		return errors.Errorf("Invalid runtime type %s", planData.Type.ValueString())
	}

	_, err := r.client.LinkCluster(ctx, req)
	if err != nil {
		return err
	}

	return r.refresh(ctx, planData)
}

func (r *RuntimeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *RuntimeResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.createOrUpdate(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create runtime, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "created runtime resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RuntimeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *RuntimeResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.refresh(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read runtime state for %s, got error: %s", data.Name.ValueString(), err))
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RuntimeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData *RuntimeResourceModel
	var stateData *RuntimeResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.createOrUpdate(ctx, planData)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update runtime, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "updated runtime resource")

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &planData)...)
}

func (r *RuntimeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *RuntimeResourceModel

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
	tflog.Trace(ctx, "deleted runtime resource")
}

func (r *RuntimeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data RuntimeResourceModel

	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
	if resp.Diagnostics.HasError() {
		return
	}

	data.Name = types.StringValue(req.ID)
	err := r.refresh(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to import runtime state for %s, got error: %s", data.Name.ValueString(), err))
		return
	}

	// Save imported data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
