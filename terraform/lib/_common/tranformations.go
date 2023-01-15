package common

import (
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/dataawsdynamodbtable"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/lambdafunction"
)

func TableToIdPair(table DataAwsDynamodbTable) ArnIdPair {
	return ArnIdPair{Arn: table.Arn(), Id: table.Id()}
}

func FunctionToIdPair(function LambdaFunction) ArnIdPair {
	return ArnIdPair{Arn: function.Arn(), Id: function.FunctionName()}
}

func ArnsToList(pairs map[string]ArnIdPair) (result []*string) {
	for _, pair := range pairs {
		result = append(result, pair.Arn)
	}
	return
}
