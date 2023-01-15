package api

import (
	"github.com/ContinentalBreakfast17/peppy/terraform/lib/_common"
	"github.com/aws/jsii-runtime-go"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/dataawsdynamodbtable"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/dynamodbtable"
)

type apiTables struct {
	Regions map[string]apiTablesInstance
}

type apiTablesInstance struct {
	IpCache DataAwsDynamodbTable
	User    DataAwsDynamodbTable
}

type apiTablesConfig struct {
	providers common.Providers
	name      *string
	kmsArns   common.MultiRegionId
}

type tablesInstanceConfig struct {
	apiTablesConfig
	region  string
	ipCache *string
	user    *string
}

func (cfg apiTablesConfig) new(ctx common.TfContext) apiTables {
	// create tables w/ replicas in each region
	tableReplicas := []DynamodbTableReplica{}
	for region, arn := range cfg.kmsArns.Replicas {
		tableReplicas = append(tableReplicas, DynamodbTableReplica{
			RegionName:    jsii.String(region),
			KmsKeyArn:     arn,
			PropagateTags: jsii.Bool(true),
		})
	}

	ipCacheTable := NewDynamodbTable(ctx.Scope, jsii.String(ctx.Id+"_ip_cache"), &DynamodbTableConfig{
		Provider:       ctx.Provider,
		Name:           jsii.String(*cfg.name + "-ip-cache"),
		BillingMode:    jsii.String("PAY_PER_REQUEST"),
		TableClass:     jsii.String("STANDARD"),
		HashKey:        jsii.String("ip"),
		StreamEnabled:  jsii.Bool(true),
		StreamViewType: jsii.String("NEW_AND_OLD_IMAGES"),
		Replica:        &tableReplicas,
		ServerSideEncryption: &DynamodbTableServerSideEncryption{
			Enabled:   jsii.Bool(true),
			KmsKeyArn: cfg.kmsArns.Primary,
		},
		Ttl: &DynamodbTableTtl{
			Enabled:       jsii.Bool(true),
			AttributeName: jsii.String("ttl"),
		},
		Attribute: &[]DynamodbTableAttribute{
			{
				Name: jsii.String("ip"),
				Type: jsii.String("S"),
			},
		},
	})

	userTable := NewDynamodbTable(ctx.Scope, jsii.String(ctx.Id+"_user_table"), &DynamodbTableConfig{
		Provider:       ctx.Provider,
		Name:           jsii.String(*cfg.name + "-users"),
		BillingMode:    jsii.String("PAY_PER_REQUEST"),
		TableClass:     jsii.String("STANDARD"),
		HashKey:        jsii.String("user"),
		StreamEnabled:  jsii.Bool(true),
		StreamViewType: jsii.String("NEW_AND_OLD_IMAGES"),
		Replica:        &tableReplicas,
		ServerSideEncryption: &DynamodbTableServerSideEncryption{
			Enabled:   jsii.Bool(true),
			KmsKeyArn: cfg.kmsArns.Primary,
		},
		Ttl: &DynamodbTableTtl{
			Enabled:       jsii.Bool(true),
			AttributeName: jsii.String("ttl"),
		},
		Attribute: &[]DynamodbTableAttribute{
			{
				Name: jsii.String("user"),
				Type: jsii.String("S"),
			},
		},
	})

	// create an instance of the service in each region
	instances := map[string]apiTablesInstance{}
	for region, provider := range cfg.providers {
		instances[region] = tablesInstanceConfig{
			apiTablesConfig: cfg,
			region:          region,
			ipCache:         ipCacheTable.Id(),
			user:            userTable.Id(),
		}.new(common.SimpleContext(ctx.Scope, ctx.Id+"_"+region, provider))
	}

	return apiTables{instances}
}

func (cfg tablesInstanceConfig) new(ctx common.TfContext) apiTablesInstance {
	ipCacheTable := NewDataAwsDynamodbTable(ctx.Scope, jsii.String(ctx.Id+"_ip_cache_table"), &DataAwsDynamodbTableConfig{
		Provider: ctx.Provider,
		Name:     cfg.ipCache,
	})

	userTable := NewDataAwsDynamodbTable(ctx.Scope, jsii.String(ctx.Id+"_user_table"), &DataAwsDynamodbTableConfig{
		Provider: ctx.Provider,
		Name:     cfg.user,
	})

	return apiTablesInstance{IpCache: ipCacheTable, User: userTable}
}

func (app apiTables) userTableIds() map[string]common.ArnIdPair {
	return common.TransformMapValues(app.Regions, func(instance apiTablesInstance) common.ArnIdPair {
		return common.TableToIdPair(instance.User)
	})
}

func (app apiTables) ipCacheTableIds() map[string]common.ArnIdPair {
	return common.TransformMapValues(app.Regions, func(instance apiTablesInstance) common.ArnIdPair {
		return common.TableToIdPair(instance.IpCache)
	})
}
