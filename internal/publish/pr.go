package publish

import (
	"fmt"
	"os/exec"
	"strings"
)

// Injection points so tests can run the PR flow against local bare
// repositories without touching GitHub.
var (
	runGH             = execGH
	resolveGitHubRepo = parseGitHubRepo
	forkPushURL       = githubForkURL
)

type prRequest struct {
	RegistryURL string
	Branch      string
	Title       string
	Identity    string
	Version     string
	Dest        string
	Checksum    string
}

// createPullRequest pushes the committed branch and opens a pull request
// against the registry repository. When the authenticated gh user does not
// own the repository, the branch is pushed to a fork first.
func createPullRequest(root string, request prRequest) (string, error) {
	owner, repo, err := resolveGitHubRepo(request.RegistryURL)
	if err != nil {
		return "", err
	}
	login, err := runGH(root, "api", "user", "-q", ".login")
	if err != nil {
		return "", fmt.Errorf("gh authentication required for --pr: %w", err)
	}
	login = strings.TrimSpace(login)
	head := request.Branch
	if login == owner {
		if err := runGit(root, "push", "origin", "HEAD:refs/heads/"+request.Branch); err != nil {
			return "", err
		}
	} else {
		if _, err := runGH(root, "repo", "fork", owner+"/"+repo, "--clone=false"); err != nil {
			return "", fmt.Errorf("fork %s/%s failed: %w", owner, repo, err)
		}
		forkURL := forkPushURL(login, repo)
		if err := runGit(root, "remote", "add", "fork", forkURL); err != nil {
			if err := runGit(root, "remote", "set-url", "fork", forkURL); err != nil {
				return "", err
			}
		}
		if err := runGit(root, "push", "fork", "HEAD:refs/heads/"+request.Branch); err != nil {
			return "", err
		}
		head = login + ":" + request.Branch
	}
	body := fmt.Sprintf(
		"Publish `%s@%s`.\n\n- destination: `%s`\n- checksum: `%s`\n\nOpened with `skillhub publish --pr`.",
		request.Identity, request.Version, request.Dest, request.Checksum,
	)
	output, err := runGH(root, "pr", "create",
		"--repo", owner+"/"+repo,
		"--head", head,
		"--title", request.Title,
		"--body", body,
	)
	if err != nil {
		return "", fmt.Errorf("gh pr create failed: %w", err)
	}
	return prURLFrom(output), nil
}

func prURLFrom(output string) string {
	lines := strings.Fields(output)
	for i := len(lines) - 1; i >= 0; i-- {
		if strings.HasPrefix(lines[i], "https://") {
			return lines[i]
		}
	}
	return strings.TrimSpace(output)
}

func parseGitHubRepo(url string) (string, string, error) {
	remainder := ""
	switch {
	case strings.HasPrefix(url, "https://github.com/"):
		remainder = strings.TrimPrefix(url, "https://github.com/")
	case strings.HasPrefix(url, "http://github.com/"):
		remainder = strings.TrimPrefix(url, "http://github.com/")
	case strings.HasPrefix(url, "git@github.com:"):
		remainder = strings.TrimPrefix(url, "git@github.com:")
	case strings.HasPrefix(url, "ssh://git@github.com/"):
		remainder = strings.TrimPrefix(url, "ssh://git@github.com/")
	default:
		return "", "", fmt.Errorf("--pr requires a github.com registry URL, got %q", url)
	}
	remainder = strings.TrimSuffix(strings.TrimSuffix(remainder, "/"), ".git")
	owner, repo, ok := strings.Cut(remainder, "/")
	if !ok || owner == "" || repo == "" || strings.Contains(repo, "/") {
		return "", "", fmt.Errorf("could not parse owner/repo from registry URL %q", url)
	}
	return owner, repo, nil
}

func githubForkURL(login string, repo string) string {
	return "https://github.com/" + login + "/" + repo + ".git"
}

func execGH(dir string, args ...string) (string, error) {
	cmd := exec.Command("gh", args...)
	if dir != "" {
		cmd.Dir = dir
	}
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("gh %v failed: %w\n%s", args, err, strings.TrimSpace(string(output)))
	}
	return string(output), nil
}
