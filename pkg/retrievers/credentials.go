package retrievers

import (
	"context"
	"fmt"

	"google.golang.org/api/option"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func Credentials(client client.Reader, ctx context.Context, key types.NamespacedName) (option.ClientOption, error) {
	logger := log.FromContext(ctx)

	var jsonKey []byte
	var bucketSecret v1.Secret
	err := client.Get(
		ctx,
		key,
		&bucketSecret,
	)
	if err != nil {
		return nil, fmt.Errorf("unable to extract bucket secret: %w", err)
	}
	jsonKey = bucketSecret.Data["service_account_private_key"]
	if jsonKey != nil {
		logger.Info("adding json to gcs")
		return option.WithCredentialsJSON(jsonKey), nil
	}
	return nil, fmt.Errorf("unable to extract json key from secret")
}
