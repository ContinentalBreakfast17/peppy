package base

import (
	"github.com/ContinentalBreakfast17/peppy/terraform/lib/_common"
	"github.com/aws/jsii-runtime-go"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/provider"
)

type providers struct {
	Main   AwsProvider
	Copies map[string]AwsProvider
}

type providerConfig struct {
	regions []string
	tags    *map[string]*string
	name    *string
}

func (cfg providerConfig) new(ctx common.TfContext) providers {
	if cfg.tags == nil || *cfg.tags == nil {
		m := map[string]*string{}
		cfg.tags = &m
	}

	// we _need_ us-east-1
	region := "us-east-1"
	main := NewAwsProvider(ctx.Scope, jsii.String(ctx.Id+"_"+region), &AwsProviderConfig{
		Region: jsii.String(region),
		DefaultTags: &AwsProviderDefaultTags{
			Tags: cfg.getTags(region),
		},
	})

	// additional regions to deploy to for HA
	copies := map[string]AwsProvider{}
	for _, region := range cfg.regions {
		copies[region] = NewAwsProvider(ctx.Scope, jsii.String(ctx.Id+"_"+region), &AwsProviderConfig{
			Region: jsii.String(region),
			Alias:  jsii.String(region),
			DefaultTags: &AwsProviderDefaultTags{
				Tags: cfg.getTags(region),
			},
		})
	}

	return providers{main, copies}
}

func (cfg providerConfig) getTags(region string) *map[string]*string {
	m := map[string]*string{"region": jsii.String(region), "app": cfg.name}
	for key, val := range *cfg.tags {
		m[key] = val
	}
	return &m
}

func (p providers) All() map[string]AwsProvider {
	m := map[string]AwsProvider{}
	m["us-east-1"] = p.Main
	for region, provider := range p.Copies {
		m[region] = provider
	}
	return m
}
