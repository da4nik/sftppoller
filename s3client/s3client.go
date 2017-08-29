package s3client

import (
	"fmt"

	"github.com/Sirupsen/logrus"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/da4nik/sftppoller/config"
)

var awsSession *session.Session

// GetSession - returns aws session
func GetSession() (*session.Session, error) {
	if awsSession != nil {
		return awsSession, nil
	}

	creds := credentials.NewStaticCredentials(config.AWSSecretID, config.AWSSecretKey, "")
	sess, err := session.NewSession(&aws.Config{
		Credentials: creds,
		Region:      aws.String("us-east-1"),
	})

	if err != nil {
		logrus.Errorf("Unable to create aws session: %s", err.Error())
		return nil, fmt.Errorf("Unable to create aws session: %s", err.Error())
	}

	awsSession = sess
	return sess, nil
}
