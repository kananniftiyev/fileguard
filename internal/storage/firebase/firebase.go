// Package firebase provides functionalities to interact with Firebase services such as Firestore and Storage.
package firebase

import (
	"cloud.google.com/go/storage"
	"context"
	"errors"
	"fileguard/internal/common"
	"fileguard/internal/db"
	"fileguard/utils"
	"fmt"
	"google.golang.org/api/iterator"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
)

// TODO: Fix all Params of func. and whole file name ect.

const maxFolderSize = 200 * 1024 * 1024

type Storage struct {
	Bucket  *storage.BucketHandle
	Context context.Context
}

func NewStorage() (*Storage, error) {
	app, ctx, err := common.GetFirebaseApp()
	if err != nil {
		panic(err)
	}

	client, err := app.Storage(ctx)
	if err != nil {
		return nil, err
	}

	bucket, err := client.DefaultBucket()
	if err != nil {
		return nil, err
	}

	return &Storage{Bucket: bucket, Context: ctx}, nil
}

// TODO: Detect file content difference.
func (s *Storage) UploadFile(localFilePath string, userToken utils.Token) error {
	folderPath := "x"

	totalFolderSize, err := s.GetFolderSize(folderPath)

	if err != nil {
		return err
	}

	if totalFolderSize >= maxFolderSize {
		return errors.New("Cannot surpass max folder size")
	}

	// Open the file
	file, err := os.Open(localFilePath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Read the file content into a byte slice
	content, err := ioutil.ReadAll(file)
	if err != nil {
		return err
	}

	// Username will be folder name
	dst := "y/" + file.Name()

	// Create a writer
	w := s.Bucket.Object(dst).NewWriter(s.Context)

	// Write the content to the writer
	if _, err := w.Write(content); err != nil {
		return err
	}

	// Close the writer
	if err := w.Close(); err != nil {
		return err
	}

	db, err := db.NewDatabase()
	if err != nil {
		return err
	}

	err = db.CreateNewFileRecord(file, userToken.UserID)

	if err != nil {
		return err
	}
	log.Println("File uploaded successfully!")
	return nil
}

// TODO: Redesign this better.
func (s *Storage) DownloadFile(objectPath string, localPath string) error {
	rc, err := s.Bucket.Object(objectPath).NewReader(s.Context)
	if err != nil {
		return err
	}
	defer rc.Close()

	// Create local file
	file, err := os.Create(localPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Copy object data to local file
	if _, err := io.Copy(file, rc); err != nil {
		return err
	}

	return nil
}

func (s *Storage) DownloadAllFiles(folderPath string, localPath string) error {
	if folderPath == "" {
		return errors.New("You should add Folder Path")
	}

	it := s.Bucket.Objects(context.Background(), &storage.Query{Prefix: folderPath})

	for {
		objs, err := it.Next()
		if err == iterator.Done {
			break
		}

		if err != nil {
			return err
		}
		fmt.Println(objs.Name)
		fileDir := filepath.Join(localPath, filepath.Base(objs.Name))
		err = s.DownloadFile(objs.Name, fileDir)
		if err != nil {
			return err
		}

		log.Printf("Downloaded file: %s\n", objs.Name)
	}

	return nil

}

func (s *Storage) GetFolderSize(folderPathInStorage string) (int64, error) {
	var totalFolderSize int64
	it := s.Bucket.Objects(s.Context, &storage.Query{Prefix: folderPathInStorage})

	for {
		objAttrs, err := it.Next()
		if err == iterator.Done {
			break
		}

		if err != nil {
			return 0, err
		}

		totalFolderSize += objAttrs.Size
	}

	return totalFolderSize, nil
}
