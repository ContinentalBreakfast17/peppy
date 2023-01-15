package base

import (
	"github.com/ContinentalBreakfast17/peppy/terraform/lib/_common"
	"github.com/aws/jsii-runtime-go"
)

type base struct {
	Providers   providers
	DataSources dataSources
	KmsMain     keySet
	Policies    policies
}

type BaseConfig struct {
	Regions        []string
	Tags           *map[string]*string
	AdminGroupName *string
	Name           *string
	IamPath        *string
	Domain         *string
}

func (cfg BaseConfig) New(ctx common.TfContext) base {
	providers := providerConfig{
		regions: cfg.Regions,
		tags:    cfg.Tags,
		name:    cfg.Name,
	}.new(common.SimpleContext(ctx.Scope, ctx.Id+"_providers", nil))

	datasources := dataSourceConfig{
		adminGroupName: cfg.AdminGroupName,
		domain:         cfg.Domain,
	}.new(common.SimpleContext(ctx.Scope, ctx.Id+"_data", providers.Main))

	kmsMain := keyConfig{
		providers:   providers,
		name:        jsii.String(*cfg.Name + "-main"),
		description: jsii.String(*cfg.Name + " main key"),
		accountId:   datasources.AccountId(),
		keyAdmins:   *datasources.AdminUsers(),
	}.new(common.SimpleContext(ctx.Scope, ctx.Id+"_kms_main", providers.Main))

	policies := policyConfig{
		kmsMain: kmsMain,
		path:    cfg.IamPath,
		name:    cfg.Name,
	}.new(common.SimpleContext(ctx.Scope, ctx.Id+"_policies", providers.Main))

	resourceGroupConfig{
		providers: providers,
		name:      cfg.Name,
	}.new(common.SimpleContext(ctx.Scope, ctx.Id+"_rgs", providers.Main))

	return base{
		Providers:   providers,
		DataSources: datasources,
		KmsMain:     kmsMain,
		Policies:    policies,
	}
}
