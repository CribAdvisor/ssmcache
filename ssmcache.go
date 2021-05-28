package ssmcache

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

type SSMCacheOptions struct {
	// Secret is used to set SSM parameter type, true for SecureString, false for String
	Secret *bool

	// BasePath is where the parameters are stored within SSM, excluding trailing slash
	//
	// Be sure to update the IAM policy (see README.md) to match this
	BasePath *string

	// KeyId is the ARN of KMS key id to use to encrypt parameter value
	KeyId *string
}

type ssmcache struct {
	options SSMCacheOptions
	ssm     paramStore
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

// New creates a new cache, if options are nil, the defaults are used
//
// Defaults:
// Secret=true
// BasePath="/cache"
// KeyId=nil
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

func (cache *ssmcache) getParameterName(key string) string {
	return fmt.Sprintf("%s/%s", *cache.options.BasePath, escapeParameterName(key))
}

func (cache *ssmcache) getParamType() types.ParameterType {
	if *cache.options.Secret {
		return types.ParameterTypeSecureString
	} else {
		return types.ParameterTypeString
	}
}

type paramValue struct {
	TTL   uint
	Value string
}

// Set puts a parameter into SSM for the given key, excluding the BasePath of the cache
func (cache *ssmcache) Set(key string, value string, ttl time.Duration) error {
	parameterName := cache.getParameterName(key)
	param := paramValue{uint(ttl.Seconds()), value}

	paramJSON, err := json.Marshal(param)
	if err != nil {
		return err
	}

	jsonString := new(string)
	*jsonString = string(paramJSON)

	paramType := cache.getParamType()

	_, err = cache.ssm.PutParameter(context.TODO(), &ssm.PutParameterInput{Name: &parameterName, Value: jsonString, Type: paramType})
	return err
}

// Get retrieves a parameter from SSM with the given key, excluding the BasePath of the cache
func (cache *ssmcache) Get(key string) (*string, error) {
	parameterName := cache.getParameterName(key)

	parameterOutput, err := cache.ssm.GetParameter(context.TODO(), &ssm.GetParameterInput{Name: &parameterName, WithDecryption: *cache.options.Secret})
	if err != nil {
		return nil, err
	}
	if *parameterOutput.Parameter.Value == "" {
		return nil, fmt.Errorf("no parameter found: %s", parameterName)
	}

	var value paramValue
	err = json.Unmarshal([]byte(*parameterOutput.Parameter.Value), &value)
	if err != nil {
		return nil, err
	}

	timestamp := (*parameterOutput.Parameter.LastModifiedDate).Unix()
	if time.Now().Unix() > (timestamp + int64(time.Second*time.Duration(value.TTL))) {
		cache.ssm.DeleteParameter(context.TODO(), &ssm.DeleteParameterInput{Name: &parameterName})
		return nil, nil
	}

	return &value.Value, nil
}
