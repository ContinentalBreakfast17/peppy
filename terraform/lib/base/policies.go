package base

import (
	"github.com/ContinentalBreakfast17/peppy/terraform/lib/_common"
	"github.com/aws/jsii-runtime-go"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/dataawsiampolicy"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/dataawsiampolicydocument"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/iampolicy"
)

type policies struct {
	KmsMain          kmsPolicies
	LambdaExec       DataAwsIamPolicy
	LambdaAssumeRole DataAwsIamPolicyDocument
}

type kmsPolicies struct {
	Read  IamPolicy
	Write IamPolicy
}

type policyConfig struct {
	kmsMain keySet
	path    *string
	name    *string
}

type kmsPoliciesConfig struct {
	keySet
	name string
	path *string
}

func (cfg policyConfig) new(ctx common.TfContext) policies {
	kmsMain := kmsPoliciesConfig{
		keySet: cfg.kmsMain,
		name:   *cfg.name + "-kms-main",
		path:   cfg.path,
	}.new(common.SimpleContext(ctx.Scope, ctx.Id+"_kms_main", ctx.Provider))

	lambdaExec := NewDataAwsIamPolicy(ctx.Scope, jsii.String(ctx.Id+"_lambda_exec"), &DataAwsIamPolicyConfig{
		Name: jsii.String("AWSLambdaBasicExecutionRole"),
	})

	lambdaAssume := common.ServiceAssumeRoleConfig{
		Service: "lambda.amazonaws.com",
	}.Doc(common.SimpleContext(ctx.Scope, ctx.Id+"_lambda_assume_role", ctx.Provider))

	return policies{
		KmsMain:          kmsMain,
		LambdaExec:       lambdaExec,
		LambdaAssumeRole: lambdaAssume,
	}
}

func (cfg kmsPoliciesConfig) new(ctx common.TfContext) kmsPolicies {
	read := NewIamPolicy(ctx.Scope, jsii.String(ctx.Id+"_read"), &IamPolicyConfig{
		Provider: ctx.Provider,
		Name:     jsii.String(cfg.name + "-read"),
		Path:     cfg.path,
		Policy: cfg.iamPolicyDoc(
			common.SimpleContext(ctx.Scope, ctx.Id+"_read_doc", ctx.Provider),
			[]string{"kms:Decrypt"},
		).Json(),
	})

	write := NewIamPolicy(ctx.Scope, jsii.String(ctx.Id+"_write"), &IamPolicyConfig{
		Provider: ctx.Provider,
		Name:     jsii.String(cfg.name + "-write"),
		Path:     cfg.path,
		Policy: cfg.iamPolicyDoc(
			common.SimpleContext(ctx.Scope, ctx.Id+"_write_doc", ctx.Provider),
			[]string{"kms:CreateGrant", "kms:Decrypt", "kms:Encrypt", "kms:GenerateDataPair*", "kms:ReEncrypt*"},
		).Json(),
	})

	return kmsPolicies{Read: read, Write: write}
}

func (cfg kmsPoliciesConfig) iamPolicyDoc(ctx common.TfContext, actions []string) DataAwsIamPolicyDocument {
	resources := []*string{cfg.Primary.Key.Arn()}
	for _, key := range cfg.Replicas {
		resources = append(resources, key.Key.Arn())
	}

	doc := NewDataAwsIamPolicyDocument(ctx.Scope, jsii.String(ctx.Id), &DataAwsIamPolicyDocumentConfig{
		Statement: []DataAwsIamPolicyDocumentStatement{
			{
				Sid:       jsii.String("AllowCryptoOps"),
				Effect:    jsii.String("Allow"),
				Actions:   jsii.Strings(actions...),
				Resources: &resources,
			},
		},
	})

	return doc
}
