package htmx

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	gitview "github.com/dkotik/cuebook/htmx/git"
)

type gitChangesRequest struct{}

func (*gitChangesRequest) Validate(context.Context) error { return nil }

type gitChangesResponse struct {
	pageData
}

func (a *handler) changes(_ context.Context, _ *gitChangesRequest) (gitChangesResponse, error) {
	if a.gitFileLinker == nil {
		return gitChangesResponse{}, errors.New("The Git changes view is not configured.")
	}
	changes, err := gitview.New(a.gitSource, gitview.WithFileLinker(a.gitFileLinker))
	if err != nil {
		return gitChangesResponse{}, fmt.Errorf("Unable to load Git changes: %w", err)
	}

	data, status := a.loadPage("", "")
	if status != http.StatusOK {
		if data.Error != "" {
			return gitChangesResponse{}, errors.New(data.Error)
		}
		return gitChangesResponse{}, errors.New("Unable to load the Git changes view.")
	}
	data.GitSelected = true
	data.GitChanges = changes.Changes()
	return gitChangesResponse{pageData: data}, nil
}
