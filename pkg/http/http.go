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
package http

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"runtime"
	"strings"
	"syscall"
	"time"

	"github.com/DopplerHQ/cli/pkg/utils"
	"github.com/DopplerHQ/cli/pkg/version"
)

type queryParam struct {
	Key   string
	Value string
}

type errorResponse struct {
	Messages []string
	Success  bool
}

// DNS resolver
var UseCustomDNSResolver = false
var DNSResolverAddress = "1.1.1.1:53"
var DNSResolverProto = "udp"
var DNSResolverTimeout = time.Duration(5) * time.Second

func generateURL(host string, uri string, params []queryParam) (*url.URL, error) {
	host = strings.TrimSuffix(host, "/")
	if !strings.HasPrefix(uri, "/") {
		uri = fmt.Sprintf("/%s", uri)
	}
	url, err := url.Parse(fmt.Sprintf("%s%s", host, uri))
	if err != nil {
		return nil, err
	}

	values := url.Query()
	for _, param := range params {
		values.Add(param.Key, param.Value)
	}
	url.RawQuery = values.Encode()

	return url, nil
}

// GetRequest perform HTTP GET
func GetRequest(url *url.URL, verifyTLS bool, headers map[string]string) (int, http.Header, []byte, error) {
	req, err := http.NewRequest("GET", url.String(), nil)
	if err != nil {
		return 0, nil, nil, err
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	statusCode, respHeaders, body, err := performRequest(req, verifyTLS)
	if err != nil {
		return statusCode, respHeaders, body, err
	}

	return statusCode, respHeaders, body, nil
}

// PostRequest perform HTTP POST
func PostRequest(url *url.URL, verifyTLS bool, headers map[string]string, body []byte) (int, http.Header, []byte, error) {
	req, err := http.NewRequest("POST", url.String(), bytes.NewReader(body))
	if err != nil {
		return 0, nil, nil, err
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	statusCode, respHeaders, body, err := performRequest(req, verifyTLS)
	if err != nil {
		return statusCode, respHeaders, body, err
	}

	return statusCode, respHeaders, body, nil
}

// PutRequest perform HTTP PUT
func PutRequest(url *url.URL, verifyTLS bool, headers map[string]string, body []byte) (int, http.Header, []byte, error) {
	req, err := http.NewRequest("PUT", url.String(), bytes.NewReader(body))
	if err != nil {
		return 0, nil, nil, err
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	statusCode, respHeaders, body, err := performRequest(req, verifyTLS)
	if err != nil {
		return statusCode, respHeaders, body, err
	}

	return statusCode, respHeaders, body, nil
}

// DeleteRequest perform HTTP DELETE
func DeleteRequest(url *url.URL, verifyTLS bool, headers map[string]string, body []byte) (int, http.Header, []byte, error) {
	req, err := http.NewRequest("DELETE", url.String(), bytes.NewReader(body))
	if err != nil {
		return 0, nil, nil, err
	}

	for key, value := range headers {
		req.Header.Set(key, value)
	}

	statusCode, respHeaders, body, err := performRequest(req, verifyTLS)
	if err != nil {
		return statusCode, respHeaders, body, err
	}

	return statusCode, respHeaders, body, nil
}

func request(req *http.Request, verifyTLS bool, allowTimeout bool) (*http.Response, error) {
	// set headers
	req.Header.Set("client-sdk", "go-cli")
	req.Header.Set("client-version", version.ProgramVersion)
	req.Header.Set("client-os", runtime.GOOS)
	req.Header.Set("client-arch", runtime.GOARCH)
	req.Header.Set("user-agent", "doppler-go-cli-"+version.ProgramVersion)
	if req.Header.Get("Accept") == "" {
		req.Header.Set("Accept", "application/json")
	}
	req.Header.Set("Content-Type", "application/json")

	// close the connection after reading the response, to help prevent socket exhaustion
	req.Close = true

	client := &http.Client{}
	// set http timeout
	if allowTimeout && UseTimeout {
		client.Timeout = TimeoutDuration
	}

	// set TLS config
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}
	// #nosec G402
	if !verifyTLS {
		tlsConfig.InsecureSkipVerify = true
	}

	// use custom DNS resolver
	dialer := &net.Dialer{}
	if UseCustomDNSResolver {
		utils.LogDebug(fmt.Sprintf("Using custom DNS resolver %s", DNSResolverAddress))

		dialer = &net.Dialer{
			Resolver: &net.Resolver{
				PreferGo: true,
				Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
					d := net.Dialer{
						Timeout: DNSResolverTimeout,
					}
					return d.DialContext(ctx, DNSResolverProto, DNSResolverAddress)
				},
			},
		}
	}
	dialContext := func(ctx context.Context, network, addr string) (net.Conn, error) {
		return dialer.DialContext(ctx, network, addr)
	}

	proxyUrl, err := http.ProxyFromEnvironment(req)
	if err != nil {
		utils.LogDebug("Unable to read proxy from environment")
		utils.LogDebugError(err)
		proxyUrl = nil
	}
	if proxyUrl != nil {
		utils.LogDebug(fmt.Sprintf("Using proxy %s", proxyUrl))
	}

	client.Transport = &http.Transport{
		// disable keep alives to prevent multiple CLI instances from exhausting the
		// OS's available network sockets. this adds a negligible performance penalty
		DisableKeepAlives: true,
		TLSClientConfig:   tlsConfig,
		DialContext:       dialContext,
		Proxy:             http.ProxyURL(proxyUrl),
	}

	utils.LogDebug(fmt.Sprintf("Performing HTTP %s to %s", req.Method, req.URL))

	startTime := time.Now()
	var response *http.Response
	response = nil

	err = utils.Retry(RequestAttempts, 500*time.Millisecond, func() error {
		// disable semgrep rule b/c we properly check that resp isn't nil before using it within the err block
		resp, err := client.Do(req) // nosemgrep: trailofbits.go.invalid-usage-of-modified-variable.invalid-usage-of-modified-variable
		if err != nil {
			if resp != nil {
				defer func() {
					if closeErr := resp.Body.Close(); closeErr != nil {
						utils.LogDebug(closeErr.Error())
					}
				}()
			}

			utils.LogDebug(err.Error())

			if isTimeout(err) || errors.Is(err, syscall.ECONNREFUSED) {
				// retry request
				return err
			}

			return utils.StopRetryError(err)
		}

		response = resp

		if requestID := resp.Header.Get("x-request-id"); requestID != "" {
			utils.LogDebug(fmt.Sprintf("Request ID %s", requestID))
		}

		if isSuccess(resp.StatusCode) {
			return nil
		}

		contentType := resp.Header.Get("content-type")
		if IsRetry(resp.StatusCode, contentType) {
			// start logging retries after 10 seconds so it doesn't feel like we've frozen
			// we subtract 1 millisecond so that we always win the race against a request that exhausts its full 10 second time out
			if time.Now().After(startTime.Add(10 * time.Second).Add(-1 * time.Millisecond)) {
				utils.Log(fmt.Sprintf("Request failed with HTTP %d, retrying", resp.StatusCode))
			}
			return errors.New("Request failed")
		}

		// we cannot recover from this error code; accept defeat
		return utils.StopRetryError(errors.New("Request failed"))
	})

	return response, err
}

func performSSERequest(req *http.Request, verifyTLS bool, handler func([]byte)) (int, http.Header, error) {
	response, requestErr := request(req, verifyTLS, false)
	if requestErr != nil {
		statusCode := 0
		if response != nil {
			statusCode = response.StatusCode
		}
		return statusCode, nil, requestErr
	}

	if response != nil {
		defer func() {
			if closeErr := response.Body.Close(); closeErr != nil {
				utils.LogDebug(closeErr.Error())
			}
		}()
	}

	headers := response.Header.Clone()

	for {
		s := 1024
		data := make([]byte, s)
		n, err := response.Body.Read(data)
		// this shouldn't occur, but log anyway to aid with debugging
		if n == s {
			utils.LogDebug(fmt.Sprintf("Response reached max buffer size of %d bytes", s))
		}
		// From Go docs for Reader.Read:
		// "Callers should always process the n > 0 bytes returned before considering the error err."
		if n > 0 {
			go handler(data[:n])
		}
		if err != nil {
			return response.StatusCode, headers, err
		}
	}
}

func performRequest(req *http.Request, verifyTLS bool) (int, http.Header, []byte, error) {
	response, requestErr := request(req, verifyTLS, true)
	if response != nil {
		defer func() {
			if closeErr := response.Body.Close(); closeErr != nil {
				utils.LogDebug(closeErr.Error())
			}
		}()
	}

	if requestErr != nil && response == nil {
		return 0, nil, nil, requestErr
	}

	headers := response.Header.Clone()

	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return response.StatusCode, nil, nil, err
	}

	// success
	if requestErr == nil {
		return response.StatusCode, headers, body, nil
	}

	// print the response body error messages
	if contentType := response.Header.Get("content-type"); strings.HasPrefix(contentType, "application/json") {
		var errResponse errorResponse
		err = json.Unmarshal(body, &errResponse)
		if err != nil {
			utils.LogDebug(fmt.Sprintf("Unable to parse response body: \n%s", string(body)))
			return response.StatusCode, headers, nil, err
		}

		return response.StatusCode, headers, body, errors.New(strings.Join(errResponse.Messages, "\n"))
	}

	return response.StatusCode, headers, nil, fmt.Errorf("Request failed with HTTP %d", response.StatusCode)
}

func isSuccess(statusCode int) bool {
	return (statusCode >= 200 && statusCode <= 299) || (statusCode >= 300 && statusCode <= 399)
}

func IsRetry(statusCode int, contentType string) bool {
	return (statusCode == 429) ||
		(statusCode >= 100 && statusCode <= 199) ||
		// don't retry 5xx errors w/ a JSON body
		(statusCode >= 500 && statusCode <= 599 && !strings.HasPrefix(contentType, "application/json"))
}

func isTimeout(err error) bool {
	if urlErr, ok := err.(*url.Error); ok {
		if netErr, ok := urlErr.Err.(net.Error); ok {
			return netErr.Timeout()
		}
	}

	return false
}
