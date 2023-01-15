package api

import (
	"github.com/ContinentalBreakfast17/peppy/terraform/lib/_common"
	"github.com/aws/jsii-runtime-go"
)

type api struct {
	tables apiTables
	role   appsyncRole
	cert   appsyncCert
}

type ApiConfig struct {
	Providers         common.Providers
	Name              *string
	IamPath           *string
	KmsWritePolicy    *string
	KmsArns           common.MultiRegionId
	DomainName        *string
	HostedZoneId      *string
	Queues            ApiQueueConfig
	FunctionsIpLookup map[string]common.ArnIdPair
	TablesHealthcheck map[string]common.ArnIdPair
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

	role := appsyncRoleConfig{
		name:              jsii.String(*cfg.Name + "-appsync"),
		path:              cfg.IamPath,
		kmsWritePolicy:    cfg.KmsWritePolicy,
		queues:            cfg.Queues.toList(),
		functionsIpLookup: cfg.FunctionsIpLookup,
		tablesHealthcheck: cfg.TablesHealthcheck,
		tablesUser:        tables.userTableIds(),
		tablesIpCache:     tables.ipCacheTableIds(),
	}.new(common.SimpleContext(ctx.Scope, ctx.Id+"_appsync_role", ctx.Provider))

	cert := appsyncCertConfig{
		domainName: cfg.DomainName,
		hostedZone: cfg.HostedZoneId,
	}.new(common.SimpleContext(ctx.Scope, ctx.Id+"_cert", ctx.Provider))

	return api{
		tables: tables,
		role:   role,
		cert:   cert,
	}
}
