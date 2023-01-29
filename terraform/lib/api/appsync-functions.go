package api

import (
	"github.com/ContinentalBreakfast17/peppy/terraform/lib/_common"
	"github.com/aws/jsii-runtime-go"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/appsyncfunction"
)

type appsyncFunctions struct {
	Regions map[string]appsyncFunctionsInstance
}

type appsyncFunctionsInstance struct {
	CacheIp             AppsyncFunction
	CheckIpCache        AppsyncFunction
	EnqueueUnrankedSolo AppsyncFunction
	HealthResponse      AppsyncFunction
	LookupIp            AppsyncFunction
	PostHealthcheck     AppsyncFunction
	PublishMatch        AppsyncFunction
}

type appsyncFunctionsConfig struct {
	providers common.Providers
	apis      appsyncApi
	vtl       map[string]*string
}

type appsyncFunctionsInstanceConfig struct {
	appsyncFunctionsConfig
	region string
	api    appsyncApiInstance
}

func (cfg appsyncFunctionsConfig) new(ctx common.TfContext) appsyncFunctions {
	// create an instance of the service in each region
	instances := map[string]appsyncFunctionsInstance{}
	for region, provider := range cfg.providers {
		instances[region] = appsyncFunctionsInstanceConfig{
			appsyncFunctionsConfig: cfg,
			region:                 region,
			api:                    cfg.apis.Regions[region],
		}.new(common.SimpleContext(ctx.Scope, ctx.Id+"_"+region, provider))
	}

	return appsyncFunctions{instances}
}

func (cfg appsyncFunctionsInstanceConfig) new(ctx common.TfContext) appsyncFunctionsInstance {
	return appsyncFunctionsInstance{
		CacheIp: NewAppsyncFunction(ctx.Scope, jsii.String(ctx.Id+"_cache_ip"), &AppsyncFunctionConfig{
			Provider:                ctx.Provider,
			Name:                    jsii.String("cache_ip"),
			ApiId:                   cfg.api.Api.Id(),
			DataSource:              cfg.api.DataSources.IpCache.Name(),
			RequestMappingTemplate:  cfg.vtl["cache-ip.req.vm"],
			ResponseMappingTemplate: cfg.vtl["cache-ip.resp.vm"],
		}),
		CheckIpCache: NewAppsyncFunction(ctx.Scope, jsii.String(ctx.Id+"_check_ip_cache"), &AppsyncFunctionConfig{
			Provider:                ctx.Provider,
			Name:                    jsii.String("check_ip_cache"),
			ApiId:                   cfg.api.Api.Id(),
			DataSource:              cfg.api.DataSources.IpCache.Name(),
			RequestMappingTemplate:  cfg.vtl["check-ip-cache.req.vm"],
			ResponseMappingTemplate: cfg.vtl["check-ip-cache.resp.vm"],
		}),
		EnqueueUnrankedSolo: NewAppsyncFunction(ctx.Scope, jsii.String(ctx.Id+"_enqueue_unranked_solo"), &AppsyncFunctionConfig{
			Provider:                ctx.Provider,
			Name:                    jsii.String("enqueue_unranked_solo"),
			ApiId:                   cfg.api.Api.Id(),
			DataSource:              cfg.api.DataSources.Queues.UnrankedSolo.Name(),
			RequestMappingTemplate:  cfg.vtl["enqueue.req.vm"],
			ResponseMappingTemplate: cfg.vtl["enqueue.resp.vm"],
		}),
		HealthResponse: NewAppsyncFunction(ctx.Scope, jsii.String(ctx.Id+"_health_response"), &AppsyncFunctionConfig{
			Provider:                ctx.Provider,
			Name:                    jsii.String("health_response"),
			ApiId:                   cfg.api.Api.Id(),
			DataSource:              cfg.api.DataSources.Healthcheck.Name(),
			RequestMappingTemplate:  cfg.vtl["healthcheck-response.req.vm"],
			ResponseMappingTemplate: cfg.vtl["healthcheck-response.resp.vm"],
		}),
		LookupIp: NewAppsyncFunction(ctx.Scope, jsii.String(ctx.Id+"_lookup_ip"), &AppsyncFunctionConfig{
			Provider:                ctx.Provider,
			Name:                    jsii.String("lookup_ip"),
			ApiId:                   cfg.api.Api.Id(),
			DataSource:              cfg.api.DataSources.IpLookup.Name(),
			RequestMappingTemplate:  cfg.vtl["lookup-ip.req.vm"],
			ResponseMappingTemplate: cfg.vtl["lookup-ip.resp.vm"],
		}),
		PostHealthcheck: NewAppsyncFunction(ctx.Scope, jsii.String(ctx.Id+"_post_healthcheck"), &AppsyncFunctionConfig{
			Provider:                ctx.Provider,
			Name:                    jsii.String("post_healthcheck"),
			ApiId:                   cfg.api.Api.Id(),
			DataSource:              cfg.api.DataSources.Healthcheck.Name(),
			RequestMappingTemplate:  cfg.vtl["post-healthcheck.req.vm"],
			ResponseMappingTemplate: cfg.vtl["post-healthcheck.resp.vm"],
		}),
		PublishMatch: NewAppsyncFunction(ctx.Scope, jsii.String(ctx.Id+"_publish_match"), &AppsyncFunctionConfig{
			Provider:                ctx.Provider,
			Name:                    jsii.String("publish_match"),
			ApiId:                   cfg.api.Api.Id(),
			DataSource:              cfg.api.DataSources.Noop.Name(),
			RequestMappingTemplate:  cfg.vtl["match.req.vm"],
			ResponseMappingTemplate: cfg.vtl["match.resp.vm"],
		}),
	}
}
