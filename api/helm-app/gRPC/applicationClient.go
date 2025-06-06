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

package gRPC

import (
	"context"
	"fmt"
	"github.com/caarlos0/env"
	grpcUtil "github.com/devtron-labs/common-lib/utils/grpc"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.opentelemetry.io/otel"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"time"
)

type HelmAppClient interface {
	ListApplication(ctx context.Context, req *AppListRequest) (ApplicationService_ListApplicationsClient, error)
	ListFluxApplication(ctx context.Context, req *AppListRequest) (ApplicationService_ListFluxApplicationsClient, error)
	GetAppDetail(ctx context.Context, in *AppDetailRequest) (*AppDetail, error)
	GetResourceTreeForExternalResources(ctx context.Context, in *ExternalResourceTreeRequest) (*ResourceTreeResponse, error)
	GetAppStatus(ctx context.Context, in *AppDetailRequest) (*AppStatus, error)
	GetAppStatusV2(ctx context.Context, in *AppDetailRequest) (*AppStatus, error)
	Hibernate(ctx context.Context, in *HibernateRequest) (*HibernateResponse, error)
	UnHibernate(ctx context.Context, in *HibernateRequest) (*HibernateResponse, error)
	GetDeploymentHistory(ctx context.Context, in *AppDetailRequest) (*HelmAppDeploymentHistory, error)
	GetValuesYaml(ctx context.Context, in *AppDetailRequest) (*ReleaseInfo, error)
	GetDesiredManifest(ctx context.Context, in *ObjectRequest) (*DesiredManifestResponse, error)
	DeleteApplication(ctx context.Context, in *ReleaseIdentifier) (*UninstallReleaseResponse, error)
	UpdateApplication(ctx context.Context, in *UpgradeReleaseRequest) (*UpgradeReleaseResponse, error)
	GetDeploymentDetail(ctx context.Context, in *DeploymentDetailRequest) (*DeploymentDetailResponse, error)
	InstallRelease(ctx context.Context, in *InstallReleaseRequest) (*InstallReleaseResponse, error)
	UpdateApplicationWithChartInfo(ctx context.Context, in *InstallReleaseRequest) (*UpgradeReleaseResponse, error)
	IsReleaseInstalled(ctx context.Context, in *ReleaseIdentifier) (*BooleanResponse, error)
	RollbackRelease(ctx context.Context, in *RollbackReleaseRequest) (*BooleanResponse, error)
	TemplateChart(ctx context.Context, in *InstallReleaseRequest) (*TemplateChartResponse, error)
	TemplateChartBulk(ctx context.Context, in *BulkInstallReleaseRequest) (*BulkTemplateChartResponse, error)
	InstallReleaseWithCustomChart(ctx context.Context, in *HelmInstallCustomRequest) (*HelmInstallCustomResponse, error)
	GetNotes(ctx context.Context, request *InstallReleaseRequest) (*ChartNotesResponse, error)
	ValidateOCIRegistry(ctx context.Context, OCIRegistryRequest *RegistryCredential) (*OCIRegistryResponse, error)
	GetReleaseDetails(ctx context.Context, in *ReleaseIdentifier) (*DeployedAppDetail, error)
	GetExternalFluxAppDetail(ctx context.Context, in *FluxAppDetailRequest) (*FluxAppDetail, error)
}

type HelmAppClientImpl struct {
	logger                   *zap.SugaredLogger
	helmClientConfig         *HelmClientConfig
	applicationServiceClient ApplicationServiceClient
	grpcConfig               *grpcUtil.Configuration
}

func NewHelmAppClientImpl(logger *zap.SugaredLogger,
	helmClientConfig *HelmClientConfig,
	grpcConfig *grpcUtil.Configuration) *HelmAppClientImpl {
	return &HelmAppClientImpl{
		logger:           logger,
		helmClientConfig: helmClientConfig,
		grpcConfig:       grpcConfig,
	}
}

// CATEGORY=INFRA_SETUP
type HelmClientConfig struct {
	Url string `env:"HELM_CLIENT_URL" envDefault:"127.0.0.1:50051" description:"Kubelink micro-service url "`
}

func GetConfig() (*HelmClientConfig, error) {
	cfg := &HelmClientConfig{}
	err := env.Parse(cfg)
	return cfg, err
}

func (impl *HelmAppClientImpl) getApplicationClient() (ApplicationServiceClient, error) {
	if impl.applicationServiceClient == nil {
		connection, err := impl.getConnection()
		if err != nil {
			return nil, err
		}
		impl.applicationServiceClient = NewApplicationServiceClient(connection)
	}
	return impl.applicationServiceClient, nil
}

func (impl *HelmAppClientImpl) getConnection() (*grpc.ClientConn, error) {
	var opts []grpc.DialOption
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	opts = append(opts,
		grpc.WithUnaryInterceptor(otelgrpc.UnaryClientInterceptor()),
		grpc.WithStreamInterceptor(otelgrpc.StreamClientInterceptor()),
		grpc.WithBlock(),
		grpc.WithInsecure(),
		grpc.WithDefaultCallOptions(
			grpc.MaxCallRecvMsgSize(impl.grpcConfig.KubelinkMaxSendMsgSize*1024*1024), // GRPC Request size
			grpc.MaxCallSendMsgSize(impl.grpcConfig.KubelinkMaxRecvMsgSize*1024*1024), // GRPC Response size
		),
		grpc.WithDefaultServiceConfig(impl.grpcConfig.KubelinkGRPCServiceConfig),
	)
	endpoint := fmt.Sprintf("dns:///%s", impl.helmClientConfig.Url)
	conn, err := grpc.DialContext(ctx, endpoint, opts...)
	if err != nil {
		return nil, err
	}
	return conn, err
}

func (impl *HelmAppClientImpl) ListApplication(ctx context.Context, req *AppListRequest) (ApplicationService_ListApplicationsClient, error) {
	applicationClient, err := impl.getApplicationClient()
	if err != nil {
		return nil, err
	}
	stream, err := applicationClient.ListApplications(ctx, req)
	if err != nil {
		return nil, err
	}
	return stream, nil
}

///	GetAppDetail(ctx context.Context, in *AppDetailRequest, opts ...grpc.CallOption) (*AppDetail, error)

func (impl *HelmAppClientImpl) GetAppDetail(ctx context.Context, in *AppDetailRequest) (*AppDetail, error) {
	applicationClient, err := impl.getApplicationClient()
	if err != nil {
		return nil, err
	}
	detail, err := applicationClient.GetAppDetail(ctx, in)
	if err != nil {
		return nil, err
	}
	return detail, nil
}

func (impl *HelmAppClientImpl) GetResourceTreeForExternalResources(ctx context.Context, in *ExternalResourceTreeRequest) (*ResourceTreeResponse, error) {
	applicationClient, err := impl.getApplicationClient()
	if err != nil {
		return nil, err
	}
	detail, err := applicationClient.GetResourceTreeForExternalResources(ctx, in)
	if err != nil {
		return nil, err
	}
	return detail, nil
}

func (impl *HelmAppClientImpl) GetAppStatus(ctx context.Context, in *AppDetailRequest) (*AppStatus, error) {
	applicationClient, err := impl.getApplicationClient()
	if err != nil {
		return nil, err
	}
	appStatus, err := applicationClient.GetAppStatus(ctx, in)
	if err != nil {
		return nil, err
	}
	return appStatus, nil
}

func (impl *HelmAppClientImpl) GetAppStatusV2(ctx context.Context, in *AppDetailRequest) (*AppStatus, error) {
	applicationClient, err := impl.getApplicationClient()
	if err != nil {
		return nil, err
	}
	appStatus, err := applicationClient.GetAppStatusV2(ctx, in)
	if err != nil {
		return nil, err
	}
	return appStatus, nil
}

func (impl *HelmAppClientImpl) Hibernate(ctx context.Context, in *HibernateRequest) (*HibernateResponse, error) {
	applicationClient, err := impl.getApplicationClient()
	if err != nil {
		return nil, err
	}
	detail, err := applicationClient.Hibernate(ctx, in)
	if err != nil {
		return nil, err
	}
	return detail, nil
}

func (impl *HelmAppClientImpl) UnHibernate(ctx context.Context, in *HibernateRequest) (*HibernateResponse, error) {
	applicationClient, err := impl.getApplicationClient()
	if err != nil {
		return nil, err
	}
	detail, err := applicationClient.UnHibernate(ctx, in)
	if err != nil {
		return nil, err
	}
	return detail, nil
}

func (impl *HelmAppClientImpl) GetDeploymentHistory(ctx context.Context, in *AppDetailRequest) (*HelmAppDeploymentHistory, error) {
	applicationClient, err := impl.getApplicationClient()
	if err != nil {
		return nil, err
	}
	history, err := applicationClient.GetDeploymentHistory(ctx, in)
	if err != nil {
		return nil, err
	}
	return history, nil
}

func (impl *HelmAppClientImpl) GetValuesYaml(ctx context.Context, in *AppDetailRequest) (*ReleaseInfo, error) {
	applicationClient, err := impl.getApplicationClient()
	if err != nil {
		return nil, err
	}
	values, err := applicationClient.GetValuesYaml(ctx, in)
	if err != nil {
		return nil, err
	}
	return values, nil
}

func (impl *HelmAppClientImpl) GetDesiredManifest(ctx context.Context, in *ObjectRequest) (*DesiredManifestResponse, error) {
	applicationClient, err := impl.getApplicationClient()
	if err != nil {
		return nil, err
	}
	manifest, err := applicationClient.GetDesiredManifest(ctx, in)
	if err != nil {
		return nil, err
	}
	return manifest, nil
}

func (impl *HelmAppClientImpl) DeleteApplication(ctx context.Context, in *ReleaseIdentifier) (*UninstallReleaseResponse, error) {
	applicationClient, err := impl.getApplicationClient()
	if err != nil {
		return nil, err
	}
	manifest, err := applicationClient.UninstallRelease(ctx, in)
	if err != nil {
		return nil, err
	}
	return manifest, nil
}

func (impl *HelmAppClientImpl) UpdateApplication(ctx context.Context, in *UpgradeReleaseRequest) (*UpgradeReleaseResponse, error) {
	newCtx, span := otel.Tracer("orchestrator").Start(ctx, "HelmAppClientImpl.UpdateApplication")
	defer span.End()
	applicationClient, err := impl.getApplicationClient()
	if err != nil {
		return nil, err
	}
	manifest, err := applicationClient.UpgradeRelease(newCtx, in)
	if err != nil {
		return nil, err
	}
	return manifest, nil
}

func (impl *HelmAppClientImpl) GetDeploymentDetail(ctx context.Context, in *DeploymentDetailRequest) (*DeploymentDetailResponse, error) {
	applicationClient, err := impl.getApplicationClient()
	if err != nil {
		return nil, err
	}
	deploymentDetail, err := applicationClient.GetDeploymentDetail(ctx, in)
	if err != nil {
		return nil, err
	}
	return deploymentDetail, nil
}

func (impl *HelmAppClientImpl) InstallRelease(ctx context.Context, in *InstallReleaseRequest) (*InstallReleaseResponse, error) {
	applicationClient, err := impl.getApplicationClient()
	if err != nil {
		return nil, err
	}
	installReleaseResponse, err := applicationClient.InstallRelease(ctx, in)
	if err != nil {
		return nil, err
	}
	return installReleaseResponse, nil
}

func (impl *HelmAppClientImpl) UpdateApplicationWithChartInfo(ctx context.Context, in *InstallReleaseRequest) (*UpgradeReleaseResponse, error) {
	applicationClient, err := impl.getApplicationClient()
	if err != nil {
		return nil, err
	}
	updateReleaseResponse, err := applicationClient.UpgradeReleaseWithChartInfo(ctx, in)
	if err != nil {
		return nil, err
	}
	return updateReleaseResponse, nil
}

func (impl *HelmAppClientImpl) IsReleaseInstalled(ctx context.Context, in *ReleaseIdentifier) (*BooleanResponse, error) {
	applicationClient, err := impl.getApplicationClient()
	if err != nil {
		return nil, err
	}
	response, err := applicationClient.IsReleaseInstalled(ctx, in)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (impl *HelmAppClientImpl) RollbackRelease(ctx context.Context, in *RollbackReleaseRequest) (*BooleanResponse, error) {
	applicationClient, err := impl.getApplicationClient()
	if err != nil {
		return nil, err
	}
	response, err := applicationClient.RollbackRelease(ctx, in)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (impl *HelmAppClientImpl) TemplateChart(ctx context.Context, in *InstallReleaseRequest) (*TemplateChartResponse, error) {
	applicationClient, err := impl.getApplicationClient()
	if err != nil {
		return nil, err
	}
	response, err := applicationClient.TemplateChart(ctx, in)
	if err != nil {
		return nil, err
	}
	return response, nil
}
func (impl *HelmAppClientImpl) TemplateChartBulk(ctx context.Context, in *BulkInstallReleaseRequest) (*BulkTemplateChartResponse, error) {
	applicationClient, err := impl.getApplicationClient()
	if err != nil {
		return nil, err
	}
	response, err := applicationClient.TemplateChartBulk(ctx, in)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (impl *HelmAppClientImpl) InstallReleaseWithCustomChart(ctx context.Context, in *HelmInstallCustomRequest) (*HelmInstallCustomResponse, error) {
	applicationClient, err := impl.getApplicationClient()
	if err != nil {
		return nil, err
	}
	response, err := applicationClient.InstallReleaseWithCustomChart(ctx, in)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (impl *HelmAppClientImpl) GetNotes(ctx context.Context, in *InstallReleaseRequest) (*ChartNotesResponse, error) {
	applicationClient, err := impl.getApplicationClient()
	if err != nil {
		return nil, err
	}
	response, err := applicationClient.GetNotes(ctx, in)

	if err != nil {
		return nil, err
	}
	return response, nil

}

func (impl *HelmAppClientImpl) ValidateOCIRegistry(ctx context.Context, in *RegistryCredential) (*OCIRegistryResponse, error) {
	applicationClient, err := impl.getApplicationClient()
	if err != nil {
		return nil, err
	}
	response, err := applicationClient.ValidateOCIRegistry(ctx, in)
	if err != nil {
		return nil, err
	}
	return response, nil
}
func (impl *HelmAppClientImpl) ListFluxApplication(ctx context.Context, req *AppListRequest) (ApplicationService_ListFluxApplicationsClient, error) {
	applicationClient, err := impl.getApplicationClient()
	if err != nil {
		return nil, err
	}
	stream, err := applicationClient.ListFluxApplications(ctx, req)
	if err != nil {
		return nil, err
	}
	return stream, nil
}

func (impl *HelmAppClientImpl) GetReleaseDetails(ctx context.Context, in *ReleaseIdentifier) (*DeployedAppDetail, error) {
	applicationClient, err := impl.getApplicationClient()
	if err != nil {
		return nil, err
	}
	response, err := applicationClient.GetReleaseDetails(ctx, in)
	if err != nil {
		return nil, err
	}
	return response, nil
}

func (impl *HelmAppClientImpl) GetExternalFluxAppDetail(ctx context.Context, in *FluxAppDetailRequest) (*FluxAppDetail, error) {
	applicationClient, err := impl.getApplicationClient()
	if err != nil {
		return nil, err
	}
	detail, err := applicationClient.GetFluxAppDetail(ctx, in)
	if err != nil {
		return nil, err
	}
	return detail, nil
}
