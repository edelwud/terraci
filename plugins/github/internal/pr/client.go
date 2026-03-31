package pr

import (
	"context"
	"os"
	"strings"

	gh "github.com/google/go-github/v68/github"
)

type Client struct {
	client *gh.Client
	owner  string
	repo   string
	token  string
}

func NewClient(token, repository string) *Client {
	owner, repo := ParseRepository(repository)

	var client *gh.Client
	if token != "" {
		client = gh.NewClient(nil).WithAuthToken(token)
	} else {
		client = gh.NewClient(nil)
	}

	return &Client{
		client: client,
		owner:  owner,
		repo:   repo,
		token:  token,
	}
}

func NewClientFromEnv() *Client {
	return NewClient(os.Getenv("GITHUB_TOKEN"), os.Getenv("GITHUB_REPOSITORY"))
}

func (c *Client) HasToken() bool {
	return c.token != ""
}

func (c *Client) ListIssueComments(ctx context.Context, prNumber int) ([]*gh.IssueComment, error) {
	opts := &gh.IssueListCommentsOptions{
		ListOptions: gh.ListOptions{PerPage: 100},
	}

	var allComments []*gh.IssueComment
	for {
		comments, resp, err := c.client.Issues.ListComments(ctx, c.owner, c.repo, prNumber, opts)
		if err != nil {
			return nil, err
		}

		allComments = append(allComments, comments...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return allComments, nil
}

func (c *Client) CreateIssueComment(ctx context.Context, prNumber int, body string) (*gh.IssueComment, error) {
	comment := &gh.IssueComment{Body: gh.Ptr(body)}
	created, _, err := c.client.Issues.CreateComment(ctx, c.owner, c.repo, prNumber, comment)
	return created, err
}

func (c *Client) UpdateIssueComment(ctx context.Context, commentID int64, body string) (*gh.IssueComment, error) {
	comment := &gh.IssueComment{Body: gh.Ptr(body)}
	updated, _, err := c.client.Issues.EditComment(ctx, c.owner, c.repo, commentID, comment)
	return updated, err
}

func ParseRepository(repository string) (owner, repo string) {
	parts := strings.SplitN(repository, "/", 2)
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], parts[1]
}
