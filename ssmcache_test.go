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
	params map[string]string
	calls  []string
}

func (store *MockParamStore) GetParameter(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error) {
	store.calls = append(store.calls, "GetParameter")
	value := store.params[*params.Name]
	now := new(time.Time)
	*now = time.Now()
	return &ssm.GetParameterOutput{Parameter: &types.Parameter{Value: &value, LastModifiedDate: now}}, nil
}

func (store *MockParamStore) PutParameter(ctx context.Context, params *ssm.PutParameterInput, optFns ...func(*ssm.Options)) (*ssm.PutParameterOutput, error) {
	store.calls = append(store.calls, "PutParameter")
	store.params[*params.Name] = *params.Value
	return &ssm.PutParameterOutput{}, nil
}

func (store *MockParamStore) DeleteParameter(ctx context.Context, params *ssm.DeleteParameterInput, optFns ...func(*ssm.Options)) (*ssm.DeleteParameterOutput, error) {
	store.calls = append(store.calls, "DeleteParameter")
	delete(store.params, *params.Name)
	return &ssm.DeleteParameterOutput{}, nil
}

func NewMock(initial map[string]string) (*MockParamStore, *SSMCache) {
	client := MockParamStore{params: initial}
	secret := new(bool)
	*secret = true
	basePath := new(string)
	*basePath = "/testcache"
	options := SSMCacheOptions{
		Secret:   secret,
		BasePath: basePath,
		KeyId:    nil,
	}
	return &client, &SSMCache{
		options: options,
		ssm:     &client,
	}
}

func getParamName(key string) string {
	return fmt.Sprintf("/testcache/%s", key)
}

func createInitial(values ...string) (initial map[string]string) {
	initial = map[string]string{}
	for i := 0; i < len(values); i += 2 {
		initial[values[i]] = values[i+1]
	}
	return
}

func TestSet(t *testing.T) {
	mock, cache := NewMock(createInitial())

	key := "test/set"
	ttl := time.Hour
	value := uuid.NewString()

	err := cache.Set(context.Background(), key, value, ttl)
	if err != nil {
		t.Fatalf("Set(%v, %v, %v), error %v", key, value, ttl, err)
	}

	paramName := getParamName(key)
	if _, ok := mock.params[paramName]; !ok {
		t.Fatalf("Param %s not found in %v", paramName, mock.params)
	}

	expectedCalls := []string{"PutParameter"}
	if !reflect.DeepEqual(mock.calls, expectedCalls) {
		t.Fatalf("Calls = %v, expected %v", mock.calls, expectedCalls)
	}
}

func TestGet(t *testing.T) {
	key := "test/get"
	value := uuid.NewString()

	mock, cache := NewMock(createInitial(
		getParamName(key), fmt.Sprintf(`{"TTL":3600,"Value":"%s"}`, value),
	))

	got, err := cache.Get(context.Background(), key)
	if err != nil {
		t.Fatalf("Get(%v), error: %v", key, err)
	}
	if *got != value {
		t.Fatalf("Get(%v) = %v, expected %v", key, *got, value)
	}

	expectedCalls := []string{"GetParameter"}
	if !reflect.DeepEqual(mock.calls, expectedCalls) {
		t.Fatalf("Calls = %v, expected %v", mock.calls, expectedCalls)
	}
}

func TestGetMissing(t *testing.T) {
	key := "test/get/missing"

	_, cache := NewMock(createInitial())

	got, err := cache.Get(context.Background(), key)
	if err == nil {
		t.Fatalf("Expected error, got nil")
	}
	if got != nil {
		t.Fatalf("Got %v, expected nil", *got)
	}
}

func TestOverwrite(t *testing.T) {
	key := "test/overwrite"

	mock, cache := NewMock(createInitial(
		getParamName(key), uuid.NewString(),
	))

	value := uuid.NewString()
	ttl := time.Hour

	cache.Set(context.Background(), key, value, ttl)
	got, _ := cache.Get(context.Background(), key)
	if *got != value {
		t.Fatalf("Got %v, expected %v", *got, value)
	}

	expectedCalls := []string{"PutParameter", "GetParameter"}
	if !reflect.DeepEqual(mock.calls, expectedCalls) {
		t.Fatalf("Calls = %v, expected %v", mock.calls, expectedCalls)
	}
}

func TestFutureTTL(t *testing.T) {
	mock, cache := NewMock(createInitial())

	key := "test/ttl/valid"
	value := uuid.NewString()
	ttl := time.Minute * 5

	cache.Set(context.Background(), key, value, ttl)

	got, _ := cache.Get(context.Background(), key)
	if *got != value {
		t.Fatalf("Got %v, expected %v", *got, value)
	}

	expectedCalls := []string{"PutParameter", "GetParameter"}
	if !reflect.DeepEqual(mock.calls, expectedCalls) {
		t.Fatalf("Calls = %v, expected %v", mock.calls, expectedCalls)
	}
}

func TestExpiredTTL(t *testing.T) {
	mock, cache := NewMock(createInitial())

	key := "test/ttl/valid"
	value := uuid.NewString()
	ttl := -1 * time.Second

	cache.Set(context.Background(), key, value, ttl)

	got, _ := cache.Get(context.Background(), key)
	if got != nil {
		t.Fatalf("Got %v, expected nil", *got)
	}

	expectedCalls := []string{"PutParameter", "GetParameter", "DeleteParameter"}
	if !reflect.DeepEqual(mock.calls, expectedCalls) {
		t.Fatalf("Calls = %v, expected %v", mock.calls, expectedCalls)
	}
}
