package api

import (
	"fmt"

	"github.com/ContinentalBreakfast17/peppy/terraform/lib/_common"
	"github.com/aws/jsii-runtime-go"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/acmcertificate"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/acmcertificatevalidation"
	. "github.com/cdktf/cdktf-provider-aws-go/aws/v10/route53record"
)

type appsyncCert struct {
	AcmCertificate
	Validation AcmCertificateValidation
}

type appsyncCertConfig struct {
	domainName *string
	hostedZone *string
}

func (cfg appsyncCertConfig) new(ctx common.TfContext) appsyncCert {
	cert := NewAcmCertificate(ctx.Scope, jsii.String(ctx.Id), &AcmCertificateConfig{
		Provider:                ctx.Provider,
		DomainName:              cfg.domainName,
		SubjectAlternativeNames: jsii.Strings("*." + *cfg.domainName),
		ValidationMethod:        jsii.String("DNS"),
	})

	records := []Route53Record{}
	dvos := cert.DomainValidationOptions()
	i := float64(0)
	for dvo := dvos.Get(jsii.Number(i)); dvo != nil && dvo.ComplexObjectIndex().(float64) >= i; i++ {
		records = append(records, NewRoute53Record(ctx.Scope, jsii.String(fmt.Sprintf("%s_record_%d", ctx.Id, i)), &Route53RecordConfig{
			Provider:       ctx.Provider,
			ZoneId:         cfg.hostedZone,
			Name:           dvo.ResourceRecordName(),
			Records:        &[]*string{dvo.ResourceRecordValue()},
			Type:           dvo.ResourceRecordType(),
			Ttl:            jsii.Number(60),
			AllowOverwrite: jsii.Bool(true),
		}))
	}

	validationRecordFqdns := make([]*string, len(records))
	for i, record := range records {
		validationRecordFqdns[i] = record.Fqdn()
	}

	validation := NewAcmCertificateValidation(ctx.Scope, jsii.String(ctx.Id+"_validation"), &AcmCertificateValidationConfig{
		Provider:              ctx.Provider,
		CertificateArn:        cert.Arn(),
		ValidationRecordFqdns: &validationRecordFqdns,
	})

	return appsyncCert{cert, validation}
}
