package common

import (
	"github.com/aws/jsii-runtime-go"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/dataawsiampolicydocument"
)

type ServiceAssumeRoleConfig struct {
	Service string
}

func (cfg ServiceAssumeRoleConfig) Doc(ctx TfContext) DataAwsIamPolicyDocument {
	doc := NewDataAwsIamPolicyDocument(ctx.Scope, jsii.String(ctx.Id), &DataAwsIamPolicyDocumentConfig{
		Statement: []DataAwsIamPolicyDocumentStatement{
			{
				Sid:     jsii.String("AllowAwsService"),
				Effect:  jsii.String("Allow"),
				Actions: jsii.Strings("sts:AssumeRole"),
				Principals: []DataAwsIamPolicyDocumentStatementPrincipals{
					{
						Type:        jsii.String("Service"),
						Identifiers: jsii.Strings(cfg.Service),
					},
				},
			},
		},
	})

	return doc
}
