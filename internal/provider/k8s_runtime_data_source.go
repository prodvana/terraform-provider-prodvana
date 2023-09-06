package provider

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/pkg/errors"
	env_pb "github.com/prodvana/prodvana-public/go/prodvana-sdk/proto/prodvana/environment"
	"github.com/prodvana/terraform-provider-prodvana/internal/provider/labels"
	"github.com/prodvana/terraform-provider-prodvana/internal/provider/validators"
	"google.golang.org/grpc"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &K8sRuntimeDataSource{}

func NewK8sRuntimeDataSource() datasource.DataSource {
	return &K8sRuntimeDataSource{}
}

// ReleaseChannelDataSource defines the data source implementation.
type K8sRuntimeDataSource struct {
	client env_pb.EnvironmentManagerClient
}

type K8sRuntimeDataSourceModel struct {
	Name types.String `tfsdk:"name"`
	Id   types.String `tfsdk:"id"`

	AgentApiToken types.String `tfsdk:"agent_api_token"`

	Labels types.List `tfsdk:"labels"`
}

func (d *K8sRuntimeDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_k8s_runtime"
}

func (d *K8sRuntimeDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		// TODO(mike): expand description -- this feeds into the generated documentation that shows up in the registry
		MarkdownDescription: "Prodvana Kubernetes Runtime",
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Runtime name",
				Required:            true,
				Validators:          validators.DefaultNameValidators(),
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Runtime identifier",
			},
			"agent_api_token": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "API Token used for linking the Kubernetes Prodvana agent",
				Sensitive:           true,
			},
			"labels": schema.ListNestedAttribute{
				MarkdownDescription: "List of labels to apply to the runtime",
				Computed:            true,
				Optional:            true,
				NestedObject:        labels.LabelDefinitionNestedObjectDataSourceSchema(),
			},
		},
	}
}

func (d *K8sRuntimeDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *K8sRuntimeDataSource) read(ctx context.Context, diags diag.Diagnostics, data *K8sRuntimeDataSourceModel) error {
	resp, err := d.client.GetCluster(ctx, &env_pb.GetClusterReq{
		Runtime:     data.Name.ValueString(),
		IncludeAuth: true,
	})
	if err != nil {
		return errors.Wrapf(err, "Unable to read runtime state for %s", data.Name.ValueString())
	}

	data.Name = types.StringValue(resp.Cluster.Name)
	data.Id = types.StringValue(resp.Cluster.Id)
	data.Labels = labels.LabelDefinitionsToTerraformList(ctx, resp.Cluster.Config.Labels, diags)

	if resp.Cluster.Type != env_pb.ClusterType_K8S {
		return errors.Errorf("Unexpected non-Kubernetes runtime type: %s. Did the runtime change outside Terraform?", resp.Cluster.Type.String())
	}

	tokenResp, err := d.client.GetClusterAgentApiToken(ctx, &env_pb.GetClusterAgentApiTokenReq{
		Name: data.Name.ValueString(),
	})
	if err != nil {
		return errors.Wrapf(err, "Unable to read runtime state for %s", data.Name.ValueString())
	}
	data.AgentApiToken = types.StringValue(tokenResp.ApiToken)

	return nil
}

func (d *K8sRuntimeDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data K8sRuntimeDataSourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	err := d.read(ctx, resp.Diagnostics, &data)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read runtime state for %s, got error: %s", data.Name.ValueString(), err))
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
