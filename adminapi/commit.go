package adminapi

import (
	"encoding/json"
	"errors"
	"fmt"
)

// commitRequest is the payload sent to /api/dataset/commit
type commitRequest struct {
	Created []Attributes `json:"created"`
	Changed []Attributes `json:"changed"`
	Deleted []int        `json:"deleted"` // the object-ids
}

type commitResponse struct {
	Status   string `json:"status"`
	CommitID int    `json:"commit_id"`
	Type     string `json:"type"`
	Message  string `json:"message"`
}

// Commit commits all changed, created, and deleted objects in a single API call.
func (s ServerObjects) Commit() (int, error) {
	commit := buildCommit(s)

	commitID, err := sendCommit(commit)
	if err != nil {
		return 0, err
	}

	for _, obj := range s {
		obj.confirmChanges()
	}

	return commitID, nil
}

// Rollback reverts all objects to their original state.
func (s ServerObjects) Rollback() {
	for _, obj := range s {
		obj.Rollback()
	}
}

// Set calls Set(key, value) on each ServerObject in the slice.
// If any Set operation fails, all errors are collected and returned
// as a joined error. This allows identifying all problematic objects
// in a single call rather than failing on the first error.
func (s ServerObjects) Set(key string, value any) error {
	var errs []error
	for i, obj := range s {
		if err := obj.Set(key, value); err != nil {
			errs = append(errs, fmt.Errorf("object %d (id=%v): %w", i, obj.Get("object_id"), err))
		}
	}
	return errors.Join(errs...)
}

// Delete calls Delete() on each ServerObject in the slice.
// This marks all objects for deletion on the next Commit().
func (s ServerObjects) Delete() {
	for _, obj := range s {
		obj.Delete()
	}
}

// Commit commits this single object's changes to the server.
func (s *ServerObject) Commit() (int, error) {
	commit := buildCommit(ServerObjects{s})
	commitID, err := sendCommit(commit)
	if err != nil {
		return 0, err
	}

	s.confirmChanges()
	return commitID, nil
}

func buildCommit(objects ServerObjects) commitRequest {
	commit := commitRequest{
		Created: []Attributes{},
		Changed: []Attributes{},
		Deleted: []int{}, // the object-ids
	}

	for _, obj := range objects {
		switch obj.CommitState() {
		case StateCreated:
			commit.Created = append(commit.Created, obj.attributes)
		case StateChanged:
			commit.Changed = append(commit.Changed, obj.serializeChanges())
		case StateDeleted:
			commit.Deleted = append(commit.Deleted, obj.ObjectID())
		case StateConsistent:
			// No changes to commit
		}
	}

	return commit
}

func sendCommit(commit commitRequest) (int, error) {
	resp, err := sendRequest(apiEndpointCommit, commit)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	var result commitResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("failed to decode commit response: %w", err)
	}

	if result.Status == "error" {
		return 0, fmt.Errorf("commit failed: %s", result.Message)
	}

	return result.CommitID, nil
}
