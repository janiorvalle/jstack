package upgrade

import (
	"archive/tar"
	"archive/zip"
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
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"golang.org/x/mod/semver"

	"github.com/janiorvalle/squirrel/internal/setup"
)

const (
	latestReleaseURL        = "https://api.github.com/repos/janiorvalle/squirrel/releases/latest"
	installerCommand        = "curl -fsSL https://raw.githubusercontent.com/janiorvalle/squirrel/main/install.sh | sh"
	installerCommandWindows = "irm https://raw.githubusercontent.com/janiorvalle/squirrel/main/install.ps1 | iex"
	maximumMetadata         = 2 << 20
	maximumChecksums        = 1 << 20
	maximumArchive          = 100 << 20
	maximumBinary           = 100 << 20
)

func fallbackInstruction() string {
	if runtime.GOOS == "windows" {
		return fmt.Sprintf("run `%s` in PowerShell", installerCommandWindows)
	}
	return fmt.Sprintf("run `%s`", installerCommand)
}

type release struct {
	TagName string  `json:"tag_name"`
	Assets  []asset `json:"assets"`
}

type asset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
}

type options struct {
	client          *http.Client
	executable      func() (string, error)
	operatingSystem string
	architecture    string
	verifyBinary    func(context.Context, string, string) error
	rerunSetup      func(context.Context, string, io.Writer) error
}

// Run replaces the current executable with the latest verified release, then
// reruns setup so the skills embedded in the new binary land in the harnesses
// the saved picks name.
func Run(ctx context.Context, currentVersion string, output io.Writer) error {
	return run(ctx, currentVersion, output, options{
		client:          &http.Client{Timeout: 30 * time.Second},
		executable:      os.Executable,
		operatingSystem: runtime.GOOS,
		architecture:    runtime.GOARCH,
		verifyBinary:    verifyBinary,
		rerunSetup:      rerunSetup,
	})
}

func run(ctx context.Context, currentVersion string, output io.Writer, options options) error {
	if output == nil {
		output = io.Discard
	}
	latest, err := fetchRelease(ctx, options.client)
	if err != nil {
		return err
	}
	latestVersion := strings.TrimPrefix(strings.TrimSpace(latest.TagName), "v")
	latestSemanticVersion := semanticVersion(latestVersion)
	if latestSemanticVersion == "" {
		return fmt.Errorf("[SQUIRREL-UPGRADE-RELEASE] GitHub release metadata from %q has invalid tag_name %q; expected a semantic version such as v0.2.4; retry later or %s", latestReleaseURL, latest.TagName, fallbackInstruction())
	}

	executable, err := options.executable()
	if err != nil {
		return fmt.Errorf("[SQUIRREL-UPGRADE-PATH] cannot locate the running squirrel executable: %w; %s to install the latest release", err, fallbackInstruction())
	}
	executable, err = filepath.EvalSymlinks(executable)
	if err != nil {
		return fmt.Errorf("[SQUIRREL-UPGRADE-PATH] cannot resolve the running squirrel executable %q: %w; fix the path or %s", executable, err, fallbackInstruction())
	}

	currentVersion = strings.TrimPrefix(strings.TrimSpace(currentVersion), "v")
	fmt.Fprintf(output, "squirrel: version: %s -> %s\n", currentVersion, latestVersion)
	currentSemanticVersion := semanticVersion(currentVersion)
	if currentSemanticVersion != "" && semver.Compare(currentSemanticVersion, latestSemanticVersion) >= 0 {
		if semver.Compare(currentSemanticVersion, latestSemanticVersion) > 0 {
			fmt.Fprintf(output, "squirrel: current version %s is newer than latest release %s; binary kept\n", currentVersion, latestVersion)
			return finishWithSetup(ctx, options.rerunSetup, executable, output)
		}
		fmt.Fprintf(output, "squirrel: already up to date at %s\n", latestVersion)
		return finishWithSetup(ctx, options.rerunSetup, executable, output)
	}

	archiveName, archiveFormat, binaryName, err := platformAsset(latestVersion, options.operatingSystem, options.architecture)
	if err != nil {
		return err
	}
	archiveAsset, ok := findAsset(latest.Assets, archiveName)
	if !ok {
		return fmt.Errorf("[SQUIRREL-UPGRADE-ASSET] release %q has no %q asset; expected a build for %s/%s; download a compatible binary from https://github.com/janiorvalle/squirrel/releases/tag/%s", latest.TagName, archiveName, options.operatingSystem, options.architecture, latest.TagName)
	}
	checksumsAsset, ok := findAsset(latest.Assets, "checksums.txt")
	if !ok {
		return fmt.Errorf("[SQUIRREL-UPGRADE-ASSET] release %q has no checksums.txt asset; expected published release checksums; retry later or %s", latest.TagName, fallbackInstruction())
	}

	archive, err := download(ctx, options.client, archiveAsset.URL, maximumArchive, "SQUIRREL-UPGRADE-DOWNLOAD", archiveName)
	if err != nil {
		return err
	}
	checksums, err := download(ctx, options.client, checksumsAsset.URL, maximumChecksums, "SQUIRREL-UPGRADE-DOWNLOAD", "checksums.txt")
	if err != nil {
		return err
	}
	if err := verifyChecksum(archiveName, archive, checksums); err != nil {
		return err
	}
	binary, err := extractBinary(archiveName, archiveFormat, binaryName, archive)
	if err != nil {
		return err
	}
	if err := replaceExecutable(ctx, executable, binary, latestVersion, options.verifyBinary); err != nil {
		return err
	}
	fmt.Fprintf(output, "squirrel: binary updated at %q\n", executable)
	return finishWithSetup(ctx, options.rerunSetup, executable, output)
}

func semanticVersion(value string) string {
	value = strings.TrimSpace(value)
	if !strings.HasPrefix(value, "v") {
		value = "v" + value
	}
	if !semver.IsValid(value) {
		return ""
	}
	return value
}

// CleanupPreviousExecutable removes the Windows backup after the upgrading
// process exits. The path shape keeps the hidden command narrowly scoped.
func CleanupPreviousExecutable(ctx context.Context, path string, parentProcessID int) error {
	if !strings.Contains(filepath.Base(path), ".squirrel-previous-") {
		return fmt.Errorf("[SQUIRREL-UPGRADE-CLEANUP] refusing unexpected backup path %q; expected a squirrel upgrade backup", path)
	}
	if parentProcessID <= 0 {
		return fmt.Errorf("[SQUIRREL-UPGRADE-CLEANUP] refusing invalid parent process ID %d; expected a positive process ID", parentProcessID)
	}
	return cleanupReplacedExecutable(ctx, path, parentProcessID)
}

func fetchRelease(ctx context.Context, client *http.Client) (release, error) {
	content, err := download(ctx, client, latestReleaseURL, maximumMetadata, "SQUIRREL-UPGRADE-RELEASE", "release metadata")
	if err != nil {
		return release{}, err
	}
	var latest release
	if err := json.Unmarshal(content, &latest); err != nil {
		return release{}, fmt.Errorf("[SQUIRREL-UPGRADE-RELEASE] GitHub returned invalid release metadata from %q: %w; expected JSON with tag_name and assets; retry or %s", latestReleaseURL, err, fallbackInstruction())
	}
	return latest, nil
}

func download(ctx context.Context, client *http.Client, url string, limit int64, code, description string) ([]byte, error) {
	if client == nil {
		client = &http.Client{Timeout: 30 * time.Second}
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("[%s] cannot request %s from %q: %w; retry or %s", code, description, url, err, fallbackInstruction())
	}
	request.Header.Set("Accept", "application/vnd.github+json")
	request.Header.Set("User-Agent", "squirrel-upgrade")
	response, err := client.Do(request)
	if err != nil {
		return nil, fmt.Errorf("[%s] cannot download %s from %q: %w; check the network and retry, or %s", code, description, url, err, fallbackInstruction())
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("[%s] cannot download %s from %q: GitHub returned %s; expected HTTP 200; retry later or %s", code, description, url, response.Status, fallbackInstruction())
	}
	content, err := io.ReadAll(io.LimitReader(response.Body, limit+1))
	if err != nil {
		return nil, fmt.Errorf("[%s] cannot read %s from %q: %w; retry or %s", code, description, url, err, fallbackInstruction())
	}
	if int64(len(content)) > limit {
		return nil, fmt.Errorf("[%s] %s from %q exceeds the %d-byte safety limit; expected a normal squirrel release asset; use the release page to inspect it before retrying", code, description, url, limit)
	}
	return content, nil
}

func platformAsset(version, operatingSystem, architecture string) (string, string, string, error) {
	if (operatingSystem != "darwin" && operatingSystem != "linux" && operatingSystem != "windows") || (architecture != "amd64" && architecture != "arm64") {
		return "", "", "", fmt.Errorf("[SQUIRREL-UPGRADE-PLATFORM] squirrel does not publish automatic upgrades for %s/%s; expected darwin, linux, or windows on amd64 or arm64; download a compatible build from https://github.com/janiorvalle/squirrel/releases", operatingSystem, architecture)
	}
	format := "tar.gz"
	binaryName := "squirrel"
	if operatingSystem == "windows" {
		format = "zip"
		binaryName = "squirrel.exe"
	}
	return fmt.Sprintf("squirrel_%s_%s_%s.%s", version, operatingSystem, architecture, format), format, binaryName, nil
}

func findAsset(assets []asset, name string) (asset, bool) {
	for _, candidate := range assets {
		if candidate.Name == name && strings.TrimSpace(candidate.URL) != "" {
			return candidate, true
		}
	}
	return asset{}, false
}

func verifyChecksum(name string, content, checksums []byte) error {
	expected := ""
	for _, line := range strings.Split(string(checksums), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && strings.TrimPrefix(fields[1], "*") == name {
			expected = strings.ToLower(fields[0])
			break
		}
	}
	if len(expected) != sha256.Size*2 {
		return fmt.Errorf("[SQUIRREL-UPGRADE-CHECKSUM] checksums.txt has no valid SHA-256 entry for %q; expected 64 hexadecimal characters followed by the asset name; do not install this release and retry later", name)
	}
	if _, err := hex.DecodeString(expected); err != nil {
		return fmt.Errorf("[SQUIRREL-UPGRADE-CHECKSUM] checksums.txt has an invalid SHA-256 entry for %q: %q; expected hexadecimal text; do not install this release and retry later", name, expected)
	}
	actual := sha256.Sum256(content)
	if actualHex := hex.EncodeToString(actual[:]); actualHex != expected {
		return fmt.Errorf("[SQUIRREL-UPGRADE-CHECKSUM] checksum mismatch for %q (expected %s, downloaded %s); the binary was not changed; retry the upgrade or install from a trusted release", name, expected, actualHex)
	}
	return nil
}

func extractBinary(archiveName, format, binaryName string, content []byte) ([]byte, error) {
	var binary []byte
	var err error
	switch format {
	case "tar.gz":
		binary, err = extractTarGzip(binaryName, content)
	case "zip":
		binary, err = extractZip(binaryName, content)
	default:
		err = fmt.Errorf("unsupported archive format %q", format)
	}
	if err != nil {
		return nil, fmt.Errorf("[SQUIRREL-UPGRADE-ARCHIVE] cannot extract %q from %q: %w; the binary was not changed; retry or download the release manually", binaryName, archiveName, err)
	}
	return binary, nil
}

func extractTarGzip(binaryName string, content []byte) ([]byte, error) {
	compressed, err := gzip.NewReader(bytes.NewReader(content))
	if err != nil {
		return nil, err
	}
	defer compressed.Close()
	archive := tar.NewReader(compressed)
	for {
		header, err := archive.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		if header.Typeflag != tar.TypeReg || filepath.Base(header.Name) != binaryName {
			continue
		}
		return readBinary(archive, header.Size)
	}
	return nil, fmt.Errorf("archive does not contain a regular %s file", binaryName)
}

func extractZip(binaryName string, content []byte) ([]byte, error) {
	archive, err := zip.NewReader(bytes.NewReader(content), int64(len(content)))
	if err != nil {
		return nil, err
	}
	for _, file := range archive.File {
		if !file.Mode().IsRegular() || filepath.Base(file.Name) != binaryName {
			continue
		}
		if file.UncompressedSize64 > maximumBinary {
			return nil, fmt.Errorf("binary exceeds the %d-byte safety limit", maximumBinary)
		}
		reader, err := file.Open()
		if err != nil {
			return nil, err
		}
		binary, readErr := readBinary(reader, int64(file.UncompressedSize64))
		closeErr := reader.Close()
		return binary, errors.Join(readErr, closeErr)
	}
	return nil, fmt.Errorf("archive does not contain a regular %s file", binaryName)
}

func readBinary(reader io.Reader, declaredSize int64) ([]byte, error) {
	if declaredSize < 0 || declaredSize > maximumBinary {
		return nil, fmt.Errorf("binary size %d exceeds the %d-byte safety limit", declaredSize, maximumBinary)
	}
	binary, err := io.ReadAll(io.LimitReader(reader, maximumBinary+1))
	if err != nil {
		return nil, err
	}
	if int64(len(binary)) > maximumBinary {
		return nil, fmt.Errorf("binary exceeds the %d-byte safety limit", maximumBinary)
	}
	if int64(len(binary)) != declaredSize {
		return nil, fmt.Errorf("binary size is %d bytes, archive declared %d", len(binary), declaredSize)
	}
	return binary, nil
}

func replaceExecutable(ctx context.Context, destination string, binary []byte, latestVersion string, verifier func(context.Context, string, string) error) error {
	info, err := os.Stat(destination)
	if err != nil {
		return fmt.Errorf("[SQUIRREL-UPGRADE-REPLACE] cannot inspect the running executable %q: %w; make it readable and retry", destination, err)
	}
	pattern := ".squirrel-upgrade-*"
	if strings.EqualFold(filepath.Ext(destination), ".exe") {
		pattern += ".exe"
	}
	temporary, err := os.CreateTemp(filepath.Dir(destination), pattern)
	if err != nil {
		return fmt.Errorf("[SQUIRREL-UPGRADE-REPLACE] cannot stage the new binary beside %q: %w; make that directory writable and retry, or %s", destination, err, fallbackInstruction())
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	fail := func(step string, cause error) error {
		_ = temporary.Close()
		return fmt.Errorf("[SQUIRREL-UPGRADE-REPLACE] cannot %s for %q: %w; the current binary was not changed; fix its directory permissions and retry, or %s", step, destination, cause, fallbackInstruction())
	}
	if err := temporary.Chmod(info.Mode().Perm() | 0o111); err != nil {
		return fail("set executable permissions", err)
	}
	if _, err := temporary.Write(binary); err != nil {
		return fail("write the staged binary", err)
	}
	if err := temporary.Sync(); err != nil {
		return fail("sync the staged binary", err)
	}
	if err := temporary.Close(); err != nil {
		return fail("close the staged binary", err)
	}
	if verifier == nil {
		verifier = verifyBinary
	}
	if err := verifier(ctx, temporaryName, latestVersion); err != nil {
		return fmt.Errorf("[SQUIRREL-UPGRADE-SMOKE] downloaded binary %q failed its version check: %w; expected version %s; the current binary was not changed; retry later or %s", temporaryName, err, latestVersion, fallbackInstruction())
	}
	if err := os.Chmod(temporaryName, info.Mode().Perm()); err != nil {
		return fail("restore the existing executable permissions", err)
	}
	if err := replaceFileAtomically(temporaryName, destination); err != nil {
		return fail("atomically replace the running binary", err)
	}
	if directory, err := os.Open(filepath.Dir(destination)); err == nil {
		_ = directory.Sync()
		_ = directory.Close()
	}
	return nil
}

func verifyBinary(ctx context.Context, binary, expectedVersion string) error {
	result, err := exec.CommandContext(ctx, binary, "--version").CombinedOutput()
	if err != nil {
		return fmt.Errorf("run --version: %w (%s)", err, strings.TrimSpace(string(result)))
	}
	if actual := strings.TrimPrefix(strings.TrimSpace(string(result)), "v"); actual != expectedVersion {
		return fmt.Errorf("--version returned %q", actual)
	}
	return nil
}

func finishWithSetup(ctx context.Context, rerun func(context.Context, string, io.Writer) error, executable string, output io.Writer) error {
	if rerun == nil {
		rerun = rerunSetup
	}
	if err := rerun(ctx, executable, output); err != nil {
		return fmt.Errorf("[SQUIRREL-UPGRADE-SETUP] the binary is ready but setup did not finish: %w; run `squirrel setup` to install the skills from the new binary", err)
	}
	return nil
}

// rerunSetup runs setup from the new binary. With saved picks it applies
// straight away, terminal or not, and tools are still never installed without
// --install-tools. Without saved picks nobody has chosen harnesses yet, so
// setup asks on a terminal and only prints the plan without one.
func rerunSetup(ctx context.Context, executable string, output io.Writer) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("cannot find the home directory: %w", err)
	}
	command := exec.CommandContext(ctx, executable, setupArguments(setup.HasSavedPicks(home))...)
	command.Stdin = os.Stdin
	command.Stdout = output
	var stderr bytes.Buffer
	command.Stderr = &stderr
	if err := command.Run(); err != nil {
		return fmt.Errorf("%w (%s)", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

func setupArguments(savedPicks bool) []string {
	if savedPicks {
		return []string{"setup", "--yes"}
	}
	return []string{"setup"}
}
