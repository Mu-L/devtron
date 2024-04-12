/*
 * Copyright (c) 2020 Devtron Labs
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package util

type EventType int

const (
	Trigger        EventType = 1
	Success        EventType = 2
	Fail           EventType = 3
	Approval       EventType = 4
	ConfigApproval EventType = 5
	Blocked 	   EventType = 6
	ImageScanning  EventType = 8
)

var RulesSupportedEvents = []int{int(ImageScanning)}

type PipelineType string

const CI PipelineType = "CI"
const CD PipelineType = "CD"
const PRE_CD PipelineType = "PRE-CD"
const POST_CD PipelineType = "POST-CD"

type Level string

type Channel string

const (
	Slack   Channel = "slack"
	SES     Channel = "ses"
	SMTP    Channel = "smtp"
	Webhook Channel = "webhook"
)

type UpdateType string

const (
	UpdateEvents     UpdateType = "events"
	UpdateRecipients UpdateType = "recipients"
	UpdateRules      UpdateType = "rules"
)
