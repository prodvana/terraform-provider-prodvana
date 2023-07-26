package pulumi

import (
	tfp "github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/prodvana/terraform-provider-prodvana/internal/provider"
	"github.com/prodvana/terraform-provider-prodvana/version"
)

// This module exposes the internal Provider package for the bridged
// Pulumi Provider to use as an entrypoint.

func NewProvider() tfp.Provider {
	return provider.New(version.Version)()
}

func NewProviderWithVersion(version string) tfp.Provider {
	return provider.New(version)()
}
