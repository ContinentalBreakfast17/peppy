package healthcheck

import (
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
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/lambdafunction"
	"github.com/hashicorp/terraform-cdk-go/cdktf"
)

type healthchecker struct {
	Regions    map[string]healthcheckerInstance
	LambdaRole IamRole
}

type healthcheckerInstance struct {
	Function LambdaFunction
	Table    DataAwsDynamodbTable
}

type healthcheckerConfig struct {
	providers      common.Providers
	name           *string
	kmsWritePolicy *string
	code           common.ObjectConfig
	kmsArns        common.MultiRegionId
	lambdaIam      common.LambdaIamConfig
	apiUrl         string
}

type healthcheckerInstanceConfig struct {
	healthcheckerConfig
	region string
	role   *string
	table  *string
}

func (cfg healthcheckerConfig) new(ctx common.TfContext) healthchecker {
	// create tables w/ replicas in each region (I guess we don't really need to use the kms arns here, but the replica regions are convenient)
	tableReplicas := []DynamodbTableReplica{}
	for region := range cfg.kmsArns.Replicas {
		tableReplicas = append(tableReplicas, DynamodbTableReplica{
			RegionName:    jsii.String(region),
			PropagateTags: jsii.Bool(true),
		})
	}

	table := NewDynamodbTable(ctx.Scope, jsii.String(ctx.Id+"_table"), &DynamodbTableConfig{
		Provider:       ctx.Provider,
		Name:           jsii.String(*cfg.name),
		BillingMode:    jsii.String("PAY_PER_REQUEST"),
		TableClass:     jsii.String("STANDARD"),
		HashKey:        jsii.String("region"),
		RangeKey:       jsii.String("id"),
		StreamEnabled:  jsii.Bool(true),
		StreamViewType: jsii.String("NEW_AND_OLD_IMAGES"),
		Replica:        &tableReplicas,
		ServerSideEncryption: &DynamodbTableServerSideEncryption{
			// we don't encrypt this as it just has random id's + health status
			// it saves us some $ since this gets called a lot
			Enabled: jsii.Bool(false),
		},
		Ttl: &DynamodbTableTtl{
			Enabled:       jsii.Bool(true),
			AttributeName: jsii.String("ttl"),
		},
		Attribute: &[]DynamodbTableAttribute{
			{
				Name: jsii.String("region"),
				Type: jsii.String("S"),
			},
			{
				Name: jsii.String("id"),
				Type: jsii.String("S"),
			},
		},
	})

	// create lambda role
	lambdaRole := cfg.lambdaRole(common.SimpleContext(ctx.Scope, ctx.Id+"_lambda_role", ctx.Provider))

	// create an instance of the service in each region
	instances := map[string]healthcheckerInstance{}
	for region, provider := range cfg.providers {
		instances[region] = healthcheckerInstanceConfig{
			healthcheckerConfig: cfg,
			region:              region,
			role:                lambdaRole.Arn(),
			table:               table.Id(),
		}.new(common.SimpleContext(ctx.Scope, ctx.Id+"_"+region, provider))
	}

	// allow lambda to use queue tables
	tableArns := []*string{}
	for _, instance := range instances {
		tableArns = append(tableArns, instance.Table.Arn())
	}

	NewIamRolePolicy(ctx.Scope, jsii.String(ctx.Id+"_lambda_role_table_policy"), &IamRolePolicyConfig{
		Provider: ctx.Provider,
		Name:     jsii.String("table"),
		Role:     lambdaRole.Name(),
		Policy: NewDataAwsIamPolicyDocument(ctx.Scope, jsii.String(ctx.Id+"_lambda_role_table_policy_doc"), &DataAwsIamPolicyDocumentConfig{
			Statement: []DataAwsIamPolicyDocumentStatement{
				{
					Effect:    jsii.String("Allow"),
					Actions:   jsii.Strings("dynamodb:UpdateItem", "dynamodb:ConditionCheckItem"),
					Resources: &tableArns,
				},
			},
		}).Json(),
	})

	return healthchecker{instances, lambdaRole}
}

func (app healthchecker) AddApiPerms(ctx common.TfContext, arns []*string) {
	// should probably be in a sync.Once /shrug
	resources := []*string{}
	for _, arn := range arns {
		resources = append(resources, []*string{
			jsii.String(*arn + "/types/Subscription/fields/healthcheck"),
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

func (cfg healthcheckerConfig) lambdaRole(ctx common.TfContext) IamRole {
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
		PolicyArn: cfg.kmsWritePolicy,
	})

	return lambdaRole
}

func (cfg healthcheckerInstanceConfig) new(ctx common.TfContext) healthcheckerInstance {
	logGroup := NewCloudwatchLogGroup(ctx.Scope, jsii.String(ctx.Id+"_logs"), &CloudwatchLogGroupConfig{
		Provider:        ctx.Provider,
		Name:            jsii.String("/aws/lambda/" + *cfg.name),
		RetentionInDays: jsii.Number(7),
	})

	table := NewDataAwsDynamodbTable(ctx.Scope, jsii.String(ctx.Id+"_table"), &DataAwsDynamodbTableConfig{
		Provider: ctx.Provider,
		Name:     cfg.table,
	})

	code := NewDataAwsS3Object(ctx.Scope, jsii.String(ctx.Id+"_code"), &DataAwsS3ObjectConfig{
		Provider: ctx.Provider,
		Bucket:   cfg.code.ToBucket(cfg.region),
		Key:      cfg.code.ToKey("rust/target/lambda/healthcheck/bootstrap.zip"),
	})

	lambdaEnv := map[string]*string{
		"TABLE":   table.Id(),
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
		Description:     jsii.String("Performs a healthcheck on the API"),
		MemorySize:      jsii.Number(128),
		Timeout:         jsii.Number(20),
		DependsOn:       &lambdaDependsOn,
		Environment: &LambdaFunctionEnvironment{
			Variables: &lambdaEnv,
		},
	})

	return healthcheckerInstance{lambda, table}
}

func (app healthchecker) TableIds() map[string]common.ArnIdPair {
	return common.TransformMapValues(app.Regions, func(instance healthcheckerInstance) common.ArnIdPair {
		return common.TableToIdPair(instance.Table)
	})
}
