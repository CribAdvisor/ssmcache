package ssmcache

import (
	"context"
	"fmt"
	"reflect"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ssm"
	"github.com/aws/aws-sdk-go-v2/service/ssm/types"
	"github.com/google/uuid"
)

type MockParamStore struct {
	cache map[string]string
}

func (store MockParamStore) GetParameter(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error) {
	value := store.cache[*params.Name]
	return &ssm.GetParameterOutput{Parameter: &types.Parameter{Value: &value}}, nil
}

func (store MockParamStore) PutParameter(ctx context.Context, params *ssm.PutParameterInput, optFns ...func(*ssm.Options)) (*ssm.PutParameterOutput, error) {
	store.cache[*params.Name] = *params.Value
	return &ssm.PutParameterOutput{}, nil
}

func NewMock(initial map[string]string) (MockParamStore, *ssmcache) {
	client := MockParamStore{cache: initial}
	var options SSMCacheOptions
	mergeDefaults(&options)
	return client, &ssmcache{
		options: options,
		ssm:     client,
	}
}

func TestSet(t *testing.T) {
	mock, cache := NewMock(make(map[string]string))

	key := "test"
	expectedTTL := uint(time.Hour.Seconds())
	expectedValue := uuid.NewString()

	err := cache.Set(key, expectedValue, expectedTTL)
	if err != nil {
		t.Fatalf("Set(%v, %v, %v), error %v", key, expectedValue, expectedTTL, err)
	}

	paramName := fmt.Sprintf("/cache/%s", key)
	if _, ok := mock.cache[paramName]; !ok {
		t.Fatalf("Param %s not found in %v", paramName, mock.cache)
	}
}

func TestGet(t *testing.T) {
	_, cache := NewMock(map[string]string{
		"/cache/test": `{"TTL":3600,"Value":"test"}`,
	})

	key := "test"
	expected := &ParamValue{
		TTL:   3600,
		Value: "test",
	}

	got, err := cache.Get(key)
	if err != nil {
		t.Fatalf("Get(%v), error: %v", key, err)
	}
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("Get(%v) = %v, expected %v", key, got, expected)
	}
}
