package provider

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	common_config_pb "github.com/prodvana/prodvana-public/go/prodvana-sdk/proto/prodvana/common_config"
	rc_pb "github.com/prodvana/prodvana-public/go/prodvana-sdk/proto/prodvana/release_channel"
	"github.com/prodvana/terraform-provider-prodvana/internal/provider/validators"
	"golang.org/x/exp/maps"
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
var _ resource.Resource = &ReleaseChannelResource{}
var _ resource.ResourceWithImportState = &ReleaseChannelResource{}

func NewReleaseChannelResource() resource.Resource {
	return &ReleaseChannelResource{}
}

// ReleaseChannelResource defines the resource implementation.
type ReleaseChannelResource struct {
	client rc_pb.ReleaseChannelManagerClient
}

// ReleaseChannelResourcrModel describes the resource data model.
type ReleaseChannelResourceModel struct {
	Name        types.String                   `tfsdk:"name"`
	Id          types.String                   `tfsdk:"id"`
	Version     types.String                   `tfsdk:"version"`
	Policy      *policyModel                   `tfsdk:"policy"`
	Runtimes    []*releaseChannelRuntimeConfig `tfsdk:"runtimes"`
	Application types.String                   `tfsdk:"application"`
}

type policyModel struct {
	DefaultEnv map[string]*envValue `tfsdk:"default_env"`
}
type envValue struct {
	Value  types.String `tfsdk:"value"`
	Secret *envSecret   `tfsdk:"secret"`
}
type envSecret struct {
	Key     types.String `tfsdk:"key"`
	Version types.String `tfsdk:"version"`
}

type releaseChannelRuntimeConfig struct {
	Runtime types.String `tfsdk:"runtime"`
	Name    types.String `tfsdk:"name"`
	Type    types.String `tfsdk:"type"`
}

func (r *ReleaseChannelResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_release_channel"
}

func (r *ReleaseChannelResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"policy": schema.SingleNestedAttribute{
				MarkdownDescription: "Release Channel policy applied to all services",
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
					},
				},
			},
			"runtimes": schema.ListNestedAttribute{
				MarkdownDescription: "Release Channel policy applied to all services",
				Required:            true,
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
							MarkdownDescription: fmt.Sprintf("type of the runtime connection, one of (%s)", strings.Join(maps.Values(rc_pb.RuntimeConnectionType_name), ", ")),
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

func (r *ReleaseChannelResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

	r.client = rc_pb.NewReleaseChannelManagerClient(conn)
}

func readReleaseChannelData(ctx context.Context, client rc_pb.ReleaseChannelManagerClient, data *ReleaseChannelResourceModel) error {
	getRcResp, err := client.GetReleaseChannel(ctx, &rc_pb.GetReleaseChannelReq{
		Application:    data.Application.ValueString(),
		ReleaseChannel: data.Name.ValueString(),
	})

	if err != nil {
		return errors.Wrapf(err, "Unable to read release channel state for %s", data.Name.ValueString())
	}

	meta := getRcResp.ReleaseChannel.Meta
	config := getRcResp.ReleaseChannel.Config

	data.Id = types.StringValue(meta.Id)
	data.Version = types.StringValue(meta.Version)

	if config.Policy == nil {
		data.Policy = nil
	} else {
		defaultEnv := make(map[string]*envValue, len(config.Policy.DefaultEnv))
		for k, v := range config.Policy.DefaultEnv {
			envVal := &envValue{}
			switch t := v.ValueOneof.(type) {
			case *common_config_pb.EnvValue_Value:
				envVal.Value = types.StringValue(t.Value)
			case *common_config_pb.EnvValue_Secret:
				envVal.Secret = &envSecret{
					Key:     types.StringValue(t.Secret.Key),
					Version: types.StringValue(t.Secret.Version),
				}
			}
			defaultEnv[k] = envVal
		}
		data.Policy = &policyModel{
			DefaultEnv: defaultEnv,
		}
	}
	if config.Runtimes == nil {
		data.Runtimes = nil
	} else {
		runtimeConfigs := make([]*releaseChannelRuntimeConfig, len(config.Runtimes))
		for idx, rt := range config.Runtimes {
			runtimeConfigs[idx] = &releaseChannelRuntimeConfig{
				Runtime: types.StringValue(rt.Runtime),
				Name:    types.StringValue(rt.Name),
				Type:    types.StringValue(rt.Type.String()),
			}
		}
		data.Runtimes = runtimeConfigs
	}

	return nil
}

func (r *ReleaseChannelResource) refresh(ctx context.Context, data *ReleaseChannelResourceModel) error {
	return readReleaseChannelData(ctx, r.client, data)
}

func (r *ReleaseChannelResource) createOrUpdate(ctx context.Context, planData, stateData *ReleaseChannelResourceModel) error {
	runtimes := make([]*rc_pb.ReleaseChannelRuntimeConfig, len(planData.Runtimes))
	for idx, rt := range planData.Runtimes {
		connType := rc_pb.RuntimeConnectionType_UNKNOWN_CONNECTION
		if rt.Type.ValueString() != "" {
			connType = rc_pb.RuntimeConnectionType(rc_pb.RuntimeConnectionType_value[rt.Name.ValueString()])
		}
		runtimes[idx] = &rc_pb.ReleaseChannelRuntimeConfig{
			Runtime: rt.Runtime.ValueString(),
			Name:    rt.Name.ValueString(),
			Type:    connType,
		}
	}
	var policy *rc_pb.Policy
	if planData.Policy != nil {
		defaultEnv := map[string]*common_config_pb.EnvValue{}
		for k, v := range planData.Policy.DefaultEnv {
			envVal := &common_config_pb.EnvValue{}
			if !v.Value.IsNull() && v.Secret != nil {
				return fmt.Errorf("only one of Value or Secret can be set for %s", k)
			}

			if !v.Value.IsNull() {
				envVal.ValueOneof = &common_config_pb.EnvValue_Value{
					Value: v.Value.ValueString(),
				}
			} else if v.Secret != nil {
				envVal.ValueOneof = &common_config_pb.EnvValue_Secret{
					Secret: &common_config_pb.Secret{
						Key:     v.Secret.Key.ValueString(),
						Version: v.Secret.Version.ValueString(),
					},
				}
			} else {
				return fmt.Errorf("EnvValue for %s is empty", k)
			}
			defaultEnv[k] = envVal
		}
		if len(defaultEnv) > 0 {
			policy = &rc_pb.Policy{
				DefaultEnv: defaultEnv,
			}
		}
	}
	releaseChannel := &rc_pb.ReleaseChannelConfig{
		Name:     planData.Name.ValueString(),
		Runtimes: runtimes,
		Policy:   policy,
	}

	_, err := r.client.ConfigureReleaseChannel(ctx, &rc_pb.ConfigureReleaseChannelReq{
		ReleaseChannel: releaseChannel,
		Application:    planData.Application.ValueString(),
	})
	if err != nil {
		return err
	}

	return r.refresh(ctx, planData)
}

func (r *ReleaseChannelResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *ReleaseChannelResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.createOrUpdate(ctx, data, nil)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create release channel, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "created release channel resource")

	// Save data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ReleaseChannelResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data *ReleaseChannelResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.refresh(ctx, data)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to read release channel state for %s, got error: %s", data.Name.ValueString(), err))
		return
	}

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *ReleaseChannelResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var planData *ReleaseChannelResourceModel
	var stateData *ReleaseChannelResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &planData)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.createOrUpdate(ctx, planData, stateData)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update release channel, got error: %s", err))
		return
	}

	tflog.Trace(ctx, "updated release channel resource")

	// Save updated data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &planData)...)
}

func (r *ReleaseChannelResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data *ReleaseChannelResourceModel

	// Read Terraform prior state data into the model
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}
	_, err := r.client.DeleteReleaseChannel(ctx, &rc_pb.DeleteReleaseChannelReq{
		Application:    data.Application.ValueString(),
		ReleaseChannel: data.Name.ValueString(),
	})
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to delete release channel, got error: %s", err))
		return
	}
	tflog.Trace(ctx, "deleted release channel resource")
}

func (r *ReleaseChannelResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	var data ReleaseChannelResourceModel

	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
	if resp.Diagnostics.HasError() {
		return
	}

	// req.ID is of the form <application>/<relase channel>
	parts := strings.Split(req.ID, "/")
	if len(parts) != 2 {
		// rewrite this error to include the correct formatting of an ID
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to import release channel, got error: invalid id %s, expected <application>/<release channel>", req.ID))
		return
	}

	data.Application = types.StringValue(parts[0])
	data.Name = types.StringValue(parts[1])
	err := r.refresh(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to import release channel state for %s, got error: %s", data.Name.ValueString(), err))
		return
	}

	// Save imported data into Terraform state
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
