package validators

import (
	"regexp"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
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
