package main

import (
	. "github.com/ContinentalBreakfast17/peppy/terraform/lib/_common"
	"github.com/ContinentalBreakfast17/peppy/terraform/lib/data"
	"github.com/ContinentalBreakfast17/peppy/terraform/lib/kms"
	"github.com/ContinentalBreakfast17/peppy/terraform/lib/provider"
	"github.com/aws/constructs-go/constructs/v10"
	"github.com/aws/jsii-runtime-go"
	"github.com/hashicorp/terraform-cdk-go/cdktf"
)

func main() {
	app := cdktf.NewApp(nil)
	stack := cdktf.NewTerraformStack(scope, jsii.String("aws"))

	// todo: parse vars
	name := "cdktf"

	cdktf.NewS3Backend(stack, &cdktf.S3BackendProps{
		// todo: move to vars
		Bucket:        jsii.String("slippi-api-artifacts-18968913554-us-east-1"),
		Key:           jsii.String("slippi/terraform/api-cdktf.json"),
		Region:        jsii.String("us-east-1"),
		DynamodbTable: jsii.String("tf-state"),
	})

	providers := provider.ProviderConfig{
		// todo: move to vars
		Regions: []string{"us-west-1"},
	}.NewProviders(SimpleContext(stack, "providers", nil))

	datasources := data.DataSourceConfig{
		// todo: move to vars
		AdminGroupName: jsii.String("infra-admins"),
	}.NewDataSources(SimpleContext(stack, "data", providers.Main))

	kms.KeyConfig{
		Providers:   providers.Copies,
		Name:        jsii.String(name + "_main"),
		Description: jsii.String(name + " main key"),
		AccountId:   datasources.AccountId(),
		KeyAdmins:   *datasources.AdminUsers(),
	}.NewKeySet(SimpleContext(stack, "kms_main", providers.Main))

	app.Synth()
}
