package provider

import (
	"context"
	"fmt"
	"sort"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	env_pb "github.com/prodvana/prodvana-public/go/prodvana-sdk/proto/prodvana/environment"
	"github.com/prodvana/terraform-provider-prodvana/internal/provider/validators"
	"golang.org/x/exp/maps"
	"google.golang.org/grpc"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &EcsRuntimeDataSource{}

func NewEcsRuntimeDataSource() datasource.DataSource {
	return &EcsRuntimeDataSource{}
}

// EcsRuntimeDataSource defines the data source implementation.
type EcsRuntimeDataSource struct {
	client env_pb.EnvironmentManagerClient
}

func (d *EcsRuntimeDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ecs_runtime"
}

func (d *EcsRuntimeDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	clusterTypes := maps.Values(env_pb.ClusterType_name)
	sort.Strings(clusterTypes)

	resp.Schema = schema.Schema{
		// TODO(mike): expand description -- this feeds into the generated documentation that shows up in the registry
		MarkdownDescription: "Prodvana Runtime",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Runtime name",
				Required:            true,
				Validators:          validators.DefaultNameValidators(),
			},
			"id": schema.StringAttribute{
				MarkdownDescription: "Runtime identifier",
				Computed:            true,
			},
			"access_key": schema.StringAttribute{
				MarkdownDescription: "AWS Access Key ID with permissions to the ECS cluster",
				Computed:            true,
			},
			"secret_key": schema.StringAttribute{
				MarkdownDescription: "AWS Secret Key with permissions to the ECS cluster",
				Computed:            true,
			},
			"region": schema.StringAttribute{
				MarkdownDescription: "AWS Region of the ECS cluster",
				Computed:            true,
			},
			"assume_role_arn": schema.StringAttribute{
				MarkdownDescription: "AWS role to assume when accessing the ECS cluster",
				Computed:            true,
			},
			"cluster_arn": schema.StringAttribute{
				MarkdownDescription: "ARN of the ECS cluster",
				Computed:            true,
			},
		},
	}
}

func (d *EcsRuntimeDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

	d.client = env_pb.NewEnvironmentManagerClient(conn)
}

func (d *EcsRuntimeDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data EcsRuntimeResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := readEcsRuntimeData(ctx, d.client, &data)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read runtime state for %s, got error: %s", data.Name.ValueString(), err))
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
