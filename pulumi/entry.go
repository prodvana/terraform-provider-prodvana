package pulumi

import (
	tfp "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/prodvana/terraform-provider-prodvana/internal/provider"
	"github.com/prodvana/terraform-provider-prodvana/version"
)

func NewProvider() tfp.Provider {
	return provider.New(version.Version)()
}
