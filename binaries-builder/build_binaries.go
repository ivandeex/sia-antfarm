package binariesbuilder

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	"go.sia.tech/sia-antfarm/persist"
	"gitlab.com/NebulousLabs/Sia/build"
	"gitlab.com/NebulousLabs/errors"
)

const (
	// siaRepoID is Gitlab Sia repository ID taken from:
	// https://gitlab.com/NebulousLabs/Sia > Project Overview > Details.
	siaRepoID = "7508674"

	// antfarmTagSuffix is a git tag suffix given to updated Sia releases in
	// Sia repository. E.g. Sia release v1.4.8 was updated for antfarm and the
	// git commit was tagged v1.4.8-antfarm
	antfarmTagSuffix = "-antfarm"
)

var (
	// Build variables
	goos = runtime.GOOS
	arch = runtime.GOARCH
)

type (
	// bySemanticVersion is a type for implementing sort.Interface to sort by
	// semantic version.
	bySemanticVersion []string

	// Command defines a struct for parameters to call execute method.
	Command struct {
		// Specific environment variables to set
		EnvVars map[string]string
		// Name of the command
		Name string
		// Command's subcommands or arguments
		Args []string
		// Working directory. Keep unset (empty string) to use default caller
		// working directory
		Dir string
	}
)

// buildSiad builds specified siad-dev versions defined by git tags into the
// given directory. If the given directory is relative path, it is relative to
// Sia-Ant-Farm/version-test directory.
func buildSiad(logger *persist.Logger, binariesDir string, versions ...string) error {
	vs := strings.Join(versions, ", ")
	logger.Debugf("preparing to build siad versions: %v", vs)

	// Clone Sia repository if it doesn't exist locally
	goPath, ok := os.LookupEnv("GOPATH")
	if !ok {
		return errors.New("couldn't get GOPATH environment variable")
	}
	gitlabNebulous := "gitlab.com/NebulousLabs"
	gitlabSia := fmt.Sprintf("%v/Sia", gitlabNebulous)
	siaPath := fmt.Sprintf("%v/src/%v", goPath, gitlabSia)
	siaRepoURL := fmt.Sprintf("https://%v.git", gitlabSia)
	err := gitClone(logger, siaRepoURL, siaPath)
	if err != nil {
		return errors.AddContext(err, "can't clone Sia repository")
	}

	// Checkout the master
	err = gitCheckout(logger, siaPath, "master")
	if err != nil {
		return errors.AddContext(err, "can't checkout specific Sia version")
	}

	// Git reset to clean git repository
	cmd := Command{
		Name: "git",
		Args: []string{"-C", siaPath, "reset", "--hard", "HEAD"},
	}
	_, err = cmd.Execute(logger)
	if err != nil {
		return errors.AddContext(err, "can't reset Sia git repository")
	}

	// Git fetch to get new branches
	cmd = Command{
		Name: "git",
		Args: []string{"-C", siaPath, "fetch"},
	}
	_, err = cmd.Execute(logger)
	if err != nil {
		return errors.AddContext(err, "can't fetch Sia git repository")
	}

	// Git pull including tags to get latest state
	cmd = Command{
		Name: "git",
		Args: []string{"-C", siaPath, "pull", "--tags", "--prune", "--force", "origin", "master"},
	}
	_, err = cmd.Execute(logger)
	if err != nil {
		return errors.AddContext(err, "can't pull Sia git repository")
	}

	for _, version := range versions {
		logger.Debugf("building a siad version: %v", version)

		// Create directory to store each version siad binary
		binarySubDir := siadBinarySubDir(version)
		var binaryDir string
		if filepath.IsAbs(binariesDir) {
			binaryDir = filepath.Join(binariesDir, binarySubDir)
		} else {
			wd, err := os.Getwd()
			if err != nil {
				return errors.AddContext(err, "can't get current working directory")
			}
			binaryDir = filepath.Join(wd, binariesDir, binarySubDir)
		}

		err := os.MkdirAll(binaryDir, 0700)
		if err != nil {
			return errors.AddContext(err, "can't create a directory for storing built siad binary")
		}

		// Checkout merkletree repository correct commit in for Sia v1.4.0
		merkletreePath := filepath.Join(goPath, "src", gitlabNebulous, "merkletree")
		if version == "v1.4.0" {
			// Clone merkletree repo if not yet available
			gitlabMerkletree := fmt.Sprintf("%v/merkletree", gitlabNebulous)
			merkletreeRepoURL := fmt.Sprintf("https://%v.git", gitlabMerkletree)
			err := gitClone(logger, merkletreeRepoURL, merkletreePath)
			if err != nil {
				return errors.AddContext(err, "can't clone merkletree repository")
			}

			// Checkout the specific merkletree commit
			err = gitCheckout(logger, merkletreePath, "bc4a11e")
			if err != nil {
				return errors.AddContext(err, "can't checkout specific merkletree commit")
			}
		}

		// Checkout the version
		err = gitCheckout(logger, siaPath, version)
		if err != nil {
			return errors.AddContext(err, "can't checkout specific Sia version")
		}

		// Get dependencies
		cmd = Command{
			Name: "go",
			Args: []string{"get", "-d", gitlabSia + "/..."},
			Dir:  siaPath,
		}
		_, err = cmd.Execute(logger)
		if err != nil {
			return errors.AddContext(err, "can't get dependencies")
		}

		// Compile siad-dev binaries
		pkg := filepath.Join(gitlabSia, "cmd/siad")
		binaryName := "siad-dev"
		binaryPath := filepath.Join(binaryDir, binaryName)

		// Set ldflags according to Sia/Makefile
		cmd = Command{Name: "date"}
		buildTime, err := cmd.Execute(logger)
		if err != nil {
			return errors.AddContext(err, "can't get build time")
		}
		buildTime = strings.TrimSpace(buildTime)
		cmd = Command{Name: "git", Args: []string{"-C", siaPath, "rev-parse", "--short", "HEAD"}}
		gitRevision, err := cmd.Execute(logger)
		if err != nil {
			return errors.AddContext(err, "can't get git revision")
		}
		gitRevision = strings.TrimSpace(gitRevision)

		var ldFlags string
		ldFlags += fmt.Sprintf(" -X gitlab.com/NebulousLabs/Sia/build.GitRevision=%v", gitRevision)
		ldFlags += fmt.Sprintf(" -X 'gitlab.com/NebulousLabs/Sia/build.BuildTime=%v'", buildTime)

		var args []string
		args = append(args, "build")
		args = append(args, "-a")
		args = append(args, "-tags")
		args = append(args, "dev debug profile netgo")
		args = append(args, "-trimpath")
		args = append(args, "-ldflags")
		args = append(args, ldFlags)
		args = append(args, "-o")
		args = append(args, binaryPath)
		args = append(args, pkg)

		// Need to set script's working directory to Sia repo so that
		// 'go build' has access to Sia 'go.mod'.
		cmd = Command{
			EnvVars: map[string]string{
				"GOOS":   goos,
				"GOARCH": arch,
			},
			Name: "go",
			Args: args,
			Dir:  siaPath,
		}
		_, err = cmd.Execute(logger)
		if err != nil {
			return errors.AddContext(err, "can't build siad binary")
		}

		// Checkout merkletree repository back to master after Sia v1.4.0
		if version == "v1.4.0" {
			err := gitCheckout(logger, merkletreePath, "master")
			if err != nil {
				return errors.AddContext(err, "can't checkout merkletree master")
			}
		}

		// Git reset to clean git repository
		cmd := Command{
			Name: "git",
			Args: []string{"-C", siaPath, "reset", "--hard", "HEAD"},
		}
		_, err = cmd.Execute(logger)
		if err != nil {
			return errors.AddContext(err, "can't reset Sia git repository")
		}
	}

	// Checkout the master
	err = gitCheckout(logger, siaPath, "master")
	if err != nil {
		return errors.AddContext(err, "can't checkout specific Sia version")
	}

	return nil
}

// ExcludeVersions takes as an input a slice of versions to be filtered and a
// slice of versions which should be excluded from the first slice. It returns
// a slice of versions without excluded versions.
func ExcludeVersions(versions, excludeVersions []string) []string {
	result := []string{}

versionsLoop:
	for _, v := range versions {
		for _, ev := range excludeVersions {
			// Check if we have found the version we want to exclude. Exclude
			// also versions with "-antfarm" tag postfix.
			if v == ev || v == ev+"-antfarm" {
				continue versionsLoop
			}
		}
		result = append(result, v)
	}

	return result
}

// ReleasesWithMaxVersion returns releases that satisfy the given maximal
// version.
func ReleasesWithMaxVersion(releases []string, maxVersion string) []string {
	var filteredReleases []string
	for _, r := range releases {
		if build.VersionCmp(r, maxVersion) <= 0 {
			filteredReleases = append(filteredReleases, r)
		}
	}
	return filteredReleases
}

// ReleasesWithMinVersion returns releases that satisfy the given minimal
// version.
func ReleasesWithMinVersion(releases []string, minVersion string) []string {
	var filteredReleases []string
	for _, r := range releases {
		if build.VersionCmp(r, minVersion) >= 0 {
			filteredReleases = append(filteredReleases, r)
		}
	}
	return filteredReleases
}

// GetReleases returns slice of git tags of Sia Gitlab releases greater than or
// equal to the given minimal version in ascending semantic version order. If
// there is a patch tagged with "-antfarm" suffix for a Sia release, patch tag
// instead release tag is added to the return slice.
// NOTE: These patches are ONLY to enable the Sia Antfarm to run and are not
// intended to address any underlying bugs in siad.
func GetReleases(minVersion string) ([]string, error) {
	// Get tags from Gitlab Sia repository. It can be multiple pages.
	bodies, err := querySiaRepoAPI("repository/tags")
	if err != nil {
		return nil, errors.AddContext(err, "can't get Sia tags from Gitlab")
	}

	// Colect release tags and release patch tags
	var releaseTags []string
	patchTags := make(map[string]struct{})

	// Process each returned page data
	for _, body := range bodies {
		// Decode response into slice of tags data
		var tags []map[string]interface{}
		if err := json.Unmarshal(body, &tags); err != nil {
			return nil, errors.AddContext(err, "can't decode tags response from Gitlab")
		}

		for _, t := range tags {
			tag := fmt.Sprintf("%v", t["name"])

			// Collect releases from minimal version up
			tagNums := strings.TrimLeft(tag, "v")
			minVersionNums := strings.TrimLeft(minVersion, "v")
			if t["release"] != nil && build.VersionCmp(tagNums, minVersionNums) >= 0 {
				releaseTags = append(releaseTags, tag)
			}

			// Collect release patch tags
			if strings.HasSuffix(tag, antfarmTagSuffix) {
				patchTags[tag] = struct{}{}
			}
		}
	}

	// Sort releases in ascending order by semantic version
	sort.Sort(bySemanticVersion(releaseTags))

	// Workaround for v1.5.5 just being a git tag, not yet released in
	// https://gitlab.com/NebulousLabs/Sia/-/releases
	// TODO: Once v1.5.5 is properly released, this code can be removed.
	v155 := "v1.5.5"
	var found bool
	var inserted bool
	for i, t := range releaseTags {
		// v1.5.5 was already released, nothing to do.
		if t == v155 {
			found = true
			break
		}
		// Insert v1.5.5 before this release.
		if build.VersionCmp(v155, t) < 0 {
			releaseTags = append(releaseTags, "")
			copy(releaseTags[i+1:], releaseTags[i:])
			releaseTags[i] = v155
			inserted = true
			break
		}
	}
	// Append v1.5.5 as the last release.
	if !found && !inserted {
		releaseTags = append(releaseTags, v155)
	}

	// If there is an antfarm patch for a release, replace release tag with a
	// patch tag
	for i, r := range releaseTags {
		versionWithSuffix := r + antfarmTagSuffix
		if _, ok := patchTags[versionWithSuffix]; ok {
			releaseTags[i] = versionWithSuffix
		}
	}

	return releaseTags, nil
}

// gitCheckout changes working directory to the git repository, performs git
// reset and git checkout by branch, tag or commit id specified in checkoutStr
// and changes working directory back to original directory.
func gitCheckout(logger *persist.Logger, gitRepoPath, checkoutStr string) error {
	// Reset git
	cmd := Command{
		Name: "git",
		Args: []string{"-C", gitRepoPath, "reset", "--hard", "HEAD"},
	}
	_, err := cmd.Execute(logger)
	if err != nil {
		return errors.AddContext(err, "can't reset git repository")
	}

	// Git checkout by branch, tag or commit id
	cmd = Command{
		Name: "git",
		Args: []string{"-C", gitRepoPath, "checkout", checkoutStr},
	}
	_, err = cmd.Execute(logger)
	if err != nil {
		return errors.AddContext(err, "can't perform git checkout")
	}

	return nil
}

// gitClone clones git repository by given URL to the given path.
func gitClone(logger *persist.Logger, repoURL, repoPath string) error {
	// Return if directory already exists
	_, err := os.Stat(repoPath)
	if err != nil && !os.IsNotExist(err) {
		return errors.AddContext(err, "can't get directory info")
	} else if err == nil {
		return nil
	}

	// Directory doesn't exist
	logger.Debugf("cloning git repository %v to %v.", repoURL, repoPath)

	// Create repository directory
	err = os.MkdirAll(repoPath, 0700)
	if err != nil {
		return errors.AddContext(err, "can't create repository directory")
	}

	// Clone repository
	cmd := Command{
		Name: "git",
		Args: []string{"clone", repoURL, repoPath},
	}
	_, err = cmd.Execute(logger)
	if err != nil {
		return errors.AddContext(err, "can't clone repository")
	}

	return nil
}

// querySiaRepoAPI queries Sia repository using Gitlab API with the given
// endpoint. The Gitlab API results are paginated, so it returns a slice of
// response bodies from each page. Each response body contains a byte slice.
func querySiaRepoAPI(siaRepoEndpoint string) (bodies [][]byte, err error) {
	// perPage defines maximum number of items returned by Gitlab API. The API
	// pagination allows max 100 items per page
	const perPage = 100

	// Handle Gitlab API pagination
	page := 1
	for {
		url := fmt.Sprintf("https://gitlab.com/api/v4/projects/%v/%v?per_page=%v&page=%v", siaRepoID, siaRepoEndpoint, perPage, page)
		resp, err := http.Get(url) //nolint:gosec
		if err != nil {
			msg := fmt.Sprintf("can't get response from %v", url)
			return nil, errors.AddContext(err, msg)
		}
		defer func() {
			if err = resp.Body.Close(); err != nil {
				err = errors.AddContext(err, "can't close response body")
			}
		}()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("response status from Gitlab is not '200 OK' but %v", resp.Status)
		}
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, errors.AddContext(err, "can't read response body")
		}
		bodies = append(bodies, body)

		if resp.Header.Get("X-Next-Page") == "" {
			break
		}

		page++
	}
	return
}

// siadBinarySubDir returns a directory where built siad-dev of the given
// version is stored.
func siadBinarySubDir(version string) string {
	return fmt.Sprintf("Sia-%s-%s-%s", version, goos, arch)
}

// SiadBinaryPath returns built siad-dev binary path from the given Sia
// version.
func SiadBinaryPath(version string) string {
	subDir := siadBinarySubDir(version)
	return fmt.Sprintf("%s/%s/siad-dev", BinariesDir, subDir)
}

// Len implements sort.Interface to sort by semantic version
func (s bySemanticVersion) Len() int {
	return len(s)
}

// Swap implements sort.Interface to sort by semantic version
func (s bySemanticVersion) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}

// Less implements sort.Interface to sort by semantic version
func (s bySemanticVersion) Less(i, j int) bool {
	return build.VersionCmp(s[i], s[j]) < 0
}

// Execute executes a given shell command defined by command receiver.
// Command struct is used instead of passing the whole command as a string and
// parsing string arguments because parsing arguments containing spaces would
// make the parsing much complex.
func (c Command) Execute(logger *persist.Logger) (string, error) {
	cmd := exec.Command(c.Name, c.Args...) //nolint:gosec
	cmd.Env = os.Environ()
	cmd.Dir = c.Dir
	var envVars = []string{}
	for k, v := range c.EnvVars {
		ev := fmt.Sprintf("%v=%v", k, v)
		envVars = append(envVars, ev)
		cmd.Env = append(cmd.Env, ev)
	}

	out, err := cmd.CombinedOutput()

	if err != nil {
		readableEnvVars := strings.Join(envVars, " ")
		readableArgs := strings.Join(c.Args, " ")
		readableCommand := fmt.Sprintf("%v %v %v", readableEnvVars, c.Name, readableArgs)
		wd, wdErr := os.Getwd()
		if wdErr != nil {
			return "", errors.AddContext(wdErr, "can't get working directory")
		}

		logger.Errorf("error executing bash command:\nWorking directory: %v\nCommand: %v\nOutput:\n%v", wd, readableCommand, string(out))

		msg := fmt.Sprintf("can't execute command: %v", readableCommand)
		return "", errors.AddContext(err, msg)
	}
	return string(out), nil
}
