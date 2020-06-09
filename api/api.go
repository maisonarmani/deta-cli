package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
)

// injects X-Resource-Addr header from account and region
func (c *DetaClient) injectResourceHeader(headers map[string]string, account, region string) {
	resAddr := fmt.Sprintf("aws:%s:%s", account, region)
	encoded := base64.StdEncoding.EncodeToString([]byte(resAddr))
	headers["X-Resource-Addr"] = encoded
}

// DeployRequest deploy program request
type DeployRequest struct {
	ProgramID string            `json:"pid"`
	Changes   map[string]string `json:"change"`
	Deletions []string          `json:"delete"`
	Account   string            `json:"-"`
	Region    string            `json:"-"`
}

// DeployResponse deploy program response
type DeployResponse struct {
	ProgramID string `json:"program_id"`
}

// Deploy sends deploy request
func (c *DetaClient) Deploy(r *DeployRequest) (*DeployResponse, error) {
	headers := make(map[string]string)
	c.injectResourceHeader(headers, r.Account, r.Region)

	i := &requestInput{
		Path:      fmt.Sprintf("/%s/", patcherPath),
		Method:    "POST",
		Headers:   headers,
		Body:      r,
		NeedsAuth: true,
	}
	o, err := c.request(i)
	if err != nil {
		return nil, err
	}
	if o.Status != 200 {
		msg := o.Error.Message
		if msg == "" {
			msg = o.Error.Errors[0]
		}
		return nil, fmt.Errorf("failed to deploy: %v", msg)
	}

	var resp DeployResponse
	err = json.Unmarshal(o.Body, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to deploy: %v", err)
	}
	return &resp, nil
}

// NewProgramRequest request to create a new program
type NewProgramRequest struct {
	Space   int64   `json:"spaceID"`
	Group   string  `json:"group"`
	Name    string  `json:"name"`
	Runtime string  `json:"runtime"`
	Fork    *string `json:"fork"`
}

// NewProgramResponse response to a new program request
type NewProgramResponse struct {
	ID      string   `json:"id"`
	Space   int64    `json:"space"`
	Runtime string   `json:"runtime"`
	Name    string   `json:"name"`
	Path    string   `json:"path"`
	Project string   `json:"project"`
	Account string   `json:"account"`
	Region  string   `json:"region"`
	Deps    []string `json:"deps"`
	Envs    []string `json:"envs"`
	Public  bool     `json:"http_auth"`
}

// NewProgram sends a new program request
func (c *DetaClient) NewProgram(r *NewProgramRequest) (*NewProgramResponse, error) {
	i := &requestInput{
		Path:      fmt.Sprintf("/%s/", "programs"),
		Method:    "POST",
		NeedsAuth: true,
		Body:      *r,
	}

	o, err := c.request(i)
	if err != nil {
		return nil, err
	}

	if o.Status != 200 {
		msg := o.Error.Message
		if msg == "" {
			msg = o.Error.Errors[0]
		}
		return nil, fmt.Errorf("failed to create new program: %v", msg)
	}

	var resp NewProgramResponse
	err = json.Unmarshal(o.Body, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to create new program: %v", err)
	}
	return &resp, nil
}

// ViewProgramRequest request to view an existing program
type ViewProgramRequest struct {
	ProgramID string
	Runtime   string
	Account   string
	Region    string
}

// ViewProgramResponse response to view program request
type ViewProgramResponse struct {
	Entrypoint string   `json:"file"`
	Contents   string   `json:"contents"`
	FileTree   []string `json:"tree"`
}

// ViewProgram sends a view program request
// The response contains the contents of the entrypoint file and the file tree
func (c *DetaClient) ViewProgram(r *ViewProgramRequest) (*ViewProgramResponse, error) {
	headers := make(map[string]string)
	c.injectResourceHeader(headers, r.Account, r.Region)

	queryParams := map[string]string{
		"runtime": r.Runtime,
	}

	i := &requestInput{
		Path:        fmt.Sprintf("/%s/%s", viewerPath, r.ProgramID),
		Method:      "GET",
		Headers:     headers,
		QueryParams: queryParams,
		NeedsAuth:   true,
	}

	o, err := c.request(i)
	if err != nil {
		return nil, err
	}

	if o.Status != 200 {
		msg := o.Error.Message
		if msg == "" {
			msg = o.Error.Errors[0]
		}
		return nil, fmt.Errorf("failed to view program: %v", msg)
	}

	var resp ViewProgramResponse
	err = json.Unmarshal(o.Body, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to view program: %v", err)
	}
	return &resp, nil
}

// ViewProgramFileRequest view program file request
type ViewProgramFileRequest struct {
	ProgramID string
	Filepath  string
	Account   string
	Region    string
}

// ViewProgramFileResponse view program file response
type ViewProgramFileResponse string

// ViewProgramFile view a particular file of the program
func (c *DetaClient) ViewProgramFile(r *ViewProgramFileRequest) (*ViewProgramFileResponse, error) {
	headers := make(map[string]string)
	c.injectResourceHeader(headers, r.Account, r.Region)

	queryParams := map[string]string{
		"path": r.Filepath,
	}

	i := &requestInput{
		Path:        fmt.Sprintf("/%s/file/%s", viewerPath, r.ProgramID),
		Method:      "GET",
		Headers:     headers,
		QueryParams: queryParams,
		NeedsAuth:   true,
	}

	o, err := c.request(i)
	if err != nil {
		return nil, err
	}

	if o.Status != 200 {
		msg := o.Error.Message
		if msg == "" {
			msg = o.Error.Errors[0]
		}
		return nil, fmt.Errorf("failed to get '%s': %v", r.Filepath, msg)
	}

	var resp ViewProgramFileResponse
	err = json.Unmarshal(o.Body, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to get '%s': %v", r.Filepath, err)
	}
	return &resp, nil
}

// DownloadProgramRequest download program request
type DownloadProgramRequest struct {
	ProgramID string
	Runtime   string
	Account   string
	Region    string
}

// DownloadProgramResponse download program response
type DownloadProgramResponse struct {
	Files map[string]string
}

// DownloadProgram download all program files
func (c *DetaClient) DownloadProgram(req *DownloadProgramRequest) (*DownloadProgramResponse, error) {
	progFiles := make(map[string]string)

	viewProgReq := &ViewProgramRequest{
		ProgramID: req.ProgramID,
		Runtime:   req.Runtime,
		Account:   req.Account,
		Region:    req.Region,
	}
	o, err := c.ViewProgram(viewProgReq)
	if err != nil {
		return nil, err
	}

	progFiles[o.Entrypoint] = o.Contents

	for _, file := range o.FileTree {
		if file != o.Entrypoint {
			if !strings.HasSuffix(file, "/") {
				viewProgFileReq := &ViewProgramFileRequest{
					ProgramID: req.ProgramID,
					Filepath:  file,
					Account:   req.ProgramID,
					Region:    req.Region,
				}
				contents, err := c.ViewProgramFile(viewProgFileReq)
				if err != nil {
					return nil, err
				}
				progFiles[file] = string(*contents)
			}
			// empty folders
			progFiles[file] = ""
		}
	}
	return &DownloadProgramResponse{Files: progFiles}, err
}

// ListSpaceItem an item in ListSpacesResponse
type ListSpaceItem struct {
	SpaceID int64  `json:"spaceID"`
	Name    string `json:"name"`
	Role    string `json:"role"`
}

// ListSpacesResponse response to list spaces request
type ListSpacesResponse []ListSpaceItem

// ListSpaces send list a spaces request
func (c *DetaClient) ListSpaces() (ListSpacesResponse, error) {
	i := &requestInput{
		Path:      fmt.Sprintf("/%s/", "spaces"),
		Method:    "GET",
		NeedsAuth: true,
	}

	o, err := c.request(i)
	if err != nil {
		return nil, err
	}

	if o.Status != 200 {
		msg := o.Error.Message
		if msg == "" {
			msg = o.Error.Errors[0]
		}
		return nil, fmt.Errorf("failed to list spaces: %v", msg)
	}
	var resp ListSpacesResponse
	err = json.Unmarshal(o.Body, &resp)
	if err != nil {
		return nil, fmt.Errorf("failed to list spaces: %v", err)
	}
	return resp, nil
}

// UpdateProgNameRequest request to update program name
type UpdateProgNameRequest struct {
	ProgramID string `json:"-"`
	Name      string `json:"name"`
}

// UpdateProgName update program name
func (c *DetaClient) UpdateProgName(req *UpdateProgNameRequest) error {

	i := &requestInput{
		Path:      fmt.Sprintf("/programs/%s", req.ProgramID),
		Method:    "PATCH",
		Body:      req,
		NeedsAuth: true,
	}

	o, err := c.request(i)
	if err != nil {
		return err
	}

	if o.Status != 200 {
		msg := o.Error.Message
		if msg == "" {
			msg = o.Error.Errors[0]
		}
		return fmt.Errorf("failed to update program name: %s", msg)
	}
	return nil
}

// UpdateProgEnvsRequest request to update program envs
type UpdateProgEnvsRequest struct {
	ProgramID string
	Account   string
	Region    string
	Vars      map[string]*string
}

// UpdateProgEnvs update program environment variables
func (c *DetaClient) UpdateProgEnvs(req *UpdateProgEnvsRequest) error {
	headers := make(map[string]string)
	c.injectResourceHeader(headers, req.Account, req.Region)

	i := &requestInput{
		Path:      fmt.Sprintf("/programs/%s/envs", req.ProgramID),
		Headers:   headers,
		NeedsAuth: true,
		Method:    "PATCH",
		Body:      req.Vars,
	}

	o, err := c.request(i)
	if err != nil {
		return err
	}

	if o.Status != 200 {
		msg := o.Error.Message
		if msg == "" {
			msg = o.Error.Errors[0]
		}
		return fmt.Errorf("failed to update env vars: %s", msg)
	}
	return nil
}

// UpdateProgDepsRequest request to update program dependencies
type UpdateProgDepsRequest struct {
	ProgramID string `json:"program_id"`
	Command   string `json:"command"`
}

// UpdateProgDepsResponse response to update program dependencies request
type UpdateProgDepsResponse struct {
	Output string `json:"output"`
}

// UpdateProgDeps update program dependencies
func (c *DetaClient) UpdateProgDeps(req *UpdateProgDepsRequest) (*UpdateProgDepsResponse, error) {
	i := &requestInput{
		Path:      fmt.Sprintf("/%s/commands", pigeonPath),
		Method:    "POST",
		NeedsAuth: true,
		Body:      req,
	}

	o, err := c.request(i)
	if err != nil {
		return nil, err
	}

	if o.Status != 200 {
		msg := o.Error.Message
		if msg == "" {
			msg = o.Error.Errors[0]
		}
		return nil, fmt.Errorf("failed to update dependencies: %s", msg)
	}

	var resp UpdateProgDepsResponse
	err = json.Unmarshal(o.Body, &resp)
	if err != nil {
		return nil, err
	}
	return &resp, nil
}