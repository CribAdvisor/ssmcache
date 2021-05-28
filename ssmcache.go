package ssmcache

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

type SSMCacheOptions struct {
	// Parameter type, `true` for `SecureString`, false for `String`
	Secret *bool

	// Where the parameters are stored within SSM *excluding trailing slash*
	//
	// Be sure to update the IAM policy (see `README.md`) if changed
	BasePath *string

	// ARN of KMS key id to use to encrypt parameter value
	KeyId *string
}

type ssmcache struct {
	options SSMCacheOptions
	ssm     ParamStore
}

func getDefaultOptions() *SSMCacheOptions {
	secret := new(bool)
	*secret = true
	basePath := new(string)
	*basePath = "/cache"
	return &SSMCacheOptions{
		Secret:   secret,
		BasePath: basePath,
		KeyId:    nil,
	}
}

func mergeDefaults(options *SSMCacheOptions) {
	defaultOptions := getDefaultOptions()
	if options.BasePath == nil {
		options.BasePath = defaultOptions.BasePath
	}
	if options.Secret == nil {
		options.Secret = defaultOptions.Secret
	}
	if options.KeyId == nil {
		options.KeyId = defaultOptions.KeyId
	}
}

func New(options *SSMCacheOptions) (*ssmcache, error) {
	mergeDefaults(options)

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return nil, err
	}
	client := ssm.NewFromConfig(cfg)

	return &ssmcache{
		options: *options,
		ssm:     client,
	}, nil
}

func escapeParameterName(name string) string {
	re := regexp.MustCompile("/[^a-zA-Z0-9_.-/]/g")
	return re.ReplaceAllString(name, "_")
}

func (cache ssmcache) getParameterName(key string) string {
	return fmt.Sprintf("%s/%s", *cache.options.BasePath, escapeParameterName(key))
}

func (cache ssmcache) getParamType() types.ParameterType {
	if *cache.options.Secret {
		return types.ParameterTypeSecureString
	} else {
		return types.ParameterTypeString
	}
}

type ParamValue struct {
	TTL   uint
	Value string
}

func (cache ssmcache) Set(key string, value string, ttl uint) error {
	parameterName := cache.getParameterName(key)
	param := ParamValue{ttl, value}

	paramJSON, err := json.Marshal(param)
	if err != nil {
		return err
	}

	jsonString := new(string)
	*jsonString = string(paramJSON)

	paramType := cache.getParamType()

	cache.ssm.PutParameter(context.TODO(), &ssm.PutParameterInput{Name: &parameterName, Value: jsonString, Type: paramType})
	return nil
}

func (cache ssmcache) Get(key string) (*ParamValue, error) {
	parameterName := cache.getParameterName(key)

	valueJSON, err := cache.ssm.GetParameter(context.TODO(), &ssm.GetParameterInput{Name: &parameterName, WithDecryption: *cache.options.Secret})
	if *valueJSON.Parameter.Value == "" {
		return nil, fmt.Errorf("no parameter found: %s", parameterName)
	}
	if err != nil {
		return nil, err
	}

	var value ParamValue
	err = json.Unmarshal([]byte(*valueJSON.Parameter.Value), &value)
	if err != nil {
		return nil, err
	}

	return &value, nil
}
