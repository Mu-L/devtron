// Code generated by Wire. DO NOT EDIT.

//go:generate wire
//+build !wireinject

package main

import (
	"github.com/devtron-labs/authenticator/client"
	"github.com/devtron-labs/authenticator/middleware"
	"github.com/devtron-labs/devtron/api/appStore/deployment"
	"github.com/devtron-labs/devtron/api/appStore/discover"
	"github.com/devtron-labs/devtron/api/appStore/values"
	chartRepo2 "github.com/devtron-labs/devtron/api/chartRepo"
	cluster2 "github.com/devtron-labs/devtron/api/cluster"
	"github.com/devtron-labs/devtron/api/connector"
	"github.com/devtron-labs/devtron/api/dashboardEvent"
	externalLink2 "github.com/devtron-labs/devtron/api/externalLink"
	client2 "github.com/devtron-labs/devtron/api/helm-app"
	sso2 "github.com/devtron-labs/devtron/api/sso"
	team2 "github.com/devtron-labs/devtron/api/team"
	user2 "github.com/devtron-labs/devtron/api/user"
	"github.com/devtron-labs/devtron/client/dashboard"
	"github.com/devtron-labs/devtron/client/k8s/application"
	"github.com/devtron-labs/devtron/client/k8s/informer"
	"github.com/devtron-labs/devtron/client/telemetry"
	"github.com/devtron-labs/devtron/internal/sql/repository/app"
	"github.com/devtron-labs/devtron/internal/sql/repository/pipelineConfig"
	"github.com/devtron-labs/devtron/internal/util"
	"github.com/devtron-labs/devtron/pkg/appStore/deployment/common"
	repository3 "github.com/devtron-labs/devtron/pkg/appStore/deployment/repository"
	service3 "github.com/devtron-labs/devtron/pkg/appStore/deployment/service"
	"github.com/devtron-labs/devtron/pkg/appStore/deployment/tool"
	"github.com/devtron-labs/devtron/pkg/appStore/discover/repository"
	"github.com/devtron-labs/devtron/pkg/appStore/repository"
	"github.com/devtron-labs/devtron/pkg/appStore/values"
	"github.com/devtron-labs/devtron/pkg/appStore/discover/service"
	"github.com/devtron-labs/devtron/pkg/appStore/values/repository"
	service2 "github.com/devtron-labs/devtron/pkg/appStore/values/service"
	"github.com/devtron-labs/devtron/pkg/chartRepo"
	"github.com/devtron-labs/devtron/pkg/chartRepo/repository"
	"github.com/devtron-labs/devtron/pkg/cluster"
	repository2 "github.com/devtron-labs/devtron/pkg/cluster/repository"
	delete2 "github.com/devtron-labs/devtron/pkg/delete"
	"github.com/devtron-labs/devtron/pkg/externalLink"
	"github.com/devtron-labs/devtron/pkg/pipeline"
	"github.com/devtron-labs/devtron/pkg/sql"
	"github.com/devtron-labs/devtron/pkg/sso"
	"github.com/devtron-labs/devtron/pkg/team"
	"github.com/devtron-labs/devtron/pkg/terminal"
	"github.com/devtron-labs/devtron/pkg/user"
	"github.com/devtron-labs/devtron/pkg/user/casbin"
	"github.com/devtron-labs/devtron/pkg/user/repository"
	util2 "github.com/devtron-labs/devtron/pkg/util"
	util3 "github.com/devtron-labs/devtron/util"
	"github.com/devtron-labs/devtron/util/k8s"
	"github.com/devtron-labs/devtron/util/rbac"
)

// Injectors from wire.go:

func InitializeApp() (*App, error) {
	config, err := sql.GetConfig()
	if err != nil {
		return nil, err
	}
	sugaredLogger := util.NewSugardLogger()
	db, err := sql.NewDbConnection(config, sugaredLogger)
	if err != nil {
		return nil, err
	}
	runtimeConfig, err := client.GetRuntimeConfig()
	if err != nil {
		return nil, err
	}
	k8sClient, err := client.NewK8sClient(runtimeConfig)
	if err != nil {
		return nil, err
	}
	dexConfig, err := client.BuildDexConfig(k8sClient)
	if err != nil {
		return nil, err
	}
	settings, err := client.GetSettings(dexConfig)
	if err != nil {
		return nil, err
	}
	sessionManager := middleware.NewSessionManager(settings, dexConfig)
	validate, err := util.IntValidator()
	if err != nil {
		return nil, err
	}
	enforcer := casbin.Create()
	enforcerImpl := casbin.NewEnforcerImpl(enforcer, sessionManager, sugaredLogger)
	defaultAuthPolicyRepositoryImpl := repository.NewDefaultAuthPolicyRepositoryImpl(db, sugaredLogger)
	defaultAuthRoleRepositoryImpl := repository.NewDefaultAuthRoleRepositoryImpl(db, sugaredLogger)
	userAuthRepositoryImpl := repository.NewUserAuthRepositoryImpl(db, sugaredLogger, defaultAuthPolicyRepositoryImpl, defaultAuthRoleRepositoryImpl)
	userRepositoryImpl := repository.NewUserRepositoryImpl(db, sugaredLogger)
	roleGroupRepositoryImpl := repository.NewRoleGroupRepositoryImpl(db, sugaredLogger)
	userCommonServiceImpl := user.NewUserCommonServiceImpl(userAuthRepositoryImpl, sugaredLogger, userRepositoryImpl, roleGroupRepositoryImpl, sessionManager)
	userServiceImpl := user.NewUserServiceImpl(userAuthRepositoryImpl, sugaredLogger, userRepositoryImpl, roleGroupRepositoryImpl, sessionManager, userCommonServiceImpl)
	ssoLoginRepositoryImpl := sso.NewSSOLoginRepositoryImpl(db)
	k8sUtil := util.NewK8sUtil(sugaredLogger, runtimeConfig)
	ssoLoginServiceImpl := sso.NewSSOLoginServiceImpl(sugaredLogger, ssoLoginRepositoryImpl, k8sUtil)
	ssoLoginRestHandlerImpl := sso2.NewSsoLoginRestHandlerImpl(validate, sugaredLogger, enforcerImpl, userServiceImpl, ssoLoginServiceImpl)
	ssoLoginRouterImpl := sso2.NewSsoLoginRouterImpl(ssoLoginRestHandlerImpl)
	teamRepositoryImpl := team.NewTeamRepositoryImpl(db)
	loginService := middleware.NewUserLogin(sessionManager, k8sClient)
	userAuthServiceImpl := user.NewUserAuthServiceImpl(userAuthRepositoryImpl, sessionManager, loginService, sugaredLogger, userRepositoryImpl, roleGroupRepositoryImpl)
	teamServiceImpl := team.NewTeamServiceImpl(sugaredLogger, teamRepositoryImpl, userAuthServiceImpl)
	clusterRepositoryImpl := repository2.NewClusterRepositoryImpl(db, sugaredLogger)
	v := informer.NewGlobalMapClusterNamespace()
	k8sInformerFactoryImpl := informer.NewK8sInformerFactoryImpl(sugaredLogger, v, runtimeConfig)
	clusterServiceImpl := cluster.NewClusterServiceImpl(clusterRepositoryImpl, sugaredLogger, k8sUtil, k8sInformerFactoryImpl)
	environmentRepositoryImpl := repository2.NewEnvironmentRepositoryImpl(db)
	environmentServiceImpl := cluster.NewEnvironmentServiceImpl(environmentRepositoryImpl, clusterServiceImpl, sugaredLogger, k8sUtil, k8sInformerFactoryImpl, userAuthServiceImpl)
	chartRepoRepositoryImpl := chartRepoRepository.NewChartRepoRepositoryImpl(db)
	acdAuthConfig, err := util2.GetACDAuthConfig()
	if err != nil {
		return nil, err
	}
	httpClient := util.NewHttpClient()
	chartRepositoryServiceImpl := chartRepo.NewChartRepositoryServiceImpl(sugaredLogger, chartRepoRepositoryImpl, k8sUtil, clusterServiceImpl, acdAuthConfig, httpClient)
	deleteServiceImpl := delete2.NewDeleteServiceImpl(sugaredLogger, teamServiceImpl, clusterServiceImpl, environmentServiceImpl, chartRepositoryServiceImpl)
	teamRestHandlerImpl := team2.NewTeamRestHandlerImpl(sugaredLogger, teamServiceImpl, userServiceImpl, enforcerImpl, validate, userAuthServiceImpl, deleteServiceImpl)
	teamRouterImpl := team2.NewTeamRouterImpl(teamRestHandlerImpl)
	userAuthHandlerImpl := user2.NewUserAuthHandlerImpl(userAuthServiceImpl, validate, sugaredLogger)
	userAuthRouterImpl, err := user2.NewUserAuthRouterImpl(sugaredLogger, userAuthHandlerImpl, userServiceImpl, dexConfig)
	if err != nil {
		return nil, err
	}
	roleGroupServiceImpl := user.NewRoleGroupServiceImpl(userAuthRepositoryImpl, sugaredLogger, userRepositoryImpl, roleGroupRepositoryImpl, userCommonServiceImpl)
	userRestHandlerImpl := user2.NewUserRestHandlerImpl(userServiceImpl, validate, sugaredLogger, enforcerImpl, roleGroupServiceImpl)
	userRouterImpl := user2.NewUserRouterImpl(userRestHandlerImpl)
	clusterRestHandlerImpl := cluster2.NewClusterRestHandlerImpl(clusterServiceImpl, sugaredLogger, userServiceImpl, validate, enforcerImpl, deleteServiceImpl)
	clusterRouterImpl := cluster2.NewClusterRouterImpl(clusterRestHandlerImpl)
	dashboardConfig, err := dashboard.GetConfig()
	if err != nil {
		return nil, err
	}
	dashboardRouterImpl := dashboard.NewDashboardRouterImpl(sugaredLogger, dashboardConfig)
	helmClientConfig, err := client2.GetConfig()
	if err != nil {
		return nil, err
	}
	helmAppClientImpl := client2.NewHelmAppClientImpl(sugaredLogger, helmClientConfig)
	pumpImpl := connector.NewPumpImpl(sugaredLogger)
	enforcerUtilHelmImpl := rbac.NewEnforcerUtilHelmImpl(sugaredLogger, clusterRepositoryImpl)
	helmAppServiceImpl := client2.NewHelmAppServiceImpl(sugaredLogger, clusterServiceImpl, helmAppClientImpl, pumpImpl, enforcerUtilHelmImpl)
	installedAppRepositoryImpl := repository3.NewInstalledAppRepositoryImpl(sugaredLogger, db)
	appStoreDeploymentCommonServiceImpl := appStoreDeploymentCommon.NewAppStoreDeploymentCommonServiceImpl(sugaredLogger, installedAppRepositoryImpl)
	helmAppRestHandlerImpl := client2.NewHelmAppRestHandlerImpl(sugaredLogger, helmAppServiceImpl, enforcerImpl, clusterServiceImpl, enforcerUtilHelmImpl, appStoreDeploymentCommonServiceImpl)
	helmAppRouterImpl := client2.NewHelmAppRouterImpl(helmAppRestHandlerImpl)
	environmentRestHandlerImpl := cluster2.NewEnvironmentRestHandlerImpl(environmentServiceImpl, sugaredLogger, userServiceImpl, validate, enforcerImpl, deleteServiceImpl)
	environmentRouterImpl := cluster2.NewEnvironmentRouterImpl(environmentRestHandlerImpl)
	k8sClientServiceImpl := application.NewK8sClientServiceImpl(sugaredLogger, clusterRepositoryImpl)
	k8sApplicationServiceImpl := k8s.NewK8sApplicationServiceImpl(sugaredLogger, clusterServiceImpl, pumpImpl, k8sClientServiceImpl, helmAppServiceImpl)
	terminalSessionHandlerImpl := terminal.NewTerminalSessionHandlerImpl(environmentServiceImpl, clusterServiceImpl, sugaredLogger)
	k8sApplicationRestHandlerImpl := k8s.NewK8sApplicationRestHandlerImpl(sugaredLogger, k8sApplicationServiceImpl, pumpImpl, terminalSessionHandlerImpl, enforcerImpl, enforcerUtilHelmImpl, clusterServiceImpl, helmAppServiceImpl)
	k8sApplicationRouterImpl := k8s.NewK8sApplicationRouterImpl(k8sApplicationRestHandlerImpl)
	chartRefRepositoryImpl := chartRepoRepository.NewChartRefRepositoryImpl(db)
	refChartDir := _wireRefChartDirValue
	chartRepositoryRestHandlerImpl := chartRepo2.NewChartRepositoryRestHandlerImpl(sugaredLogger, userServiceImpl, chartRepositoryServiceImpl, enforcerImpl, validate, deleteServiceImpl, chartRefRepositoryImpl, refChartDir)
	chartRepositoryRouterImpl := chartRepo2.NewChartRepositoryRouterImpl(chartRepositoryRestHandlerImpl)
	appStoreApplicationVersionRepositoryImpl := appStoreDiscoverRepository.NewAppStoreApplicationVersionRepositoryImpl(sugaredLogger, db)
	appStoreServiceImpl := service.NewAppStoreServiceImpl(sugaredLogger, appStoreApplicationVersionRepositoryImpl)
	appStoreRestHandlerImpl := appStoreDiscover.NewAppStoreRestHandlerImpl(sugaredLogger, userServiceImpl, appStoreServiceImpl, enforcerImpl)
	appStoreDiscoverRouterImpl := appStoreDiscover.NewAppStoreDiscoverRouterImpl(appStoreRestHandlerImpl)
	appStoreVersionValuesRepositoryImpl := appStoreValuesRepository.NewAppStoreVersionValuesRepositoryImpl(sugaredLogger, db)
	appStoreValuesServiceImpl := service2.NewAppStoreValuesServiceImpl(sugaredLogger, appStoreApplicationVersionRepositoryImpl, installedAppRepositoryImpl, appStoreVersionValuesRepositoryImpl)
	appStoreValuesRestHandlerImpl := appStoreValues.NewAppStoreValuesRestHandlerImpl(sugaredLogger, userServiceImpl, appStoreValuesServiceImpl)
	appStoreValuesRouterImpl := appStoreValues.NewAppStoreValuesRouterImpl(appStoreValuesRestHandlerImpl)
	appRepositoryImpl := app.NewAppRepositoryImpl(db)
	pipelineRepositoryImpl := pipelineConfig.NewPipelineRepositoryImpl(db, sugaredLogger)
	ciPipelineRepositoryImpl := pipelineConfig.NewCiPipelineRepositoryImpl(db, sugaredLogger)
	enforcerUtilImpl := rbac.NewEnforcerUtilImpl(sugaredLogger, teamRepositoryImpl, appRepositoryImpl, environmentRepositoryImpl, pipelineRepositoryImpl, ciPipelineRepositoryImpl, clusterRepositoryImpl)
	clusterInstalledAppsRepositoryImpl := repository3.NewClusterInstalledAppsRepositoryImpl(db, sugaredLogger)
	appStoreDeploymentHelmServiceImpl := appStoreDeploymentTool.NewAppStoreDeploymentHelmServiceImpl(sugaredLogger, helmAppServiceImpl, appStoreApplicationVersionRepositoryImpl, environmentRepositoryImpl, helmAppClientImpl)
	globalEnvVariables, err := util3.GetGlobalEnvVariables()
	if err != nil {
		return nil, err
	}
	installedAppVersionHistoryRepositoryImpl := repository3.NewInstalledAppVersionHistoryRepositoryImpl(sugaredLogger, db)
	appStoreDeploymentServiceImpl := service3.NewAppStoreDeploymentServiceImpl(sugaredLogger, installedAppRepositoryImpl, appStoreApplicationVersionRepositoryImpl, environmentRepositoryImpl, clusterInstalledAppsRepositoryImpl, appRepositoryImpl, appStoreDeploymentHelmServiceImpl, appStoreDeploymentHelmServiceImpl, environmentServiceImpl, clusterServiceImpl, helmAppServiceImpl, appStoreDeploymentCommonServiceImpl, globalEnvVariables, installedAppVersionHistoryRepositoryImpl)
	appStoreDeploymentRestHandlerImpl := appStoreDeployment.NewAppStoreDeploymentRestHandlerImpl(sugaredLogger, userServiceImpl, enforcerImpl, enforcerUtilImpl, enforcerUtilHelmImpl, appStoreDeploymentServiceImpl, validate, helmAppServiceImpl, appStoreDeploymentCommonServiceImpl)
	appStoreDeploymentRouterImpl := appStoreDeployment.NewAppStoreDeploymentRouterImpl(appStoreDeploymentRestHandlerImpl)
	posthogClient, err := telemetry.NewPosthogClient(sugaredLogger)
	if err != nil {
		return nil, err
	}
	telemetryEventClientImpl, err := telemetry.NewTelemetryEventClientImpl(sugaredLogger, httpClient, clusterServiceImpl, k8sUtil, acdAuthConfig, userServiceImpl, posthogClient)
	if err != nil {
		return nil, err
	}
	dashboardTelemetryRestHandlerImpl := dashboardEvent.NewDashboardTelemetryRestHandlerImpl(sugaredLogger, telemetryEventClientImpl)
	dashboardTelemetryRouterImpl := dashboardEvent.NewDashboardTelemetryRouterImpl(dashboardTelemetryRestHandlerImpl)
	commonDeploymentRestHandlerImpl := appStoreDeployment.NewCommonDeploymentRestHandlerImpl(sugaredLogger, userServiceImpl, enforcerImpl, enforcerUtilImpl, enforcerUtilHelmImpl, appStoreDeploymentServiceImpl, validate, helmAppServiceImpl, appStoreDeploymentCommonServiceImpl, helmAppRestHandlerImpl)
	commonDeploymentRouterImpl := appStoreDeployment.NewCommonDeploymentRouterImpl(commonDeploymentRestHandlerImpl)
	externalLinkMonitoringToolRepositoryImpl := externalLink.NewExternalLinkMonitoringToolRepositoryImpl(db)
	externalLinkClusterMappingRepositoryImpl := externalLink.NewExternalLinkClusterMappingRepositoryImpl(db)
	externalLinkRepositoryImpl := externalLink.NewExternalLinkRepositoryImpl(db)
	externalLinkServiceImpl := externalLink.NewExternalLinkServiceImpl(sugaredLogger, externalLinkMonitoringToolRepositoryImpl, externalLinkClusterMappingRepositoryImpl, externalLinkRepositoryImpl)
	externalLinkRestHandlerImpl := externalLink2.NewExternalLinkRestHandlerImpl(sugaredLogger, externalLinkServiceImpl, userServiceImpl, enforcerImpl)
	externalLinkRouterImpl := externalLink2.NewExternalLinkRouterImpl(externalLinkRestHandlerImpl)
	muxRouter := NewMuxRouter(sugaredLogger, ssoLoginRouterImpl, teamRouterImpl, userAuthRouterImpl, userRouterImpl, clusterRouterImpl, dashboardRouterImpl, helmAppRouterImpl, environmentRouterImpl, k8sApplicationRouterImpl, chartRepositoryRouterImpl, appStoreDiscoverRouterImpl, appStoreValuesRouterImpl, appStoreDeploymentRouterImpl, dashboardTelemetryRouterImpl, commonDeploymentRouterImpl, externalLinkRouterImpl)
	mainApp := NewApp(db, sessionManager, muxRouter, telemetryEventClientImpl, sugaredLogger)
	return mainApp, nil
}

var (
	_wireRefChartDirValue = pipeline.RefChartDir("scripts/devtron-reference-helm-charts")
)
