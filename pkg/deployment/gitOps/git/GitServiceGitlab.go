/*
 * Copyright (c) 2024. Devtron Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package git

import (
	"context"
	"crypto/tls"
	"fmt"
	"github.com/devtron-labs/common-lib/utils/retryFunc"
	bean2 "github.com/devtron-labs/devtron/api/bean/gitOps"
	"github.com/devtron-labs/devtron/pkg/deployment/gitOps/git/bean"
	"github.com/devtron-labs/devtron/util"
	_ "github.com/hashicorp/go-retryablehttp"
	"github.com/xanzy/go-gitlab"
	"go.uber.org/zap"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"
	"time"
)

type GitLabClient struct {
	client       *gitlab.Client
	config       *bean.GitConfig
	logger       *zap.SugaredLogger
	gitOpsHelper *GitOpsHelper
}

func NewGitLabClient(config *bean.GitConfig, logger *zap.SugaredLogger, gitOpsHelper *GitOpsHelper, tlsConfig *tls.Config) (GitOpsClient, error) {
	gitLabClient, err := CreateGitlabClient(config.GitHost, config.GitToken, tlsConfig)
	if err != nil {
		logger.Errorw("error in creating gitlab client", "err", err)
		return nil, err
	}
	gitlabGroupId := ""
	if len(config.GitlabGroupId) > 0 {
		if _, err := strconv.Atoi(config.GitlabGroupId); err == nil {
			gitlabGroupId = config.GitlabGroupId
		} else {
			groups, res, err := gitLabClient.Groups.SearchGroup(config.GitlabGroupId)
			if err != nil {
				responseStatus := 0
				if res != nil {
					responseStatus = res.StatusCode
				}
				logger.Warnw("error connecting to gitlab", "status code", responseStatus, "err", err.Error())
			}
			logger.Debugw("gitlab groups found ", "group", groups)
			if len(groups) == 0 {
				logger.Warn("no matching namespace found for gitlab")
			}
			for _, group := range groups {
				if config.GitlabGroupId == group.Name {
					gitlabGroupId = strconv.Itoa(group.ID)
				}
			}
		}
	} else {
		return nil, fmt.Errorf("no gitlab group id found")
	}
	if gitlabGroupId == "" {
		return nil, fmt.Errorf("no gitlab group id found")
	}
	group, _, err := gitLabClient.Groups.GetGroup(gitlabGroupId, &gitlab.GetGroupOptions{})
	if err != nil {
		return nil, err
	}
	if group != nil {
		config.GitlabGroupPath = group.FullPath
	}
	logger.Debugw("gitlab config", "config", config)
	return &GitLabClient{
		client:       gitLabClient,
		config:       config,
		logger:       logger,
		gitOpsHelper: gitOpsHelper,
	}, nil
}

func CreateGitlabClient(host, token string, tlsConfig *tls.Config) (*gitlab.Client, error) {
	var gitLabClient *gitlab.Client
	var err error
	options := make([]gitlab.ClientOptionFunc, 0)

	if len(host) > 0 {
		_, err = url.ParseRequestURI(host)
		if err != nil {
			return nil, err
		}
		options = append(options, gitlab.WithBaseURL(host))
	}
	if tlsConfig != nil {
		httpClient := util.GetHTTPClientWithTLSConfig(tlsConfig)
		options = append(options, gitlab.WithHTTPClient(httpClient))
	}
	gitLabClient, err = gitlab.NewClient(token, options...)
	if err != nil {
		return nil, err
	}
	return gitLabClient, err
}

func (impl GitLabClient) DeleteRepository(config *bean2.GitOpsConfigDto) (err error) {
	start := time.Now()
	defer func() {
		util.TriggerGitOpsMetrics("DeleteRepository", "GitLabClient", start, err)
	}()

	err = impl.DeleteProject(config.GitRepoName)
	if err != nil {
		impl.logger.Errorw("error in deleting repo gitlab", "project", config.GitRepoName, "err", err)
	}
	return err
}

func (impl GitLabClient) CreateRepository(ctx context.Context, config *bean2.GitOpsConfigDto) (url string, isNew bool, isEmpty bool, detailedErrorGitOpsConfigActions DetailedErrorGitOpsConfigActions) {

	var (
		err     error
		repoUrl string
	)
	start := time.Now()

	detailedErrorGitOpsConfigActions.StageErrorMap = make(map[string]error)
	impl.logger.Debugw("gitlab app create request ", "name", config.GitRepoName, "description", config.Description)
	repoUrl, isEmpty, err = impl.GetRepoUrl(config)
	if err != nil {
		impl.logger.Errorw("error in getting repo url ", "gitlab project", config.GitRepoName, "err", err)
		detailedErrorGitOpsConfigActions.StageErrorMap[bean.GetRepoUrlStage] = err
		util.TriggerGitOpsMetrics("CreateRepository", "GitLabClient", start, err)
		return "", false, isEmpty, detailedErrorGitOpsConfigActions
	}
	if len(repoUrl) > 0 {
		detailedErrorGitOpsConfigActions.SuccessfulStages = append(detailedErrorGitOpsConfigActions.SuccessfulStages, bean.GetRepoUrlStage)
		util.TriggerGitOpsMetrics("CreateRepository", "GitLabClient", start, nil)
		return repoUrl, false, isEmpty, detailedErrorGitOpsConfigActions
	} else {
		url, err = impl.createProject(config.GitRepoName, config.Description)
		if err != nil {
			detailedErrorGitOpsConfigActions.StageErrorMap[bean.CreateRepoStage] = err
			repoUrl, isEmpty, err = impl.GetRepoUrl(config)
			if err != nil {
				impl.logger.Errorw("error in getting repo url ", "gitlab project", config.GitRepoName, "err", err)
			}
			if err != nil || len(repoUrl) == 0 {
				util.TriggerGitOpsMetrics("CreateRepository", "GitLabClient", start, err)
				return "", true, isEmpty, detailedErrorGitOpsConfigActions
			}
		}
		detailedErrorGitOpsConfigActions.SuccessfulStages = append(detailedErrorGitOpsConfigActions.SuccessfulStages, bean.CreateRepoStage)
	}
	repoUrl = url
	validated, err := impl.ensureProjectAvailability(config.GitRepoName)
	if err != nil {
		impl.logger.Errorw("error in ensuring project availability ", "gitlab project", config.GitRepoName, "err", err)
		detailedErrorGitOpsConfigActions.StageErrorMap[bean.CloneHttpStage] = err
		util.TriggerGitOpsMetrics("CreateRepository", "GitLabClient", start, err)
		return "", true, isEmpty, detailedErrorGitOpsConfigActions
	}
	if !validated {
		err = fmt.Errorf("unable to validate project:%s in given time", config.GitRepoName)
		detailedErrorGitOpsConfigActions.StageErrorMap[bean.CloneHttpStage] = err
		util.TriggerGitOpsMetrics("CreateRepository", "GitLabClient", start, err)
		return "", true, isEmpty, detailedErrorGitOpsConfigActions
	}
	detailedErrorGitOpsConfigActions.SuccessfulStages = append(detailedErrorGitOpsConfigActions.SuccessfulStages, bean.CloneHttpStage)
	_, err = impl.CreateReadme(ctx, config)
	if err != nil {
		impl.logger.Errorw("error in creating readme ", "gitlab project", config.GitRepoName, "err", err)
		detailedErrorGitOpsConfigActions.StageErrorMap[bean.CreateReadmeStage] = err
		util.TriggerGitOpsMetrics("CreateRepository", "GitLabClient", start, err)
		return "", true, isEmpty, detailedErrorGitOpsConfigActions
	}
	isEmpty = false //As we have created readme, repo is no longer empty
	detailedErrorGitOpsConfigActions.SuccessfulStages = append(detailedErrorGitOpsConfigActions.SuccessfulStages, bean.CreateReadmeStage)
	validated, err = impl.ensureProjectAvailabilityOnSsh(config.GitRepoName, repoUrl, config.TargetRevision)
	if err != nil {
		impl.logger.Errorw("error in ensuring project availability ", "gitlab project", config.GitRepoName, "err", err)
		detailedErrorGitOpsConfigActions.StageErrorMap[bean.CloneSshStage] = err
		util.TriggerGitOpsMetrics("CreateRepository", "GitLabClient", start, err)
		return "", true, isEmpty, detailedErrorGitOpsConfigActions
	}
	if !validated {
		err = fmt.Errorf("unable to validate project:%s in given time", config.GitRepoName)
		detailedErrorGitOpsConfigActions.StageErrorMap[bean.CloneSshStage] = err
		util.TriggerGitOpsMetrics("CreateRepository", "GitLabClient", start, err)
		return "", true, isEmpty, detailedErrorGitOpsConfigActions
	}
	detailedErrorGitOpsConfigActions.SuccessfulStages = append(detailedErrorGitOpsConfigActions.SuccessfulStages, bean.CloneSshStage)
	util.TriggerGitOpsMetrics("CreateRepository", "GitLabClient", start, nil)
	return url, true, isEmpty, detailedErrorGitOpsConfigActions
}

func (impl GitLabClient) DeleteProject(projectName string) (err error) {
	start := time.Now()
	defer func() {
		util.TriggerGitOpsMetrics("DeleteProject", "GitLabClient", start, err)
	}()

	impl.logger.Infow("deleting project ", "gitlab project name", projectName)
	_, err = impl.client.Projects.DeleteProject(fmt.Sprintf("%s/%s", impl.config.GitlabGroupPath, projectName), nil)
	return err
}
func (impl GitLabClient) createProject(name, description string) (url string, err error) {
	start := time.Now()
	defer func() {
		util.TriggerGitOpsMetrics("createProject", "GitLabClient", start, err)
	}()

	var namespace = impl.config.GitlabGroupId
	namespaceId, err := strconv.Atoi(namespace)
	if err != nil {
		return "", err
	}

	// Create new project
	p := &gitlab.CreateProjectOptions{
		Name:                 gitlab.String(name),
		Description:          gitlab.String(description),
		MergeRequestsEnabled: gitlab.Bool(true),
		SnippetsEnabled:      gitlab.Bool(false),
		Visibility:           gitlab.Visibility(gitlab.PrivateVisibility),
		NamespaceID:          &namespaceId,
	}
	project, _, err := impl.client.Projects.CreateProject(p)
	if err != nil {
		impl.logger.Errorw("err in creating gitlab app", "req", p, "name", name, "err", err)
		return "", err
	}
	impl.logger.Infow("gitlab app created", "name", name, "url", project.HTTPURLToRepo)
	return project.HTTPURLToRepo, nil
}

func (impl GitLabClient) ensureProjectAvailability(projectName string) (bool, error) {

	var err error
	start := time.Now()
	defer func() {
		util.TriggerGitOpsMetrics("ensureProjectAvailability", "GitLabClient", start, err)
	}()

	pid := fmt.Sprintf("%s/%s", impl.config.GitlabGroupPath, projectName)
	count := 0
	verified := false
	for count < 3 && !verified {
		count = count + 1
		_, res, err := impl.client.Projects.GetProject(pid, &gitlab.GetProjectOptions{})
		if err != nil {
			return verified, err
		}
		if res.StatusCode >= 200 && res.StatusCode <= 299 {
			verified = true
			return verified, nil
		}
		time.Sleep(10 * time.Second)
	}
	return false, nil
}

func (impl GitLabClient) ensureProjectAvailabilityOnSsh(projectName string, repoUrl, targetRevision string) (bool, error) {
	var err error
	start := time.Now()
	defer func() {
		util.TriggerGitOpsMetrics("ensureProjectAvailabilityOnSsh", "GitLabClient", start, err)
	}()

	count := 0
	for count < 3 {
		count = count + 1
		_, err := impl.gitOpsHelper.Clone(repoUrl, fmt.Sprintf("/ensure-clone/%s", projectName), targetRevision)
		if err == nil {
			impl.logger.Infow("gitlab ensureProjectAvailability clone passed", "try count", count, "repoUrl", repoUrl)
			return true, nil
		}
		if err != nil {
			impl.logger.Errorw("gitlab ensureProjectAvailability clone failed", "try count", count, "err", err)
		}
		time.Sleep(10 * time.Second)
	}
	return false, nil
}

func (impl GitLabClient) GetRepoUrl(config *bean2.GitOpsConfigDto) (repoUrl string, isRepoEmpty bool, err error) {

	start := time.Now()
	defer func() {
		util.TriggerGitOpsMetrics("GetRepoUrl", "GitLabClient", start, err)
	}()

	pid := fmt.Sprintf("%s/%s", impl.config.GitlabGroupPath, config.GitRepoName)
	prop, res, err := impl.client.Projects.GetProject(pid, &gitlab.GetProjectOptions{})
	if err != nil {
		impl.logger.Debugw("gitlab get project err", "pid", pid, "err", err)
		if res != nil && res.StatusCode == 404 {
			return "", isRepoEmpty, nil
		}
		return "", isRepoEmpty, err
	}
	if res.StatusCode >= 200 && res.StatusCode <= 299 {
		return prop.HTTPURLToRepo, prop.EmptyRepo, nil
	}
	return "", isRepoEmpty, nil
}

func (impl GitLabClient) CreateFirstCommitOnHead(ctx context.Context, config *bean2.GitOpsConfigDto) (string, error) {
	return impl.CreateReadme(ctx, config)
}

func (impl GitLabClient) CreateReadme(ctx context.Context, config *bean2.GitOpsConfigDto) (string, error) {
	var err error
	start := time.Now()
	defer func() {
		util.TriggerGitOpsMetrics("CreateReadme", "GitLabClient", start, err)
	}()

	fileAction := gitlab.FileCreate
	filePath := "README.md"
	fileContent := "devtron licence"
	exists, _ := impl.checkIfFileExists(config.GitRepoName, config.TargetRevision, filePath)
	if exists {
		fileAction = gitlab.FileUpdate
	}
	actions := &gitlab.CreateCommitOptions{
		Branch:        gitlab.Ptr(config.TargetRevision),
		CommitMessage: gitlab.Ptr("test commit"),
		Actions:       []*gitlab.CommitActionOptions{{Action: &fileAction, FilePath: &filePath, Content: &fileContent}},
		AuthorEmail:   &config.UserEmailId,
		AuthorName:    &config.Username,
	}
	gitRepoName := fmt.Sprintf("%s/%s", impl.config.GitlabGroupPath, config.GitRepoName)
	c, _, err := impl.client.Commits.CreateCommit(gitRepoName, actions, gitlab.WithContext(ctx))
	if err != nil {
		impl.logger.Errorw("gitlab commit readme file err", "gitRepoName", gitRepoName, "err", err)
		return "", err
	}
	return c.ID, err
}

func (impl GitLabClient) checkIfFileExists(projectName, ref, file string) (exists bool, err error) {
	_, _, err = impl.client.RepositoryFiles.GetFileMetaData(fmt.Sprintf("%s/%s", impl.config.GitlabGroupPath, projectName), file, &gitlab.GetFileMetaDataOptions{Ref: &ref})
	return err == nil, err
}

func (impl GitLabClient) CommitValues(ctx context.Context, config *ChartConfig, gitOpsConfig *bean2.GitOpsConfigDto, publishStatusConflictError bool) (commitHash string, commitTime time.Time, err error) {

	start := time.Now()

	branch := config.TargetRevision
	if len(branch) == 0 {
		branch = util.GetDefaultTargetRevision()
	}
	path := filepath.Join(config.ChartLocation, config.FileName)
	exists, err := impl.checkIfFileExists(config.ChartRepoName, branch, path)
	var fileAction gitlab.FileActionValue
	if exists {
		fileAction = gitlab.FileUpdate
	} else {
		fileAction = gitlab.FileCreate
	}
	actions := &gitlab.CreateCommitOptions{
		Branch:        &branch,
		CommitMessage: gitlab.String(config.ReleaseMessage),
		Actions:       []*gitlab.CommitActionOptions{{Action: &fileAction, FilePath: &path, Content: &config.FileContent}},
		AuthorEmail:   &config.UserEmailId,
		AuthorName:    &config.UserName,
	}
	c, httpRes, err := impl.client.Commits.CreateCommit(fmt.Sprintf("%s/%s", impl.config.GitlabGroupPath,
		config.ChartRepoName), actions, gitlab.WithContext(ctx))
	if err != nil && httpRes != nil && httpRes.StatusCode == http.StatusBadRequest {
		impl.logger.Warnw("conflict found in commit gitlab", "config", config, "err", err)
		if publishStatusConflictError {
			util.TriggerGitOpsMetrics("CommitValues", "GitLabClient", start, err)
		}
		return "", time.Time{}, retryFunc.NewRetryableError(err)
	} else if err != nil {
		util.TriggerGitOpsMetrics("CommitValues", "GitLabClient", start, err)
		return "", time.Time{}, err
	}
	commitTime = time.Now() //default is current time, if found then will get updated accordingly
	if c != nil {
		commitTime = *c.AuthoredDate
	}
	util.TriggerGitOpsMetrics("CommitValues", "GitLabClient", start, nil)
	return c.ID, commitTime, err
}
