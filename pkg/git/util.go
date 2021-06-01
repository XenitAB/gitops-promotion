package git

import (
	"fmt"
	"strings"

	giturls "github.com/whilp/git-urls"
)

func parseGitAddress(s string) (string, string, error) {
	u, err := giturls.Parse(s)
	if err != nil {
		return "", "", err
	}

	scheme := u.Scheme
	if u.Scheme == "ssh" {
		scheme = "https"
	}

	id := strings.TrimLeft(u.Path, "/")
	id = strings.TrimSuffix(id, ".git")
	host := fmt.Sprintf("%s://%s", scheme, u.Host)
	return host, id, nil
}
