package mdutils

import (
	"os/exec"
)

func RunMarkdownPDFConverter(inputPath, outputPath string) (err error) {
	args := []string{
		"run",
		"markdown-pdf",
		"--",
		"-o",
		outputPath,
		inputPath,
	}
	cmd := exec.Command("npm", args...)
	outBytes, err := cmd.Output()
	if err != nil {
		Log.Errorf("markdown-pdf exec error: %s", string(outBytes))
		return err
	}
	return nil
}
