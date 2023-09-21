package provider

import (
	"context"
	"fmt"

	"github.com/pkg/errors"
	env_pb "github.com/prodvana/prodvana-public/go/prodvana-sdk/proto/prodvana/environment"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ resource.Resource = &RuntimeLinkResource{}

func NewRuntimeLinkResource() resource.Resource {
	return &RuntimeLinkResource{}
}

// RuntimeLinkResource defines the resource implementation.
type RuntimeLinkResource struct {
	client env_pb.EnvironmentManagerClient
}

// RuntimeLinkResouceModel describes the resource link data model.
type RuntimeLinkResourceModel struct {
	Name    types.String `tfsdk:"name"`
	Id      types.String `tfsdk:"id"`
	Timeout types.String `tfsdk:"timeout"`
}

func (r *RuntimeLinkResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_runtime_link"
}

func (r *RuntimeLinkResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: `(Alpha! This feature is still in progress.) 
A ` + "`runtime_link`" + ` resource represents a successfully linked runtime.
This is most useful for Kubernetes runtimes -- the agent must be installed and registered with the Prodvana service before the runtime can be used.
Pair this with an explicit ` + "`depends_on`" + ` block ensures that the runtime is ready before attempting to use it. See the example below.
`,
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "Name of the runtime to wait for linking.",
				Required:            true,
			},
			"id": schema.StringAttribute{
				Computed:            true,
				MarkdownDescription: "Runtime identifier",
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"timeout": schema.StringAttribute{
				MarkdownDescription: "How long to wait for the runtime linking to complete. A valid Go duration string, e.g. `10m` or `1h`. Defaults to `10m`",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("10m"),
			},
		},
	}
}

func (r *RuntimeLinkResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *RuntimeLinkResource) refresh(ctx context.Context, diags diag.Diagnostics, data *RuntimeLinkResourceModel) (bool, error) {
	resp, err := r.client.GetCluster(ctx, &env_pb.GetClusterReq{
		Runtime: data.Name.ValueString(),
	})
	if err != nil {
		return false, errors.Wrapf(err, "Unable to read runtime state for %s", data.Name.ValueString())
	}

	data.Id = types.StringValue(resp.Cluster.Id)

	return resp.Cluster.Type == env_pb.ClusterType_K8S, nil
}

func (r *RuntimeLinkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *RuntimeLinkResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	getResp, err := r.client.GetCluster(ctx, &env_pb.GetClusterReq{
		Runtime: data.Name.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read runtime state: %s", err))
		return
	}

	data.Id = types.StringValue(getResp.Cluster.Id)

	if getResp.Cluster.Type == env_pb.ClusterType_K8S {
		// keep checking to see if linking succeeded until timeout
		err := WaitForClusterWithTimeout(ctx, r.client, data.Id.ValueString(), data.Name.ValueString(), data.Timeout.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Failed waiting for runtime linking: %s", err))
			return
		}
	}

	tflog.Trace(ctx, "created runtime link resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *RuntimeLinkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *RuntimeLinkResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	linked, err := r.refresh(ctx, resp.Diagnostics, data)
	if err != nil {
		// if runtime doesn't exist anymore, remove the resource
		if status.Code(err) == codes.NotFound {
			resp.State.RemoveResource(ctx)
			return
		}

		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read runtime link state for %s, got error: %s", data.Name.ValueString(), err))
		return
	}

	// treat being unlinked as a deleted resource since this resource must be blocked on
	if !linked {
		resp.State.RemoveResource(ctx)
	} else {
		// Save updated data into Terraform state
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
	}
}

func (r *RuntimeLinkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData *RuntimeLinkResourceModel
	var stateData *RuntimeLinkResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)

	if resp.Diagnostics.HasError() {
		return
	}

	_, err := r.client.GetCluster(ctx, &env_pb.GetClusterReq{
		Runtime: planData.Name.ValueString(),
	})
	if err != nil {
		// if runtime doesn't exist anymore, remove the resource
		if status.Code(err) == codes.NotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update runtime link status, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "updated runtime link resource")

	resp.Diagnostics.Append(resp.State.Set(ctx, &planData)...)
}

func (r *RuntimeLinkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *RuntimeLinkResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	// nothing to do since there is not a true resource on the backend this is tied to

	tflog.Trace(ctx, "deleted runtime link resource")
}
