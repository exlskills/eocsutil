package mdutils

import (
	"bytes"
	"encoding/json"
	"github.com/exlskills/eocsutil/config"
	"github.com/pkg/errors"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

var Log = config.Cfg().GetLogger()
var sdServerStartedAt = time.Now()
var serverCmd *exec.Cmd

const useREST = true

var sdShutdown = false

func init() {
	if useREST {
		// boot the server
		go runShowdownServerP()
	}
}

func waitForSDServer() error {
	if !useREST {
		return errors.New("server permanently unavailable (disabled)")
	}
	if sdShutdown {
		return errors.New("server has been shutdown")
	}
	// TODO probably time out if we add some more improved status monitoring for the process
	for {
		if sdServerStartedAt.Add(time.Second * 5).Before(time.Now()) {
			return nil
		}
		time.Sleep(1 * time.Second)
	}
}

type sdServerConversionPayload struct {
	Content string `json:"content"`
}

func callSDServerRESTAPI(conversionMethod, contents string) (respContents string, err error) {
	if err = waitForSDServer(); err != nil {
		return "", err
	}
	postBytes, err := json.Marshal(sdServerConversionPayload{Content: contents})
	if err != nil {
		return "", err
	}
	req, err := http.NewRequest("POST", "http://localhost:6222/"+conversionMethod, bytes.NewBuffer(postBytes))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	client.Timeout = time.Second * 15
	resp, err := client.Do(req)
	if err != nil {
		panic(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", errors.New("got non-200 status code from showdownjs REST API")
	}

	respData := sdServerConversionPayload{}
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&respData)
	if err != nil {
		return "", err
	}
	return respData.Content, nil
}

func GracefulTeardown() {
	if sdShutdown {
		return
	}
	sdShutdown = true
	if serverCmd != nil && serverCmd.Process != nil {
		Log.Info("Shutting down showdown server")
		if err := serverCmd.Process.Kill(); err != nil {
			Log.Error("An error occurred terminating the showdownjs markdown REST API: ", err.Error())
			Log.Error("Please kill the showdownjs node process manually using `kill`")
		}
	}
}

func MakeOLX(md string) (olx string, err error) {
	if useREST {
		return callSDServerRESTAPI("makeolx", md)
	}
	return execShowdown("makeolx", "olx", md)
}

func MakeHTML(md, flavor string) (html string, err error) {
	if useREST {
		return callSDServerRESTAPI("makehtml", md)
	}
	return execShowdown("makehtml", flavor, md)
}

func MakeMD(html, flavor string) (md string, err error) {
	if useREST {
		return callSDServerRESTAPI("makemarkdown", html)
	}
	return execShowdown("makemarkdown", flavor, html)
}

func UnescapeMD(md string) (escaped string, err error) {
	if useREST {
		return callSDServerRESTAPI("unescapemd", md)
	}
	return execShowdown("unescapemd", "github", md)
}

func runShowdownServerP() {
	args := []string{
		// We need this harmony flag for regex negative lookbehind support
		"--harmony",
		"showdownjs/server.js",
	}
	serverCmd = exec.Command("node", args...)
	serverCmd.Stdout = os.Stdout
	serverCmd.Stderr = os.Stderr
	err := serverCmd.Run()
	if !sdShutdown {
		Log.Error("")
		panic(err)
	}
}

func execShowdown(subCmd, flavor, input string) (output string, err error) {
	args := []string{
		// We need this harmony flag for regex negative lookbehind support
		"--harmony",
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
