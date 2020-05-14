package methods

import "fmt"

type BintrayZsync struct {
	username    string
	repository  string
	packageName string
	path        string
}

func (info *BintrayZsync) Name() string {
	return "bintray-zsync"
}

func (instance *BintrayZsync) Execute() error {
	fmt.Println("Running ", instance.Name())
	return nil
}

func NewBintrayZsync(parts []string) (*BintrayZsync, error) {
	if len(parts) != 5 {
		return nil, fmt.Errorf("Invalid Bintray update info. Expected: bintray-zsync|<username>|<repository>|<package name>|<path>")
	}

	info := BintrayZsync{
		username:    parts[1],
		repository:  parts[2],
		packageName: parts[3],
		path:        parts[4],
	}

	return &info, nil
}
