package healthcheck

import (
	"encoding/json"
	"strings"

	"github.com/ContinentalBreakfast17/peppy/terraform/lib/_common"
	"github.com/aws/jsii-runtime-go"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/cloudwatchloggroup"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/dataawsiampolicydocument"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/dataawss3object"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/iamrole"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/iamrolepolicy"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/iamrolepolicyattachment"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/lambdaeventsourcemapping"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/lambdafunction"
	"github.com/hashicorp/terraform-cdk-go/cdktf"
)

type healthcheckResponder struct {
	Regions    map[string]healthcheckResponderInstance
	LambdaRole IamRole
}

type healthcheckResponderInstance struct {
	Function LambdaFunction
}

type healthcheckResponderConfig struct {
	providers     common.Providers
	name          *string
	kmsReadPolicy *string
	code          common.ObjectConfig
	lambdaIam     common.LambdaIamConfig
	apiUrl        string
	healthchecker healthchecker
}

type healthcheckResponderInstanceConfig struct {
	healthcheckResponderConfig
	region string
	role   *string
}

func (cfg healthcheckResponderConfig) new(ctx common.TfContext) healthcheckResponder {
	// create lambda role
	lambdaRole := cfg.lambdaRole(common.SimpleContext(ctx.Scope, ctx.Id+"_lambda_role", ctx.Provider))

	// create an instance of the service in each region
	instances := map[string]healthcheckResponderInstance{}
	for region, provider := range cfg.providers {
		instances[region] = healthcheckResponderInstanceConfig{
			healthcheckResponderConfig: cfg,
			region:                     region,
			role:                       lambdaRole.Arn(),
		}.new(common.SimpleContext(ctx.Scope, ctx.Id+"_"+region, provider))
	}

	return healthcheckResponder{instances, lambdaRole}
}

func (app healthcheckResponder) AddApiPerms(ctx common.TfContext, arns []*string) {
	// should probably be in a sync.Once /shrug
	resources := []*string{}
	for _, arn := range arns {
		resources = append(resources, []*string{
			jsii.String(*arn + "/types/Mutation/fields/publishHealth"),
			jsii.String(*arn + "/types/HealthNotification/fields/*"),
		}...)
	}

	NewIamRolePolicy(ctx.Scope, jsii.String(ctx.Id+"_lambda_role_appsync_policy"), &IamRolePolicyConfig{
		Provider: ctx.Provider,
		Name:     jsii.String("appsync"),
		Role:     app.LambdaRole.Name(),
		Policy: NewDataAwsIamPolicyDocument(ctx.Scope, jsii.String(ctx.Id+"_lambda_role_appsync_policy_doc"), &DataAwsIamPolicyDocumentConfig{
			Statement: []DataAwsIamPolicyDocumentStatement{
				{
					Effect:    jsii.String("Allow"),
					Actions:   jsii.Strings("appsync:GraphQL"),
					Resources: &resources,
				},
			},
		}).Json(),
	})
}

func (cfg healthcheckResponderConfig) lambdaRole(ctx common.TfContext) IamRole {
	lambdaRole := NewIamRole(ctx.Scope, jsii.String(ctx.Id), &IamRoleConfig{
		Provider:         ctx.Provider,
		Name:             jsii.String(*cfg.name + "-lambda"),
		Path:             cfg.lambdaIam.Path,
		AssumeRolePolicy: cfg.lambdaIam.AssumeRole,
	})

	NewIamRolePolicyAttachment(ctx.Scope, jsii.String(ctx.Id+"_policy_exec"), &IamRolePolicyAttachmentConfig{
		Provider:  ctx.Provider,
		Role:      lambdaRole.Name(),
		PolicyArn: cfg.lambdaIam.ExecPolicy,
	})

	NewIamRolePolicyAttachment(ctx.Scope, jsii.String(ctx.Id+"_kms"), &IamRolePolicyAttachmentConfig{
		Provider:  ctx.Provider,
		Role:      lambdaRole.Name(),
		PolicyArn: cfg.kmsReadPolicy,
	})

	streamArns := []*string{}
	for _, instance := range cfg.healthchecker.Regions {
		streamArns = append(streamArns, instance.Table.StreamArn())
	}

	NewIamRolePolicy(ctx.Scope, jsii.String(ctx.Id+"_dynamo"), &IamRolePolicyConfig{
		Provider: ctx.Provider,
		Name:     jsii.String("dynamo"),
		Role:     lambdaRole.Name(),
		Policy: NewDataAwsIamPolicyDocument(ctx.Scope, jsii.String(ctx.Id+"_dynamo_doc"), &DataAwsIamPolicyDocumentConfig{
			Statement: []DataAwsIamPolicyDocumentStatement{
				{
					Effect:    jsii.String("Allow"),
					Actions:   jsii.Strings("dynamodb:GetRecords", "dynamodb:GetShardIterator", "dynamodb:DescribeStream", "dynamodb:ListStreams"),
					Resources: &streamArns,
				},
			},
		}).Json(),
	})

	return lambdaRole
}

func (cfg healthcheckResponderInstanceConfig) new(ctx common.TfContext) healthcheckResponderInstance {
	logGroup := NewCloudwatchLogGroup(ctx.Scope, jsii.String(ctx.Id+"_logs"), &CloudwatchLogGroupConfig{
		Provider:        ctx.Provider,
		Name:            jsii.String("/aws/lambda/" + *cfg.name),
		RetentionInDays: jsii.Number(7),
	})

	code := NewDataAwsS3Object(ctx.Scope, jsii.String(ctx.Id+"_code"), &DataAwsS3ObjectConfig{
		Provider: ctx.Provider,
		Bucket:   cfg.code.ToBucket(cfg.region),
		Key:      cfg.code.ToKey("rust/target/lambda/process-healthcheck/bootstrap.zip"),
	})

	lambdaEnv := map[string]*string{
		"API_URL": jsii.String(strings.Replace(cfg.apiUrl, "<region>", cfg.region, -1)),
	}

	lambdaDependsOn := []cdktf.ITerraformDependable{
		logGroup,
	}

	lambda := NewLambdaFunction(ctx.Scope, jsii.String(ctx.Id+"_lambda"), &LambdaFunctionConfig{
		Provider:        ctx.Provider,
		FunctionName:    cfg.name,
		Role:            cfg.role,
		S3Bucket:        code.Bucket(),
		S3Key:           code.Key(),
		S3ObjectVersion: code.VersionId(),
		Architectures:   jsii.Strings("arm64"),
		Runtime:         jsii.String("provided.al2"),
		Handler:         jsii.String("bootstrap"),
		Description:     jsii.String("Responds to async healthchecks"),
		MemorySize:      jsii.Number(128),
		Timeout:         jsii.Number(5),
		DependsOn:       &lambdaDependsOn,
		Environment: &LambdaFunctionEnvironment{
			Variables: &lambdaEnv,
		},
	})

	filter := map[string]any{
		// only inserts
		"eventName": []string{"INSERT"},
		// only for this region
		"dynamodb": map[string]any{
			"NewImage": map[string]any{
				"region": map[string][]string{
					"S": []string{cfg.region},
				},
			},
		},
	}
	filterBytes, _ := json.Marshal(filter)

	NewLambdaEventSourceMapping(ctx.Scope, jsii.String(ctx.Id+"_stream"), &LambdaEventSourceMappingConfig{
		Provider:                       ctx.Provider,
		FunctionName:                   lambda.FunctionName(),
		Enabled:                        jsii.Bool(true),
		EventSourceArn:                 cfg.healthchecker.Regions[cfg.region].Table.StreamArn(),
		StartingPosition:               jsii.String("LATEST"),
		MaximumBatchingWindowInSeconds: jsii.Number(2),
		FilterCriteria: &LambdaEventSourceMappingFilterCriteria{
			Filter: &[]LambdaEventSourceMappingFilterCriteriaFilter{
				{
					Pattern: jsii.String(string(filterBytes)),
				},
			},
		},
	})

	return healthcheckResponderInstance{lambda}
}
