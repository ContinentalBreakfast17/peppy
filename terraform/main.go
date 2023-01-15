package main

import (
	. "github.com/ContinentalBreakfast17/peppy/terraform/lib/_common"
	"github.com/ContinentalBreakfast17/peppy/terraform/lib/_config"
	. "github.com/ContinentalBreakfast17/peppy/terraform/lib/api"
	. "github.com/ContinentalBreakfast17/peppy/terraform/lib/base"
	. "github.com/ContinentalBreakfast17/peppy/terraform/lib/healthcheck"
	. "github.com/ContinentalBreakfast17/peppy/terraform/lib/ip-lookup"
	. "github.com/ContinentalBreakfast17/peppy/terraform/lib/lock-table"
	. "github.com/ContinentalBreakfast17/peppy/terraform/lib/match-make"
	. "github.com/ContinentalBreakfast17/peppy/terraform/lib/match-publish"
	"github.com/aws/jsii-runtime-go"
	"github.com/hashicorp/terraform-cdk-go/cdktf"
)

type stackConfig struct {
	config.Stack
	config.Config
}

func main() {
	// read config
	cfg, err := config.Paths{
		// probably a better way to form these paths
		Stacks: "config",
		Schema: "../schema",
		Vtl:    "../vtl",
	}.LoadConfig()

	if err != nil {
		panic(err)
	}

	app := cdktf.NewApp(nil)
	for _, stack := range cfg.Stacks {
		(stackConfig{stack, cfg}).addTo(app)
	}

	app.Synth()
}

func (cfg stackConfig) addTo(app cdktf.App) {
	stack := cdktf.NewTerraformStack(app, jsii.String(cfg.StackName))

	cdktf.NewS3Backend(stack, &cdktf.S3BackendProps{
		Bucket:        jsii.String(cfg.Vars.Backend.Bucket),
		Key:           jsii.String(cfg.Vars.Backend.Key),
		Region:        jsii.String(cfg.Vars.Backend.Region),
		DynamodbTable: jsii.String(cfg.Vars.Backend.Table),
	})

	codeObjectConfig := ObjectConfig{
		Bucket: jsii.String(cfg.Vars.Artifacts.BucketPrefix),
		Prefix: jsii.String(cfg.Vars.Artifacts.ObjectPrefix),
	}

	base := BaseConfig{
		Name:           jsii.String(cfg.Vars.Name),
		IamPath:        jsii.String(cfg.Vars.IamPath),
		Regions:        cfg.Vars.Regions,
		AdminGroupName: jsii.String(cfg.Vars.Groups.InfraAdmin),
		Domain:         jsii.String(cfg.Vars.Domain.Name),
	}.New(SimpleContext(stack, "base", nil))

	allProviders := base.Providers.All()
	lambdaIam := LambdaIamConfig{
		Path:       jsii.String(cfg.Vars.IamPath),
		ExecPolicy: base.Policies.LambdaExec.Arn(),
		AssumeRole: base.Policies.LambdaAssumeRole.Json(),
	}

	// meaningful resources start here

	ipLookup := IpLookupConfig{
		Providers:     allProviders,
		Name:          jsii.String(cfg.Vars.Name + "-ip-lookup"),
		LambdaIam:     lambdaIam,
		Code:          codeObjectConfig,
		KmsReadPolicy: base.Policies.KmsMain.Read.Arn(),
		KmsArns:       base.KmsMain.Arns(),
	}.New(SimpleContext(stack, "ip_lookup", base.Providers.Main))

	lockTable := LockTableConfig{
		Providers: allProviders,
		Name:      jsii.String(cfg.Vars.Name + "-process-lock"),
		KmsArns:   base.KmsMain.Arns(),
	}.New(SimpleContext(stack, "process_lock", base.Providers.Main))

	matchPublish := MatchPublishConfig{
		Providers:     allProviders,
		Name:          jsii.String(cfg.Vars.Name + "-match-publish"),
		LambdaIam:     lambdaIam,
		Code:          codeObjectConfig,
		KmsReadPolicy: base.Policies.KmsMain.Read.Arn(),
		KmsArns:       base.KmsMain.Arns(),
		ApiUrl:        cfg.Vars.Domain.RegionalUrlTemplate(),
	}.New(SimpleContext(stack, "match_publish", base.Providers.Main))

	matchMake := MatchMakeConfig{
		Providers:      allProviders,
		Name:           jsii.String(cfg.Vars.Name + "-match-make"),
		LambdaIam:      lambdaIam,
		Code:           codeObjectConfig,
		KmsWritePolicy: base.Policies.KmsMain.Write.Arn(),
		KmsArns:        base.KmsMain.Arns(),
		MatchTables:    matchPublish.TableIds(),
		LockTables:     lockTable.TableIds(),
		LockRegions:    cfg.Vars.OrderedRegions(),
	}.New(SimpleContext(stack, "match_make", base.Providers.Main))

	healthcheck := HealthcheckConfig{
		Providers:      allProviders,
		Name:           jsii.String(cfg.Vars.Name + "-healthcheck"),
		LambdaIam:      lambdaIam,
		Code:           codeObjectConfig,
		KmsReadPolicy:  base.Policies.KmsMain.Read.Arn(),
		KmsWritePolicy: base.Policies.KmsMain.Write.Arn(),
		KmsArns:        base.KmsMain.Arns(),
		ApiUrl:         cfg.Vars.Domain.RegionalUrlTemplate(),
		SendAlarmsTo:   cfg.Vars.Alarms.SendTo,
	}.New(SimpleContext(stack, "healthcheck", base.Providers.Main))

	api := ApiConfig{
		Providers:         allProviders,
		Name:              jsii.String(cfg.Vars.Name),
		Schema:            cfg.Schema,
		Vtl:               cfg.Vtl,
		KmsArns:           base.KmsMain.Arns(),
		KmsWritePolicy:    base.Policies.KmsMain.Write.Arn(),
		DomainName:        jsii.String(cfg.Vars.Domain.Fqdn()),
		HostedZoneId:      base.DataSources.HostedZone.Id(),
		FunctionsIpLookup: ipLookup.FunctionIds(),
		TablesHealthcheck: healthcheck.Healthchecker.TableIds(),
		Queues: ApiQueueConfig{
			UnrankedSolo: matchMake.UnrankedSolo,
		},
	}.New(SimpleContext(stack, "api", base.Providers.Main))

	// add api permissions to lambdas
	matchPublish.AddApiPerms(
		SimpleContext(stack, "match_publish_api_perms", base.Providers.Main),
		ArnsToList(api.ApiIds()),
	)
	healthcheck.Healthchecker.AddApiPerms(
		SimpleContext(stack, "healthcheck_api_perms", base.Providers.Main),
		ArnsToList(api.ApiIds()),
	)
	healthcheck.Responder.AddApiPerms(
		SimpleContext(stack, "healthcheck_responder_api_perms", base.Providers.Main),
		ArnsToList(api.ApiIds()),
	)
}
