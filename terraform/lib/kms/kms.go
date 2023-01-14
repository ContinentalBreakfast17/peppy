package kms

import (
	"github.com/aws/jsii-runtime-go"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/dataawsiampolicydocument"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/kmsalias"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/kmskey"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/kmsreplicakey"
	"github.com/ContinentalBreakfast17/peppy/terraform/lib/_common"
)

type keySet struct {
	Primary  primaryKey
	Replicas map[string]replicaKey
}

type primaryKey struct {
	KmsKey
	KmsAlias
}

type replicaKey struct {
	KmsReplicaKey
	KmsAlias
}

type KeyConfig struct {
	Providers   common.Providers
	Name        *string
	Description *string
	AccountId   *string
	// principal arns for users allowed admin on the key
	KeyAdmins []*string
}

func (cfg KeyConfig) NewKeySet(ctx common.TfContext) keySet {
	policy := cfg.policy(ctx)

	primary := NewKmsKey(ctx.Scope, jsii.String(ctx.Id), &KmsKeyConfig{
		Provider:              ctx.Provider,
		Description:           cfg.Description,
		MultiRegion:           jsii.Bool(true),
		EnableKeyRotation:     jsii.Bool(true),
		CustomerMasterKeySpec: jsii.String("SYMMETRIC_DEFAULT"),
		DeletionWindowInDays:  jsii.Number(7),
		IsEnabled:             jsii.Bool(true),
		KeyUsage:              jsii.String("ENCRYPT_DECRYPT"),
		Policy:                policy.Json(),
	})

	primaryAlias := NewKmsAlias(ctx.Scope, jsii.String(ctx.Id+"_alias"), &KmsAliasConfig{
		Provider:    ctx.Provider,
		TargetKeyId: primary.GetStringAttribute(jsii.String("key_id")),
		Name:        jsii.String("alias/" + *cfg.Name),
	})

	primaryKey := primaryKey{primary, primaryAlias}
	replicas := map[string]replicaKey{}

	for region, provider := range cfg.Providers {
		replicas[region] = primaryKey.replica(common.SimpleContext(ctx.Scope, ctx.Id+"_"+region, provider))
	}
	return keySet{primaryKey, replicas}
}

func (key primaryKey) replica(ctx common.TfContext) replicaKey {
	replica := NewKmsReplicaKey(ctx.Scope, jsii.String(ctx.Id), &KmsReplicaKeyConfig{
		Provider:             ctx.Provider,
		PrimaryKeyArn:        key.KmsKey.GetStringAttribute(jsii.String("arn")),
		Description:          key.Description(),
		DeletionWindowInDays: jsii.Number(7),
		Enabled:              jsii.Bool(true),
		Policy:               key.Policy(),
	})

	alias := NewKmsAlias(ctx.Scope, jsii.String(ctx.Id+"_alias"), &KmsAliasConfig{
		Provider:    ctx.Provider,
		TargetKeyId: replica.GetStringAttribute(jsii.String("key_id")),
		Name:        jsii.String("alias/" + *key.Name()),
	})
	return replicaKey{replica, alias}
}

func (cfg KeyConfig) policy(ctx common.TfContext) DataAwsIamPolicyDocument {
	admins := append(cfg.KeyAdmins, cfg.AccountId)
	doc := NewDataAwsIamPolicyDocument(ctx.Scope, jsii.String(ctx.Id+"_policy"), &DataAwsIamPolicyDocumentConfig{
		Statement: []DataAwsIamPolicyDocumentStatement{
			{
				Sid:       jsii.String("AllowRoot"),
				Effect:    jsii.String("Allow"),
				Actions:   jsii.Strings("kms:*"),
				Resources: jsii.Strings("*"),
				Principals: []DataAwsIamPolicyDocumentStatementPrincipals{
					{
						Type:        jsii.String("AWS"),
						Identifiers: &admins,
					},
				},
			},
			{
				Sid:       jsii.String("AllowCloudwatchAlarms"),
				Effect:    jsii.String("Allow"),
				Actions:   jsii.Strings("kms:Decrypt", "kms:GenerateDataKey*"),
				Resources: jsii.Strings("*"),
				Principals: []DataAwsIamPolicyDocumentStatementPrincipals{
					{
						Type:        jsii.String("Service"),
						Identifiers: jsii.Strings("cloudwatch.amazonaws.com", "sns.amazonaws.com"),
					},
				},
			},
		},
	})

	return doc
}
