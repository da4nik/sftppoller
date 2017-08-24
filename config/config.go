package config

import (
	"os"
	"strconv"
)

var (
	// SFTPPollIntervalSeconds inteval of ftp polling (seconds)
	SFTPPollIntervalSeconds = 60

	// SFTPAddr address
	SFTPAddr = "localhost:2222"

	// SFTPUser sftp username
	SFTPUser = "sftp_user"

	// SFTPPassword sftp password
	SFTPPassword = "sftp_pass"

	// DownloadDir - download directory
	DownloadDir = "."

	// AWSSecretID aws secret key id
	AWSSecretID = "secret_id"

	// AWSSecretKey aws secret key
	AWSSecretKey = "secret_key"

	// AWSBucketName aws bucket for csvs
	AWSBucketName = "sn-wearable-csvs"

	// LogFileName - log file name
	LogFileName = ""
)

// Load - loads config from env or files
func Load() {
	if os.Getenv("SFTP_POLL_INTERVAL_SECONDS") != "" {
		interval, err := strconv.Atoi(os.Getenv("SFTP_POLL_INTERVAL_SECONDS"))
		if err == nil {
			SFTPPollIntervalSeconds = interval
		}
	}

	SFTPAddr = getEnvValue("SFTP_ADDR", SFTPAddr)
	SFTPUser = getEnvValue("SFTP_USER", SFTPUser)
	SFTPPassword = getEnvValue("SFTP_PASSWORD", SFTPPassword)

	DownloadDir = getEnvValue("DOWNLOAD_DIR", DownloadDir)
	LogFileName = getEnvValue("LOG_FILE_NAME", LogFileName)

	AWSSecretID = getEnvValue("AWS_SECRET_ID", AWSSecretID)
	AWSSecretKey = getEnvValue("AWS_SECRET_KEY", AWSSecretKey)
	AWSBucketName = getEnvValue("AWS_BUCKET_NAME", AWSBucketName)
}

func getEnvValue(varName string, currentValue string) string {
	if os.Getenv(varName) != "" {
		return os.Getenv(varName)
	}
	return currentValue
}
