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

func readK8sRuntimeData(ctx context.Context, client env_pb.EnvironmentManagerClient, data *K8sRuntimeResourceModel) error {
	resp, err := client.GetCluster(ctx, &env_pb.GetClusterReq{
		Runtime:     data.Name.ValueString(),
		IncludeAuth: true,
	})
	if err != nil {
		return errors.Wrapf(err, "Unable to read runtime state for %s", data.Name.ValueString())
	}

	data.Name = types.StringValue(resp.Cluster.Name)
	data.Id = types.StringValue(resp.Cluster.Id)

	if resp.Cluster.Type != env_pb.ClusterType_K8S {
		return errors.Errorf("Unexpected non-Kubernetes runtime type: %s. Did the runtime change outside Terraform?", resp.Cluster.Type.String())
	}

	tokenResp, err := client.GetClusterAgentApiToken(ctx, &env_pb.GetClusterAgentApiTokenReq{
		Name: data.Name.ValueString(),
	})
	if err != nil {
		return errors.Wrapf(err, "Unable to read runtime state for %s", data.Name.ValueString())
	}
	data.AgentApiToken = types.StringValue(tokenResp.ApiToken)

	return nil
}

func (r *K8sRuntimeResource) refresh(ctx context.Context, data *K8sRuntimeResourceModel) error {
	return readK8sRuntimeData(ctx, r.client, data)
}

func (r *K8sRuntimeResource) createOrUpdate(ctx context.Context, planData *K8sRuntimeResourceModel) error {
	_, err := r.client.LinkCluster(ctx, &env_pb.LinkClusterReq{
		Name: planData.Name.ValueString(),
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

	return r.refresh(ctx, planData)
}

func (r *K8sRuntimeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *K8sRuntimeResourceModel

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

func (r *K8sRuntimeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *K8sRuntimeResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.refresh(ctx, data)
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

	err := r.createOrUpdate(ctx, planData)
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
	err := r.refresh(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to import runtime state for %s, got error: %s", data.Name.ValueString(), err))
		return
	}

	// Save imported data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
