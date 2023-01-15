package healthcheck

import (
	"github.com/ContinentalBreakfast17/peppy/terraform/lib/_common"
	"github.com/aws/jsii-runtime-go"
)

type healthcheck struct {
	Healthchecker healthchecker
	Responder     healthcheckResponder
	cron          healthcheckCron
	alarm         healthcheckAlarm
}

type HealthcheckConfig struct {
	Providers      common.Providers
	Name           *string
	KmsWritePolicy *string
	KmsReadPolicy  *string
	KmsArns        common.MultiRegionId
	Code           common.ObjectConfig
	LambdaIam      common.LambdaIamConfig
	ApiUrl         string
	SendAlarmsTo   []string
}

func (cfg HealthcheckConfig) New(ctx common.TfContext) healthcheck {
	healthchecker := healthcheckerConfig{
		providers:      cfg.Providers,
		name:           cfg.Name,
		kmsArns:        cfg.KmsArns,
		kmsWritePolicy: cfg.KmsWritePolicy,
		code:           cfg.Code,
		lambdaIam:      cfg.LambdaIam,
		apiUrl:         cfg.ApiUrl,
	}.new(common.SimpleContext(ctx.Scope, ctx.Id, ctx.Provider))

	healthcheckResponder := healthcheckResponderConfig{
		providers:     cfg.Providers,
		name:          jsii.String(*cfg.Name + "-responder"),
		kmsReadPolicy: cfg.KmsReadPolicy,
		code:          cfg.Code,
		lambdaIam:     cfg.LambdaIam,
		apiUrl:        cfg.ApiUrl,
		healthchecker: healthchecker,
	}.new(common.SimpleContext(ctx.Scope, ctx.Id+"_responder", ctx.Provider))

	healthcheckCron := healthcheckCronConfig{
		providers:     cfg.Providers,
		name:          jsii.String(*cfg.Name + "-cron"),
		healthchecker: healthchecker,
	}.new(common.SimpleContext(ctx.Scope, ctx.Id+"_cron", ctx.Provider))

	healthcheckAlarm := healthcheckAlarmConfig{
		providers:     cfg.Providers,
		name:          jsii.String(*cfg.Name + "-alarm"),
		healthchecker: healthchecker,
		sendAlarmsTo:  cfg.SendAlarmsTo,
	}.new(common.SimpleContext(ctx.Scope, ctx.Id+"_alarm", ctx.Provider))

	return healthcheck{
		Healthchecker: healthchecker,
		Responder:     healthcheckResponder,
		cron:          healthcheckCron,
		alarm:         healthcheckAlarm,
	}
}
