package defaults

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/defaults"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type envDefaultValue struct {
	envName string
}

// Description returns a plain text description of the default's behavior, suitable for a practitioner to understand its impact.
func (d envDefaultValue) Description(ctx context.Context) string {
	return fmt.Sprintf("If this value is not passed, it defaults to the value of %s", d.envName)
}

// MarkdownDescription returns a markdown formatted description of the default's behavior, suitable for a practitioner to understand its impact.
func (d envDefaultValue) MarkdownDescription(ctx context.Context) string {
	return fmt.Sprintf("If this value is not passed, it defaults to the value of %s", d.envName)
}

func (d envDefaultValue) DefaultString(ctx context.Context, req defaults.StringRequest, resp *defaults.StringResponse) {
	value := os.Getenv(d.envName)
	if value == "" {
		resp.PlanValue = types.StringNull()
	} else {
		resp.PlanValue = types.StringValue(value)
	}
}

func EnvStringValue(envName string) defaults.String {
	return envDefaultValue{envName: envName}
}

type envDefaultBoolValue struct {
	envName      string
	defaultValue bool
}

// Description returns a plain text description of the default's behavior, suitable for a practitioner to understand its impact.
func (d envDefaultBoolValue) Description(ctx context.Context) string {
	return fmt.Sprintf("If this value is not passed, it defaults to the value of %s (true when the env value is set to 'true', '1', or 'on')", d.envName)
}

// MarkdownDescription returns a markdown formatted description of the default's behavior, suitable for a practitioner to understand its impact.
func (d envDefaultBoolValue) MarkdownDescription(ctx context.Context) string {
	return fmt.Sprintf("If this value is not passed, it defaults to the value of %s (true when the env value is set to 'true', '1', or 'on')", d.envName)
}

func (d envDefaultBoolValue) DefaultBool(ctx context.Context, req defaults.BoolRequest, resp *defaults.BoolResponse) {
	resp.PlanValue = types.BoolValue(d.defaultValue)
	value := os.Getenv(d.envName)
	if value != "" {
		resp.PlanValue = types.BoolValue(value == "true" || value == "1" || value == "on")
	}
}

func EnvBoolValue(envName string, defaultValue bool) defaults.Bool {
	return envDefaultBoolValue{envName: envName, defaultValue: defaultValue}
}

type envDefaultPathListValue struct {
	envName string
}

// Description returns a plain text description of the default's behavior, suitable for a practitioner to understand its impact.
func (d envDefaultPathListValue) Description(ctx context.Context) string {
	return fmt.Sprintf("If this value is not passed, it defaults to the value of %s (comma separated)", d.envName)
}

// MarkdownDescription returns a markdown formatted description of the default's behavior, suitable for a practitioner to understand its impact.
func (d envDefaultPathListValue) MarkdownDescription(ctx context.Context) string {
	return fmt.Sprintf("If this value is not passed, it defaults to the value of %s (comma separated)", d.envName)
}

func (d envDefaultPathListValue) DefaultList(ctx context.Context, req defaults.ListRequest, resp *defaults.ListResponse) {
	value := os.Getenv(d.envName)
	if value != "" {
		resp.PlanValue = types.ListNull(types.StringType)
	} else {
		paths := filepath.SplitList(value)
		attrPaths := make([]attr.Value, len(paths))
		for i, path := range paths {
			attrPaths[i] = types.StringValue(path)
		}
		listValue, diags := types.ListValue(types.StringType, attrPaths)
		if diags.HasError() {
			resp.Diagnostics.Append(diags...)
			return
		}
		resp.PlanValue = listValue
	}

}

func EnvPathListValue(envName string) defaults.List {
	return envDefaultPathListValue{envName: envName}
}
