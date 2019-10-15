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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
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

// GetRequest perform HTTP GET
func GetRequest(host string, uri string, params []QueryParam, apiKey string) ([]byte, error) {
	url := fmt.Sprintf("%s/%s", host, uri)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("api-key", apiKey)
	req.Header.Set("client-sdk", "go-cli")
	req.Header.Set("client-version", "0.1")
	req.Header.Set("user-agent", "doppler go-cli 0.1")
	req.Header.Set("Accept", "application/json")

	query := req.URL.Query()
	for _, param := range params {
		query.Add(param.Key, param.Value)
	}
	req.URL.RawQuery = query.Encode()

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		var response errorResponse
		err = json.Unmarshal(body, &response)
		if err != nil {
			fmt.Println(err)
			return nil, nil
		}

		for _, message := range response.Messages {
			fmt.Println(message)
		}

		return nil, nil
	}

	return body, nil
}
