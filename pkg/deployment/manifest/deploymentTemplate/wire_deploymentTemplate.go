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

package deploymentTemplate

import (
	"github.com/devtron-labs/devtron/pkg/deployment/manifest/deploymentTemplate/chartRef"
	chartRefRead "github.com/devtron-labs/devtron/pkg/deployment/manifest/deploymentTemplate/chartRef/read"
	"github.com/devtron-labs/devtron/pkg/deployment/manifest/deploymentTemplate/read"
	"github.com/devtron-labs/devtron/pkg/deployment/manifest/deploymentTemplate/validator"
	"github.com/google/wire"
)

var DeploymentTemplateWireSet = wire.NewSet(
	NewDeploymentTemplateServiceImpl,
	wire.Bind(new(DeploymentTemplateService), new(*DeploymentTemplateServiceImpl)),

	read.NewEnvConfigOverrideReadServiceImpl,
	wire.Bind(new(read.EnvConfigOverrideService), new(*read.EnvConfigOverrideReadServiceImpl)),

	validator.NewDeploymentTemplateValidationServiceEntImpl,
	validator.NewDeploymentTemplateValidationServiceImpl,
	wire.Bind(new(validator.DeploymentTemplateValidationService), new(*validator.DeploymentTemplateValidationServiceImpl)),

	chartRefRead.NewChartRefReadServiceImpl,
	wire.Bind(new(chartRefRead.ChartRefReadService), new(*chartRefRead.ChartRefReadServiceImpl)),

	chartRef.NewChartRefServiceImpl,
	wire.Bind(new(chartRef.ChartRefService), new(*chartRef.ChartRefServiceImpl)),
	read.NewDeploymentTemplateHistoryReadServiceImpl,
	wire.Bind(new(read.DeploymentTemplateHistoryReadService), new(*read.DeploymentTemplateHistoryReadServiceImpl)),
)
