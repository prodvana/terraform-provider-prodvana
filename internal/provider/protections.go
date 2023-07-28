package provider

import (
	"time"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"google.golang.org/protobuf/types/known/durationpb"

	common_config_pb "github.com/prodvana/prodvana-public/go/prodvana-sdk/proto/prodvana/common_config"
	prot_pb "github.com/prodvana/prodvana-public/go/prodvana-sdk/proto/prodvana/protection"
)

type parameterValue struct {
	Name                types.String `tfsdk:"name"`
	StringValue         types.String `tfsdk:"string_value"`
	IntValue            types.Int64  `tfsdk:"int_value"`
	DockerImageTagValue types.String `tfsdk:"docker_image_tag_value"`
	SecretValue         *envSecret   `tfsdk:"secret_value"`
}

type protectionReference struct {
	Name       types.String      `tfsdk:"name"`
	Parameters []*parameterValue `tfsdk:"parameters"`
}
type preApproval struct {
	Enabled types.Bool `tfsdk:"enabled"`
}
type postApproval struct {
	Enabled types.Bool `tfsdk:"enabled"`
}
type deployment struct {
	Enabled types.Bool `tfsdk:"enabled"`
}

type postDeployment struct {
	Enabled            types.Bool   `tfsdk:"enabled"`
	DelayCheckDuration types.String `tfsdk:"delay_check_duration"`
	CheckDuration      types.String `tfsdk:"check_duration"`
}

type protectionAttachment struct {
	Name types.String         `tfsdk:"name"`
	Ref  *protectionReference `tfsdk:"ref"`

	PreApproval    *preApproval    `tfsdk:"pre_approval"`
	PostApproval   *postApproval   `tfsdk:"post_approval"`
	Deployment     *deployment     `tfsdk:"deployment"`
	PostDeployment *postDeployment `tfsdk:"post_deployment"`
}

func (pa *protectionAttachment) AsProto() (*prot_pb.ProtectionAttachmentConfig, error) {
	params := []*common_config_pb.ParameterValue{}
	for _, param := range pa.Ref.Parameters {
		if !param.StringValue.IsNull() {
			params = append(params, &common_config_pb.ParameterValue{
				Name: param.Name.ValueString(),
				ValueOneof: &common_config_pb.ParameterValue_String_{
					String_: param.StringValue.ValueString(),
				},
			})
		} else if !param.IntValue.IsNull() {
			params = append(params, &common_config_pb.ParameterValue{
				Name: param.Name.ValueString(),
				ValueOneof: &common_config_pb.ParameterValue_Int{
					Int: param.IntValue.ValueInt64(),
				},
			})
		} else if !param.DockerImageTagValue.IsNull() {
			params = append(params, &common_config_pb.ParameterValue{
				Name: param.Name.ValueString(),
				ValueOneof: &common_config_pb.ParameterValue_DockerImageTag{
					DockerImageTag: param.DockerImageTagValue.ValueString(),
				},
			})
		} else if param.SecretValue != nil {
			params = append(params, &common_config_pb.ParameterValue{
				Name: param.Name.ValueString(),
				ValueOneof: &common_config_pb.ParameterValue_Secret{
					Secret: &common_config_pb.SecretParameterValue{
						SecretOneof: &common_config_pb.SecretParameterValue_SecretRef{
							SecretRef: &common_config_pb.Secret{
								Key:     param.SecretValue.Key.ValueString(),
								Version: param.SecretValue.Version.ValueString(),
							},
						},
					},
				},
			})
		}
	}

	lifecycles := []*prot_pb.ProtectionLifecycle{}
	if pa.PreApproval != nil && pa.PreApproval.Enabled.ValueBool() {
		lifecycles = append(lifecycles, &prot_pb.ProtectionLifecycle{
			Lifecycle: &prot_pb.ProtectionLifecycle_PreApproval_{
				PreApproval: &prot_pb.ProtectionLifecycle_PreApproval{},
			},
		})
	}
	if pa.PostApproval != nil && pa.PostApproval.Enabled.ValueBool() {
		lifecycles = append(lifecycles, &prot_pb.ProtectionLifecycle{
			Lifecycle: &prot_pb.ProtectionLifecycle_PostApproval_{
				PostApproval: &prot_pb.ProtectionLifecycle_PostApproval{},
			},
		})
	}
	if pa.Deployment != nil && pa.Deployment.Enabled.ValueBool() {
		lifecycles = append(lifecycles, &prot_pb.ProtectionLifecycle{
			Lifecycle: &prot_pb.ProtectionLifecycle_Deployment_{
				Deployment: &prot_pb.ProtectionLifecycle_Deployment{},
			},
		})
	}
	if pa.PostDeployment != nil && pa.PostDeployment.Enabled.ValueBool() {
		delayDuration, err := time.ParseDuration(pa.PostDeployment.DelayCheckDuration.ValueString())
		if err != nil {
			return nil, err
		}
		checkDuration, err := time.ParseDuration(pa.PostDeployment.CheckDuration.ValueString())
		if err != nil {
			return nil, err
		}
		lifecycles = append(lifecycles, &prot_pb.ProtectionLifecycle{
			Lifecycle: &prot_pb.ProtectionLifecycle_PostDeployment_{
				PostDeployment: &prot_pb.ProtectionLifecycle_PostDeployment{
					DelayCheckDuration: durationpb.New(delayDuration),
					CheckDuration:      durationpb.New(checkDuration),
				},
			},
		})
	}
	return &prot_pb.ProtectionAttachmentConfig{
		Name: pa.Name.ValueString(),
		Ref: &prot_pb.ProtectionReference{
			Name:       pa.Ref.Name.ValueString(),
			Parameters: params,
		},
		Lifecycle: lifecycles,
	}, nil
}

func ProtectionAttachmentProtoToTerraform(pa *prot_pb.ProtectionAttachmentConfig) *protectionAttachment {
	params := []*parameterValue{}
	for _, param := range pa.Ref.Parameters {
		switch param.ValueOneof.(type) {
		case *common_config_pb.ParameterValue_String_:
			params = append(params, &parameterValue{
				Name:        types.StringValue(param.Name),
				StringValue: types.StringValue(param.GetString_()),
			})
		case *common_config_pb.ParameterValue_Int:
			params = append(params, &parameterValue{
				Name:     types.StringValue(param.Name),
				IntValue: types.Int64Value(param.GetInt()),
			})
		case *common_config_pb.ParameterValue_DockerImageTag:
			params = append(params, &parameterValue{
				Name:                types.StringValue(param.Name),
				DockerImageTagValue: types.StringValue(param.GetDockerImageTag()),
			})
		case *common_config_pb.ParameterValue_Secret:
			params = append(params, &parameterValue{
				Name: types.StringValue(param.Name),
				SecretValue: &envSecret{
					Key:     types.StringValue(param.GetSecret().GetSecretRef().Key),
					Version: types.StringValue(param.GetSecret().GetSecretRef().Version),
				},
			})
		}
	}

	attachment := &protectionAttachment{
		Name: types.StringValue(pa.Name),
		Ref: &protectionReference{
			Name:       types.StringValue(pa.Ref.Name),
			Parameters: params,
		},
	}
	for _, lifecycle := range pa.Lifecycle {
		switch lifecycle.Lifecycle.(type) {
		case *prot_pb.ProtectionLifecycle_PreApproval_:
			attachment.PreApproval = &preApproval{
				Enabled: types.BoolValue(true),
			}
		case *prot_pb.ProtectionLifecycle_PostApproval_:
			attachment.PostApproval = &postApproval{
				Enabled: types.BoolValue(true),
			}
		case *prot_pb.ProtectionLifecycle_Deployment_:
			attachment.Deployment = &deployment{
				Enabled: types.BoolValue(true),
			}
		case *prot_pb.ProtectionLifecycle_PostDeployment_:
			delayDuration := lifecycle.GetPostDeployment().DelayCheckDuration.AsDuration()
			checkDuration := lifecycle.GetPostDeployment().CheckDuration.AsDuration()
			attachment.PostDeployment = &postDeployment{
				Enabled:            types.BoolValue(true),
				DelayCheckDuration: types.StringValue(delayDuration.String()),
				CheckDuration:      types.StringValue(checkDuration.String()),
			}
		}
	}
	return attachment
}
