package common

import (
	"strings"

	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
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

type ObjectConfig struct {
	Bucket *string
	Prefix *string
}

func (cfg ObjectConfig) ToBucket(region string) *string {
	return jsii.String(strings.Replace(*cfg.Bucket+"-"+region, "--", "-", -1))
}

func (cfg ObjectConfig) ToKey(key string) *string {
	return jsii.String(strings.Replace(*cfg.Prefix+"/"+key, "//", "/", -1))
}

type MultiRegionId struct {
	Primary  *string
	Replicas map[string]*string
}

func NewMultiRegionId(primary *string) MultiRegionId {
	return MultiRegionId{
		Primary:  primary,
		Replicas: map[string]*string{},
	}
}

type LambdaIamConfig struct {
	Path       *string
	ExecPolicy *string
	AssumeRole *string
}

type ArnIdPair struct {
	Arn *string
	Id  *string
}
