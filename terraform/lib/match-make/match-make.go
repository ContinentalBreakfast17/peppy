package match_make

import (
	"encoding/json"
	"strings"

	"github.com/ContinentalBreakfast17/peppy/terraform/lib/_common"
	"github.com/aws/jsii-runtime-go"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/cloudwatchloggroup"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/dataawsdynamodbtable"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/dataawsiampolicydocument"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/dataawss3object"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/dynamodbtable"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/iamrole"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/iamrolepolicy"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/iamrolepolicyattachment"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/lambdaeventsourcemapping"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/lambdafunction"
	"github.com/hashicorp/terraform-cdk-go/cdktf"
)

type matchMakers struct {
	UnrankedSolo queue
}

type queue struct {
	Regions      map[string]queueInstance
	LambdaRole   IamRole
	name         string
	codeLocation string
}

type queueInstance struct {
	Function LambdaFunction
	Table    DataAwsDynamodbTable
}

type MatchMakeConfig struct {
	Providers      common.Providers
	Name           *string
	KmsWritePolicy *string
	Code           common.ObjectConfig
	KmsArns        common.MultiRegionId
	LambdaIam      common.LambdaIamConfig
	MatchTables    map[string]common.ArnIdPair
	LockTables     map[string]common.ArnIdPair
	LockRegions    []string
}

type queueConfig struct {
	MatchMakeConfig
	codeLocation string
}

type instanceConfig struct {
	queueConfig
	region string
	role   *string
	table  *string
}

const queue_sort = "queue_sort"

func (cfg MatchMakeConfig) New(ctx common.TfContext) matchMakers {
	// init queue info
	result := matchMakers{
		UnrankedSolo: queue{
			name:         common.QUEUE_UNRANKED_SOLO,
			codeLocation: "rust/target/lambda/process-queue-unranked-solo/bootstrap.zip",
		},
	}

	// loop to create all queues
	queues := []*queue{
		&result.UnrankedSolo,
	}

	for _, queue := range queues {
		qResult := queueConfig{
			MatchMakeConfig: cfg,
			codeLocation:    queue.codeLocation,
		}.new(common.SimpleContext(ctx.Scope, ctx.Id+"-"+queue.name, ctx.Provider))
		queue.Regions = qResult.Regions
		queue.LambdaRole = qResult.LambdaRole
	}
	return result
}

func (cfg queueConfig) new(ctx common.TfContext) queue {
	// create tables w/ replicas in each region
	tableReplicas := []DynamodbTableReplica{}
	for region, arn := range cfg.KmsArns.Replicas {
		tableReplicas = append(tableReplicas, DynamodbTableReplica{
			RegionName:    jsii.String(region),
			KmsKeyArn:     arn,
			PropagateTags: jsii.Bool(true),
		})
	}

	queueTable := NewDynamodbTable(ctx.Scope, jsii.String(ctx.Id+"_queue_table"), &DynamodbTableConfig{
		Provider:       ctx.Provider,
		Name:           jsii.String(*cfg.Name + "-queue"),
		BillingMode:    jsii.String("PAY_PER_REQUEST"),
		TableClass:     jsii.String("STANDARD"),
		HashKey:        jsii.String("user"),
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
		GlobalSecondaryIndex: &[]DynamodbTableGlobalSecondaryIndex{
			{
				Name:           jsii.String(queue_sort),
				HashKey:        jsii.String("queue"),
				RangeKey:       jsii.String("join_time"),
				ProjectionType: jsii.String("ALL"),
			},
		},
		Attribute: &[]DynamodbTableAttribute{
			{
				Name: jsii.String("user"),
				Type: jsii.String("S"),
			},
			{
				Name: jsii.String("queue"),
				Type: jsii.String("S"),
			},
			{
				Name: jsii.String("join_time"),
				Type: jsii.String("N"),
			},
		},
	})

	// create lambda role
	lambdaRole := cfg.lambdaRole(common.SimpleContext(ctx.Scope, ctx.Id+"_lambda_role", ctx.Provider))

	// create an instance of the service in each region
	instances := map[string]queueInstance{}
	for region, provider := range cfg.Providers {
		instances[region] = instanceConfig{
			queueConfig: cfg,
			region:      region,
			role:        lambdaRole.Arn(),
			table:       queueTable.Id(),
		}.new(common.SimpleContext(ctx.Scope, ctx.Id+"_"+region, provider))
	}

	// allow lambda to use queue tables
	tableArns := []*string{}
	streamArns := []*string{}
	indexArns := []*string{}
	for _, instance := range instances {
		tableArns = append(tableArns, instance.Table.Arn())
		streamArns = append(streamArns, instance.Table.StreamArn())
		indexArns = append(indexArns, jsii.String(*instance.Table.Arn()+"/index/"+queue_sort))
	}

	NewIamRolePolicy(ctx.Scope, jsii.String(ctx.Id+"_lambda_role_queue_policy"), &IamRolePolicyConfig{
		Provider: ctx.Provider,
		Name:     jsii.String("queue-use"),
		Role:     lambdaRole.Name(),
		Policy: NewDataAwsIamPolicyDocument(ctx.Scope, jsii.String(ctx.Id+"_lambda_role_queue_policy_doc"), &DataAwsIamPolicyDocumentConfig{
			Statement: []DataAwsIamPolicyDocumentStatement{
				{
					Effect:    jsii.String("Allow"),
					Actions:   jsii.Strings("dynamodb:GetRecords", "dynamodb:GetShardIterator", "dynamodb:DescribeStream", "dynamodb:ListStreams"),
					Resources: &streamArns,
				},
				{
					Effect:    jsii.String("Allow"),
					Actions:   jsii.Strings("dynamodb:Query"),
					Resources: &indexArns,
				},
				{
					Effect:    jsii.String("Allow"),
					Actions:   jsii.Strings("dynamodb:ConditionCheckItem", "dynamodb:DeleteItem", "dynamodb:PutItem"),
					Resources: &tableArns,
				},
			},
		}).Json(),
	})

	return queue{Regions: instances, LambdaRole: lambdaRole}
}

func (cfg queueConfig) lambdaRole(ctx common.TfContext) IamRole {
	lambdaRole := NewIamRole(ctx.Scope, jsii.String(ctx.Id), &IamRoleConfig{
		Provider:         ctx.Provider,
		Name:             jsii.String(*cfg.Name + "-lambda"),
		Path:             cfg.LambdaIam.Path,
		AssumeRolePolicy: cfg.LambdaIam.AssumeRole,
	})

	NewIamRolePolicyAttachment(ctx.Scope, jsii.String(ctx.Id+"_policy_exec"), &IamRolePolicyAttachmentConfig{
		Provider:  ctx.Provider,
		Role:      lambdaRole.Name(),
		PolicyArn: cfg.LambdaIam.ExecPolicy,
	})

	NewIamRolePolicyAttachment(ctx.Scope, jsii.String(ctx.Id+"_kms"), &IamRolePolicyAttachmentConfig{
		Provider:  ctx.Provider,
		Role:      lambdaRole.Name(),
		PolicyArn: cfg.KmsWritePolicy,
	})

	lockTables := common.ArnsToList(cfg.LockTables)
	matchTables := common.ArnsToList(cfg.MatchTables)

	NewIamRolePolicy(ctx.Scope, jsii.String(ctx.Id+"_tables_policy"), &IamRolePolicyConfig{
		Provider: ctx.Provider,
		Name:     jsii.String("table-use"),
		Role:     lambdaRole.Name(),
		Policy: NewDataAwsIamPolicyDocument(ctx.Scope, jsii.String(ctx.Id+"_tables_policy_doc"), &DataAwsIamPolicyDocumentConfig{
			Statement: []DataAwsIamPolicyDocumentStatement{
				{
					Effect:    jsii.String("Allow"),
					Actions:   jsii.Strings("dynamodb:Query", "dynamodb:*Item"),
					Resources: &lockTables,
				},
				{
					Effect:    jsii.String("Allow"),
					Actions:   jsii.Strings("dynamodb:GetItem", "dynamodb:PutItem"),
					Resources: &matchTables,
				},
			},
		}).Json(),
	})

	return lambdaRole
}

func (cfg instanceConfig) new(ctx common.TfContext) queueInstance {
	logGroup := NewCloudwatchLogGroup(ctx.Scope, jsii.String(ctx.Id+"_logs"), &CloudwatchLogGroupConfig{
		Provider:        ctx.Provider,
		Name:            jsii.String("/aws/lambda/" + *cfg.Name),
		RetentionInDays: jsii.Number(7),
	})

	table := NewDataAwsDynamodbTable(ctx.Scope, jsii.String(ctx.Id+"_table"), &DataAwsDynamodbTableConfig{
		Provider: ctx.Provider,
		Name:     cfg.table,
	})

	code := NewDataAwsS3Object(ctx.Scope, jsii.String(ctx.Id+"_code"), &DataAwsS3ObjectConfig{
		Provider: ctx.Provider,
		Bucket:   cfg.Code.ToBucket(cfg.region),
		Key:      cfg.Code.ToKey(cfg.codeLocation),
	})

	lambdaEnv := map[string]*string{
		"QUEUE_TABLE":  table.Id(),
		"QUEUE_INDEX":  jsii.String(queue_sort),
		"MATCH_TABLE":  cfg.MatchTables[cfg.region].Id,
		"LOCK_TABLE":   cfg.LockTables[cfg.region].Id,
		"LOCK_REGIONS": jsii.String(strings.Join(cfg.LockRegions, ",")),
	}

	lambdaDependsOn := []cdktf.ITerraformDependable{
		logGroup,
	}

	lambda := NewLambdaFunction(ctx.Scope, jsii.String(ctx.Id+"_lambda"), &LambdaFunctionConfig{
		Provider:        ctx.Provider,
		FunctionName:    cfg.Name,
		Role:            cfg.role,
		S3Bucket:        code.Bucket(),
		S3Key:           code.Key(),
		S3ObjectVersion: code.VersionId(),
		Architectures:   jsii.Strings("arm64"),
		Runtime:         jsii.String("provided.al2"),
		Handler:         jsii.String("bootstrap"),
		Description:     jsii.String("Makes matches based on the configured queue"),
		MemorySize:      jsii.Number(128),
		Timeout:         jsii.Number(15),
		DependsOn:       &lambdaDependsOn,
		Environment: &LambdaFunctionEnvironment{
			Variables: &lambdaEnv,
		},
	})

	filter := map[string]any{
		"eventName": []string{"MODIFY", "INSERT"},
	}
	filterBytes, _ := json.Marshal(filter)

	NewLambdaEventSourceMapping(ctx.Scope, jsii.String(ctx.Id+"_stream"), &LambdaEventSourceMappingConfig{
		Provider:                       ctx.Provider,
		FunctionName:                   lambda.FunctionName(),
		Enabled:                        jsii.Bool(true),
		EventSourceArn:                 table.StreamArn(),
		StartingPosition:               jsii.String("LATEST"),
		MaximumBatchingWindowInSeconds: jsii.Number(2),
		MaximumRetryAttempts:           jsii.Number(6),
		FilterCriteria: &LambdaEventSourceMappingFilterCriteria{
			Filter: &[]LambdaEventSourceMappingFilterCriteriaFilter{
				{
					// todo: enforce region match?
					Pattern: jsii.String(string(filterBytes)),
				},
			},
		},
	})

	return queueInstance{lambda, table}
}

func (queue queue) Name() string {
	return queue.name
}

func (queue queue) Tables() map[string]common.ArnIdPair {
	return common.TransformMapValues(queue.Regions, func(instance queueInstance) common.ArnIdPair {
		return common.TableToIdPair(instance.Table)
	})
}
