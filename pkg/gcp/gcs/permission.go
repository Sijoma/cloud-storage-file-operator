package gcs

import (
	"context"
	"fmt"
	"time"

	"cloud.google.com/go/iam"
)

// getBucketPolicy gets the bucket IAM policy.
func (g StorageClient) GetBucketPolicy(ctx context.Context, bucketName string) (*iam.Policy3, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()

	policy, err := g.client.Bucket(bucketName).IAM().V3().Policy(ctx)
	if err != nil {
		return nil, fmt.Errorf("Bucket(%q).IAM().V3().Policy: %w", bucketName, err)
	}
	for _, binding := range policy.Bindings {
		fmt.Printf("%q: %q (condition: %v)\n", binding.Role, binding.Members, binding.Condition)
	}
	return policy, nil
}
