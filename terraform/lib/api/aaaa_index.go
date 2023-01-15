package api

import (
	"github.com/ContinentalBreakfast17/peppy/terraform/lib/_common"
)

type api struct {
	tables apiTables
}

type ApiConfig struct {
	Providers         common.Providers
	Name              *string
	KmsArns           common.MultiRegionId
	Queues            ApiQueueConfig
	FunctionIpLookup  map[string]common.ArnIdPair
	TablesHealthcheck map[string]common.ArnIdPair      
}

type ApiQueueConfig struct {
	UnrankedSolo queue
}

type queue interface {
	Name()   string
	Tables() map[string]common.ArnIdPair
}

func (cfg ApiConfig) New(ctx common.TfContext) api {
	tables := apiTablesConfig{
		providers: cfg.Providers,
		name:      cfg.Name,
		kmsArns:   cfg.KmsArns,
	}.new(common.SimpleContext(ctx.Scope, ctx.Id+"_tables", ctx.Provider))

	return api{
		tables: tables,
	}
}