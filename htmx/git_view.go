package htmx

import (
	"context"
	"fmt"

	gitview "github.com/dkotik/cuebook/htmx/git"
)

type gitChangesRequest struct{}

func (*gitChangesRequest) Validate(context.Context) error { return nil }

type gitChangesResponse struct {
	pageData
}

func (a *handler) changes(_ context.Context, _ *gitChangesRequest) (gitChangesResponse, error) {
	if a.gitFileLinker == nil {
		return gitChangesResponse{}, ErrGitChangesViewNotConfigured
	}
	changes, err := gitview.New(a.gitSource, gitview.WithFileLinker(a.gitFileLinker))
	if err != nil {
		return gitChangesResponse{}, fmt.Errorf("Unable to load Git changes: %w", err)
	}

	data, err := a.loadPage("", "")
	if err != nil {
		if data.Error != "" {
			return gitChangesResponse{}, err
		}
		return gitChangesResponse{}, ErrUnableToLoadGitChangesView
	}
	data.GitSelected = true
	data.GitLostLines = changes.LostLines()
	data.GitChanges = changes.Changes()
	return gitChangesResponse{pageData: data}, nil
}
