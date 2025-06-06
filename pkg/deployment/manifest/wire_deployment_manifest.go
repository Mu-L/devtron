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

package manifest

import (
	"github.com/devtron-labs/devtron/pkg/deployment/manifest/configMapAndSecret"
	"github.com/devtron-labs/devtron/pkg/deployment/manifest/deployedAppMetrics"
	"github.com/devtron-labs/devtron/pkg/deployment/manifest/deploymentTemplate"
	"github.com/google/wire"
)

var DeploymentManifestWireSet = wire.NewSet(
	deployedAppMetrics.AppMetricsWireSet,
	deploymentTemplate.DeploymentTemplateWireSet,
	configMapAndSecret.ConfigMapAndSecretWireSet,

	NewManifestCreationServiceImpl,
	wire.Bind(new(ManifestCreationService), new(*ManifestCreationServiceImpl)),
)
