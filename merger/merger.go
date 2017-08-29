package merger

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"

	"github.com/da4nik/sftppoller/config"
	"github.com/da4nik/sftppoller/s3client"
	"github.com/da4nik/sftppoller/uploader"
)

var input = make(chan string, 50)

// Init starts merger
func Init(ctx context.Context) {
	go startListening(ctx)
}

// Enqueue enqueues deviceID for merging
func Enqueue(deviceID string) {
	input <- deviceID
}

func startListening(ctx context.Context) {
	log().Debug("Merger starting listening.")
	for {
		select {
		case deviceID := <-input:
			merge(deviceID)
		case <-ctx.Done():
			return
		}
	}
}

// Merge - merges specific device id files
func merge(deviceID string) {
	sess, err := s3client.GetSession()
	if err != nil {
		log().Errorf("Unable to create aws session: %s", err.Error())
		return
	}

	svc := s3.New(sess)
	input := &s3.ListObjectsInput{
		Bucket: aws.String(config.AWSBucketName),
		Prefix: aws.String(deviceID),
	}

	result, err := svc.ListObjects(input)
	if err != nil {
		return
	}

	if len(result.Contents) == 0 {
		log().Debugf("No files found for device id %s", deviceID)
		return
	}

	var wg sync.WaitGroup
	files := make([]string, 0)
	for _, item := range result.Contents {
		if strings.Index(*item.Key, config.ResultFileName) > -1 {
			continue
		}

		wg.Add(1)
		files = append(files, *item.Key)
		go download(*item.Key, &wg)
	}
	wg.Wait()

	if len(files) == 0 {
		log().Debugf("No files to merge for %s", deviceID)
		return
	}

	sort.Strings(files)

	resultFilename := filepath.Join(config.DownloadDir, "merger", deviceID, config.ResultFileName)
	mergeFiles(deviceID, resultFilename, files)

	uploader.Upload(resultFilename, filepath.Join(deviceID, config.ResultFileName))
	log().Infof("Merging for %s complete.", deviceID)
}

func mergeFiles(deviceID, resultFilename string, files []string) error {
	f, err := os.Create(resultFilename)
	if err != nil {
		log().Errorf("Unable to create result file %s %v", resultFilename, err)
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	defer w.Flush()

	for index, filename := range files {
		name := filepath.Join(config.DownloadDir, "merger", filename)

		file, err := os.Open(name)
		if err != nil {
			log().Errorf("Unable to open file %s %v", name, err)
			return err
		}

		scanner := bufio.NewScanner(file)
		skipped := 0
		for scanner.Scan() {
			// Skipping device id string and headers, first 2 lines
			// for second and subsequent files
			if index > 0 && skipped < 2 {
				skipped = skipped + 1
				continue
			}
			fmt.Fprintln(w, scanner.Text())
		}

		if err := scanner.Err(); err != nil {
			log().Errorf("Error reading file %s %v", name, err)
		}
		file.Close()
	}

	return nil
}

// downloads file from s3
// key - full path to s3 file
func download(key string, wg *sync.WaitGroup) {
	defer wg.Done()

	// The session the S3 Downloader will use
	sess, err := s3client.GetSession()
	if err != nil {
		log().Errorf("Unable to create aws session: %s", err.Error())
		return
	}

	// Create a downloader with the session and default options
	downloader := s3manager.NewDownloader(sess)

	// Create a file to write the S3 Object contents to.
	filename := filepath.Join(config.DownloadDir, "merger", key)
	os.MkdirAll(filepath.Dir(filename), os.ModePerm)
	f, err := os.Create(filename)
	if err != nil {
		log().Errorf("Failed to create file %q, %v", filename, err)
		return
	}

	// Write the contents of S3 Object to the file
	_, err = downloader.Download(f, &s3.GetObjectInput{
		Bucket: aws.String(config.AWSBucketName),
		Key:    aws.String(key),
	})
	if err != nil {
		log().Errorf("Failed to upload file (%s), %v", key, err)
		return
	}

	log().Debugf("Downloaded file: %s", key)
}

func log() *logrus.Entry {
	return logrus.WithField("module", "merger")
}
