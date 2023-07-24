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
var _ resource.Resource = &EcsRuntimeResource{}
var _ resource.ResourceWithImportState = &EcsRuntimeResource{}

func NewEcsRuntimeResource() resource.Resource {
	return &EcsRuntimeResource{}
}

// EcsRuntimeResource defines the resource implementation.
type EcsRuntimeResource struct {
	client env_pb.EnvironmentManagerClient
}

// RuntimeK8sResouceModel describes the resource data model.
type EcsRuntimeResourceModel struct {
	Name types.String `tfsdk:"name"`
	Id   types.String `tfsdk:"id"`

	AccessKey     types.String `tfsdk:"access_key"`
	SecretKey     types.String `tfsdk:"secret_key"`
	Region        types.String `tfsdk:"region"`
	AssumeRoleArn types.String `tfsdk:"assume_role_arn"`
	ClusterArn    types.String `tfsdk:"cluster_arn"`
}

func (r *EcsRuntimeResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ecs_runtime"
}

func (r *EcsRuntimeResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "(Alpha! This feature is still in progress.) This resource allows you to manage a Prodvana ECS [Runtime](https://docs.prodvana.io/docs/prodvana-concepts#runtime).",
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
			"access_key": schema.StringAttribute{
				MarkdownDescription: "AWS Access Key ID with permissions to the ECS cluster",
				Required:            true,
			},
			"secret_key": schema.StringAttribute{
				MarkdownDescription: "AWS Secret Key with permissions to the ECS cluster",
				Required:            true,
			},
			"region": schema.StringAttribute{
				MarkdownDescription: "AWS Region of the ECS cluster",
				Required:            true,
			},
			"assume_role_arn": schema.StringAttribute{
				MarkdownDescription: "AWS role to assume when accessing the ECS cluster",
				Optional:            true,
			},
			"cluster_arn": schema.StringAttribute{
				MarkdownDescription: "ARN of the ECS cluster",
				Required:            true,
			},
		},
	}
}

func (r *EcsRuntimeResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func readEcsRuntimeData(ctx context.Context, client env_pb.EnvironmentManagerClient, data *EcsRuntimeResourceModel) error {
	resp, err := client.GetCluster(ctx, &env_pb.GetClusterReq{
		Runtime:     data.Name.ValueString(),
		IncludeAuth: true,
	})
	if err != nil {
		return errors.Wrapf(err, "Unable to read runtime state for %s", data.Name.ValueString())
	}

	data.Name = types.StringValue(resp.Cluster.Name)
	data.Id = types.StringValue(resp.Cluster.Id)

	if resp.Cluster.Type != env_pb.ClusterType_ECS {
		return errors.Errorf("Unexpected non-ECS runtime type: %s. Did the runtime change outside Terraform?", resp.Cluster.Type.String())
	}
	return nil
}

func (r *EcsRuntimeResource) refresh(ctx context.Context, data *EcsRuntimeResourceModel) error {
	return readEcsRuntimeData(ctx, r.client, data)
}

func (r *EcsRuntimeResource) createOrUpdate(ctx context.Context, planData *EcsRuntimeResourceModel) error {
	ecsAuth := &env_pb.ClusterAuth_ECSAuth{
		AccessKey:  planData.AccessKey.ValueString(),
		SecretKey:  planData.SecretKey.ValueString(),
		Region:     planData.Region.ValueString(),
		ClusterArn: planData.ClusterArn.ValueString(),
	}
	if !planData.AssumeRoleArn.IsNull() {
		ecsAuth.AssumeRoleArn = planData.AssumeRoleArn.ValueString()
	}

	_, err := r.client.LinkCluster(ctx, &env_pb.LinkClusterReq{
		Name: planData.Name.ValueString(),
		Type: env_pb.ClusterType_ECS,
		Auth: &env_pb.ClusterAuth{
			AuthOneof: &env_pb.ClusterAuth_Ecs{
				Ecs: ecsAuth,
			},
		},
		Source: version.Source_IAC,
	})
	if err != nil {
		return err
	}

	return r.refresh(ctx, planData)
}

func (r *EcsRuntimeResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *EcsRuntimeResourceModel

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

func (r *EcsRuntimeResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *EcsRuntimeResourceModel

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

func (r *EcsRuntimeResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData *EcsRuntimeResourceModel
	var stateData *EcsRuntimeResourceModel

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

func (r *EcsRuntimeResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *EcsRuntimeResourceModel

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

func (r *EcsRuntimeResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data EcsRuntimeResourceModel

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
