package updaters

import (
	"context"
	"fmt"
	"strings"

	"github.com/danwakefield/fnmatch"
	"github.com/google/go-github/v31/github"
)

type GitHubDirect struct {
	direct Direct

	gitHubUsername   string
	gitHubRepository string
	gitHubRelease    string
	gitHubFilename   string
}

func NewGitHubDirectUpdater(updateInfoString *string, target string) (*GitHubDirect, error) {
	parts := strings.Split(*updateInfoString, "|")

	if len(parts) != 5 {
		return nil, fmt.Errorf("Invalid GitHub update info. Expected: gh-releases-direct|<username>|<repo>|<release>|<file name>")
	}

	info := GitHubDirect{
		direct: Direct{
			seed: target,
		},

		gitHubUsername:   parts[1],
		gitHubRepository: parts[2],
		gitHubRelease:    parts[3],
		gitHubFilename:   parts[4],
	}

	return &info, nil
}

func (g *GitHubDirect) Method() string {
	return "gh-releases-direct"
}

func (g *GitHubDirect) Lookup() (updateAvailable bool, err error) {
	client := github.NewClient(nil)
	releases, _, err := client.Repositories.ListReleases(context.Background(), g.gitHubUsername, g.gitHubRepository, nil)
	if err != nil {
		return false, err
	}

	var latest_release *github.RepositoryRelease

	for _, release := range releases {
		if *release.Draft == true {
			continue
		}

		if g.gitHubRelease != "latest" &&
			fnmatch.Match(g.gitHubRelease, *release.TagName, fnmatch.FNM_IGNORECASE) == false {
			continue
		}

		latest_release = release
		break
	}

	if latest_release == nil {
		return false, fmt.Errorf("no releases found")
	}

	// Support fallback to direct download
	if strings.HasSuffix(g.gitHubFilename, ".zsync") {
		g.gitHubFilename = g.gitHubFilename[:len(g.gitHubFilename)-6]
	}

	for _, asset := range latest_release.Assets {
		if fnmatch.Match(g.gitHubFilename, *asset.Name, fnmatch.FNM_IGNORECASE) {
			g.direct.url = *asset.BrowserDownloadURL
			return g.direct.Lookup()
		}
	}

	return false, nil
}

func (g *GitHubDirect) Download() (output string, err error) {
	return g.direct.Download()
}
