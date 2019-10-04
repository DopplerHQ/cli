package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
)

// TODO add fallback support
func main() {
	config := map[string]string{
		"api_key":  os.Getenv("key"),
		"pipeline": os.Getenv("pipeline"),
		"env":      os.Getenv("environment"),
		"api":      "https://deploy.doppler.com/v1/variables",
	}

	argLocation := indexOf(os.Args, "--")
	if argLocation == -1 || argLocation == (len(os.Args)-1) {
		fmt.Println("Command to execute not specified")
		os.Exit(1)
	}

	// make variables request to doppler api
	body := getVariables(config["api"], config["api_key"], config["pipeline"], config["env"])

	var result map[string]interface{}
	json.Unmarshal([]byte(body), &result)
	variables := result["variables"].(map[string]interface{})

	command := os.Args[argLocation+1 : len(os.Args)]
	runCommand(command, variables)
}

func runCommand(command []string, variables map[string]interface{}) {
	cmd := exec.Command(command[0], command[1:]...)
	env := os.Environ()

	excludedKeys := []string{"PATH", "PS1", "HOME"}
	for key, value := range variables {
		if indexOf(excludedKeys, key) == -1 {
			env = append(env, fmt.Sprintf("%s=%s", key, value))
		}
	}

	cmd.Env = env

	cmdOut, _ := cmd.StdoutPipe()
	cmdErr, _ := cmd.StderrPipe()

	startErr := cmd.Start()
	if startErr != nil {
		fmt.Println(fmt.Sprintf("Error trying to execute command: %s", command))
		fmt.Println(startErr)
	}

	stdOutput, _ := ioutil.ReadAll(cmdOut)
	errOutput, _ := ioutil.ReadAll(cmdErr)

	fmt.Printf(string(stdOutput))
	fmt.Printf(string(errOutput))

	err := cmd.Wait()
	if err == nil {
		os.Exit(0)
	}

	os.Exit(1)
}

func indexOf(arr []string, str string) int {
	for i, a := range arr {
		if a == str {
			return i
		}
	}
	return -1
}

func getVariables(api string, apiKey string, pipeline string, environment string) string {
	req, err := http.NewRequest("GET", api, nil)
	if err != nil {
		fmt.Println(fmt.Sprintf("Error fetching variables: %s", err))
		os.Exit(1)
	}

	req.Header.Set("api-key", apiKey)
	req.Header.Set("client-sdk", "go-cli")
	req.Header.Set("client-version", "0.1")
	req.Header.Set("user-agent", "doppler go-cli 0.1")
	req.Header.Set("Accept", "application/json")

	params := req.URL.Query()
	params.Add("environment", environment)
	params.Add("pipeline", pipeline)
	req.URL.RawQuery = params.Encode()

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(fmt.Sprintf("Error fetching variables: %s", err))
		os.Exit(1)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(fmt.Sprintf("Error fetching variables: %s", err.Error()))
		os.Exit(1)
	}
	if resp.StatusCode != 200 {
		// TODO parse out and just log body.messages
		fmt.Println(fmt.Sprintf("Error fetching variables: %s", string(body)))
		os.Exit(1)
	}
	return string(body)
}
