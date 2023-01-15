package healthcheck

import (
	"fmt"

	"github.com/ContinentalBreakfast17/peppy/terraform/lib/_common"
	"github.com/aws/jsii-runtime-go"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/cloudwatchmetricalarm"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/dataawsiampolicydocument"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/snstopic"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/snstopicpolicy"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/snstopicsubscription"
)

type healthcheckAlarm struct {
	Regions map[string]healthcheckAlarmInstance
}

type healthcheckAlarmInstance struct {
	Topic SnsTopic
	Alarm CloudwatchMetricAlarm
}

type healthcheckAlarmConfig struct {
	providers     common.Providers
	name          *string
	healthchecker healthchecker
	sendAlarmsTo  []string
	// may need "kmsIds"
	// kmsArns       common.MultiRegionId
}

type healthcheckAlarmInstanceConfig struct {
	healthcheckAlarmConfig
	region string
}

func (cfg healthcheckAlarmConfig) new(ctx common.TfContext) healthcheckAlarm {
	// create an instance of the service in each region
	instances := map[string]healthcheckAlarmInstance{}
	for region, provider := range cfg.providers {
		instances[region] = healthcheckAlarmInstanceConfig{
			healthcheckAlarmConfig: cfg,
			region:                 region,
		}.new(common.SimpleContext(ctx.Scope, ctx.Id+"_"+region, provider))
	}

	return healthcheckAlarm{instances}
}

func (cfg healthcheckAlarmInstanceConfig) new(ctx common.TfContext) healthcheckAlarmInstance {
	topic := NewSnsTopic(ctx.Scope, jsii.String(ctx.Id+"_topic"), &SnsTopicConfig{
		Provider:       ctx.Provider,
		Name:           cfg.name,
		// skipping encryption out of sheer laziness for now
		// KmsMasterKeyId: cfg.healthchecker.Regions[cfg.region].Table.ServerSideEncryptionInput().KmsKeyArn(),
	})

	for _, address := range cfg.sendAlarmsTo {
		NewSnsTopicSubscription(ctx.Scope, jsii.String(ctx.Id+"_topic_sub_"+address), &SnsTopicSubscriptionConfig{
			Provider: ctx.Provider,
			TopicArn: topic.Arn(),
			Protocol: jsii.String("email"),
			Endpoint: jsii.String(address),
		})
	}

	alarm := NewCloudwatchMetricAlarm(ctx.Scope, jsii.String(ctx.Id+"_alarm"), &CloudwatchMetricAlarmConfig{
		Provider:           ctx.Provider,
		AlarmName:          jsii.String(*cfg.name + "-system-down"),
		AlarmDescription:   jsii.String(fmt.Sprintf("[%s/healthcheck-failures/system-down] - healthcheck function is experiencing errors", *cfg.name)),
		Namespace:          jsii.String("AWS/Lambda"),
		MetricName:         jsii.String("Errors"),
		Statistic:          jsii.String("Sum"),
		ComparisonOperator: jsii.String("GreaterThanOrEqualToThreshold"),
		Threshold:          jsii.Number(1),
		EvaluationPeriods:  jsii.Number(5),
		Period:             jsii.Number(60),
		DatapointsToAlarm:  jsii.Number(3),
		TreatMissingData:   jsii.String("breaching"),
		AlarmActions:       jsii.Strings(*topic.Arn()),
		OkActions:          jsii.Strings(*topic.Arn()),
		Dimensions: &map[string]*string{
			"FunctionName": cfg.healthchecker.Regions[cfg.region].Function.FunctionName(),
		},
	})

	NewSnsTopicPolicy(ctx.Scope, jsii.String(ctx.Id+"_topic_policy"), &SnsTopicPolicyConfig{
		Provider: ctx.Provider,
		Arn:      topic.Arn(),
		Policy: NewDataAwsIamPolicyDocument(ctx.Scope, jsii.String(ctx.Id+"_topic_policy_doc"), &DataAwsIamPolicyDocumentConfig{
			Statement: []DataAwsIamPolicyDocumentStatement{
				{
					Effect:    jsii.String("Allow"),
					Actions:   jsii.Strings("sns:Publish"),
					Resources: jsii.Strings(*topic.Arn()),
					Principals: []DataAwsIamPolicyDocumentStatementPrincipals{
						{
							Type:        jsii.String("Service"),
							Identifiers: jsii.Strings("cloudwatch.amazonaws.com"),
						},
					},
					Condition: []DataAwsIamPolicyDocumentStatementCondition{
						{
							Test:     jsii.String("ArnEquals"),
							Variable: jsii.String("aws:SourceArn"),
							Values:   jsii.Strings(*alarm.Arn()),
						},
					},
				},
			},
		}).Json(),
	})

	return healthcheckAlarmInstance{Topic: topic, Alarm: alarm}
}
