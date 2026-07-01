package updater

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	version "github.com/gustmrg/lofi"
)

const (
	defaultOwner     = "gustmrg"
	defaultRepo      = "lofi"
	defaultAPIBase   = "https://api.github.com"
	defaultUserAgent = "lofi-update-checker"
	cacheMaxAge      = 24 * time.Hour
)

var ErrAlreadyCurrent = errors.New("already current")

type Client struct {
	HTTPClient *http.Client
	APIBase    string
	Owner      string
	Repo       string
	UserAgent  string
}

type Release struct {
	TagName string
	Assets  []Asset
}

type Asset struct {
	Name string
	URL  string
}

type CheckResult struct {
	Current string
	Latest  string
	Newer   bool
}

func DefaultClient() Client {
	return Client{
		HTTPClient: http.DefaultClient,
		APIBase:    defaultAPIBase,
		Owner:      defaultOwner,
		Repo:       defaultRepo,
		UserAgent:  defaultUserAgent,
	}
}

func (c Client) CheckLatest(ctx context.Context, current string) (CheckResult, error) {
	rel, err := c.LatestRelease(ctx)
	if err != nil {
		return CheckResult{}, err
	}
	latest := version.Normalize(rel.TagName)
	cmp, err := version.Compare(latest, current)
	if err != nil {
		return CheckResult{}, err
	}
	return CheckResult{
		Current: version.Normalize(current),
		Latest:  latest,
		Newer:   cmp > 0,
	}, nil
}

func (c Client) LatestRelease(ctx context.Context) (Release, error) {
	var body struct {
		TagName string `json:"tag_name"`
		Assets  []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}
	url := strings.TrimRight(c.apiBase(), "/") + "/repos/" + c.owner() + "/" + c.repo() + "/releases/latest"
	if err := c.getJSON(ctx, url, &body); err != nil {
		return Release{}, err
	}
	if body.TagName == "" {
		return Release{}, errors.New("latest release has no tag")
	}
	assets := make([]Asset, 0, len(body.Assets))
	for _, a := range body.Assets {
		if a.Name == "" || a.BrowserDownloadURL == "" {
			continue
		}
		assets = append(assets, Asset{Name: a.Name, URL: a.BrowserDownloadURL})
	}
	return Release{TagName: body.TagName, Assets: assets}, nil
}

func (c Client) Download(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	c.addHeaders(req)
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("download %s: %s", url, resp.Status)
	}
	return io.ReadAll(resp.Body)
}

func (c Client) getJSON(ctx context.Context, url string, dst any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	c.addHeaders(req)
	resp, err := c.httpClient().Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("github releases: %s", resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(dst)
}

func (c Client) addHeaders(req *http.Request) {
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", c.userAgent())
}

func (c Client) httpClient() *http.Client {
	if c.HTTPClient != nil {
		return c.HTTPClient
	}
	return http.DefaultClient
}

func (c Client) apiBase() string {
	if c.APIBase != "" {
		return c.APIBase
	}
	return defaultAPIBase
}

func (c Client) owner() string {
	if c.Owner != "" {
		return c.Owner
	}
	return defaultOwner
}

func (c Client) repo() string {
	if c.Repo != "" {
		return c.Repo
	}
	return defaultRepo
}

func (c Client) userAgent() string {
	if c.UserAgent != "" {
		return c.UserAgent
	}
	return defaultUserAgent
}

type NoticeChecker struct {
	Client       Client
	Current      string
	CachePath    string
	Now          func() time.Time
	CacheMaxAge  time.Duration
	SilentErrors bool
}

func DefaultNoticeChecker(current string) NoticeChecker {
	return NoticeChecker{
		Client:       DefaultClient(),
		Current:      current,
		SilentErrors: true,
	}
}

func (c NoticeChecker) Notice(ctx context.Context) (string, error) {
	now := c.now()
	cachePath, err := c.cachePath()
	if err != nil {
		if c.SilentErrors {
			return "", nil
		}
		return "", err
	}
	if cached, ok := readFreshCache(cachePath, now, c.maxAge()); ok {
		return cached.Notice, nil
	}
	res, err := c.Client.CheckLatest(ctx, c.Current)
	if err != nil {
		if c.SilentErrors {
			return "", nil
		}
		return "", err
	}
	notice := ""
	if res.Newer {
		notice = fmt.Sprintf("New version available: %s (you are using %s). Run 'lofi update' to update.", version.Tag(res.Latest), version.Tag(res.Current))
	}
	_ = writeCache(cachePath, updateCache{
		CheckedAt: now,
		Latest:    res.Latest,
		Notice:    notice,
	})
	return notice, nil
}

func (c NoticeChecker) now() time.Time {
	if c.Now != nil {
		return c.Now()
	}
	return time.Now()
}

func (c NoticeChecker) maxAge() time.Duration {
	if c.CacheMaxAge > 0 {
		return c.CacheMaxAge
	}
	return cacheMaxAge
}

func (c NoticeChecker) cachePath() (string, error) {
	if c.CachePath != "" {
		return c.CachePath, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".lofi", "update-check.json"), nil
}

type updateCache struct {
	CheckedAt time.Time `json:"checked_at"`
	Latest    string    `json:"latest_version"`
	Notice    string    `json:"notice"`
}

func readFreshCache(path string, now time.Time, maxAge time.Duration) (updateCache, bool) {
	data, err := os.ReadFile(path)
	if err != nil {
		return updateCache{}, false
	}
	var c updateCache
	if err := json.Unmarshal(data, &c); err != nil {
		return updateCache{}, false
	}
	if c.CheckedAt.IsZero() || now.Sub(c.CheckedAt) > maxAge {
		return updateCache{}, false
	}
	return c, true
}

func writeCache(path string, c updateCache) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	return nil
}

type Updater struct {
	Client         Client
	Current        string
	GOOS           string
	GOARCH         string
	ExecutablePath string
	TempDir        string
}

func DefaultUpdater(current string) Updater {
	return Updater{
		Client:  DefaultClient(),
		Current: current,
	}
}

func (u Updater) Run(ctx context.Context) (CheckResult, error) {
	if u.goos() != "darwin" && u.goos() != "linux" {
		return CheckResult{}, fmt.Errorf("unsupported platform %s/%s", u.goos(), u.goarch())
	}
	rel, err := u.Client.LatestRelease(ctx)
	if err != nil {
		return CheckResult{}, err
	}
	latest := version.Normalize(rel.TagName)
	cmp, err := version.Compare(latest, u.Current)
	if err != nil {
		return CheckResult{}, err
	}
	result := CheckResult{Current: version.Normalize(u.Current), Latest: latest, Newer: cmp > 0}
	if !result.Newer {
		return result, ErrAlreadyCurrent
	}

	archiveName := fmt.Sprintf("lofi_%s_%s_%s.tar.gz", version.Tag(latest), u.goos(), u.goarch())
	archiveAsset, ok := rel.asset(archiveName)
	if !ok {
		return result, fmt.Errorf("missing release asset %s", archiveName)
	}
	checksumAsset, ok := rel.asset("checksums.txt")
	if !ok {
		return result, errors.New("missing release asset checksums.txt")
	}

	checksums, err := u.Client.Download(ctx, checksumAsset.URL)
	if err != nil {
		return result, fmt.Errorf("download checksums: %w", err)
	}
	want, err := checksumFor(checksums, archiveName)
	if err != nil {
		return result, err
	}
	archive, err := u.Client.Download(ctx, archiveAsset.URL)
	if err != nil {
		return result, fmt.Errorf("download %s: %w", archiveName, err)
	}
	if err := verifyChecksum(archive, want); err != nil {
		return result, err
	}
	bin, mode, err := extractBinary(archive)
	if err != nil {
		return result, err
	}
	if err := u.replaceExecutable(bin, mode); err != nil {
		return result, err
	}
	return result, nil
}

func (r Release) asset(name string) (Asset, bool) {
	for _, a := range r.Assets {
		if a.Name == name {
			return a, true
		}
	}
	return Asset{}, false
}

func (u Updater) goos() string {
	if u.GOOS != "" {
		return u.GOOS
	}
	return runtime.GOOS
}

func (u Updater) goarch() string {
	if u.GOARCH != "" {
		return u.GOARCH
	}
	return runtime.GOARCH
}

func checksumFor(data []byte, filename string) (string, error) {
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		name := strings.TrimPrefix(fields[len(fields)-1], "*")
		if filepath.Base(name) == filename {
			return fields[0], nil
		}
	}
	return "", fmt.Errorf("checksum for %s not found", filename)
}

func verifyChecksum(data []byte, want string) error {
	sum := sha256.Sum256(data)
	got := hex.EncodeToString(sum[:])
	if !strings.EqualFold(got, want) {
		return fmt.Errorf("checksum mismatch: got %s, want %s", got, want)
	}
	return nil
}

func extractBinary(data []byte) ([]byte, os.FileMode, error) {
	gz, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, 0, fmt.Errorf("read archive: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, 0, fmt.Errorf("read archive: %w", err)
		}
		if h.FileInfo().IsDir() || filepath.Base(h.Name) != "lofi" {
			continue
		}
		bin, err := io.ReadAll(tr)
		if err != nil {
			return nil, 0, fmt.Errorf("read binary: %w", err)
		}
		mode := h.FileInfo().Mode()
		if mode&0o111 == 0 {
			mode |= 0o755
		}
		return bin, mode, nil
	}
	return nil, 0, errors.New("archive does not contain lofi binary")
}

func (u Updater) replaceExecutable(bin []byte, mode os.FileMode) error {
	exe := u.ExecutablePath
	if exe == "" {
		var err error
		exe, err = os.Executable()
		if err != nil {
			return fmt.Errorf("locate executable: %w", err)
		}
	}
	exe, err := filepath.EvalSymlinks(exe)
	if err != nil {
		return fmt.Errorf("resolve executable: %w", err)
	}
	dir := filepath.Dir(exe)
	tmp, err := os.CreateTemp(dir, ".lofi-new-*")
	if err != nil {
		return fmt.Errorf("create replacement: %w", err)
	}
	tmpPath := tmp.Name()
	cleanupTmp := true
	defer func() {
		if cleanupTmp {
			_ = os.Remove(tmpPath)
		}
	}()
	if _, err := tmp.Write(bin); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write replacement: %w", err)
	}
	if err := tmp.Chmod(mode.Perm() | 0o111); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("chmod replacement: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close replacement: %w", err)
	}

	backup := exe + ".bak"
	_ = os.Remove(backup)
	if err := os.Rename(exe, backup); err != nil {
		return fmt.Errorf("backup current binary: %w", err)
	}
	if err := os.Rename(tmpPath, exe); err != nil {
		_ = os.Rename(backup, exe)
		return fmt.Errorf("install replacement: %w", err)
	}
	cleanupTmp = false
	return nil
}
