package poller

import (
	"bufio"
	"context"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/da4nik/sftppoller/config"
	"github.com/da4nik/sftppoller/merger"
	"github.com/da4nik/sftppoller/uploader"
	"github.com/pkg/sftp"

	"golang.org/x/crypto/ssh"
)

var csvFile = regexp.MustCompile(`(?i)\.csv$`)

var updatedDeviceIDs = make(map[string]bool)
var updatedDeviceIDsMutex sync.Mutex

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

	var wg sync.WaitGroup
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
		wg.Add(1)
		go processFile(sftp, walker.Path(), &wg)
	}
	wg.Wait()

	if len(updatedDeviceIDs) > 0 {
		updatedDeviceIDsMutex.Lock()
		for k := range updatedDeviceIDs {
			merger.Enqueue(k)
			delete(updatedDeviceIDs, k)
		}
		updatedDeviceIDsMutex.Unlock()
	}
}

func processFile(sftp *sftp.Client, file string, wg *sync.WaitGroup) {
	defer wg.Done()
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

	deviceID := strings.ToUpper(getDeviceID(dstFilePath))
	key := filekey(dstFilePath, deviceID)
	uploader.Upload(dstFilePath, key)

	logrus.Debugf("Removing file from sftp: %s", file)
	err = sftp.Remove(file)
	if err != nil {
		logrus.Errorf("Unable to delete file from sftp (%s): %s", file, err.Error())
	}

	updatedDeviceIDsMutex.Lock()
	updatedDeviceIDs[deviceID] = true
	updatedDeviceIDsMutex.Unlock()
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

func filekey(path, key string) string {
	t := time.Now()
	currentTimestamp := t.Format("20060102150405")
	return filepath.Join(key, currentTimestamp+"-"+filepath.Base(path))
}
