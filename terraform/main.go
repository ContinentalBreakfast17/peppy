package main

import (
	. "github.com/ContinentalBreakfast17/peppy/terraform/lib/_common"
	. "github.com/ContinentalBreakfast17/peppy/terraform/lib/base"
	. "github.com/ContinentalBreakfast17/peppy/terraform/lib/ip-lookup"
	"github.com/aws/jsii-runtime-go"
	"github.com/hashicorp/terraform-cdk-go/cdktf"
)

func main() {
	app := cdktf.NewApp(nil)
	stack := cdktf.NewTerraformStack(app, jsii.String("aws"))

	// todo: parse vars
	name := jsii.String("slippi-api")
	codeObjectConfig := ObjectConfig{
		Bucket: jsii.String("slippi-api-artifacts-18968913554"),
		Prefix: jsii.String("slippi/functions"),
	}
	iamPath := jsii.String("/slippi-api/")

	cdktf.NewS3Backend(stack, &cdktf.S3BackendProps{
		// todo: move to vars
		Bucket:        jsii.String("slippi-api-artifacts-18968913554-us-east-1"),
		Key:           jsii.String("slippi/terraform/api.json"),
		Region:        jsii.String("us-east-1"),
		DynamodbTable: jsii.String("tf-state"),
	})

	base := BaseConfig{
		Name:           name,
		IamPath:        iamPath,
		Regions:        []string{"us-west-1"},
		AdminGroupName: jsii.String("infra-admins"),
	}.New(SimpleContext(stack, "base", nil))

	allProviders := base.Providers.All()
	lambdaIam := LambdaIamConfig{
		Path:       iamPath,
		ExecPolicy: base.Policies.LambdaExec.Arn(),
		AssumeRole: base.Policies.LambdaAssumeRole.Json(),
	}

	IpLookupConfig{
		Providers:     allProviders,
		Name:          jsii.String(*name + "-ip-lookup"),
		LambdaIam:     lambdaIam,
		KmsReadPolicy: base.Policies.KmsMain.Read.Arn(),
		KmsArns:       base.KmsMain.Arns(),
		Code:          codeObjectConfig,
	}.New(SimpleContext(stack, "ip_lookup", base.Providers.Main))

	app.Synth()
}
