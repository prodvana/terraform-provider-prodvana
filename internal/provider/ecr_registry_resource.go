package provider

import (
	"context"
	"fmt"

	workflow_pb "github.com/prodvana/prodvana-public/go/prodvana-sdk/proto/prodvana/workflow"
	"github.com/prodvana/terraform-provider-prodvana/internal/provider/validators"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ECRRegistryResource{}

func NewECRRegistryResource() resource.Resource {
	return &ECRRegistryResource{}
}

// ECRRegistryResource defines the resource implementation.
type ECRRegistryResource struct {
	client workflow_pb.WorkflowManagerClient
}

// ContainerRegistryResouceModel describes the resource link data model.
type ECRRegistryResourceModel struct {
	Name   types.String `tfsdk:"name"`
	Id     types.String `tfsdk:"id"`
	Region types.String `tfsdk:"region"`

	// this is encapsulated in its own nested object because we will
	// support other authentication methods in the future
	CredentialsAuth *CredentialAuthModel `tfsdk:"credentials_auth"`
}

type CredentialAuthModel struct {
	AccessKeyID     types.String `tfsdk:"access_key_id"`
	SecretAccessKey types.String `tfsdk:"secret_access_key"`
}

func (r *ECRRegistryResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ecr_registry"
}

func (r *ECRRegistryResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "This resource allows you to link an [ECR registry](https://docs.prodvana.io/docs/ecr) to Prodvana.",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Name for the ECR registry, used to reference it in Prodvana configuration.",
				Required:            true,
				Validators:          validators.DefaultNameValidators(),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "ECR Registry Identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"region": schema.StringAttribute{
				MarkdownDescription: "AWS region where the ECR registry is located.",
				Required:            true,
			},
			"credentials_auth": schema.SingleNestedAttribute{
				MarkdownDescription: "Credentials to authenticate with the ECR registry.",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"access_key_id": schema.StringAttribute{
						MarkdownDescription: "AWS Access Key ID with permissions to the ECR registry",
						Required:            true,
					},
					"secret_access_key": schema.StringAttribute{
						MarkdownDescription: "AWS Secret Access Key with permissions to the ECR registry",
						Required:            true,
						Sensitive:           true,
					},
				},
			},
		},
	}
}

func (r *ECRRegistryResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *ECRRegistryResource) refresh(ctx context.Context, diags diag.Diagnostics, data *ECRRegistryResourceModel) error {
	resp, err := r.client.GetContainerRegistryIntegration(ctx, &workflow_pb.GetContainerRegistryIntegrationReq{
		RegistryName: data.Name.ValueString(),
	})
	if err != nil {
		return errors.Wrapf(err, "Unable to read ecr registry state for %s", data.Name.ValueString())
	}
	if resp.Registry.Type != workflow_pb.RegistryType_ECR.String() {
		return fmt.Errorf("registry %s is not an ECR registry, registry of type %s found", data.Name.ValueString(), resp.Registry.Type)
	}

	data.Id = types.StringValue(resp.Registry.IntegrationId)
	if resp.Registry.GetEcrInfo() != nil {
		data.Region = types.StringValue(resp.Registry.GetEcrInfo().Region)
	}

	return nil
}

func (r *ECRRegistryResource) createOrUpdate(ctx context.Context, planData *ECRRegistryResourceModel) error {
	createReq := &workflow_pb.CreateContainerRegistryIntegrationReq{
		Name: planData.Name.ValueString(),
		Type: workflow_pb.RegistryType_ECR,
		RegistryOptions: &workflow_pb.CreateContainerRegistryIntegrationReq_EcrOptions{
			EcrOptions: &workflow_pb.CreateContainerRegistryIntegrationReq_ECROptions{
				Region:    planData.Region.ValueString(),
				AccessKey: planData.CredentialsAuth.AccessKeyID.ValueString(),
				SecretKey: planData.CredentialsAuth.SecretAccessKey.ValueString(),
			},
		},
	}

	createResp, err := r.client.CreateContainerRegistryIntegration(ctx, createReq)
	if err != nil {
		return err
	}
	planData.Id = types.StringValue(createResp.IntegrationId)

	return nil
}

func (r *ECRRegistryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *ECRRegistryResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.createOrUpdate(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create ecr registry, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "created ecr registry resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ECRRegistryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *ECRRegistryResourceModel

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

		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read ecr registry state for %s, got error: %s", data.Name.ValueString(), err))
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ECRRegistryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData *ECRRegistryResourceModel
	var stateData *ECRRegistryResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.createOrUpdate(ctx, planData)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update ecr registry, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "updated ecr registry resource")

	resp.Diagnostics.Append(resp.State.Set(ctx, &planData)...)
}

func (r *ECRRegistryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *ECRRegistryResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
	_, err := r.client.DeleteContainerRegistryIntegration(ctx, &workflow_pb.DeleteContainerRegistryIntegrationReq{
		RegistryName: data.Name.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete ecr registry, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "deleted ecr registry resource")
}
