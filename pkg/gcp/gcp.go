package gcp

import (
	"context"
	"fmt"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iam/v1"
	"google.golang.org/api/option"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

type Client struct {
	gcs *storage.Client
	// For creating GCP Service Accounts
	client    *iam.Service
	projectID string
	// Custom client for ManagedFolders (no official support in storage client)
	// https://cloud.google.com/storage/docs/access-control/using-iam-permissions#managed-folder-iam
	folderService *jsonFolderClient
}

func (p Client) ProjectID() string {
	return p.projectID
}

func NewGCPClient(ctx context.Context, gcpProjectID string, opts ...option.ClientOption) (*Client, error) {
	gcsClient, err := storage.NewClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create gcp client: %w", err)
	}

	client, err := iam.NewService(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("NewGCPClient: %w", err)
	}

	folderService, err := newFolderClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("NewGCPClient: %w", err)
	}

	return &Client{
		gcs:           gcsClient,
		client:        client,
		folderService: folderService,
		projectID:     gcpProjectID,
	}, nil
}

// CreateServiceAccount creates a service account.
func (p Client) CreateServiceAccount(ctx context.Context, saName, kubernetesSA, kubernetesNamespace string) (*iam.ServiceAccount, error) {
	logger := log.FromContext(ctx)

	displayName := "storage-" + saName + "-" + kubernetesNamespace
	account, err := p.getOrCreateServiceAccount(ctx, saName, displayName)
	if err != nil {
		return nil, fmt.Errorf("CreateServiceAccount: %w", err)
	}
	logger.Info("service account connected", "serviceAccount", account.Name)

	// Workload identity binding - This needs to be on the service account
	member := fmt.Sprintf("serviceAccount:%s.svc.id.goog[%s/%s]", account.ProjectId, kubernetesNamespace, kubernetesSA)
	err = p.addBindingOnSA(ctx, account, member, "roles/iam.workloadIdentityUser")
	if err != nil {
		return nil, fmt.Errorf("CreateServiceAccount: %w", err)
	}
	logger.V(2).Info("workload identity added", "member", member)

	return account, nil
}

func (p Client) CreateManagedFolder(ctx context.Context, prefix, bucketName string) (string, error) {
	folder, err := p.folderService.getOrCreateManagedFolder(ctx, prefix, bucketName)
	if err != nil {
		return "", fmt.Errorf("CreateManagedFolder: %w", err)
	}

	return folder, nil
}

func (p Client) GrantRoleOnFolder(ctx context.Context, folder, bucketName, role, principal string) error {
	err := p.folderService.addIAMBinding(ctx, folder, bucketName, role, principal)
	if err != nil {
		return fmt.Errorf("GrantRoleOnFolder: %w", err)
	}
	return nil
}
