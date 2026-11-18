/*
Copyright 2025. The Jumpstarter Authors.

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

// Package flightctl delivers exporter files to hosts managed by Flight Control.
package flightctl

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"sigs.k8s.io/yaml"
)

// clientConfig is the part of the flightctl CLI client config ("flightctl login") that we use.
type clientConfig struct {
	Service struct {
		Server                   string `json:"server"`
		TLSServerName            string `json:"tls-server-name"`
		CertificateAuthority     string `json:"certificate-authority"`
		CertificateAuthorityData []byte `json:"certificate-authority-data"`
		InsecureSkipVerify       bool   `json:"insecureSkipVerify"`
	} `json:"service"`
	Authentication struct {
		AccessToken string `json:"access-token"`
		IDToken     string `json:"id-token"`
		TokenToUse  string `json:"token-to-use"`
	} `json:"authentication"`
	Organization string `json:"organization"`
}

// Client talks to the Flight Control API.
type Client struct {
	server       *url.URL
	httpClient   *http.Client
	token        string
	organization string
}

// DefaultConfigPath is where "flightctl login" stores the client config.
func DefaultConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("locating the flightctl client config: %w", err)
	}
	return filepath.Join(configDir, "flightctl", "client.yaml"), nil
}

// NewClientFromConfigFile builds a client from a flightctl CLI client config file.
// It uses the token stored there as is; it does not refresh it.
func NewClientFromConfigFile(path string) (*Client, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading flightctl client config (run \"flightctl login\" first): %w", err)
	}
	var cfg clientConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing flightctl client config %s: %w", path, err)
	}

	server, err := url.Parse(cfg.Service.Server)
	if err != nil || server.Host == "" {
		return nil, fmt.Errorf("flightctl client config %s has no valid service.server", path)
	}

	token := cfg.Authentication.AccessToken
	if cfg.Authentication.TokenToUse == "id" {
		token = cfg.Authentication.IDToken
	}
	if token == "" {
		return nil, fmt.Errorf("flightctl client config %s has no token (run \"flightctl login\" first)", path)
	}

	tlsConfig := &tls.Config{
		MinVersion:         tls.VersionTLS12,
		ServerName:         cfg.Service.TLSServerName,
		InsecureSkipVerify: cfg.Service.InsecureSkipVerify, //nolint:gosec // the operator's explicit choice in the flightctl config
	}
	caData := cfg.Service.CertificateAuthorityData
	if len(caData) == 0 && cfg.Service.CertificateAuthority != "" {
		caPath := cfg.Service.CertificateAuthority
		if !filepath.IsAbs(caPath) {
			caPath = filepath.Join(filepath.Dir(path), caPath)
		}
		if caData, err = os.ReadFile(caPath); err != nil {
			return nil, fmt.Errorf("reading flightctl certificate authority: %w", err)
		}
	}
	if len(caData) > 0 {
		pool := x509.NewCertPool()
		if !pool.AppendCertsFromPEM(caData) {
			return nil, fmt.Errorf("flightctl client config %s has no usable certificate authority", path)
		}
		tlsConfig.RootCAs = pool
	}

	return &Client{
		server: server,
		httpClient: &http.Client{
			Timeout:   30 * time.Second,
			Transport: &http.Transport{TLSClientConfig: tlsConfig, Proxy: http.ProxyFromEnvironment},
		},
		token:        token,
		organization: cfg.Organization,
	}, nil
}

// Device is the part of a Flight Control device that we read.
type Device struct {
	Metadata DeviceMetadata `json:"metadata"`
	// Spec is absent on a device that was never configured.
	Spec *DeviceSpec `json:"spec"`
}

// DeviceMetadata is the part of a device's metadata that we read.
type DeviceMetadata struct {
	Name string `json:"name"`
	// Owner is set when a fleet owns the device, for example "Fleet/lab".
	Owner string `json:"owner"`
	// ResourceVersion changes with every change to the device.
	ResourceVersion string `json:"resourceVersion"`
}

// DeviceSpec is the part of a device's spec that we read.
type DeviceSpec struct {
	// Config holds the config providers. Only ours is decoded; the rest is passed through untouched.
	Config []json.RawMessage `json:"config"`
}

// Config returns the config providers of the device, none when it has no spec.
func (d *Device) Config() []json.RawMessage {
	if d.Spec == nil {
		return nil
	}
	return d.Spec.Config
}

// FindDevice returns the one device carrying all of the given labels.
func (c *Client) FindDevice(ctx context.Context, selector map[string]string) (*Device, error) {
	pairs := make([]string, 0, len(selector))
	for key, value := range selector {
		pairs = append(pairs, key+"="+value)
	}
	sort.Strings(pairs)
	labelSelector := strings.Join(pairs, ",")

	var list struct {
		Items []Device `json:"items"`
	}
	if err := c.get(ctx, "/api/v1/devices", url.Values{"labelSelector": {labelSelector}}, &list); err != nil {
		return nil, err
	}
	if len(list.Items) != 1 {
		return nil, fmt.Errorf("%d flightctl devices match %q, expected exactly one", len(list.Items), labelSelector)
	}
	return &list.Items[0], nil
}

// patchOp is one operation of a JSON patch.
type patchOp struct {
	Op    string `json:"op"`
	Path  string `json:"path"`
	Value any    `json:"value"`
}

// SetDeviceConfig replaces the config providers of a device and leaves the rest of its spec alone.
// It fails when the device changed since it was read, instead of overwriting that change.
func (c *Client) SetDeviceConfig(ctx context.Context, device *Device, config []json.RawMessage) error {
	unchanged := patchOp{Op: "test", Path: "/metadata/resourceVersion", Value: device.Metadata.ResourceVersion}
	// A JSON patch "add" on an object member creates or replaces it, but it needs the parent to exist.
	set := patchOp{Op: "add", Path: "/spec/config", Value: config}
	if device.Spec == nil {
		set = patchOp{Op: "add", Path: "/spec", Value: map[string]any{"config": config}}
	}
	return c.patch(ctx, "/api/v1/devices/"+url.PathEscape(device.Metadata.Name), []patchOp{unchanged, set})
}

// get decodes the response to a GET of path into result.
func (c *Client) get(ctx context.Context, path string, query url.Values, result any) error {
	req, err := c.newRequest(ctx, http.MethodGet, path, query, nil)
	if err != nil {
		return err
	}
	resp, err := c.send(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
		return fmt.Errorf("decoding flightctl response for GET %s: %w", path, err)
	}
	return nil
}

// patch applies a JSON patch to the resource at path.
func (c *Client) patch(ctx context.Context, path string, ops []patchOp) error {
	body, err := json.Marshal(ops)
	if err != nil {
		return fmt.Errorf("encoding flightctl patch: %w", err)
	}
	req, err := c.newRequest(ctx, http.MethodPatch, path, url.Values{}, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json-patch+json")
	resp, err := c.send(req)
	if err != nil {
		return err
	}
	return resp.Body.Close()
}

// newRequest builds an authenticated request for path in the client's organization.
func (c *Client) newRequest(
	ctx context.Context, method, path string, query url.Values, body io.Reader,
) (*http.Request, error) {
	if c.organization != "" {
		query.Set("org_id", c.organization)
	}
	target := c.server.JoinPath(path)
	target.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, method, target.String(), body)
	if err != nil {
		return nil, fmt.Errorf("building flightctl request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	return req, nil
}

// send sends req and turns a response outside 2xx into an error.
func (c *Client) send(req *http.Request) (*http.Response, error) {
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("flightctl %s %s: %w", req.Method, req.URL.Path, err)
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		message, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		_ = resp.Body.Close()
		return nil, fmt.Errorf("flightctl %s %s: %s: %s",
			req.Method, req.URL.Path, resp.Status, strings.TrimSpace(string(message)))
	}
	return resp, nil
}
