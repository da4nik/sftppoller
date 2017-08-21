package uploader

import (
	"os"
	"path/filepath"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/da4nik/sftppoller/config"
)

// Upload - uploads file to S3
func Upload(path, key string) {
	logrus.Debugf("Uploading file %s with key %s", path, key)

	creds := credentials.NewStaticCredentials(config.AWSSecretID, config.AWSSecretKey, "")
	sess, err := session.NewSession(&aws.Config{
		Credentials: creds,
		Region:      aws.String("us-east-1"),
	})
	if err != nil {
		logrus.Errorf("Unable to create aws session: %s", err.Error())
		return
	}

	uploader := s3manager.NewUploader(sess)

	f, err := os.Open(path)
	if err != nil {
		logrus.Errorf("Unable to open local file (%s): %s", path, err.Error())
		return
	}

	result, err := uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(config.AWSBucketName),
		Key:    aws.String(filekey(path, key)),
		Body:   f,
	})
	if err != nil {
		logrus.Errorf("Failed to upload file to s3 (%s): %s", path, err.Error())
		return
	}
	logrus.Debugf("File uploaded to: %s\n", result.Location)
	f.Close()

	err = os.Remove(path)
	if err != nil {
		logrus.Errorf("Failed to delete source file (%s): %s", path, err.Error())
		return
	}
}

func filekey(path, key string) string {
	t := time.Now()
	currentTimestamp := t.Format("20060102150405")
	return filepath.Join(key, currentTimestamp+"-"+filepath.Base(path))
}
