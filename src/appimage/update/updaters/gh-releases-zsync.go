package updaters

import (
	"context"
	"fmt"
	"github.com/danwakefield/fnmatch"
	"github.com/google/go-github/v31/github"
	"strings"
)

type GitHubZsync struct {
	zsync ZSync

	gitHubUsername   string
	gitHubRepository string
	gitHubRelease    string
	gitHubFilename   string
}

func NewGitHubUpdater(updateInfoString *string, target *string) (*GitHubZsync, error) {
	parts := strings.Split(*updateInfoString, "|")

	if len(parts) != 5 {
		return nil, fmt.Errorf("Invalid GitHub update info. Expected: gh-releases-zsync|<username>|<repo>|<release>|<file name>")
	}

	info := GitHubZsync{
		zsync: ZSync{
			seed: *target,
		},

		gitHubUsername:   parts[1],
		gitHubRepository: parts[2],
		gitHubRelease:    parts[3],
		gitHubFilename:   parts[4],
	}

	return &info, nil
}

func (g *GitHubZsync) Method() string {
	return "gh-releases-zsync"
}

func (g *GitHubZsync) Lookup() (updateAvailable bool, err error) {
	client := github.NewClient(nil)
	releases, _, err := client.Repositories.ListReleases(context.Background(), g.gitHubUsername, g.gitHubRepository, nil)
	if err != nil {
		return false, err
	}

	for _, release := range releases {
		if *release.Draft == true {
			continue
		}

		if g.gitHubRelease != "latest" &&
			fnmatch.Match(g.gitHubRelease, *release.TagName, fnmatch.FNM_IGNORECASE) == false {
			continue
		}

		for _, asset := range release.Assets {
			if fnmatch.Match(g.gitHubFilename, *asset.Name, fnmatch.FNM_IGNORECASE) {
				g.zsync.url = *asset.BrowserDownloadURL
				return g.zsync.Lookup()
			}
		}
	}

	return false, nil
}

func (g *GitHubZsync) Download() (output string, err error) {
	return g.zsync.Download()
}
