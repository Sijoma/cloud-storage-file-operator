package gcs

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

type StorageClient struct {
	client *storage.Client
}

func NewGcsClient(ctx context.Context, opts ...option.ClientOption) (*StorageClient, error) {
	client, err := storage.NewClient(ctx, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed creating gcs client %w", err)
	}

	return &StorageClient{
		client: client,
	}, nil
}

func (g StorageClient) FindObjects(ctx context.Context, bucket string, sq storage.Query) ([]string, error) {
	var foundObjects []string
	obj := g.client.Bucket(bucket).Objects(ctx, &sq)
	for {
		objectArrs, err := obj.Next()
		if errors.Is(err, iterator.Done) {
			break
		}
		if err != nil {
			return nil, err
		}
		foundObjects = append(foundObjects, objectArrs.Name)
	}
	return foundObjects, nil
}

func (g StorageClient) CopyFiles(ctx context.Context, bucketName string, sq storage.Query, targetPrefix string) error {
	foundObjects, err := g.FindObjects(ctx, bucketName, sq)
	if err != nil {
		fmt.Println("naaah")
		return err
	}
	if len(foundObjects) == 0 {
		return fmt.Errorf("no objects found")
	}

	err = g.copyFiles(ctx, bucketName, sq.Prefix, targetPrefix, foundObjects)
	if err != nil {
		return err
	}
	return nil
}

func (g StorageClient) copyFiles(ctx context.Context, bucket, prefix, targetPrefix string, objectKeys []string) error {
	wg := sync.WaitGroup{}
	for i, obj := range objectKeys {
		targetPath, found := strings.CutPrefix(obj, prefix)
		if !found {
			return errors.New("did not found prefix on obj")
		}

		src := obj
		dst := targetPrefix + targetPath

		wg.Add(1)
		go func() {
			err := g.copyFile(ctx, bucket, src, dst)
			if err != nil {
				// We should bubble these up
				fmt.Println(err)
			}
			wg.Done()
		}()
		// Todo: WIP Artificial slowdown - use buffered channel
		if i%100 == 0 {
			time.Sleep(time.Millisecond * 200)
		}
	}
	wg.Wait()
	return nil
}

func (g StorageClient) copyFile(ctx context.Context, bucket, srcObj, dstObj string) error {
	src := g.client.Bucket(bucket).Object(srcObj)
	dst := g.client.Bucket(bucket).Object(dstObj)

	//dst = dst.If(storage.Conditions{DoesNotExist: true}).

	copier := dst.CopierFrom(src)
	_, err := copier.Run(ctx)
	if err != nil {
		return fmt.Errorf("Object(%q).CopierFrom(%q).Run: %w", dstObj, srcObj, err)
	}
	return nil
}
