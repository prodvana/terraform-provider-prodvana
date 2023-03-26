package application

import (
	"context"
	"fmt"
	"strings"

	"github.com/pkg/errors"
	app_pb "github.com/prodvana/prodvana-public/go/prodvana-sdk/proto/prodvana/application"
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
	Name            types.String                    `tfsdk:"name"`
	Id              types.String                    `tfsdk:"id"`
	Version         types.String                    `tfsdk:"version"`
	ReleaseChannels map[string]*releaseChannelModel `tfsdk:"release_channels"`
}

type releaseChannelModel struct {
	Id       types.String                   `tfsdk:"id"`
	Policy   *policyModel                   `tfsdk:"policy"`
	Runtimes []*releaseChannelRuntimeConfig `tfsdk:"runtimes"`
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
	Runtime        types.String `tfsdk:"runtime"`
	ConnectionName types.String `tfsdk:"connection_name"`
	ConnectionType types.String `tfsdk:"connection_type"`
}

func (r *ApplicationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application"
}

func (r *ApplicationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
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
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"release_channels": schema.MapNestedAttribute{
				Required: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
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
									"connection_name": schema.StringAttribute{
										MarkdownDescription: "optional identifier for this runtime connection within this release channel",
										Optional:            true,
										Computed:            true,
									},
									"connection_type": schema.StringAttribute{
										MarkdownDescription: fmt.Sprintf("type of the runtime connection, one of (%s)", strings.Join(maps.Values(rc_pb.RuntimeConnectionType_name), ", ")),
										Optional:            true,
										Computed:            true,
										Validators:          validators.DefaultNameValidators(),
									},
								},
							},
						},
					},
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

func (r *ApplicationResource) refresh(ctx context.Context, data *ApplicationResourceModel) error {
	getAppResp, err := r.client.GetApplication(ctx, &app_pb.GetApplicationReq{
		Application: data.Name.ValueString(),
	})

	if err != nil {
		return errors.Wrapf(err, "Unable to read application state for %s", data.Name.ValueString())
	}

	appMeta := getAppResp.Application.Meta

	data.Id = types.StringValue(appMeta.Id)
	data.Version = types.StringValue(appMeta.Version)

	for name, rc := range data.ReleaseChannels {
		getRcResp, err := r.rcClient.GetReleaseChannel(ctx, &rc_pb.GetReleaseChannelReq{
			Application:    data.Name.ValueString(),
			ReleaseChannel: name,
		})
		if err != nil {
			return errors.Wrapf(err, "Unable to read release channel state for %s", data.Name.ValueString())
		}

		meta := getRcResp.ReleaseChannel.Meta
		config := getRcResp.ReleaseChannel.Config

		rc.Id = types.StringValue(meta.Id)
		if config.Policy == nil {
			rc.Policy = nil
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
			rc.Policy = &policyModel{
				DefaultEnv: defaultEnv,
			}
		}
		if config.Runtimes == nil {
			rc.Runtimes = nil
		} else {
			runtimeConfigs := make([]*releaseChannelRuntimeConfig, len(config.Runtimes))
			for idx, rt := range config.Runtimes {
				runtimeConfigs[idx] = &releaseChannelRuntimeConfig{
					Runtime:        types.StringValue(rt.Runtime),
					ConnectionName: types.StringValue(rt.ConnectionName),
					ConnectionType: types.StringValue(rt.ConnectionType.String()),
				}
			}
			rc.Runtimes = runtimeConfigs
		}
	}

	return nil
}

func (r *ApplicationResource) createOrUpdate(ctx context.Context, planData, stateData *ApplicationResourceModel) error {
	// this is an update. if a release channel was removed,
	// we need to set the appropriate ApprovedDangerousActionIds value
	var approvedDangerIds []string
	if stateData != nil {
		for name := range stateData.ReleaseChannels {
			if _, ok := planData.ReleaseChannels[name]; ok {
				continue
			}
			approvedDangerIds = append(approvedDangerIds, "delete:"+name)
		}
	}
	tflog.Trace(ctx, fmt.Sprintf("built danger ids: %v", approvedDangerIds))

	var releaseChannels []*rc_pb.ReleaseChannelConfig
	for name, rc := range planData.ReleaseChannels {
		runtimes := make([]*rc_pb.ReleaseChannelRuntimeConfig, len(rc.Runtimes))
		for idx, rt := range rc.Runtimes {
			connType := rc_pb.RuntimeConnectionType_UNKNOWN_CONNECTION
			if rt.ConnectionType.ValueString() != "" {
				connType = rc_pb.RuntimeConnectionType(rc_pb.RuntimeConnectionType_value[rt.ConnectionName.ValueString()])
			}
			runtimes[idx] = &rc_pb.ReleaseChannelRuntimeConfig{
				Runtime:        rt.Runtime.ValueString(),
				ConnectionName: rt.ConnectionName.ValueString(),
				ConnectionType: connType,
			}
		}
		var policy *rc_pb.Policy
		if rc.Policy != nil {
			defaultEnv := map[string]*common_config_pb.EnvValue{}
			for k, v := range rc.Policy.DefaultEnv {
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
		releaseChannels = append(releaseChannels, &rc_pb.ReleaseChannelConfig{
			Name:     name,
			Runtimes: runtimes,
			Policy:   policy,
		})
	}

	_, err := r.client.ConfigureApplication(ctx, &app_pb.ConfigureApplicationReq{
		ApplicationConfig: &app_pb.ApplicationConfig{
			Name:               planData.Name.ValueString(),
			UseDynamicDelivery: true,
			ReleaseChannels:    releaseChannels,
		},
		ApprovedDangerousActionIds: approvedDangerIds,
	})
	if err != nil {
		return err
	}

	return r.refresh(ctx, planData)
}

func (r *ApplicationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data *ApplicationResourceModel

	// Read Terraform plan data into the model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)

	if resp.Diagnostics.HasError() {
		return
	}

	err := r.createOrUpdate(ctx, data, nil)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to create application, got error: %s", err))
		return
	}

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

	err := r.createOrUpdate(ctx, planData, stateData)
	if err != nil {
		resp.Diagnostics.AddError("Client Error", fmt.Sprintf("Unable to update application, got error: %s", err))
		return
	}

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
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
