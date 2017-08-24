package poller

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/da4nik/sftppoller/config"
	"github.com/da4nik/sftppoller/uploader"
	"github.com/pkg/sftp"

	"golang.org/x/crypto/ssh"
)

var csvFile = regexp.MustCompile(`(?i)\.csv$`)

// Start - starts poller
func Start(ctx context.Context) {
	go startPolling(ctx)
}

func startPolling(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(time.Second * time.Duration(config.SFTPPollIntervalSeconds)):
			logrus.Infof("Polling sftp %s", config.SFTPAddr)
			pollSFTP()
		}
	}
}

func getSSHConnection() (*ssh.Client, error) {
	addr := config.SFTPAddr
	config := &ssh.ClientConfig{
		User: config.SFTPUser,
		Auth: []ssh.AuthMethod{
			ssh.Password(config.SFTPPassword),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	return ssh.Dial("tcp", addr, config)
}

func pollSFTP() {
	conn, err := getSSHConnection()
	if err != nil {
		logrus.Errorf("Unable to connect to sftp: %s", err.Error())
		return
	}

	sftp, err := sftp.NewClient(conn)
	if err != nil {
		logrus.Errorf("Unable to create sftp client: %s", err.Error())
		return
	}
	defer conn.Close()

	walker := sftp.Walk("/")
	for walker.Step() {
		if err := walker.Err(); err != nil {
			logrus.Errorf("Walker error: %s", err.Error())
			continue
		}

		// Skipping dirs and non csv files
		fi := walker.Stat()
		if fi.IsDir() || !csvFile.MatchString(walker.Path()) {
			continue
		}

		logrus.Debugf("Found new file: %s", walker.Path())
		processFile(sftp, walker.Path())
	}
}

func processFile(sftp *sftp.Client, file string) {
	// Open the source file
	srcFile, err := sftp.Open(file)
	if err != nil {
		logrus.Errorf("Unable to open remote file (%s): %s", file, err.Error())
		return
	}

	// Create the destination file
	filename := filepath.Base(file)
	dstFilePath := filepath.Join(config.DownloadDir, filename)
	dstFile, err := os.Create(dstFilePath)
	if err != nil {
		logrus.Errorf("Unable to open local file (%s): %s", dstFilePath, err.Error())
		srcFile.Close()
		return
	}

	srcFile.WriteTo(dstFile)

	dstFile.Close()
	srcFile.Close()

	logrus.Debugf("Uploading file to s3: %s", dstFilePath)
	uploader.Upload(dstFilePath, getDeviceID(dstFilePath))

	logrus.Debugf("Removing file from sftp: %s", file)
	err = sftp.Remove(file)
	if err != nil {
		logrus.Errorf("Unable to delete file from sftp (%s): %s", file, err.Error())
	}
}

func getDeviceID(path string) string {
	file, err := os.Open(path)
	if err != nil {
		logrus.Errorf("Error opening file to get deviceID: %s", err.Error())
		return ""
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	// get just first line.
	scanner.Scan()
	parts := strings.Split(scanner.Text(), ";")
	if len(parts) < 2 || !strings.HasPrefix(strings.ToLower(parts[0]), "device id") {
		return ""
	}

	return strings.ToUpper(parts[1])
}
