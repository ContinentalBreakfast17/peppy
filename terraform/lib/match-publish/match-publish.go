package match_publish

import (
	"encoding/json"
	"strings"

	"github.com/ContinentalBreakfast17/peppy/terraform/lib/_common"
	"github.com/aws/jsii-runtime-go"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/cloudwatchloggroup"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/dataawsdynamodbtable"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/dataawsiampolicydocument"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/dataawss3object"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/dynamodbtable"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/iamrole"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/iamrolepolicy"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/iamrolepolicyattachment"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/lambdaeventsourcemapping"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/lambdafunction"
	"github.com/hashicorp/terraform-cdk-go/cdktf"
)

type matchPublish struct {
	Regions    map[string]matchPublishInstance
	LambdaRole IamRole
}

type matchPublishInstance struct {
	Function LambdaFunction
	Table    DataAwsDynamodbTable
}

type MatchPublishConfig struct {
	Providers     common.Providers
	Name          *string
	KmsReadPolicy *string
	Code          common.ObjectConfig
	KmsArns       common.MultiRegionId
	LambdaIam     common.LambdaIamConfig
	ApiUrl        string
}

type instanceConfig struct {
	MatchPublishConfig
	region string
	role   *string
	table  *string
}

func (cfg MatchPublishConfig) New(ctx common.TfContext) matchPublish {
	// create tables w/ replicas in each region
	tableReplicas := []DynamodbTableReplica{}
	for region, arn := range cfg.KmsArns.Replicas {
		tableReplicas = append(tableReplicas, DynamodbTableReplica{
			RegionName:    jsii.String(region),
			KmsKeyArn:     arn,
			PropagateTags: jsii.Bool(true),
		})
	}

	table := NewDynamodbTable(ctx.Scope, jsii.String(ctx.Id+"_table"), &DynamodbTableConfig{
		Provider:       ctx.Provider,
		Name:           jsii.String(*cfg.Name),
		BillingMode:    jsii.String("PAY_PER_REQUEST"),
		TableClass:     jsii.String("STANDARD"),
		HashKey:        jsii.String("match"),
		StreamEnabled:  jsii.Bool(true),
		StreamViewType: jsii.String("NEW_AND_OLD_IMAGES"),
		Replica:        &tableReplicas,
		ServerSideEncryption: &DynamodbTableServerSideEncryption{
			Enabled:   jsii.Bool(true),
			KmsKeyArn: cfg.KmsArns.Primary,
		},
		Ttl: &DynamodbTableTtl{
			Enabled:       jsii.Bool(true),
			AttributeName: jsii.String("ttl"),
		},
		Attribute: &[]DynamodbTableAttribute{
			{
				Name: jsii.String("match"),
				Type: jsii.String("S"),
			},
		},
	})

	// create lambda role
	lambdaRole := cfg.lambdaRole(common.SimpleContext(ctx.Scope, ctx.Id+"_lambda_role", ctx.Provider))

	// create an instance of the service in each region
	instances := map[string]matchPublishInstance{}
	for region, provider := range cfg.Providers {
		instances[region] = instanceConfig{
			MatchPublishConfig: cfg,
			region:             region,
			role:               lambdaRole.Arn(),
			table:              table.Id(),
		}.new(common.SimpleContext(ctx.Scope, ctx.Id+"_"+region, provider))
	}

	// allow lambda to use queue tables
	streamArns := []*string{}
	for _, instance := range instances {
		streamArns = append(streamArns, instance.Table.StreamArn())
	}

	NewIamRolePolicy(ctx.Scope, jsii.String(ctx.Id+"_lambda_role_table_policy"), &IamRolePolicyConfig{
		Provider: ctx.Provider,
		Name:     jsii.String("table"),
		Role:     lambdaRole.Name(),
		Policy: NewDataAwsIamPolicyDocument(ctx.Scope, jsii.String(ctx.Id+"_lambda_role_table_policy_doc"), &DataAwsIamPolicyDocumentConfig{
			Statement: []DataAwsIamPolicyDocumentStatement{
				{
					Effect:    jsii.String("Allow"),
					Actions:   jsii.Strings("dynamodb:GetRecords", "dynamodb:GetShardIterator", "dynamodb:DescribeStream", "dynamodb:ListStreams"),
					Resources: &streamArns,
				},
			},
		}).Json(),
	})

	return matchPublish{instances, lambdaRole}
}

func (app matchPublish) AddApiPerms(ctx common.TfContext, arns []*string) {
	// should probably be in a sync.Once /shrug
	resources := []*string{}
	for _, arn := range arns {
		resources = append(resources, []*string{
			jsii.String(*arn + "/types/Mutation/fields/publishMatch"),
			jsii.String(*arn + "/types/Match/fields/*"),
			jsii.String(*arn + "/types/Player/fields/*"),
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

func (cfg MatchPublishConfig) lambdaRole(ctx common.TfContext) IamRole {
	lambdaRole := NewIamRole(ctx.Scope, jsii.String(ctx.Id), &IamRoleConfig{
		Provider:         ctx.Provider,
		Name:             jsii.String(*cfg.Name + "-lambda"),
		Path:             cfg.LambdaIam.Path,
		AssumeRolePolicy: cfg.LambdaIam.AssumeRole,
	})

	NewIamRolePolicyAttachment(ctx.Scope, jsii.String(ctx.Id+"_policy_exec"), &IamRolePolicyAttachmentConfig{
		Provider:  ctx.Provider,
		Role:      lambdaRole.Name(),
		PolicyArn: cfg.LambdaIam.ExecPolicy,
	})

	NewIamRolePolicyAttachment(ctx.Scope, jsii.String(ctx.Id+"_kms"), &IamRolePolicyAttachmentConfig{
		Provider:  ctx.Provider,
		Role:      lambdaRole.Name(),
		PolicyArn: cfg.KmsReadPolicy,
	})

	return lambdaRole
}

func (cfg instanceConfig) new(ctx common.TfContext) matchPublishInstance {
	logGroup := NewCloudwatchLogGroup(ctx.Scope, jsii.String(ctx.Id+"_logs"), &CloudwatchLogGroupConfig{
		Provider:        ctx.Provider,
		Name:            jsii.String("/aws/lambda/" + *cfg.Name),
		RetentionInDays: jsii.Number(7),
	})

	table := NewDataAwsDynamodbTable(ctx.Scope, jsii.String(ctx.Id+"_table"), &DataAwsDynamodbTableConfig{
		Provider: ctx.Provider,
		Name:     cfg.table,
	})

	code := NewDataAwsS3Object(ctx.Scope, jsii.String(ctx.Id+"_code"), &DataAwsS3ObjectConfig{
		Provider: ctx.Provider,
		Bucket:   cfg.Code.ToBucket(cfg.region),
		Key:      cfg.Code.ToKey("rust/target/lambda/process-match/bootstrap.zip"),
	})

	lambdaEnv := map[string]*string{
		"API_URL": jsii.String(strings.Replace(cfg.ApiUrl, "<region>", cfg.region, -1)),
	}

	lambdaDependsOn := []cdktf.ITerraformDependable{
		logGroup,
	}

	lambda := NewLambdaFunction(ctx.Scope, jsii.String(ctx.Id+"_lambda"), &LambdaFunctionConfig{
		Provider:        ctx.Provider,
		FunctionName:    cfg.Name,
		Role:            cfg.role,
		S3Bucket:        code.Bucket(),
		S3Key:           code.Key(),
		S3ObjectVersion: code.VersionId(),
		Architectures:   jsii.Strings("arm64"),
		Runtime:         jsii.String("provided.al2"),
		Handler:         jsii.String("bootstrap"),
		Description:     jsii.String("Notifies players of matches"),
		MemorySize:      jsii.Number(128),
		Timeout:         jsii.Number(15),
		DependsOn:       &lambdaDependsOn,
		Environment: &LambdaFunctionEnvironment{
			Variables: &lambdaEnv,
		},
	})

	filter := map[string]any{
		"eventName": []string{"MODIFY", "INSERT"},
	}
	filterBytes, _ := json.Marshal(filter)

	NewLambdaEventSourceMapping(ctx.Scope, jsii.String(ctx.Id+"_stream"), &LambdaEventSourceMappingConfig{
		Provider:                       ctx.Provider,
		FunctionName:                   lambda.FunctionName(),
		Enabled:                        jsii.Bool(true),
		EventSourceArn:                 table.StreamArn(),
		StartingPosition:               jsii.String("LATEST"),
		MaximumBatchingWindowInSeconds: jsii.Number(2),
		MaximumRetryAttempts:           jsii.Number(6),
		FilterCriteria: &LambdaEventSourceMappingFilterCriteria{
			Filter: &[]LambdaEventSourceMappingFilterCriteriaFilter{
				{
					// todo: enforce region match?
					Pattern: jsii.String(string(filterBytes)),
				},
			},
		},
	})

	return matchPublishInstance{lambda, table}
}

func (app matchPublish) TableIds() map[string]common.ArnIdPair {
	return common.TransformMapValues(app.Regions, func(instance matchPublishInstance) common.ArnIdPair{
		return common.TableToIdPair(instance.Table)
	})
}
