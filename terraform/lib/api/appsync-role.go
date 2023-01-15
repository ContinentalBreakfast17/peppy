package api

import (
	"github.com/ContinentalBreakfast17/peppy/terraform/lib/_common"
	"github.com/aws/jsii-runtime-go"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/dataawsiampolicy"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/dataawsiampolicydocument"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/iamrole"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/iamrolepolicy"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/iamrolepolicyattachment"
)

type appsyncRole struct {
	IamRole
}

type appsyncRoleConfig struct {
	name              *string
	path              *string
	kmsWritePolicy    *string
	queues            []queue
	functionsIpLookup map[string]common.ArnIdPair
	tablesHealthcheck map[string]common.ArnIdPair
	tablesUser        map[string]common.ArnIdPair
	tablesIpCache     map[string]common.ArnIdPair
}

func (cfg appsyncRoleConfig) new(ctx common.TfContext) appsyncRole {
	appsyncAssume := common.ServiceAssumeRoleConfig{
		Service: "appsync.amazonaws.com",
	}.Doc(common.SimpleContext(ctx.Scope, ctx.Id+"_assume_role", ctx.Provider))

	logs := NewDataAwsIamPolicy(ctx.Scope, jsii.String(ctx.Id+"_logs_policy"), &DataAwsIamPolicyConfig{
		Name: jsii.String("AWSAppSyncPushToCloudWatchLogs"),
	})

	role := NewIamRole(ctx.Scope, jsii.String(ctx.Id), &IamRoleConfig{
		Provider:         ctx.Provider,
		Name:             cfg.name,
		Path:             cfg.path,
		AssumeRolePolicy: appsyncAssume.Json(),
	})

	NewIamRolePolicyAttachment(ctx.Scope, jsii.String(ctx.Id+"_logs"), &IamRolePolicyAttachmentConfig{
		Provider:  ctx.Provider,
		Role:      role.Name(),
		PolicyArn: logs.Arn(),
	})

	NewIamRolePolicyAttachment(ctx.Scope, jsii.String(ctx.Id+"_kms"), &IamRolePolicyAttachmentConfig{
		Provider:  ctx.Provider,
		Role:      role.Name(),
		PolicyArn: cfg.kmsWritePolicy,
	})

	lambdaArns := common.ArnsToList(cfg.functionsIpLookup)
	tableArns := common.ArnsToList(cfg.tablesHealthcheck)
	tableArns = append(tableArns, common.ArnsToList(cfg.tablesUser)...)
	tableArns = append(tableArns, common.ArnsToList(cfg.tablesIpCache)...)
	for _, queue := range cfg.queues {
		tableArns = append(tableArns, common.ArnsToList(queue.Tables())...)
	}

	NewIamRolePolicy(ctx.Scope, jsii.String(ctx.Id+"_custom_policy"), &IamRolePolicyConfig{
		Provider: ctx.Provider,
		Name:     jsii.String("custom"),
		Role:     role.Name(),
		Policy: NewDataAwsIamPolicyDocument(ctx.Scope, jsii.String(ctx.Id+"_custom_policy_doc"), &DataAwsIamPolicyDocumentConfig{
			Statement: []DataAwsIamPolicyDocumentStatement{
				{
					Effect:    jsii.String("Allow"),
					Actions:   jsii.Strings("dynamodb:*Item"),
					Resources: &tableArns,
				},
				{
					Effect:    jsii.String("Allow"),
					Actions:   jsii.Strings("lambda:Invoke"),
					Resources: &lambdaArns,
				},
			},
		}).Json(),
	})

	return appsyncRole{role}
}
