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
	"fmt"
	"time"
)

type DetailedErrorGitOpsConfigActions struct {
	SuccessfulStages []string         `json:"successfulStages"`
	StageErrorMap    map[string]error `json:"stageErrorMap"`
	ValidatedOn      time.Time        `json:"validatedOn"`
	DeleteRepoFailed bool             `json:"deleteRepoFailed"`
}

type ChartConfig struct {
	ChartName      string
	ChartLocation  string
	FileName       string //filename
	FileContent    string
	ReleaseMessage string
	ChartRepoName  string
	TargetRevision string
	// UseDefaultBranch will override the TargetRevision and use the default branch of the repo
	// This is currently implemented for the bitbucket client only.
	// This is used to create the first commit on default branch.
	UseDefaultBranch bool
	UserName         string
	UserEmailId      string
	bitBucketBaseDir string // base directory is required for bitbucket to load the
}

func (c *ChartConfig) SetBitBucketBaseDir(dir string) {
	c.bitBucketBaseDir = fmt.Sprintf("temp-%s", dir)
	return
}

func (c *ChartConfig) GetBitBucketBaseDir() string {
	return c.bitBucketBaseDir
}
