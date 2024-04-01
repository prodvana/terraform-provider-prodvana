package validators

import (
	"context"
	"fmt"
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/helpers/validatordiag"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

func DefaultNameValidators() []validator.String {
	return []validator.String{
		stringvalidator.LengthBetween(1, 63),
		stringvalidator.RegexMatches(
			regexp.MustCompile(`^[a-z]([a-z0-9-]*[a-z0-9]){0,1}$`),
			"must contain only lowercase alphanumeric characters, and start with a letter.",
		),
	}
}

func URLHasHTTPProtocolValidator() validator.String {
	return stringvalidator.RegexMatches(
		regexp.MustCompile(`^https?://`),
		"must start with http:// or https://",
	)
}

func CheckAttributeAtPath(path path.Expression, value attr.Value) validator.Object {
	return checkPathObjectValidator{
		path:  path,
		value: value,
	}
}

var _ validator.Object = checkPathObjectValidator{}

type checkPathObjectValidator struct {
	path  path.Expression
	value attr.Value
}

func (v checkPathObjectValidator) Description(_ context.Context) string {
	return fmt.Sprintf("value at path %s must be set to %s", v.path, v.value)
}

func (v checkPathObjectValidator) MarkdownDescription(ctx context.Context) string {
	return v.Description(ctx)
}

func (v checkPathObjectValidator) ValidateObject(ctx context.Context, req validator.ObjectRequest, resp *validator.ObjectResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		// If code block does not exist, config is valid.
		return
	}

	matchedPaths, diags := req.Config.PathMatches(ctx, v.path)

	resp.Diagnostics.Append(diags...)
	if diags.HasError() {
		return
	}

	for _, mp := range matchedPaths {
		var mpVal attr.Value
		diags := req.Config.GetAttribute(ctx, mp, &mpVal)
		resp.Diagnostics.Append(diags...)

		// Collect all errors
		if diags.HasError() {
			continue
		}

		// Delay validation until all involved attribute have a known value
		if mpVal.IsUnknown() {
			return
		}

		if !mpVal.Equal(v.value) {
			resp.Diagnostics.Append(validatordiag.InvalidAttributeCombinationDiagnostic(
				req.Path,
				fmt.Sprintf("Attribute %q must be set to %q be specified when %q is specified", mp, v.value, req.Path),
			))
		}
	}
}
