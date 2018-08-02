package mdutils

import (
	"os/exec"
	"strings"
)

func MakeOLX(md string) (olx string, err error) {
	return execShowdown("makeolx", "olx", md)
}

func MakeHTML(md, flavor string) (html string, err error) {
	return execShowdown("makehtml", flavor, md)
}

func MakeMD(html, flavor string) (md string, err error) {
	return execShowdown("makemarkdown", flavor, html)
}

func execShowdown(subCmd, flavor, input string) (output string, err error) {
	args := []string{
		"showdownjs",
		subCmd,
		"-m",
		"-p",
		flavor,
	}
	cmd := exec.Command("node", args...)
	cmd.Stdin = strings.NewReader(input)
	outBytes, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(outBytes), nil
}
