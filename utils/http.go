/*
Copyright © 2019 Doppler <support@doppler.com>

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package utils

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

// QueryParam a url query parameter. ex: ?foo=bar
type QueryParam struct {
	Key   string
	Value string
}

type errorResponse struct {
	Messages []string
	Success  bool
}

// Insecure whether we should support https connections without a valid cert
var Insecure = false

// GetRequest perform HTTP GET
func GetRequest(host string, uri string, params []QueryParam, apiKey string) ([]byte, error) {
	url := fmt.Sprintf("%s%s", host, uri)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("api-key", apiKey)

	body, err := performRequest(req, params)
	if err != nil {
		return nil, err
	}

	return body, nil
}

// PostRequest perform HTTP POST
func PostRequest(host string, uri string, params []QueryParam, apiKey string, body []byte) ([]byte, error) {
	url := fmt.Sprintf("%s%s", host, uri)
	req, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("api-key", apiKey)

	body, err = performRequest(req, params)
	if err != nil {
		return nil, err
	}

	return body, nil
}

// DeleteRequest perform HTTP DELETE
func DeleteRequest(host string, uri string, params []QueryParam, apiKey string) ([]byte, error) {
	url := fmt.Sprintf("%s%s", host, uri)
	req, err := http.NewRequest("DELETE", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("api-key", apiKey)

	body, err := performRequest(req, params)
	if err != nil {
		return nil, err
	}

	return body, nil
}

func performRequest(req *http.Request, params []QueryParam) ([]byte, error) {
	req.Header.Set("client-sdk", "go-cli")
	req.Header.Set("client-version", ProgramVersion)
	req.Header.Set("user-agent", "doppler-go-cli-"+ProgramVersion)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")

	query := req.URL.Query()
	for _, param := range params {
		query.Add(param.Key, param.Value)
	}
	req.URL.RawQuery = query.Encode()

	client := &http.Client{Timeout: 10 * time.Second}
	if Insecure {
		tr := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		client.Transport = tr
	}

	var res *http.Response
	res = nil
	err := retry(5, 100*time.Millisecond, func() error {
		resp, err := client.Do(req)
		if err != nil {
			if Debug {
				fmt.Println(err)
			}
			return StopRetry{errors.New("Unable to reach host " + req.Host + ". Please ensure you are connected to the internet.")}
		}

		// TODO make status code handling more granular
		if resp.StatusCode != 200 {
			if Debug && !JSON {
				requestID := resp.Header.Get("x-request-id")
				fmt.Println("Request ID:", requestID)
			}

			var response errorResponse
			var body []byte
			err = json.Unmarshal(body, &response)
			if err != nil {
				return err
			}

			var sb strings.Builder
			for i, message := range response.Messages {
				if i != 0 {
					sb.WriteString("\n")
				}
				sb.WriteString(message)
			}

			return errors.New(sb.String())
		}

		res = resp
		return nil
	})

	if res != nil {
		defer res.Body.Close()
	}

	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		if Debug && !JSON {
			requestID := res.Header.Get("x-request-id")
			fmt.Println("Request ID:", requestID)
		}
		return nil, err
	}

	return body, nil
}
