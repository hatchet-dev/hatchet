package vcs

import (
	"fmt"
	"io"
	"net/url"

	"github.com/hatchet-dev/hatchet/pkg/repository/prisma/db"
)

type VCSRepositoryKind string

const (
	// enumeration of in-tree repository kinds
	VCSRepositoryKindGithub VCSRepositoryKind = "github"
	VCSRepositoryKindGitlab VCSRepositoryKind = "gitlab"
)

type VCSCheckRunStatus string

const (
	VCSCheckRunStatusQueued     VCSCheckRunStatus = "queued"
	VCSCheckRunStatusInProgress VCSCheckRunStatus = "in_progress"
	VCSCheckRunStatusCompleted  VCSCheckRunStatus = "completed"
)

type VCSCheckRunConclusion string

const (
	VCSCheckRunConclusionSuccess        VCSCheckRunConclusion = "success"
	VCSCheckRunConclusionFailure        VCSCheckRunConclusion = "failure"
	VCSCheckRunConclusionCancelled      VCSCheckRunConclusion = "cancelled"
	VCSCheckRunConclusionSkipped        VCSCheckRunConclusion = "skipped"
	VCSCheckRunConclusionTimedOut       VCSCheckRunConclusion = "timed_out"
	VCSCheckRunConclusionActionRequired VCSCheckRunConclusion = "action_required"
)

type VCSCheckRun struct {
	Name       string
	Status     VCSCheckRunStatus
	Conclusion VCSCheckRunConclusion
}

type DirectoryItem struct {
	Type string `json:"type"`
	Name string `json:"name"`
}

type CreatePullRequestOpts struct {
	GitRepoOwner string
	GitRepoName  string

	// (optional) the base branch. If not specified, the default branch is used
	BaseBranch     *string
	Title          string
	HeadBranchName string
	Files          map[string][]byte
}

// VCSRepository provides an interface to implement for new version control providers
// (github, gitlab, etc)
type VCSRepository interface {
	// GetKind returns the kind of VCS provider -- used for downstream integrations
	GetKind() VCSRepositoryKind

	// GetRepoOwner returns the owner of the repository
	GetRepoOwner() string

	// GetRepoName returns the name of the repository
	GetRepoName() string

	// SetupRepository sets up a VCS repository on Hatchet.
	SetupRepository(tenantId string) error

	// GetArchiveLink returns an archive link for a specific repo SHA
	GetArchiveLink(ref string) (*url.URL, error)

	// GetBranch gets a full branch (name and sha)
	GetBranch(name string) (VCSBranch, error)

	// CreateOrUpdatePullRequest creates a new pull request or updates an existing one
	CreateOrUpdatePullRequest(tenantId, workflowRunId string, opts *CreatePullRequestOpts) (*db.GithubPullRequestModel, error)

	// ReadFile returns a file by a SHA reference or path
	ReadFile(ref, path string) (io.ReadCloser, error)

	// ReadDirectory returns a list of directory items
	ReadDirectory(ref, path string) ([]DirectoryItem, error)

	// CompareCommits compares a base commit with a head commit
	CompareCommits(base, head string) (VCSCommitsComparison, error)
}

// VCSObjectID is a generic method for retrieving IDs from the underlying VCS repository.
// Depending on the provider, object IDs may be int64 or strings.
//
// Object ids are meant to be passed between methods in the same VCS repository, so they should
// only be read by the same VCS repository that wrote them.
type VCSObjectID interface {
	GetIDString() *string
	GetIDInt64() *int64
}

type vcsObjectString struct {
	id string
}

func NewVCSObjectString(id string) VCSObjectID {
	return vcsObjectString{id}
}

func (v vcsObjectString) GetIDString() *string {
	return &v.id
}

func (v vcsObjectString) GetIDInt64() *int64 {
	return nil
}

type vcsObjectInt struct {
	id int64
}

func NewVCSObjectInt(id int64) VCSObjectID {
	return vcsObjectInt{id}
}

func (v vcsObjectInt) GetIDString() *string {
	return nil
}

func (v vcsObjectInt) GetIDInt64() *int64 {
	return &v.id
}

type VCSProvider interface {
	// GetVCSRepositoryFromModule returns the corresponding VCS repository for the module.
	// Callers should likely use the package method GetVCSProviderFromModule.
	GetVCSRepositoryFromWorkflow(workflow *db.WorkflowModel) (VCSRepository, error)
}

// GetVCSRepositoryFromWorkflow returns the corresponding VCS repository for the workflow
func GetVCSRepositoryFromWorkflow(allProviders map[VCSRepositoryKind]VCSProvider, workflow *db.WorkflowModel) (VCSRepository, error) {
	var repoKind VCSRepositoryKind

	if deploymentConf, ok := workflow.DeploymentConfig(); ok {
		if installationId, ok := deploymentConf.GithubAppInstallationID(); ok && installationId != "" {
			repoKind = VCSRepositoryKindGithub
		}
	}

	provider, exists := allProviders[repoKind]

	if !exists {
		return nil, fmt.Errorf("VCS provider kind '%s' is not registered on this Hatchet instance", repoKind)
	}

	return provider.GetVCSRepositoryFromWorkflow(workflow)
}

// VCSRepositoryPullRequest abstracts the underlying pull or merge request methods to only
// extract relevant information.
type VCSRepositoryPullRequest interface {
	GetRepoOwner() string
	GetVCSID() VCSObjectID
	GetPRNumber() int64
	GetRepoName() string
	GetBaseSHA() string
	GetHeadSHA() string
	GetBaseBranch() string
	GetHeadBranch() string
	GetTitle() string
	GetState() string
}

type VCSBranch interface {
	GetName() string
	GetLatestRef() string
}

type VCSCommitsComparison interface {
	GetFiles() []CommitFile
}

type CommitFile struct {
	Name string
}

func (f CommitFile) GetFilename() string {
	return f.Name
}
