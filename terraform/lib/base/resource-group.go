package base

import (
	"encoding/json"

	"github.com/ContinentalBreakfast17/peppy/terraform/lib/_common"
	"github.com/aws/jsii-runtime-go"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/resourcegroupsgroup"
)

type resourceGroups struct {
	Regions map[string]ResourcegroupsGroup
}

type resourceGroupConfig struct {
	providers providers
	name      *string
}

func (cfg resourceGroupConfig) new(ctx common.TfContext) resourceGroups {
	filter := map[string]any{
		"ResourceTypeFilters": []string{"AWS::AllSupported"},
		"TagFilters": []map[string]any{
			{
				"Key":    "app",
				"Values": []string{*cfg.name},
			},
		},
	}
	filterBytes, _ := json.Marshal(filter)

	regions := map[string]ResourcegroupsGroup{}
	for region, provider := range cfg.providers.All() {
		regions[region] = NewResourcegroupsGroup(ctx.Scope, jsii.String(ctx.Id+"_"+region), &ResourcegroupsGroupConfig{
			Provider: provider,
			Name:     cfg.name,
			ResourceQuery: &ResourcegroupsGroupResourceQuery{
				Type:  jsii.String("TAG_FILTERS_1_0"),
				Query: jsii.String(string(filterBytes)),
			},
		})
	}

	return resourceGroups{regions}
}
