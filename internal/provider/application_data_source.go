package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	app_pb "github.com/prodvana/prodvana-public/go/prodvana-sdk/proto/prodvana/application"
	"github.com/prodvana/terraform-provider-prodvana/internal/provider/validators"
	"google.golang.org/grpc"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &ReleaseChannelDataSource{}

func NewApplicationDataSource() datasource.DataSource {
	return &ApplicationDataSource{}
}

// ApplicationDataSource defines the data source implementation.
type ApplicationDataSource struct {
	client app_pb.ApplicationManagerClient
}

func (d *ApplicationDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application"
}

func (d *ApplicationDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Prodvana Application",
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
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "Application description",
				Optional:            true,
			},
		},
	}
}

func (d *ApplicationDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	// Prevent panic if the provider has not been configured.
	if req.ProviderData == nil {
		return
	}

	conn, ok := req.ProviderData.(*grpc.ClientConn)

	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *grpc.ClientConn, got: %T. Please report this issue to the provider developers.", req.ProviderData),
		)

		return
	}

	d.client = app_pb.NewApplicationManagerClient(conn)
}

func (d *ApplicationDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ApplicationResourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := readApplicationData(ctx, d.client, &data)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read application state for %s, got error: %s", data.Name.ValueString(), err))
		return
	}

	tflog.Trace(ctx, "read application data source")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
