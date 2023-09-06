package provider

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	env_pb "github.com/prodvana/prodvana-public/go/prodvana-sdk/proto/prodvana/environment"
	"github.com/prodvana/prodvana-public/go/prodvana-sdk/proto/prodvana/version"
	"github.com/prodvana/terraform-provider-prodvana/internal/provider/validators"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &K8sRuntimeResource{}
var _ resource.ResourceWithImportState = &K8sRuntimeResource{}

func NewK8sRuntimeResource() resource.Resource {
	return &K8sRuntimeResource{}
}

// K8sRuntimeResource defines the resource implementation.
type K8sRuntimeResource struct {
	client env_pb.EnvironmentManagerClient
}

// K8sRuntimeResouceModel describes the resource data model.
type K8sRuntimeResourceModel struct {
	Name types.String `tfsdk:"name"`
	Id   types.String `tfsdk:"id"`

	AgentApiToken types.String `tfsdk:"agent_api_token"`

	AgentURL   types.String `tfsdk:"agent_url"`
	AgentImage types.String `tfsdk:"agent_image"`
	AgentArgs  types.List   `tfsdk:"agent_args"`
}

func (r *K8sRuntimeResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_k8s_runtime"
}

func (r *K8sRuntimeResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {

	resp.Schema = schema.Schema{
		MarkdownDescription: "This resource allows you to manage a Prodvana Kubernetes [Runtime](https://docs.prodvana.io/docs/prodvana-concepts#runtime). You are responsible for managing the agent lifetime. Also see `prodvana_managed_k8s_runtime`.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Runtime name",
				Required:            true,
				Validators:          validators.DefaultNameValidators(),
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Runtime identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"agent_api_token": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "API Token used for linking the Kubernetes Prodvana agent",
				Sensitive:           true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"agent_url": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "URL of the Kubernetes Prodvana agent server",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"agent_image": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "URL of the Kubernetes Prodvana agent container image.",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"agent_args": schema.ListAttribute{
				Computed:            true,
				Sensitive:           true,
				MarkdownDescription: "Arguments to pass to the Kubernetes Prodvana agent container.",
				ElementType:         types.StringType,
			},
		},
	}
}

func (r *K8sRuntimeResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func readK8sRuntimeData(ctx context.Context, diags diag.Diagnostics, client env_pb.EnvironmentManagerClient, data *K8sRuntimeResourceModel) error {
	linkResp, err := client.LinkCluster(ctx, &env_pb.LinkClusterReq{
		Name: data.Name.ValueString(),
		Type: env_pb.ClusterType_K8S,
		Auth: &env_pb.ClusterAuth{
			K8SAgentAuth: true,
			AuthOneof: &env_pb.ClusterAuth_K8S{
				K8S: &env_pb.ClusterAuth_K8SAuth{
					AgentExternallyManaged: true,
				},
			},
		},
		Source: version.Source_IAC,
	})
	if err != nil {
		return err
	}
	data.Id = types.StringValue(linkResp.ClusterId)
	data.AgentApiToken = types.StringValue(linkResp.K8SAgentApiToken)
	data.AgentURL = types.StringValue(linkResp.K8SAgentUrl)
	data.AgentImage = types.StringValue(linkResp.K8SAgentImage)
	args, valDiags := types.ListValueFrom(ctx, types.StringType, linkResp.K8SAgentArgs)
	if valDiags.HasError() {
		return errors.Errorf("Failed to convert agent args: %v", valDiags.Errors())
	}
	data.AgentArgs = args

	return nil
}

func (r *K8sRuntimeResource) refresh(ctx context.Context, diags diag.Diagnostics, data *K8sRuntimeResourceModel) error {
	return readK8sRuntimeData(ctx, diags, r.client, data)
}

func (r *K8sRuntimeResource) createOrUpdate(ctx context.Context, diags diag.Diagnostics, planData *K8sRuntimeResourceModel) error {
	// linkResp, err := r.client.LinkCluster(ctx, &env_pb.LinkClusterReq{
	// 	Name: planData.Name.ValueString(),
	// 	Type: env_pb.ClusterType_K8S,
	// 	Auth: &env_pb.ClusterAuth{
	// 		K8SAgentAuth: true,
	// 		AuthOneof: &env_pb.ClusterAuth_K8S{
	// 			K8S: &env_pb.ClusterAuth_K8SAuth{
	// 				AgentExternallyManaged: true,
	// 			},
	// 		},
	// 	},
	// 	Source: version.Source_IAC,
	// })
	// if err != nil {
	// 	return err
	// }
	// planData.AgentApiToken = types.StringValue(linkResp.K8SAgentApiToken)
	// planData.AgentURL = types.StringValue(linkResp.K8SAgentUrl)
	// planData.AgentImage = types.StringValue(linkResp.K8SAgentImage)
	// args, valDiags := types.ListValueFrom(ctx, types.StringType, linkResp.K8SAgentArgs)
	// if valDiags.HasError() {
	// 	return errors.Errorf("Failed to convert agent args: %v", valDiags.Errors())
	// }
	// planData.AgentArgs = args

	return r.refresh(ctx, diags, planData)
}

func (r *K8sRuntimeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *K8sRuntimeResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.createOrUpdate(ctx, resp.Diagnostics, data)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create runtime, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "created runtime resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *K8sRuntimeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *K8sRuntimeResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.refresh(ctx, resp.Diagnostics, data)
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

func (r *K8sRuntimeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData *K8sRuntimeResourceModel
	var stateData *K8sRuntimeResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.createOrUpdate(ctx, resp.Diagnostics, planData)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update runtime, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "updated runtime resource")

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &planData)...)
}

func (r *K8sRuntimeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *K8sRuntimeResourceModel

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

func (r *K8sRuntimeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data K8sRuntimeResourceModel

	resource.ImportStatePassthroughID(ctx, path.Root("name"), req, resp)
	if resp.Diagnostics.HasError() {
		return
	}

	data.Name = types.StringValue(req.ID)
	err := r.refresh(ctx, resp.Diagnostics, &data)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to import runtime state for %s, got error: %s", data.Name.ValueString(), err))
		return
	}

	// Save imported data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
