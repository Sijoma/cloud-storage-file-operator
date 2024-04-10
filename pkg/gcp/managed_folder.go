package gcp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"golang.org/x/oauth2/google"
	"google.golang.org/api/iam/v1"
)

type jsonFolderClient struct {
	client   *http.Client
	endpoint string
}

// managedFolderEndpointFormat contains %s to insert the BucketName
const managedFolderEndpointFormat = "https://storage.googleapis.com/storage/v1/b/%s/managedFolders"

func newFolderClient(ctx context.Context) (*jsonFolderClient, error) {
	// Todo: When do we need other scopes?
	scopes := []string{}
	httpClient, err := google.DefaultClient(ctx, scopes...)
	if err != nil {
		return nil, fmt.Errorf("newFolderClient: %w", err)
	}

	return &jsonFolderClient{
		client:   httpClient,
		endpoint: managedFolderEndpointFormat,
	}, nil
}

func (c *jsonFolderClient) listManagedFolders(prefix, bucketName string) (string, error) {
	req, err := c.client.Get(fmt.Sprintf(c.endpoint, bucketName))
	if req != nil {
		defer req.Body.Close()
	}

	if err != nil {
		return "", fmt.Errorf("listManagedFolders: %w", err)
	}

	if req == nil {
		return "", fmt.Errorf("listManagedFolders: nil request")
	}

	all, err := io.ReadAll(req.Body)
	if err != nil {
		return "", err
	}
	fmt.Printf(string(all))

	return "", nil
}

type managedFolderRequest struct {
	Name string `json:"name"`
}

type managedFolderResource struct {
	Kind           string    `json:"kind"`
	Id             string    `json:"id"`
	SelfLink       string    `json:"selfLink"`
	Name           string    `json:"name"`
	Bucket         string    `json:"bucket"`
	CreateTime     time.Time `json:"createTime"`
	UpdateTime     time.Time `json:"updateTime"`
	Metageneration string    `json:"metageneration"`
}

// Not the same as googleapi.Error
type apiError struct {
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
		Errors  []struct {
			Message string `json:"message"`
			Domain  string `json:"domain"`
			Reason  string `json:"reason"`
		} `json:"errors"`
	} `json:"error"`
}

type notFoundError struct {
	folderName string
}

func (n *notFoundError) Error() string {
	return fmt.Sprintf("folder %s not found", n.folderName)
}

func (c *jsonFolderClient) getManagedFolder(ctx context.Context, prefix string, bucketName string) (*managedFolderResource, error) {
	endpoint := fmt.Sprintf(c.endpoint, bucketName)
	endpoint = endpoint + "/" + url.PathEscape(prefix)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("getManagedFolder: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("getManagedFolder: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, &notFoundError{prefix}
	}

	if resp.StatusCode >= 300 {
		var reqErr apiError
		err := json.NewDecoder(resp.Body).Decode(&reqErr)
		if err != nil {
			return nil, fmt.Errorf("getManagedFolder: %w", err)
		}
		return nil, fmt.Errorf("getManagedFolder: %s, %v", resp.Status, reqErr)
	}

	var managedFolder managedFolderResource
	err = json.NewDecoder(resp.Body).Decode(&managedFolder)
	if err != nil {
		return nil, fmt.Errorf("getManagedFolder: %w", err)
	}
	return &managedFolder, nil
}

func (c *jsonFolderClient) createManagedFolder(ctx context.Context, prefix, bucketName string) (*managedFolderResource, error) {
	folder := managedFolderRequest{Name: prefix}
	body, err := json.Marshal(folder)
	if err != nil {
		return nil, fmt.Errorf("createManagedFolder: %w", err)
	}
	endpoint := fmt.Sprintf(c.endpoint, bucketName)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("createManagedFolder: %w", err)
	}
	defer req.Body.Close()
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("createManagedFolder: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var folderError apiError
		err := json.NewDecoder(resp.Body).Decode(&folderError)
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("createManagedFolder: %s", folderError.Error.Message)
	}

	var managedFolder managedFolderResource
	err = json.NewDecoder(resp.Body).Decode(&managedFolder)
	if err != nil {
		return nil, fmt.Errorf("createManagedFolder: %w", err)
	}

	return &managedFolder, nil
}

func (c *jsonFolderClient) getOrCreateManagedFolder(ctx context.Context, prefix string, bucketName string) (string, error) {
	folder, err := c.getManagedFolder(ctx, prefix, bucketName)
	if err != nil {
		var notFoundErr *notFoundError
		switch {
		case errors.As(err, &notFoundErr):
			createdFolder, err := c.createManagedFolder(ctx, prefix, bucketName)
			if err != nil {
				return "", fmt.Errorf("getOrCreateManagedFolder: %w", err)
			}
			return createdFolder.Name, nil
		default:
			return "", fmt.Errorf("getOrCreateManagedFolder: %w", err)
		}
	}

	return folder.Name, nil
}

func (c *jsonFolderClient) getIAMPolicy(ctx context.Context, folder, bucketName string) (*iam.Policy, error) {
	endpoint := fmt.Sprintf(c.endpoint, bucketName)
	endpoint += "/" + url.PathEscape(folder) + "/iam"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("getIAMPolicy: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("getIAMPolicy: %w", err)
	}

	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("getIAMPolicy: %s, %v", resp.Status, resp.StatusCode)
	}

	var iamPolicy iam.Policy
	err = json.NewDecoder(resp.Body).Decode(&iamPolicy)
	if err != nil {
		return nil, fmt.Errorf("getIAMPolicy: %w", err)
	}

	return &iamPolicy, nil
}

func (c *jsonFolderClient) setIAMPolicy(ctx context.Context, folder, bucketName string, policy iam.Policy) error {
	endpoint := fmt.Sprintf(c.endpoint, bucketName)
	endpoint += "/" + url.PathEscape(folder) + "/iam"

	body, err := json.Marshal(policy)
	if err != nil {
		return fmt.Errorf("setIAMPolicy: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, endpoint, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("getIAMPolicy: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("getIAMPolicy: %w", err)
	}

	if resp.StatusCode >= 300 {
		return fmt.Errorf("getIAMPolicy: %s, %v", resp.Status, resp.StatusCode)
	}

	return nil
}

func (c *jsonFolderClient) addIAMBinding(ctx context.Context, folder, bucketName, role, principal string) error {
	iamPolicy, err := c.getIAMPolicy(ctx, folder, bucketName)
	if err != nil {
		return err
	}

	var binding *iam.Binding
	for _, b := range iamPolicy.Bindings {
		if b.Role == role {
			binding = b
			break
		}
	}

	if binding != nil {
		// If the binding exists, adds the member to the binding
		binding.Members = append(binding.Members, principal)
	} else {
		// If the binding does not exist, adds a new binding to the policy
		binding = &iam.Binding{
			Role:    role,
			Members: []string{principal},
		}
		iamPolicy.Bindings = append(iamPolicy.Bindings, binding)
	}

	err = c.setIAMPolicy(ctx, folder, bucketName, *iamPolicy)
	if err != nil {
		return err
	}

	return nil
}
