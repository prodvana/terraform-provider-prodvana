package provider

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	rc_pb "github.com/prodvana/prodvana-public/go/prodvana-sdk/proto/prodvana/release_channel"
	"github.com/prodvana/terraform-provider-prodvana/internal/provider/validators"
	"golang.org/x/exp/maps"
	"google.golang.org/grpc"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &ReleaseChannelDataSource{}

func NewReleaseChannelDataSource() datasource.DataSource {
	return &ReleaseChannelDataSource{}
}

// ReleaseChannelDataSource defines the data source implementation.
type ReleaseChannelDataSource struct {
	client rc_pb.ReleaseChannelManagerClient
}

func (d *ReleaseChannelDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_release_channel"
}

func (d *ReleaseChannelDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	connectionTypes := maps.Values(rc_pb.RuntimeConnectionType_name)
	sort.Slice(connectionTypes, func(i int, j int) bool {
		return connectionTypes[i] < connectionTypes[j]
	})
	resp.Schema = schema.Schema{
		MarkdownDescription: "Prodvana Release Channel",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Release Channel name",
				Required:            true,
				Validators:          validators.DefaultNameValidators(),
			},
			"application": schema.StringAttribute{
				MarkdownDescription: "Name of the Application this Release Channel belongs to",
				Required:            true,
				Validators:          validators.DefaultNameValidators(),
			},
			"version": schema.StringAttribute{
				MarkdownDescription: "Current application version",
				Computed:            true,
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Release channel identifier",
			},
			"policy": schema.SingleNestedAttribute{
				MarkdownDescription: "Release Channel policy applied to all services",
				Computed:            true,
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"default_env": schema.MapNestedAttribute{
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"value": schema.StringAttribute{
									MarkdownDescription: "Non-sensitive environment variable value",
									Optional:            true,
								},
								"secret": schema.SingleNestedAttribute{
									MarkdownDescription: "Reference to a secret value stored in Prodvana.",
									Optional:            true,
									Attributes: map[string]schema.Attribute{
										"key": schema.StringAttribute{
											MarkdownDescription: "Name of the secret.",
											Optional:            true,
										},
										"version": schema.StringAttribute{
											MarkdownDescription: "Version of the secret",
											Optional:            true,
										},
									},
								},
							},
						},
						MarkdownDescription: "default environment variables for services in this Release Channel",
						Optional:            true,
						Computed:            true,
					},
				},
			},
			"runtimes": schema.ListNestedAttribute{
				MarkdownDescription: "Release Channel policy applied to all services",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"runtime": schema.StringAttribute{
							MarkdownDescription: "name of the a runtime",
							Optional:            true,
							Validators:          validators.DefaultNameValidators(),
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "optional identifier for this runtime connection within this release channel",
							Optional:            true,
							Computed:            true,
						},
						"type": schema.StringAttribute{
							MarkdownDescription: fmt.Sprintf("type of the runtime connection, one of (%s)", strings.Join(connectionTypes, ", ")),
							Optional:            true,
							Computed:            true,
							Validators:          validators.DefaultNameValidators(),
						},
					},
				},
			},
		},
	}
}

func (d *ReleaseChannelDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	d.client = rc_pb.NewReleaseChannelManagerClient(conn)
}

func (d *ReleaseChannelDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ReleaseChannelResourceModel

	// Read Terraform configuration data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := readReleaseChannelData(ctx, d.client, &data)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read release channel state for %s, got error: %s", data.Name.ValueString(), err))
		return
	}

	tflog.Trace(ctx, "read release channel data source")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
