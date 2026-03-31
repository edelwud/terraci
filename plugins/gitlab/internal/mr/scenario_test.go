package mr

import (
	"context"
	"testing"

	gitlab "gitlab.com/gitlab-org/api/client-go"

	configpkg "github.com/edelwud/terraci/plugins/gitlab/internal/config"
)

type fakeNoteClient struct {
	hasToken bool

	notes []*gitlab.Note

	getErr    error
	createErr error
	updateErr error

	createdProjectID string
	createdMRIID     int64
	createdBody      string

	updatedProjectID string
	updatedMRIID     int64
	updatedNoteID    int64
	updatedBody      string
}

func (f *fakeNoteClient) HasToken() bool {
	return f.hasToken
}

func (f *fakeNoteClient) GetMRNotes(_ string, _ int64) ([]*gitlab.Note, error) {
	if f.getErr != nil {
		return nil, f.getErr
	}
	return f.notes, nil
}

func (f *fakeNoteClient) CreateMRNote(projectID string, mrIID int64, body string) (*gitlab.Note, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	f.createdProjectID = projectID
	f.createdMRIID = mrIID
	f.createdBody = body
	return &gitlab.Note{ID: 1, Body: body}, nil
}

func (f *fakeNoteClient) UpdateMRNote(projectID string, mrIID, noteID int64, body string) (*gitlab.Note, error) {
	if f.updateErr != nil {
		return nil, f.updateErr
	}
	f.updatedProjectID = projectID
	f.updatedMRIID = mrIID
	f.updatedNoteID = noteID
	f.updatedBody = body
	return &gitlab.Note{ID: noteID, Body: body}, nil
}

type serviceScenario struct {
	t       *testing.T
	service *Service
	client  *fakeNoteClient
}

func newServiceScenario(t *testing.T) *serviceScenario {
	t.Helper()
	client := &fakeNoteClient{hasToken: true}
	svc := NewService(nil, client, &Context{
		InMR:      true,
		ProjectID: "123",
		MRIID:     1,
	})
	return &serviceScenario{
		t:       t,
		service: svc,
		client:  client,
	}
}

func (s *serviceScenario) withContext(ctx *Context) *serviceScenario {
	s.t.Helper()
	s.service.context = ctx
	return s
}

func (s *serviceScenario) withConfig(cfg *configpkg.MRConfig) *serviceScenario {
	s.t.Helper()
	s.service.config = cfg
	return s
}

func (s *serviceScenario) withToken(hasToken bool) *serviceScenario {
	s.t.Helper()
	s.client.hasToken = hasToken
	return s
}

func (s *serviceScenario) withNotes(notes ...*gitlab.Note) *serviceScenario {
	s.t.Helper()
	s.client.notes = notes
	return s
}

func (s *serviceScenario) withGetError(err error) *serviceScenario {
	s.t.Helper()
	s.client.getErr = err
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
