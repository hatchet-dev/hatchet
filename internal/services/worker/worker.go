package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"
	"sync"

	"golang.org/x/sync/errgroup"

	"github.com/hatchet-dev/hatchet/internal/encryption"
	"github.com/hatchet-dev/hatchet/internal/integrations/vcs"
	"github.com/hatchet-dev/hatchet/internal/integrations/vcs/vcsutils"
	"github.com/hatchet-dev/hatchet/internal/repository"
	"github.com/hatchet-dev/hatchet/pkg/client"
	"github.com/hatchet-dev/hatchet/pkg/worker"
)

const (
	PullRequestWorkflow string = "create-pull-request"
	StartPullRequest    string = "pull_request:start"
)

type StartPullRequestEvent struct {
	TenantID   string `json:"tenant_id"`
	StepRunID  string `json:"step_run_id"`
	BranchName string `json:"branch_name"`
}

type WorkerOpt func(*WorkerOpts)

type WorkerOpts struct {
	client client.Client

	repo         repository.Repository
	vcsProviders map[vcs.VCSRepositoryKind]vcs.VCSProvider
}

func defaultWorkerOpts() *WorkerOpts {
	return &WorkerOpts{}
}

func WithRepository(r repository.Repository) WorkerOpt {
	return func(opts *WorkerOpts) {
		opts.repo = r
	}
}

func WithVCSProviders(vcsProviders map[vcs.VCSRepositoryKind]vcs.VCSProvider) WorkerOpt {
	return func(opts *WorkerOpts) {
		opts.vcsProviders = vcsProviders
	}
}

func WithClient(c client.Client) WorkerOpt {
	return func(opts *WorkerOpts) {
		opts.client = c
	}
}

type WorkerImpl struct {
	*worker.Worker

	repo         repository.Repository
	vcsProviders map[vcs.VCSRepositoryKind]vcs.VCSProvider
}

func NewWorker(fs ...WorkerOpt) (*WorkerImpl, error) {
	opts := defaultWorkerOpts()

	for _, f := range fs {
		f(opts)
	}

	if opts.repo == nil {
		return nil, fmt.Errorf("repository is required. use WithRepository")
	}

	if opts.client == nil {
		return nil, fmt.Errorf("client is required. use WithClient")
	}

	hatchetWorker, err := worker.NewWorker(
		worker.WithClient(opts.client),
	)

	if err != nil {
		return nil, fmt.Errorf("could not create worker: %w", err)
	}

	return &WorkerImpl{
		Worker:       hatchetWorker,
		repo:         opts.repo,
		vcsProviders: opts.vcsProviders,
	}, nil
}

func (w *WorkerImpl) Start(ctx context.Context) error {
	err := w.On(
		worker.Event(StartPullRequest),
		&worker.WorkflowJob{
			Name:        PullRequestWorkflow,
			Description: "Workflow that creates a new pull request.",
			Timeout:     "60s",
			Steps: []*worker.WorkflowStep{
				worker.Fn(w.handleStartPullRequest).SetName("start-pull-request"),
			},
		},
	)

	if err != nil {
		return fmt.Errorf("could not register workflow: %w", err)
	}

	// start the worker
	return w.Worker.Start(ctx)
}

func (w *WorkerImpl) handleStartPullRequest(ctx worker.HatchetContext) error {
	var event StartPullRequestEvent

	err := ctx.WorkflowInput(&event)

	if err != nil {
		return err
	}

	stepRun, err := w.repo.StepRun().GetStepRunById(event.TenantID, event.StepRunID)

	if err != nil {
		return fmt.Errorf("could not get step run: %w", err)
	}

	workflowRun, err := w.repo.WorkflowRun().GetWorkflowRunById(event.TenantID, stepRun.JobRun().WorkflowRunID)

	if err != nil {
		return fmt.Errorf("could not get workflow run: %w", err)
	}

	// read workflow
	workflow, err := w.repo.Workflow().GetWorkflowById(workflowRun.WorkflowVersion().WorkflowID)

	if err != nil {
		return fmt.Errorf("could not get workflow: %w", err)
	}

	if deploymentConfig, ok := workflow.DeploymentConfig(); ok {
		if installationId, ok := deploymentConfig.GithubAppInstallationID(); ok && installationId != "" {
			git, err := vcs.GetVCSRepositoryFromWorkflow(w.vcsProviders, workflow)

			if err != nil {
				return fmt.Errorf("could not get VCS repository from workflow: %w", err)
			}

			callerFilepaths := map[string]string{}

			callerFilepathsJSON, _ := stepRun.CallerFiles()

			err = json.Unmarshal(callerFilepathsJSON, &callerFilepaths)

			if err != nil {
				return fmt.Errorf("could not unmarshal caller filepaths: %w", err)
			}

			callerFilepathsToRepoFiles := sync.Map{}

			var errGroup errgroup.Group

			for _, file := range callerFilepaths {
				fileCp := file

				errGroup.Go(func() error {
					gitBranch, _ := stepRun.GitRepoBranch()

					foundSearchPath, fileReader, err := searchForFile(fileCp, git, gitBranch)

					if err != nil {
						return fmt.Errorf("could not search for file: %w", err)
					}

					if fileReader != nil {
						fileBytes, err := io.ReadAll(fileReader)

						if err != nil {
							return fmt.Errorf("could not read file: %w", err)
						}

						callerFilepathsToRepoFiles.Store(foundSearchPath, fileBytes)
					}

					return nil
				})
			}

			if err = errGroup.Wait(); err != nil {
				return fmt.Errorf("could not search for files: %w", err)
			}

			diffs, _, err := vcsutils.GetStepRunOverrideDiffs(w.repo.StepRun(), stepRun)

			if err != nil {
				return fmt.Errorf("could not get step run override diffs: %w", err)
			}

			newFiles := make(map[string][]byte)

			callerFilepathsToRepoFiles.Range(func(key, value interface{}) bool {
				fileBytes := value.([]byte)

				for diffKey, diffValue := range diffs {
					fileBytes, err = findAndReplace(fileBytes, diffKey, diffValue)

					if err != nil {
						return true
					}
				}

				newFiles[key.(string)] = fileBytes

				return true
			})

			prSuffix, err := encryption.GenerateRandomBytes(4)

			if err != nil {
				return fmt.Errorf("could not generate random bytes: %w", err)
			}

			var baseBranch *string

			if event.BranchName != "" {
				baseBranch = &event.BranchName
			}

			// create the pull request
			_, err = git.CreateOrUpdatePullRequest(stepRun.TenantID, workflowRun.ID, &vcs.CreatePullRequestOpts{
				GitRepoOwner:   git.GetRepoOwner(),
				GitRepoName:    git.GetRepoName(),
				BaseBranch:     baseBranch,
				Files:          newFiles,
				Title:          fmt.Sprintf("[Hatchet] Update %s", workflow.Name),
				HeadBranchName: fmt.Sprintf("hatchet/pr-%s", prSuffix),
			})

			if err != nil {
				return fmt.Errorf("could not create pull request: %w", err)
			}
		}
	}

	return nil
}

func searchForFile(targetPath string, vcs vcs.VCSRepository, ref string) (string, io.ReadCloser, error) {
	if !filepath.IsAbs(targetPath) {
		return "", nil, fmt.Errorf("filepath must be absolute")
	}

	searchPaths := getSearchPaths(targetPath)

	var res io.ReadCloser
	var foundSearchPath string

	for _, searchPath := range searchPaths {
		searchPathCp := searchPath
		file, err := vcs.ReadFile(ref, searchPathCp)

		if err == nil {
			foundSearchPath = searchPathCp
			res = file
		}
	}

	return foundSearchPath, res, nil
}

// getSearchPaths returns paths from the most specific path to the least specific in the path. For
// example, if the caller file is under `/usr/local/hatchet/src/worker.py`, and the file is under ./src/worker.py
// in the repository, it will search in this order:
// - `/usr/local/hatchet/src/worker.py`
// - `/local/hatchet/src/worker.py`
// - `/hatchet/src/worker.py`
// - `/src/worker.py`
func getSearchPaths(targetPath string) []string {
	base := filepath.Base(targetPath)
	searchBases := []string{}

	if base != "" && base != "/" && base != "." && base != ".." {
		searchBases = append(searchBases, base)
	}

	currDir := targetPath

	for {
		currDir = filepath.Dir(currDir)

		base := filepath.Base(currDir)

		if base != "" && base != "/" && base != "." && base != ".." {
			searchBases = append([]string{base}, searchBases...)
		}

		if currDir == "." || currDir == "/" {
			break
		}
	}

	searchPaths := []string{}

	// construct search bases in reverse order
	for i := range searchBases {
		searchPath := strings.Join(searchBases[i:], "/")

		if searchPath != "" {
			searchPaths = append(searchPaths, searchPath)
		}

	}

	return searchPaths
}

func searchWithRegex(input []byte, pattern string, replacements map[string]string) (string, error) {
	// Compile the regular expression with named capturing groups
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", err
	}

	// Convert input to string for easier manipulation and because we work with indexes
	inputStr := string(input)
	offset := 0 // Keep track of the offset caused by replacements

	// Find all matches
	matches := re.FindAllStringSubmatchIndex(inputStr, -1)
	for _, match := range matches {
		// Process each named group for replacement
		for i := 2; i < len(match); i += 2 {
			// Adjust group indexes by offset
			groupStart, groupEnd := match[i]+offset, match[i+1]+offset
			if groupStart == -1 || groupEnd == -1 {
				continue // Skip unmatched groups
			}

			name := re.SubexpNames()[i/2] // Get the name of the current capturing group
			if replacement, ok := replacements[name]; ok && name != "" {
				// Perform the replacement in the input string
				before := inputStr[:groupStart]
				after := inputStr[groupEnd:]
				inputStr = before + replacement + after

				// Update the offset
				offset += len(replacement) - (groupEnd - groupStart)
			}
		}
	}

	return inputStr, nil
}

func getPatternForValue(value string) string {
	return `(?P<instance>\w+)\.override\(\s*(?P<param1>('|"|""")\s*` + value + `\s*('|"|"""))\s*,\s*(?P<override>[\s\S]*?)\s*\)`
}

func findAndReplace(input []byte, value, override string) ([]byte, error) {
	pattern := getPatternForValue(value)

	replacements := map[string]string{
		"override": override,
	}

	modifiedInput, err := searchWithRegex(input, pattern, replacements)
	if err != nil {
		return nil, err
	}

	return []byte(modifiedInput), nil
}
