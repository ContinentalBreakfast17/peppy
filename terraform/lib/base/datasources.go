package base

import (
	"github.com/ContinentalBreakfast17/peppy/terraform/lib/_common"
	"github.com/aws/jsii-runtime-go"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/dataawscalleridentity"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/dataawsiamgroup"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/dataawsroute53zone"
)

type dataSources struct {
	DataAwsCallerIdentity
	Admins     DataAwsIamGroup
	HostedZone DataAwsRoute53Zone
}

type dataSourceConfig struct {
	adminGroupName *string
	domain         *string
}

func (cfg dataSourceConfig) new(ctx common.TfContext) dataSources {
	caller := NewDataAwsCallerIdentity(ctx.Scope, jsii.String(ctx.Id), &DataAwsCallerIdentityConfig{
		Provider: ctx.Provider,
	})

	admins := NewDataAwsIamGroup(ctx.Scope, jsii.String(ctx.Id+"_admin_group"), &DataAwsIamGroupConfig{
		Provider:  ctx.Provider,
		GroupName: cfg.adminGroupName,
	})

	zone := NewDataAwsRoute53Zone(ctx.Scope, jsii.String(ctx.Id+"_zone"), &DataAwsRoute53ZoneConfig{
		Provider: ctx.Provider,
		Name:     jsii.String(*cfg.domain + "."),
	})

	return dataSources{caller, admins, zone}
}

func (data dataSources) AdminUsers() *[]*string {
	list := []*string{}
	users := data.Admins.Users()
	i := float64(0)
	for user := users.Get(jsii.Number(i)); user != nil && user.ComplexObjectIndex().(float64) >= i; i++ {
		list = append(list, user.Arn())
	}
	return &list
}
