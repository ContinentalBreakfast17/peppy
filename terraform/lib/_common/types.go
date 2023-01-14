package common

import (
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/cdktf/cdktf-provider-aws-go/aws/v10/provider"
)

type Providers map[string]provider.AwsProvider

type TfContext struct {
	Scope    constructs.Construct
	Id       string
	Provider provider.AwsProvider
}

func SimpleContext(scope constructs.Construct, id string, provider provider.AwsProvider) TfContext {
	return TfContext{
		Scope:    scope,
		Id:       id,
		Provider: provider,
	}
}
