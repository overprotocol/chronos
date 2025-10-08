package checkpointsync

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/prysmaticlabs/prysm/v5/io/file"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var uploadFlags = struct {
	StateFile string
	BlockFile string
	ServerURL string
	Timeout   time.Duration
}{}

var uploadCmd = &cli.Command{
	Name:    "upload",
	Aliases: []string{"up"},
	Usage:   "Upload checkpoint files to a checkpoint server",
	Action: func(cliCtx *cli.Context) error {
		if err := cliActionUpload(cliCtx); err != nil {
			log.WithError(err).Fatal("Could not upload checkpoint data")
		}
		return nil
	},
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:        "state-file",
			Usage:       "Path to the SSZ-encoded state file",
			Destination: &uploadFlags.StateFile,
			Required:    true,
		},
		&cli.StringFlag{
			Name:        "block-file",
			Usage:       "Path to the SSZ-encoded block file",
			Destination: &uploadFlags.BlockFile,
			Required:    true,
		},
		&cli.StringFlag{
			Name:        "server-url",
			Usage:       "Checkpoint server URL (e.g., http://checkpoint-server:8080/upload)",
			Destination: &uploadFlags.ServerURL,
			Required:    true,
		},
		&cli.DurationFlag{
			Name:        "timeout",
			Usage:       "Upload timeout duration",
			Destination: &uploadFlags.Timeout,
			Value:       10 * time.Minute,
		},
	},
}

func cliActionUpload(_ *cli.Context) error {
	f := uploadFlags

	log.WithFields(log.Fields{
		"stateFile": f.StateFile,
		"blockFile": f.BlockFile,
		"serverURL": f.ServerURL,
	}).Info("Starting checkpoint upload")

	// Read state file
	stateBytes, err := file.ReadFileAsBytes(f.StateFile)
	if err != nil {
		return fmt.Errorf("failed to read state file: %w", err)
	}

	// Read block file
	blockBytes, err := file.ReadFileAsBytes(f.BlockFile)
	if err != nil {
		return fmt.Errorf("failed to read block file: %w", err)
	}

	// Create multipart form
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Add state file
	stateFileName := filepath.Base(f.StateFile)
	statePart, err := writer.CreateFormFile("state", stateFileName)
	if err != nil {
		return fmt.Errorf("failed to create state form field: %w", err)
	}
	if _, err := statePart.Write(stateBytes); err != nil {
		return fmt.Errorf("failed to write state data: %w", err)
	}

	// Add block file
	blockFileName := filepath.Base(f.BlockFile)
	blockPart, err := writer.CreateFormFile("block", blockFileName)
	if err != nil {
		return fmt.Errorf("failed to create block form field: %w", err)
	}
	if _, err := blockPart.Write(blockBytes); err != nil {
		return fmt.Errorf("failed to write block data: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close multipart writer: %w", err)
	}

	// Create HTTP request
	client := &http.Client{Timeout: f.Timeout}
	req, err := http.NewRequest("POST", f.ServerURL, body)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}
	req.Header.Set("Content-Type", writer.FormDataContentType())

	log.WithFields(log.Fields{
		"stateSize": len(stateBytes),
		"blockSize": len(blockBytes),
		"url":       f.ServerURL,
	}).Info("Uploading checkpoint files")

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to upload files: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.WithError(err).Warn("Failed to close response body")
		}
	}()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	log.WithFields(log.Fields{
		"status":   resp.StatusCode,
		"response": string(respBody),
	}).Info("Checkpoint files uploaded successfully")

	return nil
}

// SimpleCopyToServer is a simpler alternative using SCP or direct file copy
func SimpleCopyToServer(stateFile, blockFile, destDir string) error {
	log.WithFields(log.Fields{
		"stateFile": stateFile,
		"blockFile": blockFile,
		"destDir":   destDir,
	}).Info("Copying checkpoint files to destination")

	// Ensure destination directory exists
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Copy state file
	stateDestPath := filepath.Join(destDir, filepath.Base(stateFile))
	if err := copyFile(stateFile, stateDestPath); err != nil {
		return fmt.Errorf("failed to copy state file: %w", err)
	}
	log.WithField("dest", stateDestPath).Info("Copied state file")

	// Copy block file
	blockDestPath := filepath.Join(destDir, filepath.Base(blockFile))
	if err := copyFile(blockFile, blockDestPath); err != nil {
		return fmt.Errorf("failed to copy block file: %w", err)
	}
	log.WithField("dest", blockDestPath).Info("Copied block file")

	return nil
}

func copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		if err := sourceFile.Close(); err != nil {
			log.WithError(err).Warn("Failed to close source file")
		}
	}()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		if err := destFile.Close(); err != nil {
			log.WithError(err).Warn("Failed to close destination file")
		}
	}()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return err
	}

	return destFile.Sync()
}
