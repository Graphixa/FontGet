package update

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/blang/semver"
)

const (
	defaultReleaseBaseURL = "https://github.com/Graphixa/FontGet/releases"

	releaseRequestTimeout  = 30 * time.Second
	archiveDownloadTimeout = 5 * time.Minute
	maxChecksumsBytes      = 1 << 20   // 1 MiB
	maxReleaseAssetBytes   = 200 << 20 // 200 MiB
	updateUserAgent        = "FontGet (https://github.com/Graphixa/FontGet)"
)

var (
	// releaseBaseURL is overridden by tests with a loopback server.
	releaseBaseURL = defaultReleaseBaseURL

	errReleaseNotFound = errors.New("release not found")
)

type releaseClient struct {
	baseURL *url.URL
	client  *http.Client
}

func newReleaseClient() (*releaseClient, error) {
	base, err := url.Parse(strings.TrimRight(releaseBaseURL, "/"))
	if err != nil {
		return nil, fmt.Errorf("invalid release base URL: %w", err)
	}
	if err := validateReleaseBaseURL(base); err != nil {
		return nil, err
	}

	c := &releaseClient{baseURL: base}
	c.client = &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return fmt.Errorf("stopped after 10 redirects")
			}
			return validateDownloadURL(req.URL, c.baseURL)
		},
	}
	return c, nil
}

// latestVersion resolves GitHub's stable-release permalink without following
// the redirect. The redirect target is treated only as a version identifier;
// all asset URLs are constructed locally from the validated tag.
func (c *releaseClient) latestVersion(ctx context.Context) (semver.Version, error) {
	latest := *c.baseURL
	latest.Path = strings.TrimRight(c.baseURL.Path, "/") + "/latest"
	latest.RawQuery = ""
	latest.Fragment = ""

	ctx, cancel := context.WithTimeout(ctx, releaseRequestTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, latest.String(), nil)
	if err != nil {
		return semver.Version{}, fmt.Errorf("failed to create latest-release request: %w", err)
	}
	req.Header.Set("User-Agent", updateUserAgent)

	noRedirect := *c.client
	noRedirect.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}

	resp, err := noRedirect.Do(req)
	if err != nil {
		return semver.Version{}, fmt.Errorf("failed to resolve latest release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return semver.Version{}, errReleaseNotFound
	}
	if !isRedirectStatus(resp.StatusCode) {
		return semver.Version{}, fmt.Errorf("latest release returned HTTP %d instead of a redirect", resp.StatusCode)
	}

	location := resp.Header.Get("Location")
	if location == "" {
		return semver.Version{}, fmt.Errorf("latest release redirect did not include Location")
	}
	return parseLatestRedirect(c.baseURL, location)
}

func (c *releaseClient) checksums(ctx context.Context, version semver.Version) ([]byte, error) {
	return c.downloadAsset(ctx, version, "checksums.txt", maxChecksumsBytes, releaseRequestTimeout)
}

func (c *releaseClient) archive(ctx context.Context, version semver.Version, name string) ([]byte, error) {
	return c.downloadAsset(ctx, version, name, maxReleaseAssetBytes, archiveDownloadTimeout)
}

func (c *releaseClient) downloadAsset(ctx context.Context, version semver.Version, name string, limit int64, timeout time.Duration) ([]byte, error) {
	assetURL, err := c.assetURL(version, name)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, assetURL.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create asset request: %w", err)
	}
	req.Header.Set("User-Agent", updateUserAgent)
	req.Header.Set("Accept", "application/octet-stream")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download %s: %w", name, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("%w: %s", errReleaseNotFound, name)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to download %s (HTTP %d)", name, resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, fmt.Errorf("failed to read %s: %w", name, err)
	}
	if int64(len(body)) > limit {
		return nil, fmt.Errorf("%s exceeded %d bytes", name, limit)
	}
	return body, nil
}

func (c *releaseClient) assetURL(version semver.Version, name string) (*url.URL, error) {
	if name == "" || name != url.PathEscape(name) || strings.ContainsAny(name, `/\`) {
		return nil, fmt.Errorf("invalid release asset name %q", name)
	}

	u := *c.baseURL
	u.Path = strings.TrimRight(c.baseURL.Path, "/") + "/download/v" + version.String() + "/" + name
	u.RawQuery = ""
	u.Fragment = ""
	if err := validateDownloadURL(&u, c.baseURL); err != nil {
		return nil, err
	}
	return &u, nil
}

func parseLatestRedirect(base *url.URL, location string) (semver.Version, error) {
	target, err := base.Parse(location)
	if err != nil {
		return semver.Version{}, fmt.Errorf("invalid latest release redirect: %w", err)
	}
	if target.User != nil || target.RawQuery != "" || target.Fragment != "" {
		return semver.Version{}, fmt.Errorf("latest release redirect contains unexpected URL components")
	}
	if !strings.EqualFold(target.Scheme, base.Scheme) || !strings.EqualFold(target.Host, base.Host) {
		return semver.Version{}, fmt.Errorf("latest release redirected to unexpected origin %q", target.Host)
	}

	tagPrefix := strings.TrimRight(base.Path, "/") + "/tag/v"
	if !strings.HasPrefix(target.EscapedPath(), tagPrefix) {
		return semver.Version{}, fmt.Errorf("latest release redirected to unexpected path %q", target.EscapedPath())
	}
	rawVersion := strings.TrimPrefix(target.EscapedPath(), tagPrefix)
	if rawVersion == "" || strings.Contains(rawVersion, "/") {
		return semver.Version{}, fmt.Errorf("latest release redirect contains invalid version %q", rawVersion)
	}
	decodedVersion, err := url.PathUnescape(rawVersion)
	if err != nil {
		return semver.Version{}, fmt.Errorf("latest release redirect contains invalid version encoding: %w", err)
	}
	version, err := semver.Parse(decodedVersion)
	if err != nil {
		return semver.Version{}, fmt.Errorf("latest release redirect contains invalid semantic version %q: %w", decodedVersion, err)
	}
	return version, nil
}

func validateReleaseBaseURL(base *url.URL) error {
	if base == nil || base.Hostname() == "" {
		return fmt.Errorf("release base URL has no host")
	}
	if base.User != nil || base.RawQuery != "" || base.Fragment != "" {
		return fmt.Errorf("release base URL contains unexpected URL components")
	}
	if base.Scheme != "https" && !(base.Scheme == "http" && isLoopbackHost(base.Hostname())) {
		return fmt.Errorf("release base URL must use HTTPS")
	}
	if base.Path == "" || strings.HasSuffix(base.Path, "/") {
		return fmt.Errorf("release base URL has invalid path")
	}
	if !isLoopbackHost(base.Hostname()) && base.String() != defaultReleaseBaseURL {
		return fmt.Errorf("release base URL must target Graphixa/FontGet on GitHub")
	}
	return nil
}

func validateDownloadURL(target, base *url.URL) error {
	if target == nil || target.Hostname() == "" {
		return fmt.Errorf("refusing download URL with empty host")
	}
	if target.User != nil {
		return fmt.Errorf("refusing download URL with user information")
	}
	host := strings.ToLower(target.Hostname())
	sameBaseOrigin := strings.EqualFold(target.Scheme, base.Scheme) &&
		strings.EqualFold(target.Host, base.Host)
	if sameBaseOrigin {
		return nil
	}
	if target.Scheme != "https" {
		return fmt.Errorf("refusing non-HTTPS download URL")
	}
	if isGitHubDownloadHost(host) {
		return nil
	}
	return fmt.Errorf("refusing download from untrusted host %q", host)
}

func isGitHubDownloadHost(host string) bool {
	host = strings.ToLower(host)
	return host == "github.com" ||
		host == "www.github.com" ||
		host == "githubusercontent.com" ||
		strings.HasSuffix(host, ".githubusercontent.com")
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func isRedirectStatus(status int) bool {
	switch status {
	case http.StatusMovedPermanently,
		http.StatusFound,
		http.StatusSeeOther,
		http.StatusTemporaryRedirect,
		http.StatusPermanentRedirect:
		return true
	default:
		return false
	}
}
