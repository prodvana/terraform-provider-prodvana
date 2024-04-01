package provider

import (
	"context"
	"fmt"

	workflow_pb "github.com/prodvana/prodvana-public/go/prodvana-sdk/proto/prodvana/workflow"
	"github.com/prodvana/terraform-provider-prodvana/internal/provider/validators"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ContainerRegistryResource{}

func NewContainerRegistryResource() resource.Resource {
	return &ContainerRegistryResource{}
}

// ContainerRegistryResource defines the resource implementation.
type ContainerRegistryResource struct {
	client workflow_pb.WorkflowManagerClient
}

// ContainerRegistryResouceModel describes the resource link data model.
type ContainerRegistryResourceModel struct {
	Name     types.String `tfsdk:"name"`
	Id       types.String `tfsdk:"id"`
	URL      types.String `tfsdk:"url"`
	Username types.String `tfsdk:"username"`
	Password types.String `tfsdk:"password"`
	Public   types.Bool   `tfsdk:"public"`
}

func (r *ContainerRegistryResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_container_registry"
}

func (r *ContainerRegistryResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "This resource allows you to link a [container registry](https://docs.prodvana.io/docs/container-image-registries) to Prodvana.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Name for the container registry, used to reference it in Prodvana configuration.",
				Required:            true,
				Validators:          validators.DefaultNameValidators(),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Container Registry Identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"url": schema.StringAttribute{
				MarkdownDescription: "URL pointing to the container registry.",
				Required:            true,
				Validators: []validator.String{
					validators.URLHasHTTPProtocolValidator(),
				},
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "Username to authenticate with the container registry.",
				Optional:            true,
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "Password to authenticate with the container registry.",
				Sensitive:           true,
				Optional:            true,
			},
			"public": schema.BoolAttribute{
				MarkdownDescription: "Whether the container registry is public (no authentication required) or not.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
				Validators: []validator.Bool{
					validators.ExclusiveBool(
						path.MatchRelative().AtName("username"),
						path.MatchRelative().AtName("password"),
					),
				},
			},
		},
	}
}

func (r *ContainerRegistryResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	r.client = workflow_pb.NewWorkflowManagerClient(conn)
}

func (r *ContainerRegistryResource) refresh(ctx context.Context, diags diag.Diagnostics, data *ContainerRegistryResourceModel) error {
	resp, err := r.client.GetContainerRegistryIntegration(ctx, &workflow_pb.GetContainerRegistryIntegrationReq{
		RegistryName: data.Name.ValueString(),
	})
	if err != nil {
		return errors.Wrapf(err, "Unable to read container registry state for %s", data.Name.ValueString())
	}

	data.Id = types.StringValue(resp.Registry.IntegrationId)
	data.URL = types.StringValue(resp.Registry.Url)

	return nil
}

func (r *ContainerRegistryResource) createOrUpdate(ctx context.Context, planData *ContainerRegistryResourceModel) error {
	createReq := &workflow_pb.CreateContainerRegistryIntegrationReq{
		Name:     planData.Name.ValueString(),
		Url:      planData.URL.ValueString(),
		Username: planData.Username.ValueString(),
		Secret:   planData.Password.ValueString(),
		Type:     workflow_pb.RegistryType_DOCKER_REGISTRY,
	}

	if planData.Public.ValueBool() {
		createReq.RegistryOptions = &workflow_pb.CreateContainerRegistryIntegrationReq_PublicRegistryOptions_{
			PublicRegistryOptions: &workflow_pb.CreateContainerRegistryIntegrationReq_PublicRegistryOptions{},
		}
	}

	createResp, err := r.client.CreateContainerRegistryIntegration(ctx, createReq)
	if err != nil {
		return err
	}
	planData.Id = types.StringValue(createResp.IntegrationId)

	return nil
}

func (r *ContainerRegistryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *ContainerRegistryResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.createOrUpdate(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create container registry, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "created container registry resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ContainerRegistryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *ContainerRegistryResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.refresh(ctx, resp.Diagnostics, data)
	if err != nil {
		// if registry doesn't exist anymore, remove the resource
		if status.Code(err) == codes.NotFound {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read container registry state for %s, got error: %s", data.Name.ValueString(), err))
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ContainerRegistryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData *ContainerRegistryResourceModel
	var stateData *ContainerRegistryResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.createOrUpdate(ctx, planData)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update container registry, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "updated container registry resource")

	resp.Diagnostics.Append(resp.State.Set(ctx, &planData)...)
}

func (r *ContainerRegistryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *ContainerRegistryResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
	_, err := r.client.DeleteContainerRegistryIntegration(ctx, &workflow_pb.DeleteContainerRegistryIntegrationReq{
		RegistryName: data.Name.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete container registry, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "deleted container registry resource")
}
