package pr

import (
	"context"
	"testing"

	gh "github.com/google/go-github/v68/github"

	"github.com/edelwud/terraci/plugins/github/internal/config"
)

type fakeIssueCommentClient struct {
	hasToken  bool
	comments  []*gh.IssueComment
	listErr   error
	createErr error
	updateErr error

	createdPRNumber  int
	createdBody      string
	updatedCommentID int64
	updatedBody      string
}

func (f *fakeIssueCommentClient) HasToken() bool {
	return f.hasToken
}

func (f *fakeIssueCommentClient) ListIssueComments(_ context.Context, _ int) ([]*gh.IssueComment, error) {
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.comments, nil
}

func (f *fakeIssueCommentClient) CreateIssueComment(_ context.Context, prNumber int, body string) (*gh.IssueComment, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	f.createdPRNumber = prNumber
	f.createdBody = body
	return &gh.IssueComment{ID: gh.Ptr(int64(1)), Body: gh.Ptr(body)}, nil
}

func (f *fakeIssueCommentClient) UpdateIssueComment(_ context.Context, commentID int64, body string) (*gh.IssueComment, error) {
	if f.updateErr != nil {
		return nil, f.updateErr
	}
	f.updatedCommentID = commentID
	f.updatedBody = body
	return &gh.IssueComment{ID: gh.Ptr(commentID), Body: gh.Ptr(body)}, nil
}

type serviceScenario struct {
	t       *testing.T
	service *Service
	client  *fakeIssueCommentClient
}

func newServiceScenario(t *testing.T) *serviceScenario {
	t.Helper()
	client := &fakeIssueCommentClient{hasToken: true}
	service := NewService(nil, client, &Context{
		InPR:     true,
		PRNumber: 1,
	})
	return &serviceScenario{
		t:       t,
		service: service,
		client:  client,
	}
}

func (s *serviceScenario) withContext(ctx *Context) *serviceScenario {
	s.t.Helper()
	s.service.context = ctx
	return s
}

func (s *serviceScenario) withConfig(cfg *config.PRConfig) *serviceScenario {
	s.t.Helper()
	s.service.config = cfg
	return s
}

func (s *serviceScenario) withToken(hasToken bool) *serviceScenario {
	s.t.Helper()
	s.client.hasToken = hasToken
	return s
}

func (s *serviceScenario) withComments(comments ...*gh.IssueComment) *serviceScenario {
	s.t.Helper()
	s.client.comments = comments
	return s
}

func (s *serviceScenario) withListError(err error) *serviceScenario {
	s.t.Helper()
	s.client.listErr = err
	return s
}

func (s *serviceScenario) withCreateError(err error) *serviceScenario {
	s.t.Helper()
	s.client.createErr = err
	return s
}

func (s *serviceScenario) withUpdateError(err error) *serviceScenario {
	s.t.Helper()
	s.client.updateErr = err
	return s
}

func (s *serviceScenario) upsert(body string) error {
	s.t.Helper()
	return s.service.UpsertComment(context.Background(), body)
}
