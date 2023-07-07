package provider

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	app_pb "github.com/prodvana/prodvana-public/go/prodvana-sdk/proto/prodvana/application"
	rc_pb "github.com/prodvana/prodvana-public/go/prodvana-sdk/proto/prodvana/release_channel"
	"github.com/prodvana/terraform-provider-prodvana/internal/provider/validators"
	"google.golang.org/grpc"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &ApplicationResource{}
var _ resource.ResourceWithImportState = &ApplicationResource{}

func NewApplicationResource() resource.Resource {
	return &ApplicationResource{}
}

// ApplicationResource defines the resource implementation.
type ApplicationResource struct {
	client   app_pb.ApplicationManagerClient
	rcClient rc_pb.ReleaseChannelManagerClient
}

// ApplicationResourceModel describes the resource data model.
type ApplicationResourceModel struct {
	Name    types.String `tfsdk:"name"`
	Id      types.String `tfsdk:"id"`
	Version types.String `tfsdk:"version"`
}

func (r *ApplicationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application"
}

func (r *ApplicationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// TODO(mike): expand description -- this feeds into the generated documentation that shows up in the registry
		MarkdownDescription: "This resource allows you to manage a Prodvana [Application](https://docs.prodvana.io/docs/prodvana-concepts#application).",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Application name",
				Required:            true,
				Validators:          validators.DefaultNameValidators(),
			},
			"version": schema.StringAttribute{
				MarkdownDescription: "Current application version",
				Computed:            true,
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Application identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *ApplicationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	r.client = app_pb.NewApplicationManagerClient(conn)
	r.rcClient = rc_pb.NewReleaseChannelManagerClient(conn)
}

func readApplicationData(ctx context.Context, client app_pb.ApplicationManagerClient, data *ApplicationResourceModel) error {
	getAppResp, err := client.GetApplication(ctx, &app_pb.GetApplicationReq{
		Application: data.Name.ValueString(),
	})

	if err != nil {
		return errors.Wrapf(err, "Unable to read application state for %s", data.Name.ValueString())
	}

	appMeta := getAppResp.Application.Meta

	data.Name = types.StringValue(appMeta.Name)
	data.Id = types.StringValue(appMeta.Id)
	data.Version = types.StringValue(appMeta.Version)

	return nil
}

func (r *ApplicationResource) refresh(ctx context.Context, data *ApplicationResourceModel) error {
	return readApplicationData(ctx, r.client, data)
}

func (r *ApplicationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *ApplicationResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	configResp, err := r.client.ConfigureApplication(ctx, &app_pb.ConfigureApplicationReq{
		ApplicationConfig: &app_pb.ApplicationConfig{
			Name: data.Name.ValueString(),
		},
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create application, got error: %s", err))
		return
	}
	data.Id = types.StringValue(configResp.Meta.Id)
	data.Version = types.StringValue(configResp.Meta.Version)

	tflog.Trace(ctx, "created application resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ApplicationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *ApplicationResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.refresh(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read application state for %s, got error: %s", data.Name.ValueString(), err))
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ApplicationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData *ApplicationResourceModel
	var stateData *ApplicationResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)

	if resp.Diagnostics.HasError() {
		return
	}

	// get current application config so we don't override values that either are not yet supported in TF,
	// or are updated as separate resources, e.g. Release Channels
	getAppResp, err := r.client.GetApplication(ctx, &app_pb.GetApplicationReq{
		Application: planData.Name.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update application, got error: %s", err))
		return
	}

	appConfig := getAppResp.Application.Config

	// this is not really needed since changing the application's name is not supported
	appConfig.Name = planData.Name.ValueString()
	// this is where newly supported fields should be set

	configResp, err := r.client.ConfigureApplication(ctx, &app_pb.ConfigureApplicationReq{
		ApplicationConfig: appConfig,
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update application, got error: %s", err))
		return
	}

	planData.Id = types.StringValue(configResp.Meta.Id)
	planData.Version = types.StringValue(configResp.Meta.Version)

	tflog.Trace(ctx, "updated application resource")

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &planData)...)
}

func (r *ApplicationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *ApplicationResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
	_, err := r.client.DeleteApplication(ctx, &app_pb.DeleteApplicationReq{
		Application: data.Name.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete application, got error: %s", err))
		return
	}
	tflog.Trace(ctx, "deleted application resource")
}

func (r *ApplicationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data ApplicationResourceModel

	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
	if resp.Diagnostics.HasError() {
		return
	}

	data.Name = types.StringValue(req.ID)
	err := r.refresh(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to import application state for %s, got error: %s", data.Name.ValueString(), err))
		return
	}

	// Save imported data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
