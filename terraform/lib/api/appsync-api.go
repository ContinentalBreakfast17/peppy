package api

import (
	"github.com/ContinentalBreakfast17/peppy/terraform/lib/_common"
	"github.com/aws/jsii-runtime-go"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/acmcertificatevalidation"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/appsyncdatasource"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/appsyncdomainname"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/appsyncdomainnameapiassociation"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/appsyncgraphqlapi"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/lambdapermission"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/route53healthcheck"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/route53record"
	"github.com/hashicorp/terraform-cdk-go/cdktf"
)

type appsyncApi struct {
	Regions map[string]appsyncApiInstance
}

type appsyncApiInstance struct {
	Api         AppsyncGraphqlApi
	DataSources appsyncDataSources
	DomainName  AppsyncDomainName
}

type appsyncDataSources struct {
	Noop        AppsyncDatasource
	IpLookup    AppsyncDatasource
	IpCache     AppsyncDatasource
	User        AppsyncDatasource
	Healthcheck AppsyncDatasource
	Queues      appsyncQueueDataSources
}

type appsyncQueueDataSources struct {
	UnrankedSolo AppsyncDatasource
}

type appsyncApiConfig struct {
	providers           common.Providers
	name                *string
	schema              string
	domainName          *string
	hostedZone          *string
	cert                *string
	certValidation      AcmCertificateValidation
	role                *string
	queues              ApiQueueConfig
	functionsIpLookup   map[string]common.ArnIdPair
	functionsAuthorizer map[string]common.ArnIdPair
	tablesHealthcheck   map[string]common.ArnIdPair
	tablesUser          map[string]common.ArnIdPair
	tablesIpCache       map[string]common.ArnIdPair
	alarmsHealthCheck   map[string]common.ArnIdPair
}

type appsyncApiInstanceConfig struct {
	appsyncApiConfig
	region string
}

func (cfg appsyncApiConfig) new(ctx common.TfContext) appsyncApi {
	// create an instance of the service in each region
	instances := map[string]appsyncApiInstance{}
	for region, provider := range cfg.providers {
		instances[region] = appsyncApiInstanceConfig{
			appsyncApiConfig: cfg,
			region:           region,
		}.new(common.SimpleContext(ctx.Scope, ctx.Id+"_"+region, provider))
	}

	return appsyncApi{instances}
}

func (cfg appsyncApiInstanceConfig) new(ctx common.TfContext) appsyncApiInstance {
	api := NewAppsyncGraphqlApi(ctx.Scope, jsii.String(ctx.Id+"_api"), &AppsyncGraphqlApiConfig{
		Provider:           ctx.Provider,
		Name:               cfg.name,
		Schema:             jsii.String(cfg.schema),
		AuthenticationType: jsii.String("AWS_IAM"),
		AdditionalAuthenticationProvider: &[]AppsyncGraphqlApiAdditionalAuthenticationProvider{
			{
				AuthenticationType: jsii.String("AWS_LAMBDA"),
				LambdaAuthorizerConfig: &AppsyncGraphqlApiAdditionalAuthenticationProviderLambdaAuthorizerConfig{
					AuthorizerUri:                cfg.functionsAuthorizer[cfg.region].Arn,
					AuthorizerResultTtlInSeconds: jsii.Number(60),
				},
			},
		},
		LogConfig: &AppsyncGraphqlApiLogConfig{
			CloudwatchLogsRoleArn: cfg.role,
			FieldLogLevel:         jsii.String("ALL"),
		},
	})

	NewLambdaPermission(ctx.Scope, jsii.String(ctx.Id+"_lambda_perm"), &LambdaPermissionConfig{
		Provider:     ctx.Provider,
		StatementId:  jsii.String("AllowExecutionFromAppsync"),
		Action:       jsii.String("lambda:InvokeFunction"),
		FunctionName: cfg.functionsAuthorizer[cfg.region].Arn,
		Principal:    jsii.String("appsync.amazonaws.com"),
		SourceArn:    api.Arn(),
	})

	domainName := NewAppsyncDomainName(ctx.Scope, jsii.String(ctx.Id+"_domain"), &AppsyncDomainNameConfig{
		Provider:       ctx.Provider,
		DomainName:     jsii.String(cfg.region + "." + *cfg.domainName),
		CertificateArn: cfg.cert,
		DependsOn:      &[]cdktf.ITerraformDependable{cfg.certValidation},
	})

	NewAppsyncDomainNameApiAssociation(ctx.Scope, jsii.String(ctx.Id+"_domain_assoc"), &AppsyncDomainNameApiAssociationConfig{
		Provider:   ctx.Provider,
		ApiId:      api.Id(),
		DomainName: domainName.DomainName(),
	})

	healthcheck := NewRoute53HealthCheck(ctx.Scope, jsii.String(ctx.Id+"_healthcheck"), &Route53HealthCheckConfig{
		Provider:                     ctx.Provider,
		ReferenceName:                jsii.String(*cfg.name + "-" + cfg.region),
		Type:                         jsii.String("CLOUDWATCH_METRIC"),
		CloudwatchAlarmName:          cfg.alarmsHealthCheck[cfg.region].Id,
		CloudwatchAlarmRegion:        jsii.String(cfg.region),
		InsufficientDataHealthStatus: jsii.String("Unhealthy"),
	})

	NewRoute53Record(ctx.Scope, jsii.String(ctx.Id+"_dns_global"), &Route53RecordConfig{
		Provider:      ctx.Provider,
		ZoneId:        cfg.hostedZone,
		Name:          cfg.domainName,
		Records:       jsii.Strings(*domainName.DomainName()),
		Type:          jsii.String("CNAME"),
		Ttl:           jsii.Number(300),
		SetIdentifier: jsii.String(cfg.region),
		HealthCheckId: healthcheck.Id(),
		LatencyRoutingPolicy: &[]Route53RecordLatencyRoutingPolicy{
			{
				Region: jsii.String(cfg.region),
			},
		},
	})

	NewRoute53Record(ctx.Scope, jsii.String(ctx.Id+"_dns_regional"), &Route53RecordConfig{
		Provider: ctx.Provider,
		ZoneId:   cfg.hostedZone,
		Name:     domainName.DomainName(),
		Type:     jsii.String("A"),
		Alias: &[]Route53RecordAlias{
			{
				EvaluateTargetHealth: jsii.Bool(false),
				Name:                 domainName.AppsyncDomainName(),
				ZoneId:               domainName.HostedZoneId(),
			},
		},
	})

	dataSources := appsyncDataSources{
		Noop: NewAppsyncDatasource(ctx.Scope, jsii.String(ctx.Id+"_datasource_noop"), &AppsyncDatasourceConfig{
			Provider: ctx.Provider,
			ApiId:    api.Id(),
			Name:     jsii.String("no_op"),
			Type:     jsii.String("NONE"),
		}),
		IpLookup: NewAppsyncDatasource(ctx.Scope, jsii.String(ctx.Id+"_datasource_ip_lookup"), &AppsyncDatasourceConfig{
			Provider:       ctx.Provider,
			ApiId:          api.Id(),
			Name:           jsii.String("ip_lookup"),
			Type:           jsii.String("AWS_LAMBDA"),
			ServiceRoleArn: cfg.role,
			LambdaConfig: &AppsyncDatasourceLambdaConfig{
				FunctionArn: cfg.functionsIpLookup[cfg.region].Arn,
			},
		}),
		IpCache: NewAppsyncDatasource(ctx.Scope, jsii.String(ctx.Id+"_datasource_ip_cache"), &AppsyncDatasourceConfig{
			Provider:       ctx.Provider,
			ApiId:          api.Id(),
			Name:           jsii.String("ip_cache"),
			Type:           jsii.String("AMAZON_DYNAMODB"),
			ServiceRoleArn: cfg.role,
			DynamodbConfig: &AppsyncDatasourceDynamodbConfig{
				TableName: cfg.tablesIpCache[cfg.region].Id,
			},
		}),
		User: NewAppsyncDatasource(ctx.Scope, jsii.String(ctx.Id+"_datasource_user"), &AppsyncDatasourceConfig{
			Provider:       ctx.Provider,
			ApiId:          api.Id(),
			Name:           jsii.String("user"),
			Type:           jsii.String("AMAZON_DYNAMODB"),
			ServiceRoleArn: cfg.role,
			DynamodbConfig: &AppsyncDatasourceDynamodbConfig{
				TableName: cfg.tablesUser[cfg.region].Id,
			},
		}),
		Healthcheck: NewAppsyncDatasource(ctx.Scope, jsii.String(ctx.Id+"_datasource_healthcheck"), &AppsyncDatasourceConfig{
			Provider:       ctx.Provider,
			ApiId:          api.Id(),
			Name:           jsii.String("healthcheck"),
			Type:           jsii.String("AMAZON_DYNAMODB"),
			ServiceRoleArn: cfg.role,
			DynamodbConfig: &AppsyncDatasourceDynamodbConfig{
				TableName: cfg.tablesHealthcheck[cfg.region].Id,
			},
		}),
		Queues: appsyncQueueDataSources{
			UnrankedSolo: NewAppsyncDatasource(ctx.Scope, jsii.String(ctx.Id+"_datasource_q_unranked_solo"), &AppsyncDatasourceConfig{
				Provider:       ctx.Provider,
				ApiId:          api.Id(),
				Name:           jsii.String("q_unranked_solo"),
				Type:           jsii.String("AMAZON_DYNAMODB"),
				ServiceRoleArn: cfg.role,
				DynamodbConfig: &AppsyncDatasourceDynamodbConfig{
					TableName: cfg.queues.UnrankedSolo.Tables()[cfg.region].Id,
				},
			}),
		},
	}

	return appsyncApiInstance{
		Api:         api,
		DomainName:  domainName,
		DataSources: dataSources,
	}
}

func (app appsyncApi) ApiIds() map[string]common.ArnIdPair {
	return common.TransformMapValues(app.Regions, func(instance appsyncApiInstance) common.ArnIdPair {
		return common.ArnIdPair{Arn: instance.Api.Arn(), Id: instance.Api.Id()}
	})
}
