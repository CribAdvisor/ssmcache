# ssmcache

Cache un/encrypted expiring values as AWS SSM parameters with a TTL

## Usage

[Docs](https://pkg.go.dev/github.com/CribAdvisor/ssmcache)

```go
import (
    "github.com/CribAdvisor/ssmcache"
)

func main() {
    cache := ssmcache.New(ssmcache.SSMCacheOptions{
        Secret: true,
        BasePath: "/cache",
        KeyId: nil,
    })

    accessToken, err := cache.Get("my_token")
    if err != nil {
        panic(err)
    }
    if token == nil {
        // obtain a new token
        // newToken, ttl := getNewToken(...)
        accessToken = newToken
        cache.Set("my_token", accessToken, ttl)
    }
}
```

## Required IAM permissions

**NOTE:**
1. Replace `Resource` with your AWS region and account ID
2. Replace `/cache` with the modified `BasePath` if applicable
3. Add `kms:Encrypt` and `kms:Decrypt` actions and resources for your KMS key (if applicable)
```
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "Stmt1585412677747",
      "Action": [
        "ssm:DeleteParameter",
        "ssm:GetParameter",
        "ssm:PutParameter"
      ],
      "Effect": "Allow",
      "Resource": "arn:aws:ssm:<region>:<account_ID>:parameter/cache/*"
    }
  ]
}
```
