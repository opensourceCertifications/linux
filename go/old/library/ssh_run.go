package library

import (
	"bytes"
	"fmt"
)

// RunCommandOverWarmSSH runs a single command over the existing warm SSH
// connection. It returns the combined stdout+stderr output.
// If no warm connection is available, it returns an error.
func RunCommandOverWarmSSH(cmd string) (string, error) {
	client := GetWarmClient()
	if client == nil {
		return "", fmt.Errorf("no warm SSH client available")
	}

	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("failed to create SSH session: %w", err)
	}
	defer session.Close()

	var outBuf, errBuf bytes.Buffer
	session.Stdout = &outBuf
	session.Stderr = &errBuf

	if err := session.Run(cmd); err != nil {
		// Include stderr in the error to make debugging easier
		return outBuf.String() + errBuf.String(), fmt.Errorf("failed to run command %q: %w", cmd, err)
	}

	return outBuf.String() + errBuf.String(), nil
}
