package healthcheck

import (
	"github.com/ContinentalBreakfast17/peppy/terraform/lib/_common"
	"github.com/aws/jsii-runtime-go"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/cloudwatcheventrule"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/cloudwatcheventtarget"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/lambdapermission"
)

type healthcheckCron struct {
	Regions map[string]healthcheckCronInstance
}

type healthcheckCronInstance struct {
	Rule CloudwatchEventRule
}

type healthcheckCronConfig struct {
	providers     common.Providers
	name          *string
	healthchecker healthchecker
}

type healthcheckCronInstanceConfig struct {
	healthcheckCronConfig
	region string
}

func (cfg healthcheckCronConfig) new(ctx common.TfContext) healthcheckCron {
	// create an instance of the service in each region
	instances := map[string]healthcheckCronInstance{}
	for region, provider := range cfg.providers {
		instances[region] = healthcheckCronInstanceConfig{
			healthcheckCronConfig: cfg,
			region:                region,
		}.new(common.SimpleContext(ctx.Scope, ctx.Id+"_"+region, provider))
	}

	return healthcheckCron{instances}
}

func (cfg healthcheckCronInstanceConfig) new(ctx common.TfContext) healthcheckCronInstance {
	rule := NewCloudwatchEventRule(ctx.Scope, jsii.String(ctx.Id+"_rule"), &CloudwatchEventRuleConfig{
		Provider:           ctx.Provider,
		Name:               cfg.name,
		Description:        jsii.String("Triggers a healthcheck on a schedule"),
		IsEnabled:          jsii.Bool(true),
		ScheduleExpression: jsii.String("rate(1 minute)"),
	})

	lambdaArn := cfg.healthchecker.Regions[cfg.region].Function.Arn()

	NewCloudwatchEventTarget(ctx.Scope, jsii.String(ctx.Id+"_target"), &CloudwatchEventTargetConfig{
		Provider: ctx.Provider,
		Rule:     rule.Name(),
		Arn:      lambdaArn,
		RetryPolicy: &CloudwatchEventTargetRetryPolicy{
			MaximumEventAgeInSeconds: jsii.Number(60),
			MaximumRetryAttempts:     jsii.Number(1),
		},
	})

	NewLambdaPermission(ctx.Scope, jsii.String(ctx.Id+"_perm"), &LambdaPermissionConfig{
		Provider:     ctx.Provider,
		StatementId:  jsii.String("AllowExecutionFromEventBridge"),
		Action:       jsii.String("lambda:InvokeFunction"),
		FunctionName: lambdaArn,
		Principal:    jsii.String("events.amazonaws.com"),
		SourceArn:    rule.Arn(),
	})

	return healthcheckCronInstance{rule}
}
