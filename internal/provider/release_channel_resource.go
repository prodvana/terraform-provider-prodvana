package provider

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/pkg/errors"
	common_config_pb "github.com/prodvana/prodvana-public/go/prodvana-sdk/proto/prodvana/common_config"
	prot_pb "github.com/prodvana/prodvana-public/go/prodvana-sdk/proto/prodvana/protection"
	rc_pb "github.com/prodvana/prodvana-public/go/prodvana-sdk/proto/prodvana/release_channel"
	runtimes_pb "github.com/prodvana/prodvana-public/go/prodvana-sdk/proto/prodvana/runtimes"
	"github.com/prodvana/terraform-provider-prodvana/internal/provider/validators"
	"golang.org/x/exp/maps"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/durationpb"

	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/objectvalidator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
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

	ReleaseChannelStablePreconditions []*releaseChannelStable `tfsdk:"release_channel_stable_preconditions"`
	ManualApprovalPreconditions       []*manualApproval       `tfsdk:"manual_approval_preconditions"`

	Protections                []*protectionAttachment `tfsdk:"protections"`
	ConvergenceProtections     []*protectionAttachment `tfsdk:"convergence_protections"`
	ServiceInstanceProtections []*protectionAttachment `tfsdk:"service_instance_protections"`
}

type releaseChannelStable struct {
	ReleaseChannel types.String `tfsdk:"release_channel"`
	Duration       types.String `tfsdk:"duration"`
}

type manualApproval struct {
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	EveryAction types.Bool   `tfsdk:"every_action"`
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

	K8sNamespace types.String `tfsdk:"k8s_namespace"`
	EcsPrefix    types.String `tfsdk:"ecs_prefix"`
}

var runtimeConnectionTypes []string

func init() {
	runtimeConnectionTypes = maps.Values(rc_pb.RuntimeConnectionType_name)
	sort.Strings(runtimeConnectionTypes)
}

func (r *ReleaseChannelResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_release_channel"
}

func (r *ReleaseChannelResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {

	protectionSchema := schema.NestedAttributeObject{
		Attributes: map[string]schema.Attribute{
			"name": schema.StringAttribute{
				MarkdownDescription: "name of the protection",
				Optional:            true,
				Computed:            true,
				Validators:          validators.DefaultNameValidators(),
			},
			"ref": schema.SingleNestedAttribute{
				MarkdownDescription: "reference to a protection stored in Prodvana",
				Required:            true,
				Attributes: map[string]schema.Attribute{
					"name": schema.StringAttribute{
						MarkdownDescription: "name of the protection",
						Required:            true,
						Validators:          validators.DefaultNameValidators(),
					},
					"parameters": schema.ListNestedAttribute{
						MarkdownDescription: "parameters to pass to the protection",
						Optional:            true,
						NestedObject: schema.NestedAttributeObject{
							Attributes: map[string]schema.Attribute{
								"name": schema.StringAttribute{
									MarkdownDescription: "name of the parameter",
									Required:            true,
								},
								"string_value": schema.StringAttribute{
									MarkdownDescription: "parameter string value, only one of (string_value, int_value, docker_image_tag_value, secret_value) can be set",
									Optional:            true,
									Validators: []validator.String{
										stringvalidator.ExactlyOneOf(
											path.MatchRelative().AtParent().AtName("int_value"),
											path.MatchRelative().AtParent().AtName("docker_image_tag_value"),
											path.MatchRelative().AtParent().AtName("secret_value"),
										),
									},
								},
								"int_value": schema.Int64Attribute{
									MarkdownDescription: "parameter int value, only one of (string_value, int_value, docker_image_tag_value, secret_value) can be set",
									Optional:            true,
									Validators: []validator.Int64{
										int64validator.ExactlyOneOf(
											path.MatchRelative().AtParent().AtName("string_value"),
											path.MatchRelative().AtParent().AtName("docker_image_tag_value"),
											path.MatchRelative().AtParent().AtName("secret_value"),
										),
									},
								},
								"docker_image_tag_value": schema.StringAttribute{
									MarkdownDescription: "parameter docker image tag value, only one of (string_value, int_value, docker_image_tag_value, secret_value) can be set",
									Optional:            true,
									Validators: []validator.String{
										stringvalidator.ExactlyOneOf(
											path.MatchRelative().AtParent().AtName("string_value"),
											path.MatchRelative().AtParent().AtName("int_value"),
											path.MatchRelative().AtParent().AtName("secret_value"),
										),
									},
								},
								"secret_value": schema.SingleNestedAttribute{
									MarkdownDescription: "parameter secret value, only one of (string_value, int_value, docker_image_tag_value, secret_value) can be set",
									Optional:            true,
									Validators: []validator.Object{
										objectvalidator.ExactlyOneOf(
											path.MatchRelative().AtParent().AtName("string_value"),
											path.MatchRelative().AtParent().AtName("int_value"),
											path.MatchRelative().AtParent().AtName("docker_image_tag_value"),
										),
									},
									Attributes: map[string]schema.Attribute{
										"key": schema.StringAttribute{
											MarkdownDescription: "Name of the secret.",
											Required:            true,
										},
										"version": schema.StringAttribute{
											MarkdownDescription: "Version of the secret",
											Required:            true,
										},
									},
								},
							},
						},
					},
				},
			},
			"pre_approval": schema.SingleNestedAttribute{
				MarkdownDescription: "pre-approval lifecycle options, enabled if present",
				Optional:            true,
			},
			"post_approval": schema.SingleNestedAttribute{
				MarkdownDescription: "post-approval lifecycle options, enabled if present",
				Optional:            true,
			},
			"deployment": schema.SingleNestedAttribute{
				MarkdownDescription: "deployment lifecycle options, enabled if present",
				Optional:            true,
			},
			"post_deployment": schema.SingleNestedAttribute{
				MarkdownDescription: "post-deployment lifecycle options, enabled if present",
				Optional:            true,
				Attributes: map[string]schema.Attribute{
					"delay_check_duration": schema.StringAttribute{
						MarkdownDescription: "delay between the deployment completing and when this protection starts checking. A valid Go duration string, e.g. `10m` or `1h`. Defaults to `10m`",
						Optional:            true,
					},
					"check_duration": schema.StringAttribute{
						MarkdownDescription: "how long to keep checking. A valid Go duration string, e.g. `10m` or `1h`. Defaults to `10m`",
						Optional:            true,
					},
				},
			},
		},
	}
	resp.Schema = schema.Schema{
		MarkdownDescription: "This resource allows you to manage a Prodvana [Release Channel](https://docs.prodvana.io/docs/prodvana-concepts#release-channel).",
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
							MarkdownDescription: fmt.Sprintf("type of the runtime connection, one of (%s)", strings.Join(runtimeConnectionTypes, ", ")),
							Optional:            true,
							Computed:            true,
							Validators: []validator.String{
								stringvalidator.OneOf(runtimeConnectionTypes...),
							},
						},
						"k8s_namespace": schema.StringAttribute{
							MarkdownDescription: "Optionally set a custom namespace. If not set, Prodvana will create and manage the namespace. If set, the namespace *must* already exist and Prodvana will not try to create or delete it. Can only be set on a Kubernetes Runtime.",
							Optional:            true,
							Computed:            true,
							Validators: []validator.String{
								stringvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("ecs_prefix")),
							},
						},
						"ecs_prefix": schema.StringAttribute{
							MarkdownDescription: "Prefix used when naming ECS resources. Can only be set on an ECS Runtime.",
							Optional:            true,
							Computed:            true,
							Validators: []validator.String{
								stringvalidator.ConflictsWith(path.MatchRelative().AtParent().AtName("kubernetes_namespace")),
							},
						},
					},
				},
			},
			"release_channel_stable_preconditions": schema.ListNestedAttribute{
				MarkdownDescription: "Preconditions requiring other release channels to be stable before this release channel can be deployed",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"release_channel": schema.StringAttribute{
							MarkdownDescription: "name of a release channel that must be in a stable deployment state",
							Required:            true,
							Validators:          validators.DefaultNameValidators(),
						},
						"duration": schema.StringAttribute{
							MarkdownDescription: "duration to wait for the release channel to be stable. A valid Go duration string, e.g. `10m` or `1h`. Defaults to `10m`",
							Required:            true,
						},
					},
				},
			},
			"manual_approval_preconditions": schema.ListNestedAttribute{
				MarkdownDescription: "Preconditions requiring manual approval before this release channel can be deployed",
				Optional:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: "name of the manual approval",
							Required:            true,
							Validators:          validators.DefaultNameValidators(),
						},
						"description": schema.StringAttribute{
							MarkdownDescription: "description of the manual approval",
							Optional:            true,
							Computed:            true,
							Default:             stringdefault.StaticString(""),
						},
						"every_action": schema.BoolAttribute{
							MarkdownDescription: "whether this approval is required for every convergence action, or just the first",
							Optional:            true,
							Computed:            true,
							Default:             booldefault.StaticBool(false),
						},
					},
				},
			},
			"protections": schema.ListNestedAttribute{
				MarkdownDescription: "Protections applied this release channel",
				Optional:            true,
				NestedObject:        protectionSchema,
			},
			"service_instance_protections": schema.ListNestedAttribute{
				MarkdownDescription: "Protections applied to service instances in this release channel",
				Optional:            true,
				NestedObject:        protectionSchema,
			},
			"convergence_protections": schema.ListNestedAttribute{
				MarkdownDescription: "Feature Coming Soon",
				Optional:            true,
				NestedObject:        protectionSchema,
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

func attachmentProtosToTerraform(protections []*prot_pb.ProtectionAttachmentConfig) []*protectionAttachment {
	attachments := make([]*protectionAttachment, len(protections))
	for idx, pa := range protections {
		attachments[idx] = ProtectionAttachmentProtoToTerraform(pa)
	}
	return attachments
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
			if rt.GetContainerOrchestration() != nil && rt.GetContainerOrchestration().Backend != nil {
				backend := rt.GetContainerOrchestration().Backend
				switch t := backend.(type) {
				case *runtimes_pb.ContainerOrchestrationRuntime_K8S_:
					runtimeConfigs[idx].K8sNamespace = types.StringValue(t.K8S.Namespace)
				case *runtimes_pb.ContainerOrchestrationRuntime_Ecs:
					runtimeConfigs[idx].EcsPrefix = types.StringValue(t.Ecs.Prefix)
				}
			}
		}
		data.Runtimes = runtimeConfigs
	}

	if config.Preconditions != nil {
		rcStable := []*releaseChannelStable{}
		approvals := []*manualApproval{}
		for _, rc := range config.Preconditions {
			switch rc.Precondition.(type) {
			case *rc_pb.Precondition_ReleaseChannelStable_:
				duration := rc.GetReleaseChannelStable().Duration.AsDuration()
				precon := &releaseChannelStable{
					ReleaseChannel: types.StringValue(rc.GetReleaseChannelStable().ReleaseChannel),
					Duration:       types.StringValue(duration.String()),
				}

				rcStable = append(rcStable, precon)
			case *rc_pb.Precondition_ManualApproval_:
				precon := &manualApproval{
					Name:        types.StringValue(rc.GetManualApproval().Name),
					Description: types.StringValue(rc.GetManualApproval().Description),
					EveryAction: types.BoolValue(rc.GetManualApproval().EveryAction),
				}
				approvals = append(approvals, precon)
			}
		}
		if len(rcStable) > 0 {
			data.ReleaseChannelStablePreconditions = rcStable
		}
		if len(approvals) > 0 {
			data.ManualApprovalPreconditions = approvals
		}
	}

	if config.Protections != nil {
		protections := attachmentProtosToTerraform(config.Protections)
		if len(protections) > 0 {
			data.Protections = protections
		}
	}
	if config.ConvergenceProtections != nil {
		protections := attachmentProtosToTerraform(config.ConvergenceProtections)
		if len(protections) > 0 {
			data.ConvergenceProtections = protections
		}
	}
	if config.ServiceInstanceProtections != nil {
		protections := attachmentProtosToTerraform(config.ServiceInstanceProtections)
		if len(protections) > 0 {
			data.ServiceInstanceProtections = protections
		}
	}

	return nil
}

func (r *ReleaseChannelResource) refresh(ctx context.Context, data *ReleaseChannelResourceModel) error {
	return readReleaseChannelData(ctx, r.client, data)
}

func protectionAttachmentsToProtos(attachments []*protectionAttachment) ([]*prot_pb.ProtectionAttachmentConfig, error) {
	protections := []*prot_pb.ProtectionAttachmentConfig{}
	for _, protection := range attachments {
		protoAttachment, err := protection.AsProto()
		if err != nil {
			return nil, err
		}
		protections = append(protections, protoAttachment)
	}
	return protections, nil
}

func (r *ReleaseChannelResource) createOrUpdate(ctx context.Context, planData *ReleaseChannelResourceModel) error {
	runtimes := make([]*rc_pb.ReleaseChannelRuntimeConfig, len(planData.Runtimes))
	for idx, rt := range planData.Runtimes {
		runtimes[idx] = &rc_pb.ReleaseChannelRuntimeConfig{
			Runtime: rt.Runtime.ValueString(),
			Name:    rt.Name.ValueString(),
		}
		if rt.Type.ValueString() != "" {
			connVal, found := rc_pb.RuntimeConnectionType_value[rt.Type.ValueString()]
			if !found {
				return errors.Errorf("Invalid runtime connection type %s, must be one of (%s)", rt.Type.ValueString(), strings.Join(runtimeConnectionTypes, ", "))
			}
			runtimes[idx].Type = rc_pb.RuntimeConnectionType(connVal)
		}
		if !rt.K8sNamespace.IsUnknown() {
			runtimes[idx].Capability = &rc_pb.ReleaseChannelRuntimeConfig_ContainerOrchestration{
				ContainerOrchestration: &runtimes_pb.ContainerOrchestrationRuntime{
					Backend: &runtimes_pb.ContainerOrchestrationRuntime_K8S_{
						K8S: &runtimes_pb.ContainerOrchestrationRuntime_K8S{
							Namespace: rt.K8sNamespace.ValueString(),
						},
					},
				},
			}
		} else if !rt.EcsPrefix.IsUnknown() {
			runtimes[idx].Capability = &rc_pb.ReleaseChannelRuntimeConfig_ContainerOrchestration{
				ContainerOrchestration: &runtimes_pb.ContainerOrchestrationRuntime{
					Backend: &runtimes_pb.ContainerOrchestrationRuntime_Ecs{
						Ecs: &runtimes_pb.ContainerOrchestrationRuntime_ECS{
							Prefix: rt.EcsPrefix.ValueString(),
						},
					},
				},
			}
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

	preconditions := []*rc_pb.Precondition{}
	if planData.ReleaseChannelStablePreconditions != nil {
		for _, rc := range planData.ReleaseChannelStablePreconditions {
			dur, err := time.ParseDuration(rc.Duration.ValueString())
			if err != nil {
				return err
			}

			preconditions = append(preconditions, &rc_pb.Precondition{
				Precondition: &rc_pb.Precondition_ReleaseChannelStable_{
					ReleaseChannelStable: &rc_pb.Precondition_ReleaseChannelStable{
						ReleaseChannel: rc.ReleaseChannel.ValueString(),
						Duration:       durationpb.New(dur),
					},
				},
			})
		}
	}

	if planData.ManualApprovalPreconditions != nil {
		for _, approval := range planData.ManualApprovalPreconditions {
			preconditions = append(preconditions, &rc_pb.Precondition{
				Precondition: &rc_pb.Precondition_ManualApproval_{
					ManualApproval: &rc_pb.Precondition_ManualApproval{
						Name:        approval.Name.ValueString(),
						Description: approval.Description.ValueString(),
						EveryAction: approval.EveryAction.ValueBool(),
					},
				},
			})
		}
	}

	protections, err := protectionAttachmentsToProtos(planData.Protections)
	if err != nil {
		return err
	}

	convergenceProtections, err := protectionAttachmentsToProtos(planData.ConvergenceProtections)
	if err != nil {
		return err
	}

	svcInstanceProtections, err := protectionAttachmentsToProtos(planData.ServiceInstanceProtections)
	if err != nil {
		return err
	}

	releaseChannel := &rc_pb.ReleaseChannelConfig{
		Name:                       planData.Name.ValueString(),
		Runtimes:                   runtimes,
		Policy:                     policy,
		Preconditions:              preconditions,
		Protections:                protections,
		ConvergenceProtections:     convergenceProtections,
		ServiceInstanceProtections: svcInstanceProtections,
	}

	_, err = r.client.ConfigureReleaseChannel(ctx, &rc_pb.ConfigureReleaseChannelReq{
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

	err := r.createOrUpdate(ctx, data)
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
		// if the release channel does not exist, remove the resource
		if status.Code(err) == codes.NotFound {
			resp.State.RemoveResource(ctx)
			return
		}
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

	err := r.createOrUpdate(ctx, planData)
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
