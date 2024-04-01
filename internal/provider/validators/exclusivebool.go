package validators

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// Exlusive Bool is a validator that ensures if the provided Bool attribute is true
// then the provided path expressions must unset or false. If the Bool attribute is
// false, then the path expressions must have a value.
func ExclusiveBool(expressions ...path.Expression) validator.Bool {
	return &exclusiveBoolValdiator{
		expressions: expressions,
	}
}

// Ensure our implementation satisfies the validator.Bool interface.
var _ validator.Bool = &exclusiveBoolValdiator{}

type exclusiveBoolValdiator struct {
	expressions path.Expressions
}

func (v exclusiveBoolValdiator) Description(_ context.Context) string {
	return fmt.Sprintf("If this Bool value is true, %s attributes must be unset", v.expressions)
}

// MarkdownDescription returns a Markdown formatted string describing the validator.
func (v exclusiveBoolValdiator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

// Validate performs the validation logic for the validator.
func (v exclusiveBoolValdiator) ValidateBool(ctx context.Context, req validator.BoolRequest, resp *validator.BoolResponse) {
	if req.ConfigValue.IsUnknown() {
		return
	}

	boolValue := (!req.ConfigValue.IsNull() && req.ConfigValue.ValueBool())

	if boolValue {
		// If the bool value is true, then the path expressions must be unset.
		for _, expression := range v.expressions {
			matchedPaths, diags := req.Config.PathMatches(ctx, expression)
			resp.Diagnostics.Append(diags...)

			if diags.HasError() {
				continue
			}

			for _, matchedPath := range matchedPaths {
				var matchedPathValue attr.Value

				diags := req.Config.GetAttribute(ctx, matchedPath, &matchedPathValue)

				resp.Diagnostics.Append(diags...)

				if diags.HasError() {
					continue
				}

				if matchedPathValue.IsNull() || matchedPathValue.IsUnknown() {
					continue
				}

				resp.Diagnostics.AddAttributeError(
					matchedPath,
					"Invalid Attribute Value",
					fmt.Sprintf("Must be unset when %s is true", req.Path),
				)
			}
		}
	} else {
		// If the bool value is false, then the path expressions must have a value.
		for _, expression := range v.expressions {
			matchedPaths, diags := req.Config.PathMatches(ctx, expression)

			resp.Diagnostics.Append(diags...)

			if diags.HasError() {
				continue
			}

			for _, matchedPath := range matchedPaths {
				var matchedPathValue attr.Value

				diags := req.Config.GetAttribute(ctx, matchedPath, &matchedPathValue)

				resp.Diagnostics.Append(diags...)

				if diags.HasError() {
					continue
				}

				if matchedPathValue.IsNull() || matchedPathValue.IsUnknown() {
					resp.Diagnostics.AddAttributeError(
						matchedPath,
						"Invalid Attribute Value",
						fmt.Sprintf("%s must be set when %s is false", matchedPath, req.Path),
					)
				}
			}
		}

	}
}
