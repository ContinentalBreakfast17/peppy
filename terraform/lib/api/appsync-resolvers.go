package api

import (
	"fmt"
	"strings"

	"github.com/ContinentalBreakfast17/peppy/terraform/lib/_common"
	"github.com/aws/jsii-runtime-go"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/appsyncresolver"
)

type appsyncResolvers struct {
	Regions map[string]appsyncResolversInstance
}

type appsyncResolversInstance struct {
	Query__region                       AppsyncResolver
	Mutation__publishHealth             AppsyncResolver
	Mutation__publishMatch              AppsyncResolver
	Subscription__healthcheck           AppsyncResolver
	Subscription__joinUnrankedSoloQueue AppsyncResolver
}

type appsyncResolversConfig struct {
	providers common.Providers
	apis      appsyncApi
	functions appsyncFunctions
	vtl       map[string]*string
}

type appsyncResolversInstanceConfig struct {
	appsyncResolversConfig
	region string
	api    appsyncApiInstance
	fns    appsyncFunctionsInstance
}

func (cfg appsyncResolversConfig) new(ctx common.TfContext) appsyncResolvers {
	// create an instance of the service in each region
	instances := map[string]appsyncResolversInstance{}
	for region, provider := range cfg.providers {
		instances[region] = appsyncResolversInstanceConfig{
			appsyncResolversConfig: cfg,
			region:                 region,
			api:                    cfg.apis.Regions[region],
			fns:                    cfg.functions.Regions[region],
		}.new(common.SimpleContext(ctx.Scope, ctx.Id+"_"+region, provider))
	}

	return appsyncResolvers{instances}
}

func (cfg appsyncResolversInstanceConfig) new(ctx common.TfContext) appsyncResolversInstance {
	requestTemplate := `$util.quiet($ctx.stash.put("entry_time", $util.time.nowEpochSeconds()))` + "\n"
	requestTemplate += `$util.quiet($ctx.stash.put("graphql", $ctx.info))` + "\n"
	requestTemplate += fmt.Sprintf(`$util.quiet($ctx.stash.put("region", "%s"))`, cfg.region) + "\n"
	requestTemplate += *cfg.vtl["_global.req.vm"] + "\n"
	requestTemplate += `{}`

	return appsyncResolversInstance{
		Query__region: NewAppsyncResolver(ctx.Scope, jsii.String(ctx.Id+"_Query_region"), &AppsyncResolverConfig{
			Provider:         ctx.Provider,
			ApiId:            cfg.api.Api.Id(),
			DataSource:       cfg.api.DataSources.Noop.Name(),
			Type:             jsii.String("Query"),
			Field:            jsii.String("region"),
			RequestTemplate:  jsii.String(`{ "version": "2017-02-28", "payload": {} }`),
			ResponseTemplate: jsii.String(fmt.Sprintf(`$util.toJson("%s")`, cfg.region)),
		}),
		Mutation__publishHealth: NewAppsyncResolver(ctx.Scope, jsii.String(ctx.Id+"_Mutation_publishHealth"), &AppsyncResolverConfig{
			Provider:         ctx.Provider,
			ApiId:            cfg.api.Api.Id(),
			Kind:             jsii.String("PIPELINE"),
			Type:             jsii.String("Mutation"),
			Field:            jsii.String("publishHealth"),
			RequestTemplate:  jsii.String(requestTemplate),
			ResponseTemplate: jsii.String(`$util.toJson({ "id": $ctx.args.id })`),
			PipelineConfig: &AppsyncResolverPipelineConfig{
				Functions: jsii.Strings(
					*cfg.fns.HealthResponse.FunctionId(),
				),
			},
		}),
		Mutation__publishMatch: NewAppsyncResolver(ctx.Scope, jsii.String(ctx.Id+"_Mutation_publishMatch"), &AppsyncResolverConfig{
			Provider:         ctx.Provider,
			ApiId:            cfg.api.Api.Id(),
			Kind:             jsii.String("PIPELINE"),
			Type:             jsii.String("Mutation"),
			Field:            jsii.String("publishMatch"),
			RequestTemplate:  jsii.String(requestTemplate),
			ResponseTemplate: jsii.String(`$util.toJson($ctx.result)`),
			PipelineConfig: &AppsyncResolverPipelineConfig{
				Functions: jsii.Strings(
					*cfg.fns.PublishMatch.FunctionId(),
				),
			},
		}),
		Subscription__healthcheck: NewAppsyncResolver(ctx.Scope, jsii.String(ctx.Id+"_Subscription_healthcheck"), &AppsyncResolverConfig{
			Provider: ctx.Provider,
			ApiId:    cfg.api.Api.Id(),
			Kind:     jsii.String("PIPELINE"),
			Type:     jsii.String("Subscription"),
			Field:    jsii.String("healthcheck"),
			RequestTemplate: jsii.String(strings.Join([]string{
				// using an empty ip will get info about the requesting lambda's ip
				`$util.quiet($ctx.stash.put("ip", ""))`,
				fmt.Sprintf(`$util.quiet($ctx.stash.put("healthcheck_table", "%s"))`, *cfg.api.DataSources.Healthcheck.DynamodbConfig().TableName()),
				requestTemplate,
			}, "\n")),
			ResponseTemplate: jsii.String("#return"),
			PipelineConfig: &AppsyncResolverPipelineConfig{
				Functions: jsii.Strings(
					*cfg.fns.LookupIp.FunctionId(),
					*cfg.fns.PostHealthcheck.FunctionId(),
				),
			},
		}),
		Subscription__joinUnrankedSoloQueue: NewAppsyncResolver(ctx.Scope, jsii.String(ctx.Id+"_Subscription_joinUnrankedSoloQueue"), &AppsyncResolverConfig{
			Provider: ctx.Provider,
			ApiId:    cfg.api.Api.Id(),
			Kind:     jsii.String("PIPELINE"),
			Type:     jsii.String("Subscription"),
			Field:    jsii.String("joinUnrankedSoloQueue"),
			RequestTemplate: jsii.String(strings.Join([]string{
				`$util.quiet($ctx.stash.put("mmrKey", "unrankedSolo"))`,
				// todo: consider removing dequeue entirely (no transaction, saves lot of cost)
				`$util.quiet($ctx.stash.put("dequeue_tables", []))`,
				fmt.Sprintf(`$util.quiet($ctx.stash.put("queue_table", "%s"))`, *cfg.api.DataSources.Queues.UnrankedSolo.DynamodbConfig().TableName()),
				requestTemplate,
			}, "\n")),
			ResponseTemplate: jsii.String("#return"),
			PipelineConfig: &AppsyncResolverPipelineConfig{
				Functions: jsii.Strings(
					*cfg.fns.CheckIpCache.FunctionId(),
					*cfg.fns.LookupIp.FunctionId(),
					*cfg.fns.CacheIp.FunctionId(),
					*cfg.fns.EnqueueUnrankedSolo.FunctionId(),
				),
			},
		}),
	}
}
