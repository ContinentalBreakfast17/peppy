package api

import (
	"github.com/ContinentalBreakfast17/peppy/terraform/lib/_common"
	"github.com/aws/jsii-runtime-go"
)

type api struct {
	appsyncApi
	tables     apiTables
	authorizer apiAuthorizer
	role       appsyncRole
	cert       appsyncCert
}

type ApiConfig struct {
	Providers         common.Providers
	Name              *string
	Schema            string
	Vtl               map[string]*string
	IamPath           *string
	KmsWritePolicy    *string
	KmsReadPolicy     *string
	KmsArns           common.MultiRegionId
	DomainName        *string
	HostedZoneId      *string
	Code              common.ObjectConfig
	LambdaIam         common.LambdaIamConfig
	Queues            ApiQueueConfig
	FunctionsIpLookup map[string]common.ArnIdPair
	TablesHealthcheck map[string]common.ArnIdPair
	AlarmsHealthCheck map[string]common.ArnIdPair
}

type ApiQueueConfig struct {
	UnrankedSolo queue
}

func (queues ApiQueueConfig) toList() []queue {
	return []queue{queues.UnrankedSolo}
}

type queue interface {
	Name() string
	Tables() map[string]common.ArnIdPair
}

func (cfg ApiConfig) New(ctx common.TfContext) api {
	tables := apiTablesConfig{
		providers: cfg.Providers,
		name:      cfg.Name,
		kmsArns:   cfg.KmsArns,
	}.new(common.SimpleContext(ctx.Scope, ctx.Id+"_tables", ctx.Provider))

	authorizer := apiAuthorizerConfig{
		providers:     cfg.Providers,
		name:          jsii.String(*cfg.Name + "-authorizer"),
		tables:        tables,
		kmsReadPolicy: cfg.KmsReadPolicy,
		code:          cfg.Code,
		lambdaIam:     cfg.LambdaIam,
	}.new(common.SimpleContext(ctx.Scope, ctx.Id+"_authorizer", ctx.Provider))

	role := appsyncRoleConfig{
		name:                jsii.String(*cfg.Name + "-appsync"),
		path:                cfg.IamPath,
		kmsWritePolicy:      cfg.KmsWritePolicy,
		queues:              cfg.Queues.toList(),
		functionsIpLookup:   cfg.FunctionsIpLookup,
		functionsAuthorizer: authorizer.functionIds(),
		tablesHealthcheck:   cfg.TablesHealthcheck,
		tablesUser:          tables.userTableIds(),
		tablesIpCache:       tables.ipCacheTableIds(),
	}.new(common.SimpleContext(ctx.Scope, ctx.Id+"_appsync_role", ctx.Provider))

	cert := appsyncCertConfig{
		domainName: cfg.DomainName,
		hostedZone: cfg.HostedZoneId,
	}.new(common.SimpleContext(ctx.Scope, ctx.Id+"_cert", ctx.Provider))

	appsyncApi := appsyncApiConfig{
		providers:           cfg.Providers,
		name:                cfg.Name,
		schema:              cfg.Schema,
		domainName:          cfg.DomainName,
		hostedZone:          cfg.HostedZoneId,
		role:                role.Arn(),
		cert:                cert.Arn(),
		certValidation:      cert.Validation,
		queues:              cfg.Queues,
		functionsIpLookup:   cfg.FunctionsIpLookup,
		functionsAuthorizer: authorizer.functionIds(),
		tablesHealthcheck:   cfg.TablesHealthcheck,
		tablesUser:          tables.userTableIds(),
		tablesIpCache:       tables.ipCacheTableIds(),
		alarmsHealthCheck:   cfg.AlarmsHealthCheck,
	}.new(common.SimpleContext(ctx.Scope, ctx.Id, ctx.Provider))

	appsyncFunctions := appsyncFunctionsConfig{
		providers: cfg.Providers,
		apis:      appsyncApi,
		vtl:       cfg.Vtl,
	}.new(common.SimpleContext(ctx.Scope, ctx.Id+"_functions", ctx.Provider))

	appsyncResolversConfig{
		providers: cfg.Providers,
		apis:      appsyncApi,
		functions: appsyncFunctions,
		vtl:       cfg.Vtl,
	}.new(common.SimpleContext(ctx.Scope, ctx.Id+"_resolvers", ctx.Provider))

	return api{
		tables:     tables,
		authorizer: authorizer,
		role:       role,
		cert:       cert,
		appsyncApi: appsyncApi,
	}
}
