package core

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestCompareStableVersions(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"0.6.1", "0.6.1", 0},
		{"0.6.2", "0.6.1", 1},
		{"0.7.0", "0.6.99", 1},
		{"1.0.0", "0.99.99", 1},
		{"0.6.0", "0.6.1", -1},
	}
	for _, tc := range cases {
		got, err := compareStableVersions(tc.a, tc.b)
		if err != nil {
			t.Fatalf("compare %s %s: %v", tc.a, tc.b, err)
		}
		if got != tc.want {
			t.Fatalf("compare %s %s = %d, want %d", tc.a, tc.b, got, tc.want)
		}
	}
	for _, bad := range []string{"v0.6.1", "0.6", "0.6.1-beta", "01.2.3", ""} {
		if _, err := compareStableVersions(bad, "0.6.1"); err == nil {
			t.Fatalf("expected invalid version %q to fail", bad)
		}
	}
}

func TestNormalizeUpdateReleaseAcceptsOfficialStableRelease(t *testing.T) {
	body := `{
		"tag_name":"v0.7.0",
		"html_url":"https://github.com/DaisyCloverSoftware/workbench/releases/tag/v0.7.0",
		"draft":false,
		"prerelease":false,
		"assets":[
			{"name":"Workbench.exe","size":10,"browser_download_url":"https://github.com/DaisyCloverSoftware/workbench/releases/download/v0.7.0/Workbench.exe","digest":"sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"},
			{"name":"Workbench.exe.sha256","size":80,"browser_download_url":"https://github.com/DaisyCloverSoftware/workbench/releases/download/v0.7.0/Workbench.exe.sha256","digest":"sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}
		]
	}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.WriteString(w, body)
	}))
	defer server.Close()

	release, err := fetchUpdateRelease(context.Background(), server.Client(), server.URL, false)
	if err != nil {
		t.Fatal(err)
	}
	if release.Version != "0.7.0" || release.Tag != "v0.7.0" || len(release.Assets) != 2 {
		t.Fatalf("unexpected release: %#v", release)
	}
}

func TestNormalizeUpdateReleaseRejectsUntrustedMetadata(t *testing.T) {
	base := githubLatestRelease{TagName: "v0.7.0", HTMLURL: "https://github.com/DaisyCloverSoftware/workbench/releases/tag/v0.7.0"}
	base.Assets = append(base.Assets, struct {
		Name               string `json:"name"`
		Size               int64  `json:"size"`
		BrowserDownloadURL string `json:"browser_download_url"`
		Digest             string `json:"digest"`
	}{Name: "Workbench.exe", Size: 10, BrowserDownloadURL: "https://example.invalid/Workbench.exe"})
	if _, err := normalizeUpdateRelease(base); err == nil {
		t.Fatal("expected untrusted asset URL to fail")
	}

	pre := base
	pre.Assets = nil
	pre.Prerelease = true
	if _, err := normalizeUpdateRelease(pre); err == nil {
		t.Fatal("expected prerelease to fail")
	}

	duplicate := githubLatestRelease{TagName: "v0.7.0", HTMLURL: "https://github.com/DaisyCloverSoftware/workbench/releases/tag/v0.7.0"}
	for i := 0; i < 2; i++ {
		duplicate.Assets = append(duplicate.Assets, struct {
			Name               string `json:"name"`
			Size               int64  `json:"size"`
			BrowserDownloadURL string `json:"browser_download_url"`
			Digest             string `json:"digest"`
		}{Name: "same.bin", Size: 1, BrowserDownloadURL: "https://github.com/DaisyCloverSoftware/workbench/releases/download/v0.7.0/same.bin"})
	}
	if _, err := normalizeUpdateRelease(duplicate); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("expected duplicate asset failure, got %v", err)
	}
}

func TestDownloadBoundedVerifiesSizeAndHash(t *testing.T) {
	payload := []byte("verified Workbench payload")
	digest := sha256.Sum256(payload)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(payload)
	}))
	defer server.Close()
	asset := UpdateAsset{Name: "asset", Size: int64(len(payload)), DownloadURL: server.URL}
	body, gotHash, err := downloadBounded(context.Background(), server.Client(), asset, 1024, nil)
	if err != nil {
		t.Fatal(err)
	}
	wantHash := hex.EncodeToString(digest[:])
	if string(body) != string(payload) || gotHash != wantHash {
		t.Fatalf("download mismatch: body=%q hash=%s want=%s", body, gotHash, wantHash)
	}

	asset.Size++
	if _, _, err := downloadBounded(context.Background(), server.Client(), asset, 1024, nil); err == nil || !strings.Contains(err.Error(), "size mismatch") {
		t.Fatalf("expected size mismatch, got %v", err)
	}
}

func TestChecksumAndDeclaredDigestValidation(t *testing.T) {
	const hashValue = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	got, err := parseChecksumFile([]byte(hashValue+"  Workbench.exe\n"), "Workbench.exe")
	if err != nil || got != hashValue {
		t.Fatalf("checksum parse: got=%q err=%v", got, err)
	}
	if _, err := parseChecksumFile([]byte(hashValue+"  other.exe\n"), "Workbench.exe"); err == nil {
		t.Fatal("expected checksum filename mismatch")
	}
	asset := UpdateAsset{Name: "Workbench.exe", Digest: "sha256:" + hashValue}
	if err := verifyDeclaredDigest(asset, hashValue); err != nil {
		t.Fatal(err)
	}
	if err := verifyDeclaredDigest(asset, strings.Repeat("a", 64)); err == nil {
		t.Fatal("expected declared digest mismatch")
	}
}

func TestDownloadVerifiedReleaseAssetRejectsConstructedUntrustedURLBeforeNetwork(t *testing.T) {
	release := UpdateRelease{
		Version: "0.7.0",
		Tag:     "v0.7.0",
		Assets: map[string]UpdateAsset{
			"Workbench.exe": {Name: "Workbench.exe", Size: 1, DownloadURL: "https://example.invalid/Workbench.exe"},
			"Workbench.exe.sha256": {Name: "Workbench.exe.sha256", Size: 80, DownloadURL: "https://example.invalid/Workbench.exe.sha256"},
		},
	}
	if _, err := DownloadVerifiedReleaseAsset(context.Background(), release, "Workbench.exe", "Workbench.exe.sha256", 1024); err == nil || !strings.Contains(err.Error(), "untrusted") {
		t.Fatalf("expected untrusted URL rejection, got %v", err)
	}
}

func TestTrustedGitHubDownloadHosts(t *testing.T) {
	for _, raw := range []string{
		"https://github.com/DaisyCloverSoftware/workbench/releases/download/v0.7.0/a",
		"https://release-assets.githubusercontent.com/a",
		"https://objects.githubusercontent.com/a",
	} {
		u, _ := url.Parse(raw)
		if !trustedGitHubDownloadHost(u) {
			t.Fatalf("trusted GitHub host rejected: %s", raw)
		}
	}
	for _, raw := range []string{"http://github.com/a", "https://example.invalid/a", "https://github.com.evil.invalid/a"} {
		u, _ := url.Parse(raw)
		if trustedGitHubDownloadHost(u) {
			t.Fatalf("untrusted host accepted: %s", raw)
		}
	}
}
