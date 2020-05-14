package methods

import (
	"fmt"
)

type GitHubZsync struct {
	username   string
	repository string
	release    string
	filename   string
}

func (info *GitHubZsync) Name() string {
	return "gh-releases-zsync"
}

func (instance *GitHubZsync) Execute() error {
	fmt.Println("Running ", instance.Name())
	return nil
}

func NewGitHubUpdate(parts []string) (*GitHubZsync, error) {
	if len(parts) != 5 {
		return nil, fmt.Errorf("Invalid GitHub update info. Expected: gh-releases-zsync|<username>|<repo>|<release>|<file name>")
	}

	info := GitHubZsync{
		username:   parts[1],
		repository: parts[2],
		release:    parts[3],
		filename:   parts[4],
	}

	return &info, nil
}
