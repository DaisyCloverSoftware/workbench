package core

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	officialUpdateOwner      = "DaisyCloverSoftware"
	officialUpdateRepo       = "workbench"
	officialLatestReleaseURL = "https://api.github.com/repos/DaisyCloverSoftware/workbench/releases/latest"
	maxReleaseMetadataBytes  = 2 << 20
	maxChecksumAssetBytes    = 4096
)

var stableVersionPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)

type UpdateAsset struct {
	Name        string `json:"name"`
	Size        int64  `json:"size"`
	DownloadURL string `json:"download_url"`
	Digest      string `json:"digest,omitempty"`
}

type UpdateRelease struct {
	Version string                 `json:"version"`
	Tag     string                 `json:"tag"`
	PageURL string                 `json:"page_url,omitempty"`
	Assets  map[string]UpdateAsset `json:"assets"`
}

type UpdateCheck struct {
	CurrentVersion  string        `json:"current_version"`
	LatestVersion   string        `json:"latest_version"`
	UpdateAvailable bool          `json:"update_available"`
	Release         UpdateRelease `json:"release"`
}

type VerifiedUpdateAsset struct {
	Name   string `json:"name"`
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

type githubLatestRelease struct {
	TagName    string `json:"tag_name"`
	HTMLURL    string `json:"html_url"`
	Draft      bool   `json:"draft"`
	Prerelease bool   `json:"prerelease"`
	Assets     []struct {
		Name               string `json:"name"`
		Size               int64  `json:"size"`
		BrowserDownloadURL string `json:"browser_download_url"`
		Digest             string `json:"digest"`
	} `json:"assets"`
}

func CheckOfficialUpdate(ctx context.Context, currentVersion string) (UpdateCheck, error) {
	release, err := fetchUpdateRelease(ctx, officialHTTPClient(), officialLatestReleaseURL, true)
	if err != nil {
		return UpdateCheck{}, err
	}
	cmp, err := compareStableVersions(release.Version, currentVersion)
	if err != nil {
		return UpdateCheck{}, err
	}
	return UpdateCheck{
		CurrentVersion:  currentVersion,
		LatestVersion:   release.Version,
		UpdateAvailable: cmp > 0,
		Release:         release,
	}, nil
}

func FetchOfficialLatestRelease(ctx context.Context) (UpdateRelease, error) {
	return fetchUpdateRelease(ctx, officialHTTPClient(), officialLatestReleaseURL, true)
}

func officialHTTPClient() *http.Client {
	return &http.Client{
		Timeout: 45 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 6 {
				return errors.New("too many update download redirects")
			}
			if !trustedGitHubDownloadHost(req.URL) {
				return fmt.Errorf("update download redirected to untrusted host %q", req.URL.Hostname())
			}
			return nil
		},
	}
}

func fetchUpdateRelease(ctx context.Context, client *http.Client, endpoint string, enforceOfficialEndpoint bool) (UpdateRelease, error) {
	if client == nil {
		return UpdateRelease{}, errors.New("update HTTP client is nil")
	}
	if enforceOfficialEndpoint && endpoint != officialLatestReleaseURL {
		return UpdateRelease{}, errors.New("Workbench update discovery endpoint is not the official release API")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return UpdateRelease{}, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "DaisyCloverSoftware-Workbench-Updater/"+Version)
	resp, err := client.Do(req)
	if err != nil {
		return UpdateRelease{}, fmt.Errorf("check Workbench release: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return UpdateRelease{}, fmt.Errorf("check Workbench release: HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxReleaseMetadataBytes+1))
	if err != nil {
		return UpdateRelease{}, err
	}
	if len(body) > maxReleaseMetadataBytes {
		return UpdateRelease{}, errors.New("Workbench release metadata is unexpectedly large")
	}
	var raw githubLatestRelease
	if err := json.Unmarshal(body, &raw); err != nil {
		return UpdateRelease{}, fmt.Errorf("decode Workbench release metadata: %w", err)
	}
	return normalizeUpdateRelease(raw)
}

func normalizeUpdateRelease(raw githubLatestRelease) (UpdateRelease, error) {
	if raw.Draft || raw.Prerelease {
		return UpdateRelease{}, errors.New("latest Workbench release is not a stable published release")
	}
	if !strings.HasPrefix(raw.TagName, "v") {
		return UpdateRelease{}, fmt.Errorf("unexpected Workbench release tag %q", raw.TagName)
	}
	version := strings.TrimPrefix(raw.TagName, "v")
	if _, err := parseStableVersion(version); err != nil {
		return UpdateRelease{}, err
	}
	if raw.HTMLURL != "" {
		u, err := url.Parse(raw.HTMLURL)
		if err != nil || u.Scheme != "https" || !strings.EqualFold(u.Hostname(), "github.com") || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
			return UpdateRelease{}, errors.New("Workbench release page URL is not trusted")
		}
		expectedPage := "/" + officialUpdateOwner + "/" + officialUpdateRepo + "/releases/tag/" + raw.TagName
		if u.Path != expectedPage {
			return UpdateRelease{}, errors.New("Workbench release page does not belong to the official repository")
		}
	}
	assets := make(map[string]UpdateAsset, len(raw.Assets))
	for _, a := range raw.Assets {
		name := strings.TrimSpace(a.Name)
		if name == "" {
			return UpdateRelease{}, errors.New("Workbench release contains an unnamed asset")
		}
		if _, exists := assets[name]; exists {
			return UpdateRelease{}, fmt.Errorf("Workbench release contains duplicate asset %q", name)
		}
		asset := UpdateAsset{Name: name, Size: a.Size, DownloadURL: strings.TrimSpace(a.BrowserDownloadURL), Digest: strings.TrimSpace(a.Digest)}
		if err := validateOfficialAssetURL(asset.DownloadURL, raw.TagName, name); err != nil {
			return UpdateRelease{}, err
		}
		if asset.Size <= 0 {
			return UpdateRelease{}, fmt.Errorf("Workbench release asset %q has invalid size", name)
		}
		if asset.Digest != "" {
			if _, err := parseSHA256Digest(asset.Digest); err != nil {
				return UpdateRelease{}, fmt.Errorf("Workbench release asset %q has invalid digest: %w", name, err)
			}
		}
		assets[name] = asset
	}
	return UpdateRelease{Version: version, Tag: raw.TagName, PageURL: raw.HTMLURL, Assets: assets}, nil
}

func validateOfficialRelease(release UpdateRelease) error {
	if _, err := parseStableVersion(release.Version); err != nil {
		return err
	}
	if release.Tag != "v"+release.Version {
		return errors.New("Workbench update release tag does not match its version")
	}
	if release.Assets == nil {
		return errors.New("Workbench update release has no assets")
	}
	return nil
}

func validateOfficialAssetURL(rawURL, tag, assetName string) error {
	u, err := url.Parse(rawURL)
	if err != nil || u.Scheme != "https" || !strings.EqualFold(u.Hostname(), "github.com") || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return fmt.Errorf("Workbench release asset %q has an untrusted download URL", assetName)
	}
	expected := "/" + officialUpdateOwner + "/" + officialUpdateRepo + "/releases/download/" + tag + "/" + assetName
	if u.Path != expected {
		return fmt.Errorf("Workbench release asset %q does not belong to the official release", assetName)
	}
	return nil
}

func trustedGitHubDownloadHost(u *url.URL) bool {
	if u == nil || u.Scheme != "https" || u.User != nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	return host == "github.com" || strings.HasSuffix(host, ".githubusercontent.com")
}

func compareStableVersions(a, b string) (int, error) {
	av, err := parseStableVersion(a)
	if err != nil {
		return 0, err
	}
	bv, err := parseStableVersion(b)
	if err != nil {
		return 0, err
	}
	for i := range av {
		if av[i] < bv[i] {
			return -1, nil
		}
		if av[i] > bv[i] {
			return 1, nil
		}
	}
	return 0, nil
}

func parseStableVersion(v string) ([3]uint64, error) {
	var out [3]uint64
	if !stableVersionPattern.MatchString(v) {
		return out, fmt.Errorf("invalid stable Workbench version %q", v)
	}
	parts := strings.Split(v, ".")
	for i, part := range parts {
		n, err := strconv.ParseUint(part, 10, 64)
		if err != nil {
			return out, fmt.Errorf("invalid stable Workbench version %q", v)
		}
		out[i] = n
	}
	return out, nil
}

func DownloadVerifiedReleaseAsset(ctx context.Context, release UpdateRelease, assetName, checksumName string, maxBytes int64) (VerifiedUpdateAsset, error) {
	if err := validateOfficialRelease(release); err != nil {
		return VerifiedUpdateAsset{}, err
	}
	if maxBytes <= 0 {
		return VerifiedUpdateAsset{}, errors.New("update asset size limit must be positive")
	}
	asset, ok := release.Assets[assetName]
	if !ok {
		return VerifiedUpdateAsset{}, fmt.Errorf("Workbench release is missing %s", assetName)
	}
	checksumAsset, ok := release.Assets[checksumName]
	if !ok {
		return VerifiedUpdateAsset{}, fmt.Errorf("Workbench release is missing %s", checksumName)
	}
	if asset.Name != assetName || checksumAsset.Name != checksumName {
		return VerifiedUpdateAsset{}, errors.New("Workbench release asset metadata does not match its name")
	}
	if err := validateOfficialAssetURL(asset.DownloadURL, release.Tag, assetName); err != nil {
		return VerifiedUpdateAsset{}, err
	}
	if err := validateOfficialAssetURL(checksumAsset.DownloadURL, release.Tag, checksumName); err != nil {
		return VerifiedUpdateAsset{}, err
	}
	if checksumAsset.Size > maxChecksumAssetBytes {
		return VerifiedUpdateAsset{}, errors.New("Workbench checksum asset is unexpectedly large")
	}
	if asset.Size > maxBytes {
		return VerifiedUpdateAsset{}, fmt.Errorf("Workbench release asset %s exceeds the allowed size", assetName)
	}

	client := officialHTTPClient()
	checksumBody, checksumHash, err := downloadBounded(ctx, client, checksumAsset, maxChecksumAssetBytes, nil)
	if err != nil {
		return VerifiedUpdateAsset{}, fmt.Errorf("download %s: %w", checksumName, err)
	}
	if err := verifyDeclaredDigest(checksumAsset, checksumHash); err != nil {
		return VerifiedUpdateAsset{}, err
	}
	expectedHash, err := parseChecksumFile(checksumBody, assetName)
	if err != nil {
		return VerifiedUpdateAsset{}, err
	}

	dir, err := NewScratchDirectory("update-")
	if err != nil {
		return VerifiedUpdateAsset{}, err
	}
	cleanup := true
	defer func() {
		if cleanup {
			_ = os.RemoveAll(dir)
		}
	}()
	filePath := pathForUpdateAsset(dir, assetName)
	f, err := os.OpenFile(filePath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return VerifiedUpdateAsset{}, err
	}
	_, actualHash, err := downloadBounded(ctx, client, asset, maxBytes, f)
	closeErr := f.Close()
	if err != nil {
		return VerifiedUpdateAsset{}, fmt.Errorf("download %s: %w", assetName, err)
	}
	if closeErr != nil {
		return VerifiedUpdateAsset{}, closeErr
	}
	if err := verifyDeclaredDigest(asset, actualHash); err != nil {
		return VerifiedUpdateAsset{}, err
	}
	if !strings.EqualFold(actualHash, expectedHash) {
		return VerifiedUpdateAsset{}, fmt.Errorf("Workbench release checksum mismatch for %s", assetName)
	}
	cleanup = false
	return VerifiedUpdateAsset{Name: assetName, Path: filePath, SHA256: actualHash, Size: asset.Size}, nil
}

func pathForUpdateAsset(dir, assetName string) string {
	return dir + string(os.PathSeparator) + strings.ReplaceAll(strings.ReplaceAll(assetName, "/", "_"), "\\", "_")
}

func downloadBounded(ctx context.Context, client *http.Client, asset UpdateAsset, maxBytes int64, dst io.Writer) ([]byte, string, error) {
	if asset.Size <= 0 || asset.Size > maxBytes {
		return nil, "", errors.New("update asset size is outside the allowed range")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, asset.DownloadURL, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("User-Agent", "DaisyCloverSoftware-Workbench-Updater/"+Version)
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	if resp.ContentLength > maxBytes {
		return nil, "", errors.New("update download exceeds the allowed size")
	}
	h := sha256.New()
	var buffer bytes.Buffer
	writers := []io.Writer{h}
	if dst != nil {
		writers = append(writers, dst)
	} else {
		writers = append(writers, &buffer)
	}
	n, err := io.Copy(io.MultiWriter(writers...), io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return nil, "", err
	}
	if n > maxBytes {
		return nil, "", errors.New("update download exceeds the allowed size")
	}
	if n != asset.Size {
		return nil, "", fmt.Errorf("update download size mismatch: got %d, want %d", n, asset.Size)
	}
	return buffer.Bytes(), hex.EncodeToString(h.Sum(nil)), nil
}

func parseChecksumFile(body []byte, expectedName string) (string, error) {
	line := strings.TrimSpace(string(body))
	if strings.ContainsAny(line, "\r\n") {
		return "", errors.New("Workbench checksum file must contain exactly one record")
	}
	fields := strings.Fields(line)
	if len(fields) != 2 {
		return "", errors.New("Workbench checksum file has an invalid format")
	}
	hashValue := strings.ToLower(fields[0])
	name := strings.TrimPrefix(fields[1], "*")
	if name != expectedName {
		return "", fmt.Errorf("Workbench checksum file names %q instead of %q", name, expectedName)
	}
	if len(hashValue) != sha256.Size*2 {
		return "", errors.New("Workbench checksum is not SHA-256")
	}
	if _, err := hex.DecodeString(hashValue); err != nil {
		return "", errors.New("Workbench checksum is not valid hexadecimal SHA-256")
	}
	return hashValue, nil
}

func parseSHA256Digest(digest string) (string, error) {
	parts := strings.SplitN(strings.TrimSpace(digest), ":", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "sha256") || len(parts[1]) != sha256.Size*2 {
		return "", errors.New("digest is not SHA-256")
	}
	value := strings.ToLower(parts[1])
	if _, err := hex.DecodeString(value); err != nil {
		return "", errors.New("digest is not valid hexadecimal SHA-256")
	}
	return value, nil
}

func verifyDeclaredDigest(asset UpdateAsset, actual string) error {
	if strings.TrimSpace(asset.Digest) == "" {
		return nil
	}
	want, err := parseSHA256Digest(asset.Digest)
	if err != nil {
		return fmt.Errorf("Workbench release asset %s has an invalid declared digest: %w", asset.Name, err)
	}
	if !strings.EqualFold(want, actual) {
		return fmt.Errorf("Workbench release asset digest mismatch for %s", asset.Name)
	}
	return nil
}
