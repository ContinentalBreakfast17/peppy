package base

import (
	"github.com/ContinentalBreakfast17/peppy/terraform/lib/_common"
	"github.com/aws/jsii-runtime-go"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/dataawscalleridentity"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/dataawsiamgroup"
)

type dataSources struct {
	DataAwsCallerIdentity
	Admins DataAwsIamGroup
}

type dataSourceConfig struct {
	adminGroupName *string
}

func (cfg dataSourceConfig) new(ctx common.TfContext) dataSources {
	caller := NewDataAwsCallerIdentity(ctx.Scope, jsii.String(ctx.Id), &DataAwsCallerIdentityConfig{
		Provider: ctx.Provider,
	})

	admins := NewDataAwsIamGroup(ctx.Scope, jsii.String(ctx.Id+"_admin_group"), &DataAwsIamGroupConfig{
		Provider:  ctx.Provider,
		GroupName: cfg.adminGroupName,
	})
	return dataSources{caller, admins}
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
