package labels

import (
	"context"
	"regexp"

	labels_pb "github.com/prodvana/prodvana-public/go/prodvana-sdk/proto/prodvana/labels"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	ds_schema "github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var LabelDefinitionObjectType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"label": types.StringType,
		"value": types.StringType,
	},
}

type LabelDefinition struct {
	Label string `tfsdk:"label"`
	Value string `tfsdk:"value"`
}

func (l LabelDefinition) ToProto() *labels_pb.LabelDefinition {
	return &labels_pb.LabelDefinition{
		Label: l.Label,
		Value: l.Value,
	}
}

func LabelDefinitionFromProto(label *labels_pb.LabelDefinition) LabelDefinition {
	return LabelDefinition{
		Label: label.Label,
		Value: label.Value,
	}
}

func LabelDefinitionsToProtos(labelDefinitions []LabelDefinition) []*labels_pb.LabelDefinition {
	labels := make([]*labels_pb.LabelDefinition, len(labelDefinitions))
	for idx, label := range labelDefinitions {
		labels[idx] = label.ToProto()
	}
	return labels
}

func LabelDefinitionProtosToTerraform(labelDefinitions []*labels_pb.LabelDefinition) []LabelDefinition {
	labels := make([]LabelDefinition, len(labelDefinitions))
	for idx, label := range labelDefinitions {
		labels[idx] = LabelDefinitionFromProto(label)
	}
	return labels
}
func LabelDefinitionsToTerraformListWithValidation(ctx context.Context, labelDefinitions []*labels_pb.LabelDefinition, userProvided []LabelDefinition, diags diag.Diagnostics) types.List {
	// we can't guarantee the order of the label definitions returned from the API, so rather than having Terraform think that
	// the labels have changed because the order is different, we'll validate that the labels returned from the API match the
	// user provided labels and then return the API provided labels in the same order, and let Terraform figure out if the
	// label values have changed.
	if len(labelDefinitions) != len(userProvided) {
		diags = append(diags, diag.NewErrorDiagnostic(
			"Inconsistent state",
			"The label definitions returned from the API do not match the user provided labels. This is an internal error.",
		))
		return types.List{}
	}
	labelDefinitionsMap := make(map[string]*labels_pb.LabelDefinition)
	for _, label := range labelDefinitions {
		labelDefinitionsMap[label.Label] = label
	}
	labels := make([]LabelDefinition, len(userProvided))
	for idx, label := range userProvided {
		if _, ok := labelDefinitionsMap[label.Label]; !ok {
			diags = append(diags, diag.NewErrorDiagnostic(
				"Inconsistent state",
				"The label definitions returned from the API do not match the user provided labels. This is an internal error.",
			))
			return types.List{}
		}
		labels[idx] = label
	}
	list, d := types.ListValueFrom(ctx, LabelDefinitionObjectType, labels)
	diags.Append(d...)
	return list
}

func LabelDefinitionsToTerraformList(ctx context.Context, labelDefinitions []*labels_pb.LabelDefinition, diags diag.Diagnostics) types.List {
	list, d := types.ListValueFrom(ctx, LabelDefinitionObjectType, LabelDefinitionProtosToTerraform(labelDefinitions))
	diags.Append(d...)
	return list
}
func LabelDefinitionsFromTerraformList(ctx context.Context, list types.List, diags diag.Diagnostics) []LabelDefinition {
	labelDefinitions := []LabelDefinition{}
	d := list.ElementsAs(ctx, &labelDefinitions, false)
	diags.Append(d...)
	return labelDefinitions
}

func LabelDefinitionProtosFromTerraformList(ctx context.Context, list types.List, diags diag.Diagnostics) []*labels_pb.LabelDefinition {
	labelDefinitions := LabelDefinitionsFromTerraformList(ctx, list, diags)
	if diags.HasError() {
		return []*labels_pb.LabelDefinition{}
	}
	return LabelDefinitionsToProtos(labelDefinitions)
}

func LabelDefinitionNestedObjectDataSourceSchema() ds_schema.NestedAttributeObject {
	return ds_schema.NestedAttributeObject{
		Attributes: map[string]ds_schema.Attribute{
			"label": schema.StringAttribute{
				MarkdownDescription: "Label name",
				Required:            true,
			},
			"value": schema.StringAttribute{
				MarkdownDescription: "Label value",
				Required:            true,
			},
		},
	}
}

func LabelDefinitionNestedObjectResourceSchema() schema.NestedAttributeObject {
	return schema.NestedAttributeObject{
		Attributes: map[string]schema.Attribute{
			"label": schema.StringAttribute{
				MarkdownDescription: "Label name",
				Required:            true,
				Validators: []validator.String{
					stringvalidator.LengthAtLeast(1),
					labelValueValidator(),
				},
			},
			"value": schema.StringAttribute{
				MarkdownDescription: "Label value",
				Required:            true,
				Validators: []validator.String{
					labelValueValidator(),
				},
			},
		},
	}
}

var labelValueRegex = regexp.MustCompile(`^[a-zA-Z0-9.\\\-_@+]*$`)

func labelValueValidator() validator.String {
	return stringvalidator.RegexMatches(
		labelValueRegex,
		"must contain only alphanumeric characters, @, -, _, \\, and start with a letter.",
	)
}
