package provider

import (
	"github.com/aws/jsii-runtime-go"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/provider"
	"github.com/ContinentalBreakfast17/peppy/terraform/lib/_common"
)

type providers struct {
	Main   AwsProvider
	Copies map[string]AwsProvider
}

type ProviderConfig struct {
	Regions []string
	Tags    *map[string]*string
}

func (cfg ProviderConfig) NewProviders(ctx common.TfContext) providers {
	if cfg.Tags == nil || *cfg.Tags == nil {
		m := map[string]*string{}
		cfg.Tags = &m
	}

	// we _need_ us-east-1
	region := "us-east-1"
	main := NewAwsProvider(ctx.Scope, jsii.String(ctx.Id+"_"+region), &AwsProviderConfig{
		Region: jsii.String(region),
		DefaultTags: &AwsProviderDefaultTags{
			Tags: cfg.tags(region),
		},
	})

	// additional regions to deploy to for HA
	copies := map[string]AwsProvider{}
	for _, region := range cfg.Regions {
		copies[region] = NewAwsProvider(ctx.Scope, jsii.String(ctx.Id+"_"+region), &AwsProviderConfig{
			Region: jsii.String(region),
			Alias:  jsii.String(region),
			DefaultTags: &AwsProviderDefaultTags{
				Tags: cfg.tags(region),
			},
		})
	}

	return providers{main, copies}
}

func (cfg ProviderConfig) tags(region string) *map[string]*string {
	m := map[string]*string{"region": jsii.String(region)}
	for key, val := range *cfg.Tags {
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
