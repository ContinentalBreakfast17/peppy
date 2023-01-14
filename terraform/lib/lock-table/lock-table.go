package lock_table

import (
	"github.com/ContinentalBreakfast17/peppy/terraform/lib/_common"
	"github.com/aws/jsii-runtime-go"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/dataawsdynamodbtable"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/dynamodbtable"
)

type lockTable struct {
	Regions map[string]lockTableInstance
}

type lockTableInstance struct {
	Table DataAwsDynamodbTable
}

type LockTableConfig struct {
	Providers common.Providers
	Name      *string
	KmsArns   common.MultiRegionId
}

type instanceConfig struct {
	LockTableConfig
	region string
	role   *string
	table  *string
}

func (cfg LockTableConfig) New(ctx common.TfContext) lockTable {
	// create tables w/ replicas in each region
	tableReplicas := []DynamodbTableReplica{}
	for region, arn := range cfg.KmsArns.Replicas {
		tableReplicas = append(tableReplicas, DynamodbTableReplica{
			RegionName:    jsii.String(region),
			KmsKeyArn:     arn,
			PropagateTags: jsii.Bool(true),
		})
	}

	table := NewDynamodbTable(ctx.Scope, jsii.String(ctx.Id+"_table"), &DynamodbTableConfig{
		Provider:       ctx.Provider,
		Name:           jsii.String(*cfg.Name),
		BillingMode:    jsii.String("PAY_PER_REQUEST"),
		TableClass:     jsii.String("STANDARD"),
		HashKey:        jsii.String("process"),
		RangeKey:       jsii.String("sk"),
		StreamEnabled:  jsii.Bool(true),
		StreamViewType: jsii.String("NEW_AND_OLD_IMAGES"),
		Replica:        &tableReplicas,
		ServerSideEncryption: &DynamodbTableServerSideEncryption{
			Enabled:   jsii.Bool(true),
			KmsKeyArn: cfg.KmsArns.Primary,
		},
		Ttl: &DynamodbTableTtl{
			Enabled:       jsii.Bool(true),
			AttributeName: jsii.String("ttl"),
		},
		Attribute: &[]DynamodbTableAttribute{
			{
				Name: jsii.String("process"),
				Type: jsii.String("S"),
			},
			{
				Name: jsii.String("sk"),
				Type: jsii.String("S"),
			},
		},
	})

	// create an instance of the service in each region
	instances := map[string]lockTableInstance{}
	for region, provider := range cfg.Providers {
		instances[region] = instanceConfig{
			LockTableConfig: cfg,
			region:          region,
			table:           table.Id(),
		}.new(common.SimpleContext(ctx.Scope, ctx.Id+"_"+region, provider))
	}

	return lockTable{instances}
}

func (cfg instanceConfig) new(ctx common.TfContext) lockTableInstance {
	table := NewDataAwsDynamodbTable(ctx.Scope, jsii.String(ctx.Id+"_table"), &DataAwsDynamodbTableConfig{
		Provider: ctx.Provider,
		Name:     cfg.table,
	})

	return lockTableInstance{table}
}

func (app lockTable) TableArns() map[string]common.ArnIdPair {
	result := map[string]common.ArnIdPair{}
	for region, instance := range app.Regions {
		result[region] = common.ArnIdPair{Arn: instance.Table.Arn(), Id: instance.Table.Id()}
	}
	return result
}
