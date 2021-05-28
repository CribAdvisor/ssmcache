package ssmcache

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ssm"
)

type ParamStore interface {
	GetParameter(ctx context.Context,
		params *ssm.GetParameterInput,
		optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error)
	PutParameter(ctx context.Context,
		params *ssm.PutParameterInput,
		optFns ...func(*ssm.Options)) (*ssm.PutParameterOutput, error)
	DeleteParameter(ctx context.Context,
		params *ssm.DeleteParameterInput,
		optFns ...func(*ssm.Options)) (*ssm.DeleteParameterOutput, error)
}
