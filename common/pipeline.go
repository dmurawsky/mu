package common

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/codepipeline"
	"github.com/aws/aws-sdk-go/service/codepipeline/codepipelineiface"
	"strings"
)

// PipelineStateLister for getting cluster instances
type PipelineStateLister interface {
	ListState(pipelineName string) ([]*codepipeline.StageState, error)
}

// PipelineGitInfoGetter for getting the git revision
type PipelineGitInfoGetter interface {
	GetGitInfo(pipelineName string) (GitInfo, error)
}

// GitInfo represents pertinent git information
type GitInfo struct {
	revision string
	repoName string
	orgName  string
}

// PipelineManager composite of all cluster capabilities
type PipelineManager interface {
	PipelineStateLister
	PipelineGitInfoGetter
}

type codePipelineManager struct {
	codePipelineAPI codepipelineiface.CodePipelineAPI
}

func newPipelineManager(sess *session.Session) (PipelineManager, error) {
	log.Debug("Connecting to CodePipeline service")
	codePipelineAPI := codepipeline.New(sess)

	return &codePipelineManager{
		codePipelineAPI: codePipelineAPI,
	}, nil
}

// ListState get the state of the pipeline
func (cplMgr *codePipelineManager) ListState(pipelineName string) ([]*codepipeline.StageState, error) {
	cplAPI := cplMgr.codePipelineAPI

	params := &codepipeline.GetPipelineStateInput{
		Name: aws.String(pipelineName),
	}

	log.Debugf("Searching for pipeline state for pipeline named '%s'", pipelineName)

	output, err := cplAPI.GetPipelineState(params)
	if err != nil {
		return nil, err
	}

	return output.StageStates, nil
}

func (cplMgr *codePipelineManager) GetGitInfo(pipelineName string) (GitInfo, error) {
	stageStates, err := cplMgr.ListState(pipelineName)
	if err != nil {
		return GitInfo{}, err
	}

	for _, stageState := range stageStates {
		for _, actionState := range stageState.ActionStates {
			if aws.StringValue(actionState.ActionName) == "Source" {
				cloneURL := *actionState.EntityUrl
				parts := strings.Split(cloneURL, "/")
				return GitInfo{*actionState.CurrentRevision.RevisionId, parts[4], parts[3]}, nil
			}
		}
	}

	return GitInfo{}, fmt.Errorf("Can not obtain git information from CodePipeline: %s", pipelineName)
}
