package api

import (
	"github.com/ContinentalBreakfast17/peppy/terraform/lib/_common"
	"github.com/aws/jsii-runtime-go"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/cloudwatchloggroup"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/dataawsiampolicydocument"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/dataawss3object"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/iamrole"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/iamrolepolicy"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/iamrolepolicyattachment"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/lambdafunction"
	"github.com/hashicorp/terraform-cdk-go/cdktf"
)

type apiAuthorizer struct {
	Regions map[string]apiAuthorizerInstance
}

type apiAuthorizerInstance struct {
	Function LambdaFunction
}

type apiAuthorizerConfig struct {
	providers     common.Providers
	name          *string
	tables        apiTables
	kmsReadPolicy *string
	code          common.ObjectConfig
	lambdaIam     common.LambdaIamConfig
}

type apiAuthorizerInstanceConfig struct {
	apiAuthorizerConfig
	region string
	role   *string
	table  *string
}

func (cfg apiAuthorizerConfig) new(ctx common.TfContext) apiAuthorizer {
	lambdaRole := cfg.lambdaRole(common.SimpleContext(ctx.Scope, ctx.Id+"_lambda_role", ctx.Provider))

	// create an instance of the service in each region
	instances := map[string]apiAuthorizerInstance{}
	for region, provider := range cfg.providers {
		instances[region] = apiAuthorizerInstanceConfig{
			apiAuthorizerConfig: cfg,
			region:              region,
			role:                lambdaRole.Arn(),
			table:               cfg.tables.Regions[region].User.Id(),
		}.new(common.SimpleContext(ctx.Scope, ctx.Id+"_"+region, provider))
	}

	return apiAuthorizer{instances}
}

func (cfg apiAuthorizerConfig) lambdaRole(ctx common.TfContext) IamRole {
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

	indexArns := []*string{}
	for _, instance := range cfg.tables.Regions {
		indexArns = append(indexArns, jsii.String(*instance.User.Arn()+"/index/"+auth_sort))
	}

	NewIamRolePolicy(ctx.Scope, jsii.String(ctx.Id+"_lambda_role_custom_policy"), &IamRolePolicyConfig{
		Provider: ctx.Provider,
		Name:     jsii.String("read-table"),
		Role:     lambdaRole.Name(),
		Policy: NewDataAwsIamPolicyDocument(ctx.Scope, jsii.String(ctx.Id+"_lambda_role_custom_policy_doc"), &DataAwsIamPolicyDocumentConfig{
			Statement: []DataAwsIamPolicyDocumentStatement{
				{
					Effect:    jsii.String("Allow"),
					Actions:   jsii.Strings("dynamodb:Query"),
					Resources: &indexArns,
				},
			},
		}).Json(),
	})

	return lambdaRole
}

func (cfg apiAuthorizerInstanceConfig) new(ctx common.TfContext) apiAuthorizerInstance {
	logGroup := NewCloudwatchLogGroup(ctx.Scope, jsii.String(ctx.Id+"_logs"), &CloudwatchLogGroupConfig{
		Provider:        ctx.Provider,
		Name:            jsii.String("/aws/lambda/" + *cfg.name),
		RetentionInDays: jsii.Number(7),
	})

	code := NewDataAwsS3Object(ctx.Scope, jsii.String(ctx.Id+"_code"), &DataAwsS3ObjectConfig{
		Provider: ctx.Provider,
		Bucket:   cfg.code.ToBucket(cfg.region),
		Key:      cfg.code.ToKey("rust/target/lambda/authorizer/bootstrap.zip"),
	})

	lambdaEnv := map[string]*string{
		"TABLE": cfg.table,
		"INDEX": jsii.String(auth_sort),
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
		Description:     jsii.String("Authorizes API users"),
		MemorySize:      jsii.Number(128),
		Timeout:         jsii.Number(3),
		DependsOn:       &lambdaDependsOn,
		Environment: &LambdaFunctionEnvironment{
			Variables: &lambdaEnv,
		},
	})

	return apiAuthorizerInstance{lambda}
}

func (app apiAuthorizer) functionIds() map[string]common.ArnIdPair {
	return common.TransformMapValues(app.Regions, func(instance apiAuthorizerInstance) common.ArnIdPair {
		return common.FunctionToIdPair(instance.Function)
	})
}
