package ip_lookup

import (
	"github.com/ContinentalBreakfast17/peppy/terraform/lib/_common"
	"github.com/aws/jsii-runtime-go"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/cloudwatchloggroup"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/dataawsiampolicydocument"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/dataawss3object"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/dataawssecretsmanagersecret"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/iamrole"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/iamrolepolicy"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/iamrolepolicyattachment"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/lambdafunction"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/secretsmanagersecret"
	"github.com/hashicorp/terraform-cdk-go/cdktf"
)

type ipLookup struct {
	Regions    map[string]ipLookupInstance
	LambdaRole IamRole
}

type ipLookupInstance struct {
	Function LambdaFunction
	Secret   DataAwsSecretsmanagerSecret
}

type IpLookupConfig struct {
	Providers     common.Providers
	Name          *string
	KmsReadPolicy *string
	Code          common.ObjectConfig
	KmsArns       common.MultiRegionId
	LambdaIam     common.LambdaIamConfig
}

type instanceConfig struct {
	IpLookupConfig
	region     string
	role       *string
	secretName *string
}

func (cfg IpLookupConfig) New(ctx common.TfContext) ipLookup {
	// create secret w/ replicas in each region
	secretReplicas := []SecretsmanagerSecretReplica{}
	for region, arn := range cfg.KmsArns.Replicas {
		secretReplicas = append(secretReplicas, SecretsmanagerSecretReplica{
			Region:   jsii.String(region),
			KmsKeyId: arn,
		})
	}

	secret := NewSecretsmanagerSecret(ctx.Scope, jsii.String(ctx.Id+"_secret"), &SecretsmanagerSecretConfig{
		Provider:    ctx.Provider,
		NamePrefix:  jsii.String(*cfg.Name + "-token"),
		Description: jsii.String("API token for ip lookup service"),
		KmsKeyId:    cfg.KmsArns.Primary,
		Replica:     &secretReplicas,
	})

	// create lambda role
	lambdaRole := cfg.lambdaRole(common.SimpleContext(ctx.Scope, ctx.Id+"_lambda_role", ctx.Provider))

	// create an instance of the service in each region
	instances := map[string]ipLookupInstance{}
	for region, provider := range cfg.Providers {
		instances[region] = instanceConfig{
			IpLookupConfig: cfg,
			region:         region,
			role:           lambdaRole.Arn(),
			secretName:     secret.Name(),
		}.new(common.SimpleContext(ctx.Scope, ctx.Id+"_"+region, provider))
	}

	// allow lambda to read secrets
	secretArns := []*string{}
	for _, instance := range instances {
		secretArns = append(secretArns, instance.Secret.Arn())
	}

	NewIamRolePolicy(ctx.Scope, jsii.String(ctx.Id+"_lambda_role_custom_policy"), &IamRolePolicyConfig{
		Provider: ctx.Provider,
		Name:     jsii.String("get-secret"),
		Role:     lambdaRole.Name(),
		Policy: NewDataAwsIamPolicyDocument(ctx.Scope, jsii.String(ctx.Id+"_lambda_role_custom_policy_doc"), &DataAwsIamPolicyDocumentConfig{
			Statement: []DataAwsIamPolicyDocumentStatement{
				{
					Effect:    jsii.String("Allow"),
					Actions:   jsii.Strings("secretsmanager:GetSecretValue"),
					Resources: &secretArns,
				},
			},
		}).Json(),
	})

	return ipLookup{instances, lambdaRole}
}

func (cfg IpLookupConfig) lambdaRole(ctx common.TfContext) IamRole {
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

func (cfg instanceConfig) new(ctx common.TfContext) ipLookupInstance {
	logGroup := NewCloudwatchLogGroup(ctx.Scope, jsii.String(ctx.Id+"_logs"), &CloudwatchLogGroupConfig{
		Provider:        ctx.Provider,
		Name:            jsii.String("/aws/lambda/" + *cfg.Name),
		RetentionInDays: jsii.Number(7),
	})

	// note: this will fail on initial deploy to a new replica region
	// not sure how to fix this short of sleeping in a provisioner/null resource
	secret := NewDataAwsSecretsmanagerSecret(ctx.Scope, jsii.String(ctx.Id+"_secret"), &DataAwsSecretsmanagerSecretConfig{
		Provider: ctx.Provider,
		Name:     cfg.secretName,
	})

	code := NewDataAwsS3Object(ctx.Scope, jsii.String(ctx.Id+"_code"), &DataAwsS3ObjectConfig{
		Provider: ctx.Provider,
		Bucket:   cfg.Code.ToBucket(cfg.region),
		Key:      cfg.Code.ToKey("rust/target/lambda/ip-lookup/bootstrap.zip"),
	})

	lambdaEnv := map[string]*string{
		"SECRET_ARN": secret.Arn(),
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
		Description:     jsii.String("Gets geolocation data from an ip address"),
		MemorySize:      jsii.Number(128),
		Timeout:         jsii.Number(3),
		DependsOn:       &lambdaDependsOn,
		Environment: &LambdaFunctionEnvironment{
			Variables: &lambdaEnv,
		},
	})

	return ipLookupInstance{lambda, secret}
}

func (app ipLookup) FunctionIds() map[string]common.ArnIdPair {
	return common.TransformMapValues(app.Regions, func(instance ipLookupInstance) common.ArnIdPair{
		return common.FunctionToIdPair(instance.Function)
	})
}
