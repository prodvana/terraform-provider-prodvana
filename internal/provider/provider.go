package provider

import (
	"context"
	"crypto/tls"
	"fmt"
	"os"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
)

// Ensure ProdvanaProvider satisfies various provider interfaces.
var _ provider.Provider = &ProdvanaProvider{}

// ProdvanaProvider defines the provider implementation.
type ProdvanaProvider struct {
	// version is set to the provider version on release, "dev" when the
	// provider is built and ran locally, and "test" when running acceptance
	// testing.
	version string
}

// ProdvanaProviderModel describes the provider data model.
type ProdvanaProviderModel struct {
	OrgSlug  types.String `tfsdk:"org_slug"`
	ApiToken types.String `tfsdk:"api_token"`
}

type AuthToken struct {
	Token string
}

func (t AuthToken) GetRequestMetadata(ctx context.Context, in ...string) (map[string]string, error) {
	return map[string]string{
		"authorization": "Bearer " + t.Token,
	}, nil
}

func (AuthToken) RequireTransportSecurity() bool {
	return true
}

func (p *ProdvanaProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "prodvana"
	resp.Version = p.version
}

func (p *ProdvanaProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Attributes: map[string]schema.Attribute{
			"org_slug": schema.StringAttribute{
				MarkdownDescription: "Prodvana organization to authenticate with (you can find this in your Org's url: <org>.prodvana.io)",
				// Optional because we support passing as an environment variable, see Configure
				Optional: true,
			},
			"api_token": schema.StringAttribute{
				MarkdownDescription: "An API token generated with permissions to this organization.",
				Sensitive:           true,
				// Optional because we support passing as an environment variable, see Configure
				Optional: true,
			},
		},
	}
}

func (p *ProdvanaProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var data ProdvanaProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)

	if data.OrgSlug.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("org_slug"),
			"Unknown Prodvana Org Slug",
			"The provider cannot create  a Prodvana API client as there is an unknown configuration value for the org_slug."+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the PVN_ORG_SLUG environment variable.",
		)
	}

	if data.ApiToken.IsUnknown() {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_token"),
			"Unknown Prodvana API Token",
			"The provider cannot create  a Prodvana API client as there is an unknown configuration value for the api_token."+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the PVN_API_TOKEN environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	// Default values to environment variables, but override
	// with Terraform configuration value if set.

	orgSlug := os.Getenv("PVN_ORG_SLUG")
	apiToken := os.Getenv("PVN_API_TOKEN")

	if !data.OrgSlug.IsNull() {
		orgSlug = data.OrgSlug.ValueString()
	}

	if !data.ApiToken.IsNull() {
		apiToken = data.ApiToken.ValueString()
	}

	// If any of the expected configurations are missing, return
	// errors with provider-specific guidance.

	if orgSlug == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("org_slug"),
			"Missing Prodvana Org Slug",
			"The provider cannot create  a Prodvana API client as there is an unknown configuration value for the org_slug."+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the PVN_ORG_SLUG environment variable."+
				"If either is already set, ensure the value is not empty.",
		)
	}

	if apiToken == "" {
		resp.Diagnostics.AddAttributeError(
			path.Root("api_token"),
			"Missing Prodvana API Token",
			"The provider cannot create  a Prodvana API client as there is an unknown configuration value for the api_token."+
				"Either target apply the source of the value first, set the value statically in the configuration, or use the PVN_API_TOKEN environment variable.",
		)
	}

	if resp.Diagnostics.HasError() {
		return
	}

	ctx = tflog.SetField(ctx, "prodvana_org_slug", orgSlug)
	ctx = tflog.SetField(ctx, "prodvana_api_token", apiToken)
	ctx = tflog.MaskFieldValuesWithFieldKeys(ctx, "prodvana_api_token")

	tflog.Debug(ctx, "Creating Prodvana client")

	domain := fmt.Sprintf("api.%s.prodvana.io", orgSlug)
	cred := credentials.NewTLS(&tls.Config{ServerName: domain})
	options := []grpc.DialOption{
		grpc.WithTransportCredentials(cred),
		grpc.WithPerRPCCredentials(AuthToken{Token: apiToken}),
	}

	conn, err := grpc.Dial(domain+":443", options...)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Prodvana API Client",
			"An unexpected error occurred when creating the Prodvana API client. "+
				"If the error is not clear, please contact the provider developers.\n\n"+
				"Prodvana Client Error: "+err.Error(),
		)
		return
	}

	resp.DataSourceData = conn
	resp.ResourceData = conn
}

func (p *ProdvanaProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		NewApplicationResource,
		NewReleaseChannelResource,
		NewRuntimeResource,
		NewRuntimeLinkResource,
		NewManagedK8sRuntimeResource,
	}
}

func (p *ProdvanaProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		NewApplicationDataSource,
		NewReleaseChannelDataSource,
		NewRuntimeDataSource,
	}
}

func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &ProdvanaProvider{
			version: version,
		}
	}
}
