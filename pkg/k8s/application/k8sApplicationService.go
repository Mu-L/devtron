package application

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/devtron-labs/common-lib/utils"
	"github.com/devtron-labs/devtron/api/helm-app/gRPC"
	"github.com/devtron-labs/devtron/enterprise/pkg/resourceFilter"
	util4 "github.com/devtron-labs/devtron/api/util"
	"github.com/devtron-labs/devtron/enterprise/pkg/deploymentWindow"
	"github.com/devtron-labs/devtron/internal/constants"
	"github.com/devtron-labs/devtron/pkg/auth/authorisation/casbin"
	"github.com/go-pg/pg"
	client2 "github.com/devtron-labs/scoop/client"
	types2 "github.com/devtron-labs/scoop/types"
	"io"
	admissionregistrationV1alpha1 "k8s.io/api/admissionregistration/v1alpha1"
	admissionregistrationV1beta1 "k8s.io/api/admissionregistration/v1beta1"
	apiserverinternalV1alpha1 "k8s.io/api/apiserverinternal/v1alpha1"
	appsV1 "k8s.io/api/apps/v1"
	autoscalingV2 "k8s.io/api/autoscaling/v2"
	batchv1 "k8s.io/api/batch/v1"
	certificatesV1 "k8s.io/api/certificates/v1"
	coordinationV1 "k8s.io/api/coordination/v1"
	discoveryV1beta1 "k8s.io/api/discovery/v1beta1"
	networkingV1 "k8s.io/api/networking/v1"
	networkingv1alpha1 "k8s.io/api/networking/v1alpha1"
	networkingV1beta1 "k8s.io/api/networking/v1beta1"
	nodeV1 "k8s.io/api/node/v1"
	resourceV1alpha1 "k8s.io/api/resource/v1alpha1"
	schedulingV1 "k8s.io/api/scheduling/v1"
	storageV1beta1 "k8s.io/api/storage/v1beta1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/duration"
	v1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/kubernetes/pkg/apis/flowcontrol"
	"k8s.io/kubernetes/pkg/apis/rbac"
	"net/http"
	"net/http/httputil"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/caarlos0/env/v6"
	k8s2 "github.com/devtron-labs/common-lib-private/utils/k8s"
	k8s3 "github.com/devtron-labs/common-lib/utils/k8s"
	k8sCommonBean "github.com/devtron-labs/common-lib/utils/k8s/commonBean"
	k8sObjectUtils "github.com/devtron-labs/common-lib/utils/k8sObjectsUtil"
	policyV1beta1 "k8s.io/api/policy/v1beta1"

	yamlUtil "github.com/devtron-labs/common-lib/utils/yaml"
	"github.com/devtron-labs/devtron/api/connector"
	"github.com/devtron-labs/devtron/api/helm-app/openapiClient"
	"github.com/devtron-labs/devtron/internal/util"
	"github.com/devtron-labs/devtron/pkg/argoApplication"
	"github.com/devtron-labs/devtron/pkg/cluster"
	"github.com/devtron-labs/devtron/pkg/cluster/repository"
	"github.com/devtron-labs/devtron/pkg/k8s"
	bean3 "github.com/devtron-labs/devtron/pkg/k8s/application/bean"
	"github.com/devtron-labs/devtron/pkg/kubernetesResourceAuditLogs"
	"github.com/devtron-labs/devtron/pkg/terminal"
	util3 "github.com/devtron-labs/devtron/pkg/util"
	util2 "github.com/devtron-labs/devtron/util"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
	apiV1 "k8s.io/api/core/v1"
	errors2 "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"

	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/kubernetes/pkg/printers"
)

var ScoopNotConfiguredErr = errors.New("scoop not configured")

type K8sApplicationService interface {
	ValidatePodLogsRequestQuery(r *http.Request) (*k8s.ResourceRequestBean, error)
	ValidateTerminalRequestQuery(r *http.Request) (*terminal.TerminalSessionRequest, *k8s.ResourceRequestBean, error)
	DecodeDevtronAppId(applicationId string) (*bean3.DevtronAppIdentifier, error)
	GetPodLogs(ctx context.Context, request *k8s.ResourceRequestBean) (io.ReadCloser, error)
	ValidateResourceRequest(ctx context.Context, appIdentifier *client.AppIdentifier, request *k8s3.K8sRequestBean) (bool, error)
	ValidateClusterResourceRequest(ctx context.Context, clusterResourceRequest *k8s.ResourceRequestBean,
		rbacCallback func(clusterName string, resourceIdentifier k8s3.ResourceIdentifier) bool) (bool, error)
	ValidateClusterResourceBean(ctx context.Context, clusterId int, manifest unstructured.Unstructured, gvk schema.GroupVersionKind, rbacCallback func(clusterName string, resourceIdentifier k8s3.ResourceIdentifier) bool) bool
	GetResourceInfo(ctx context.Context) (*bean3.ResourceInfo, error)
	GetAllApiResourceGVKWithoutAuthorization(ctx context.Context, clusterId int) (*k8s3.GetAllApiResourcesResponse, error)
	GetAllApiResources(ctx context.Context, clusterId int, isSuperAdmin bool, userId int32, token string) (*k8s3.GetAllApiResourcesResponse, error)
	GetResourceList(ctx context.Context, token string, request *k8s.ResourceRequestBean, validateResourceAccess func(token string, clusterName string, request k8s.ResourceRequestBean, casbinAction string) bool) (*k8s3.ClusterResourceListMap, error)
	ApplyResources(ctx context.Context, token string, request *k8s3.ApplyResourcesRequest, resourceRbacHandler func(token string, clusterName string, request k8s.ResourceRequestBean, casbinAction string) bool) ([]*k8s3.ApplyResourcesResponse, error)
	CreatePodEphemeralContainers(req *cluster.EphemeralContainerRequest) error
	TerminatePodEphemeralContainer(req cluster.EphemeralContainerRequest) (bool, error)
	GetPodContainersList(clusterId int, namespace, podName string) (*k8s.PodContainerList, error)
	GetPodListByLabel(clusterId int, namespace, label string) ([]apiV1.Pod, error)
	RecreateResource(ctx context.Context, request *k8s.ResourceRequestBean) (*k8s3.ManifestResponse, error)
	DeleteResourceWithAudit(ctx context.Context, request *k8s.ResourceRequestBean, userId int32) (*k8s3.ManifestResponse, error)
	GetUrlsByBatchForIngress(ctx context.Context, resp []k8s.BatchResourceResponse) []interface{}
	ValidateK8sResourceForCluster(token string, resourceName string, namespace string, resourceGVK schema.GroupVersionKind, rbacForResource func(token string, clusterName string, resourceIdentifier k8s3.ResourceIdentifier, casbinAction string) bool, clusterName string, resourceAction string) bool
	ValidateK8sResourceAccess(token string, clusterId int, namespace string, resourceGVK schema.GroupVersionKind, resourceAction string, podName string, rbacForResource func(token string, clusterName string, resourceIdentifier k8s3.ResourceIdentifier, casbinAction string) bool) (bool, error)
	GetScoopServiceProxyHandler(ctx context.Context, clusterId int) (*httputil.ReverseProxy, ScoopServiceClusterConfig, error)
	PortForwarding(ctx context.Context, clusterId int, serviceName string, namespace string, port string) (*httputil.ReverseProxy, error)
	StartProxyServer(ctx context.Context, clusterId int) (*httputil.ReverseProxy, error)
	GetClusterForK8sProxy(request *bean3.K8sProxyRequest) (*repository.Cluster, error)
	GetScoopPort(ctx context.Context, clusterId int) (int, ScoopServiceClusterConfig, error)
}

type K8sApplicationServiceImpl struct {
	logger                       *zap.SugaredLogger
	clusterService               cluster.ClusterService
	pump                         connector.Pump
	helmAppService               client.HelmAppService
	K8sUtil                      *k8s2.K8sUtilExtended
	aCDAuthConfig                *util3.ACDAuthConfig
	K8sResourceHistoryService    kubernetesResourceAuditLogs.K8sResourceHistoryService
	k8sCommonService             k8s.K8sCommonService
	terminalSession              terminal.TerminalSessionHandler
	ephemeralContainerService    cluster.EphemeralContainerService
	ephemeralContainerRepository repository.EphemeralContainersRepository
	environmentRepository        repository.EnvironmentRepository
	clusterRepository            repository.ClusterRepository
	k8sAppConfig                 *K8sAppConfig
	argoApplicationService       argoApplication.ArgoApplicationService

	// nil for EA mode
	deploymentWindowService                 deploymentWindow.DeploymentWindowService
	celEvaluatorService                     resourceFilter.CELEvaluatorService
	printers                                *printers.HumanReadableGenerator
	interClusterServiceCommunicationHandler InterClusterServiceCommunicationHandler
	scoopClusterServiceMap                  map[int]ScoopServiceClusterConfig
}

func NewK8sApplicationServiceImpl(logger *zap.SugaredLogger, clusterService cluster.ClusterService, pump connector.Pump, helmAppService client.HelmAppService, K8sUtil *k8s2.K8sUtilExtended, aCDAuthConfig *util3.ACDAuthConfig, K8sResourceHistoryService kubernetesResourceAuditLogs.K8sResourceHistoryService,
	k8sCommonService k8s.K8sCommonService, terminalSession terminal.TerminalSessionHandler,
	ephemeralContainerService cluster.EphemeralContainerService,
	ephemeralContainerRepository repository.EphemeralContainersRepository,
	environmentRepository repository.EnvironmentRepository,
	clusterRepository repository.ClusterRepository,
	argoApplicationService argoApplication.ArgoApplicationService,
	celEvaluatorService resourceFilter.CELEvaluatorService, interClusterServiceCommunicationHandler InterClusterServiceCommunicationHandler,
	deploymentWindowService deploymentWindow.DeploymentWindowService) (*K8sApplicationServiceImpl, error) {
	k8sAppConfig := &K8sAppConfig{}
	err := env.Parse(k8sAppConfig)
	if err != nil {
		logger.Errorw("error in parsing K8sAppConfig from env", "err", err)
		return nil, err
	}
	scoopConfig := make(map[int]ScoopServiceClusterConfig)
	scoopClusterConfig := k8sAppConfig.ScoopClusterConfig
	err = json.Unmarshal([]byte(scoopClusterConfig), &scoopConfig)
	if err != nil {
		logger.Warnw("error occurred while parsing scoop cluster config", "scoopClusterConfig", scoopClusterConfig, "err", err)
	}
	printers := printers.NewTableGenerator()
	util4.AddHandlers(printers)
	return &K8sApplicationServiceImpl{
		logger:                                  logger,
		clusterService:                          clusterService,
		pump:                                    pump,
		helmAppService:                          helmAppService,
		K8sUtil:                                 K8sUtil,
		aCDAuthConfig:                           aCDAuthConfig,
		K8sResourceHistoryService:               K8sResourceHistoryService,
		k8sCommonService:                        k8sCommonService,
		terminalSession:                         terminalSession,
		ephemeralContainerService:               ephemeralContainerService,
		ephemeralContainerRepository:            ephemeralContainerRepository,
		environmentRepository:                   environmentRepository,
		clusterRepository:                       clusterRepository,
		k8sAppConfig:                            k8sAppConfig,
		argoApplicationService:                  argoApplicationService,
		deploymentWindowService:                 deploymentWindowService,
		celEvaluatorService:                     celEvaluatorService,
		printers:                                printers,
		interClusterServiceCommunicationHandler: interClusterServiceCommunicationHandler,
		scoopClusterServiceMap:                  scoopConfig,
	}, nil
}

type K8sAppConfig struct {
	EphemeralServerVersionRegex string `env:"EPHEMERAL_SERVER_VERSION_REGEX" envDefault:"v[1-9]\\.\\b(2[3-9]|[3-9][0-9])\\b.*"`
	ScoopClusterConfig          string `env:"SCOOP_CLUSTER_CONFIG" envDefault:"{}"`
	UseResourceListV2           bool   `env:"USE_RESOURCE_LIST_V2"`
}

type ScoopServiceClusterConfig struct {
	ServiceName string `json:"serviceName"`
	Namespace   string `json:"namespace"`
	PassKey     string `json:"passKey"`
	Port        string `json:"port"`
}

func (impl *K8sApplicationServiceImpl) ValidatePodLogsRequestQuery(r *http.Request) (*k8s.ResourceRequestBean, error) {
	v, vars := r.URL.Query(), mux.Vars(r)
	request := &k8s.ResourceRequestBean{}
	var err error
	request.ExternalArgoApplicationName = v.Get("externalArgoApplicationName")
	appTypeStr := v.Get("appType")
	var appType int
	if len(appTypeStr) > 0 {
		appType, err = strconv.Atoi(appTypeStr)
		if err != nil {
			return nil, &util.ApiError{
				Code:            "400",
				HttpStatusCode:  400,
				UserMessage:     "invalid param: appType",
				InternalMessage: "invalid param: appType",
			}
		}
	}
	request.AppType = appType
	podName := vars["podName"]
	sinceSecondsParam := v.Get("sinceSeconds")
	var sinceSeconds int
	if len(sinceSecondsParam) > 0 {
		sinceSeconds, err = strconv.Atoi(sinceSecondsParam)
		if err != nil || sinceSeconds <= 0 {
			return nil, &util.ApiError{
				Code:            "400",
				HttpStatusCode:  400,
				UserMessage:     "invalid value provided for sinceSeconds",
				InternalMessage: "invalid value provided for sinceSeconds"}
		}
	}
	sinceTimeParam := v.Get("sinceTime")
	sinceTime := metav1.Unix(0, 0)
	if len(sinceTimeParam) > 0 {
		sinceTimeVar, err := strconv.ParseInt(sinceTimeParam, 10, 64)
		if err != nil || sinceTimeVar <= 0 {
			return nil, &util.ApiError{
				Code:            "400",
				HttpStatusCode:  400,
				UserMessage:     "invalid value provided for sinceTime",
				InternalMessage: "invalid value provided for sinceTime"}
		}
		sinceTime = metav1.Unix(sinceTimeVar, 0)
	}
	containerName, clusterIdString := v.Get("containerName"), v.Get("clusterId")
	prevContainerLogs := v.Get("previous")
	isPrevLogs, err := strconv.ParseBool(prevContainerLogs)
	if err != nil {
		isPrevLogs = false
	}
	appId := v.Get("appId")
	follow, err := strconv.ParseBool(v.Get("follow"))
	if err != nil {
		follow = false
	}
	tailLinesParam := v.Get("tailLines")
	var tailLines int
	if len(tailLinesParam) > 0 {
		tailLines, err = strconv.Atoi(tailLinesParam)
		if err != nil || tailLines <= 0 {
			return nil, &util.ApiError{
				Code:            "400",
				HttpStatusCode:  400,
				UserMessage:     "invalid value provided for tailLines",
				InternalMessage: "invalid value provided for tailLines"}
		}
	}
	k8sRequest := &k8s3.K8sRequestBean{
		ResourceIdentifier: k8s3.ResourceIdentifier{
			Name:             podName,
			GroupVersionKind: schema.GroupVersionKind{},
		},
		PodLogsRequest: k8s3.PodLogsRequest{
			SinceSeconds:               sinceSeconds,
			SinceTime:                  &sinceTime,
			TailLines:                  tailLines,
			Follow:                     follow,
			ContainerName:              containerName,
			IsPrevContainerLogsEnabled: isPrevLogs,
		},
	}
	request.K8sRequest = k8sRequest
	if appId != "" {
		if len(appTypeStr) > 0 && !(appType == bean3.DevtronAppType || appType == bean3.HelmAppType || appType == bean3.ArgoAppType) {
			impl.logger.Errorw("Invalid appType", "err", err, "appType", appType)
			return nil, err
		}
		// Validate Deployment Type
		deploymentType, err := strconv.Atoi(v.Get("deploymentType"))
		if err != nil || !(deploymentType == bean3.HelmInstalledType || deploymentType == bean3.ArgoInstalledType) {
			impl.logger.Errorw("Invalid deploymentType", "err", err, "deploymentType", deploymentType)
			return nil, err
		}
		request.DeploymentType = deploymentType
		// Validate App Id
		if request.AppType == bean3.HelmAppType {
			// For Helm App resources
			appIdentifier, err := impl.helmAppService.DecodeAppId(appId)
			if err != nil {
				impl.logger.Errorw("error in decoding appId", "err", err, "appId", appId)
				return nil, err
			}
			request.AppIdentifier = appIdentifier
			request.ClusterId = appIdentifier.ClusterId
			request.K8sRequest.ResourceIdentifier.Namespace = appIdentifier.Namespace
		} else if request.AppType == bean3.DevtronAppType {
			// For Devtron App resources
			devtronAppIdentifier, err := impl.DecodeDevtronAppId(appId)
			if err != nil {
				impl.logger.Errorw("error in decoding appId", "err", err, "appId", request.AppId)
				return nil, err
			}
			request.DevtronAppIdentifier = devtronAppIdentifier
			request.ClusterId = devtronAppIdentifier.ClusterId
			namespace := v.Get("namespace")
			if namespace == "" {
				err = fmt.Errorf("missing required field namespace")
				impl.logger.Errorw("empty namespace", "err", err, "appId", request.AppId)
				return nil, err
			}
			request.K8sRequest.ResourceIdentifier.Namespace = namespace
		}
	} else if clusterIdString != "" {
		// Validate Cluster Id
		clusterId, err := strconv.Atoi(clusterIdString)
		if err != nil {
			impl.logger.Errorw("invalid cluster id", "clusterId", clusterIdString, "err", err)
			return nil, err
		}
		request.ClusterId = clusterId
		namespace := v.Get("namespace")
		if namespace == "" {
			err = fmt.Errorf("missing required field namespace")
			impl.logger.Errorw("empty namespace", "err", err, "appId", request.AppId)
			return nil, err
		}
		request.K8sRequest.ResourceIdentifier.Namespace = namespace
		request.K8sRequest.ResourceIdentifier.GroupVersionKind = schema.GroupVersionKind{
			Group:   "",
			Kind:    "Pod",
			Version: "v1",
		}
	}
	return request, nil
}

func (impl *K8sApplicationServiceImpl) ValidateTerminalRequestQuery(r *http.Request) (*terminal.TerminalSessionRequest, *k8s.ResourceRequestBean, error) {
	request := &terminal.TerminalSessionRequest{}
	v := r.URL.Query()
	vars := mux.Vars(r)
	request.ContainerName = vars["container"]
	request.Namespace = vars["namespace"]
	request.PodName = vars["pod"]
	request.Shell = vars["shell"]
	resourceRequestBean := &k8s.ResourceRequestBean{}
	resourceRequestBean.ExternalArgoApplicationName = v.Get("externalArgoApplicationName")
	identifier := vars["identifier"]
	if strings.Contains(identifier, "|") {
		// Validate App Type
		appType, err := strconv.Atoi(v.Get("appType"))
		if err != nil || appType < bean3.DevtronAppType && appType > bean3.HelmAppType {
			impl.logger.Errorw("Invalid appType", "err", err, "appType", appType)
			return nil, nil, err
		}
		request.ApplicationId = identifier
		if appType == bean3.HelmAppType {
			appIdentifier, err := impl.helmAppService.DecodeAppId(request.ApplicationId)
			if err != nil {
				impl.logger.Errorw("invalid app id", "err", err, "appId", request.ApplicationId)
				return nil, nil, err
			}
			resourceRequestBean.AppIdentifier = appIdentifier
			resourceRequestBean.ClusterId = appIdentifier.ClusterId
			request.ClusterId = appIdentifier.ClusterId
		} else if appType == bean3.DevtronAppType {
			devtronAppIdentifier, err := impl.DecodeDevtronAppId(request.ApplicationId)
			if err != nil {
				impl.logger.Errorw("invalid app id", "err", err, "appId", request.ApplicationId)
				return nil, nil, err
			}
			resourceRequestBean.DevtronAppIdentifier = devtronAppIdentifier
			resourceRequestBean.ClusterId = devtronAppIdentifier.ClusterId
			request.ClusterId = devtronAppIdentifier.ClusterId
		}
	} else {
		// Validate Cluster Id
		clsuterId, err := strconv.Atoi(identifier)
		if err != nil || clsuterId <= 0 {
			impl.logger.Errorw("Invalid cluster id", "err", err, "clusterId", identifier)
			return nil, nil, err
		}
		resourceRequestBean.ClusterId = clsuterId
		request.ClusterId = clsuterId
		k8sRequest := &k8s3.K8sRequestBean{
			ResourceIdentifier: k8s3.ResourceIdentifier{
				Name:      request.PodName,
				Namespace: request.Namespace,
				GroupVersionKind: schema.GroupVersionKind{
					Group:   "",
					Kind:    "Pod",
					Version: "v1",
				},
			},
		}
		resourceRequestBean.K8sRequest = k8sRequest
	}
	return request, resourceRequestBean, nil
}

func (impl *K8sApplicationServiceImpl) DecodeDevtronAppId(applicationId string) (*bean3.DevtronAppIdentifier, error) {
	component := strings.Split(applicationId, "|")
	if len(component) != 3 {
		return nil, fmt.Errorf("malformed app id %s", applicationId)
	}
	clusterId, err := strconv.Atoi(component[0])
	if err != nil {
		return nil, err
	}
	appId, err := strconv.Atoi(component[1])
	if err != nil {
		return nil, err
	}
	envId, err := strconv.Atoi(component[2])
	if err != nil {
		return nil, err
	}
	if clusterId <= 0 || appId <= 0 || envId <= 0 {
		return nil, fmt.Errorf("invalid app identifier")
	}
	return &bean3.DevtronAppIdentifier{
		ClusterId: clusterId,
		AppId:     appId,
		EnvId:     envId,
	}, nil
}

func (impl *K8sApplicationServiceImpl) GetPodLogs(ctx context.Context, request *k8s.ResourceRequestBean) (io.ReadCloser, error) {
	clusterId := request.ClusterId
	resourceIdentifier := request.K8sRequest.ResourceIdentifier
	podLogsRequest := request.K8sRequest.PodLogsRequest
	var restConfigFinal *rest.Config
	if len(request.ExternalArgoApplicationName) > 0 {
		restConfig, err := impl.argoApplicationService.GetRestConfigForExternalArgo(ctx, clusterId, request.ExternalArgoApplicationName)
		if err != nil {
			impl.logger.Errorw("error in getting rest config", "err", err, "clusterId", clusterId, "externalArgoApplicationName", request.ExternalArgoApplicationName)
		}
		restConfigFinal = restConfig
	} else {
		restConfig, err, _ := impl.k8sCommonService.GetRestConfigByClusterId(ctx, clusterId)
		if err != nil {
			impl.logger.Errorw("error in getting rest config by cluster Id", "err", err, "clusterId", clusterId)
			return nil, err
		}
		restConfigFinal = restConfig
	}
	resp, err := impl.K8sUtil.GetPodLogs(ctx, restConfigFinal, resourceIdentifier.Name, resourceIdentifier.Namespace, podLogsRequest.SinceTime, podLogsRequest.TailLines, podLogsRequest.SinceSeconds, podLogsRequest.Follow, podLogsRequest.ContainerName, podLogsRequest.IsPrevContainerLogsEnabled)
	if err != nil {
		impl.logger.Errorw("error in getting pod logs", "err", err, "clusterId", clusterId)
		return nil, err
	}
	return resp, nil
}

func (impl *K8sApplicationServiceImpl) ValidateK8sResourceAccess(token string, clusterId int, namespace string, resourceGVK schema.GroupVersionKind, resourceAction string, resourceName string, rbacForResource func(token string, clusterName string, resourceIdentifier k8s3.ResourceIdentifier, casbinAction string) bool) (bool, error) {
	clusterBean, err := impl.clusterService.FindById(clusterId)
	if err != nil {
		impl.logger.Errorw("error in getting clusterBean by cluster Id", "clusterId", clusterId, "err", err)
		return false, err
	}
	clusterName := clusterBean.ClusterName
	return impl.ValidateK8sResourceForCluster(token, resourceName, namespace, resourceGVK, rbacForResource, clusterName, resourceAction), nil
}

func (impl *K8sApplicationServiceImpl) ValidateK8sResourceForCluster(token string, resourceName string, namespace string, resourceGVK schema.GroupVersionKind, rbacForResource func(token string, clusterName string, resourceIdentifier k8s3.ResourceIdentifier, casbinAction string) bool, clusterName string, resourceAction string) bool {
	resourceIdentifier := k8s3.ResourceIdentifier{
		Name:             resourceName,
		Namespace:        namespace,
		GroupVersionKind: resourceGVK,
	}
	return rbacForResource(token, clusterName, resourceIdentifier, resourceAction)
}

func (impl *K8sApplicationServiceImpl) ValidateClusterResourceRequest(ctx context.Context, clusterResourceRequest *k8s.ResourceRequestBean,
	rbacCallback func(clusterName string, resourceIdentifier k8s3.ResourceIdentifier) bool) (bool, error) {
	clusterId := clusterResourceRequest.ClusterId
	clusterBean, err := impl.clusterService.FindById(clusterId)
	if err != nil {
		impl.logger.Errorw("error in getting clusterBean by cluster Id", "clusterId", clusterId, "err", err)
		return false, err
	}
	clusterName := clusterBean.ClusterName
	k8sRequest := clusterResourceRequest.K8sRequest
	respManifest, err := impl.k8sCommonService.GetResource(ctx, clusterResourceRequest)
	if err != nil {
		impl.logger.Errorw("error in getting resource", "err", err, "request", clusterResourceRequest)
		return false, err
	}
	return impl.validateResourceManifest(clusterName, respManifest.ManifestResponse.Manifest, k8sRequest.ResourceIdentifier.GroupVersionKind, rbacCallback), nil
}

func (impl *K8sApplicationServiceImpl) validateResourceManifest(clusterName string, resourceManifest unstructured.Unstructured, gvk schema.GroupVersionKind, rbacCallback func(clusterName string, resourceIdentifier k8s3.ResourceIdentifier) bool) bool {
	validateCallback := func(namespace, group, kind, resourceName string) bool {
		resourceIdentifier := k8s3.ResourceIdentifier{
			Name:      resourceName,
			Namespace: namespace,
			GroupVersionKind: schema.GroupVersionKind{
				Group: group,
				Kind:  kind,
			},
		}
		return rbacCallback(clusterName, resourceIdentifier)
	}
	return impl.K8sUtil.ValidateResource(resourceManifest.Object, gvk, validateCallback)
}

func (impl *K8sApplicationServiceImpl) ValidateClusterResourceBean(ctx context.Context, clusterId int, manifest unstructured.Unstructured, gvk schema.GroupVersionKind, rbacCallback func(clusterName string, resourceIdentifier k8s3.ResourceIdentifier) bool) bool {
	clusterBean, err := impl.clusterService.FindById(clusterId)
	if err != nil {
		impl.logger.Errorw("error in getting clusterBean by cluster Id", "clusterId", clusterId, "err", err)
		return false
	}
	return impl.validateResourceManifest(clusterBean.ClusterName, manifest, gvk, rbacCallback)
}

func (impl *K8sApplicationServiceImpl) ValidateResourceRequest(ctx context.Context, appIdentifier *client.AppIdentifier, request *k8s3.K8sRequestBean) (bool, error) {
	app, err := impl.helmAppService.GetApplicationDetail(ctx, appIdentifier)
	if err != nil {
		impl.logger.Errorw("error in getting app detail", "err", err, "appDetails", appIdentifier)
		return false, err
	}
	valid := false
	for _, node := range app.ResourceTreeResponse.Nodes {
		nodeDetails := k8s3.ResourceIdentifier{
			Name:      node.Name,
			Namespace: node.Namespace,
			GroupVersionKind: schema.GroupVersionKind{
				Group:   node.Group,
				Version: node.Version,
				Kind:    node.Kind,
			},
		}
		if nodeDetails == request.ResourceIdentifier {
			valid = true
			break
		}
	}
	return impl.validateContainerNameIfReqd(valid, request, app), nil
}

func (impl *K8sApplicationServiceImpl) validateContainerNameIfReqd(valid bool, request *k8s3.K8sRequestBean, app *gRPC.AppDetail) bool {
	if !valid {
		requestContainerName := request.PodLogsRequest.ContainerName
		podName := request.ResourceIdentifier.Name
		for _, pod := range app.ResourceTreeResponse.PodMetadata {
			if pod.Name == podName {

				// finding the container name in main Containers
				for _, container := range pod.Containers {
					if container == requestContainerName {
						return true
					}
				}

				// finding the container name in init containers
				for _, initContainer := range pod.InitContainers {
					if initContainer == requestContainerName {
						return true
					}
				}

				// finding the container name in ephemeral containers
				for _, ephemeralContainer := range pod.EphemeralContainers {
					if ephemeralContainer.Name == requestContainerName {
						return true
					}
				}

			}
		}
	}
	return valid
}

func (impl *K8sApplicationServiceImpl) GetResourceInfo(ctx context.Context) (*bean3.ResourceInfo, error) {
	pod, err := impl.K8sUtil.GetResourceInfoByLabelSelector(ctx, impl.aCDAuthConfig.ACDConfigMapNamespace, "app=inception")
	if err != nil {
		err = &util.ApiError{Code: "404", HttpStatusCode: 404, UserMessage: "error on getting resource from k8s"}
		impl.logger.Errorw("error on getting resource from k8s, unable to fetch installer pod", "err", err)
		return nil, err
	}
	response := &bean3.ResourceInfo{PodName: pod.Name}
	return response, nil
}

// GetAllApiResourceGVKWithoutAuthorization  This function will the all the available api resource GVK list for specific cluster
func (impl *K8sApplicationServiceImpl) GetAllApiResourceGVKWithoutAuthorization(ctx context.Context, clusterId int) (*k8s3.GetAllApiResourcesResponse, error) {
	impl.logger.Infow("getting all api-resources without auth", "clusterId", clusterId)
	restConfig, err, _ := impl.k8sCommonService.GetRestConfigByClusterId(ctx, clusterId)
	if err != nil {
		impl.logger.Errorw("error in getting cluster rest config", "clusterId", clusterId, "err", err)
		return nil, err
	}
	allApiResources, err := impl.K8sUtil.GetApiResources(restConfig, bean3.LIST_VERB)
	if err != nil {
		if client.IsClusterUnReachableError(err) {
			impl.logger.Errorw("k8s cluster unreachable", "err", err)
			return nil, &util.ApiError{HttpStatusCode: http.StatusBadRequest, UserMessage: err.Error()}
		}
		return nil, err
	}
	// FILTER STARTS
	// 1) remove ""/v1 event kind if event kind exist in events.k8s.io/v1 and ""/v1
	k8sEventIndex := -1
	v1EventIndex := -1
	for index, apiResource := range allApiResources {
		gvk := apiResource.Gvk
		if gvk.Kind == bean3.EVENT_K8S_KIND && gvk.Version == "v1" {
			if gvk.Group == "" {
				v1EventIndex = index
			} else if gvk.Group == "events.k8s.io" {
				k8sEventIndex = index
			}
		}
		if gvk.Kind == "Node" {
			allApiResources = append(allApiResources[:index], allApiResources[index+1:]...)
		}
	}
	if k8sEventIndex > -1 && v1EventIndex > -1 {
		allApiResources = append(allApiResources[:v1EventIndex], allApiResources[v1EventIndex+1:]...)
	}
	// FILTER ENDS

	response := &k8s3.GetAllApiResourcesResponse{
		ApiResources: allApiResources,
	}
	return response, nil
}

func (impl *K8sApplicationServiceImpl) GetAllApiResources(ctx context.Context, clusterId int, isSuperAdmin bool, userId int32, token string) (*k8s3.GetAllApiResourcesResponse, error) {
	impl.logger.Infow("getting all api-resources", "clusterId", clusterId)
	apiResourceGVKResponse, err := impl.GetAllApiResourceGVKWithoutAuthorization(ctx, clusterId)
	if err != nil {
		return nil, err
	}
	allApiResources := apiResourceGVKResponse.ApiResources

	// RBAC FILER STARTS
	allowedAll := isSuperAdmin
	filteredApiResources := make([]*k8s3.K8sApiResource, 0)
	if !isSuperAdmin {
		clusterBean, err := impl.clusterService.FindById(clusterId)
		if err != nil {
			impl.logger.Errorw("failed to find cluster for id", "err", err, "clusterId", clusterId)
			return nil, err
		}
		roles, err := impl.clusterService.FetchRolesFromGroup(userId, token)
		if err != nil {
			impl.logger.Errorw("error on fetching user roles for cluster list", "err", err)
			return nil, err
		}

		allowedGroupKinds := make(map[string]bool) // group||kind
		for _, role := range roles {
			if clusterBean.ClusterName != role.Cluster {
				continue
			}
			kind := role.Kind
			if role.Group == "" && kind == "" {
				allowedAll = true
				break
			}
			groupName := role.Group
			if groupName == "" {
				groupName = "*"
			} else if groupName == casbin.ClusterEmptyGroupPlaceholder {
				groupName = ""
			}
			allowedGroupKinds[groupName+"||"+kind] = true
			// add children for this kind
			children, found := k8sCommonBean.KindVsChildrenGvk[kind]
			if found {
				// if rollout kind other than argo, then neglect only
				if kind != k8sCommonBean.K8sClusterResourceRolloutKind || groupName == k8sCommonBean.K8sClusterResourceRolloutGroup {
					for _, child := range children {
						allowedGroupKinds[child.Group+"||"+child.Kind] = true
					}
				}
			}
		}

		if !allowedAll {
			for _, apiResource := range allApiResources {
				gvk := apiResource.Gvk
				_, found := allowedGroupKinds[gvk.Group+"||"+gvk.Kind]
				if found {
					filteredApiResources = append(filteredApiResources, apiResource)
				} else {
					_, found = allowedGroupKinds["*"+"||"+gvk.Kind]
					if found {
						filteredApiResources = append(filteredApiResources, apiResource)
					}
				}
			}
		}
	}
	response := &k8s3.GetAllApiResourcesResponse{
		AllowedAll: allowedAll,
	}
	if allowedAll {
		response.ApiResources = allApiResources
	} else {
		response.ApiResources = filteredApiResources
	}
	// RBAC FILER ENDS

	return response, nil
}

func (impl *K8sApplicationServiceImpl) getResourceListV2(ctx context.Context, token string, request *k8s.ResourceRequestBean, validateResourceAccess func(token string, clusterName string, request k8s.ResourceRequestBean, casbinAction string) bool) (*k8s3.ClusterResourceListMap, error) {
	clusterId := request.ClusterId
	_, err, clusterBean := impl.k8sCommonService.GetRestConfigByClusterId(ctx, clusterId)
	if err != nil {
		impl.logger.Errorw("error in getting rest config by cluster Id", "err", err, "clusterId", request.ClusterId)
		return nil, err
	}
	scoopPort, scoopConfig, err := impl.GetScoopPort(ctx, clusterId)
	if err != nil {
		impl.logger.Errorw("error occurred while fetching scoop port, switching to old API", "request", request, "err", err)
		return impl.getResourceListV1(ctx, token, request, validateResourceAccess)
	}
	scoopServiceUrl := fmt.Sprintf("http://127.0.0.1:%d", scoopPort)
	scoopClient, _ := client2.NewScoopClientImpl(impl.logger, scoopServiceUrl, scoopConfig.PassKey)
	k8sRequest := request.K8sRequest
	resourceIdentifier := k8sRequest.ResourceIdentifier
	scoopK8sRequest := &types2.K8sRequestBean{
		ResourceIdentifier: types2.ResourceIdentifier{GroupVersionKind: resourceIdentifier.GroupVersionKind, Namespace: resourceIdentifier.Namespace},
		Filter:             request.Filter,
		LabelSelector:      request.LabelSelector,
		FieldSelector:      request.FieldSelector,
	}
	resourceList, err := scoopClient.GetResourceList(ctx, scoopK8sRequest)
	if err != nil {
		impl.logger.Errorw("error occurred while fetching resource list from scoop", "request", request, "err", err)
		return nil, errors.New("failed to fetch resource list")
	}

	// store the copy of requested resource identifier
	resourceIdentifierCloned := k8sRequest.ResourceIdentifier
	filteredDataList := make([]map[string]interface{}, 0)
	// containsNameHeader := impl.containsHeader(resourceList.Headers, k8sCommonBean.K8sClusterResourceNameKey)
	// if !containsNameHeader {
	//	impl.logger.Warnw("data does not contains name field, returning empty data", "headers", resourceList.Headers)
	//	resourceList.Data = []map[string]interface{}{}
	//	return resourceList, nil
	// }
	for _, dataRow := range resourceList.Data {
		resourceName := ""
		resourceNamespace := ""
		var metadata map[string]interface{}
		if metadataIntf, ok := dataRow[k8sCommonBean.K8sClusterResourceMetadataKey]; ok {
			if metadata, ok = metadataIntf.(map[string]interface{}); !ok {
				continue
			}
		} else {
			continue
		}

		if resourceNameIntf, ok := metadata[k8sCommonBean.K8sClusterResourceNameKey]; ok {
			resourceName, _ = resourceNameIntf.(string)
		}
		if len(resourceName) == 0 {
			continue
		}
		if resourceNamespaceIntf, ok := metadata[k8sCommonBean.K8sClusterResourceNamespaceKey]; ok {
			resourceNamespace, _ = resourceNamespaceIntf.(string)
		}
		resourceIdentifierCloned.Name = resourceName
		resourceIdentifierCloned.Namespace = resourceNamespace
		k8sRequest.ResourceIdentifier = resourceIdentifierCloned
		allowed := validateResourceAccess(token, clusterBean.ClusterName, *request, casbin.ActionGet)
		if allowed {
			filteredDataList = append(filteredDataList, dataRow)
		} else {
			if ownerRefIntf, ok := metadata[k8sCommonBean.K8sClusterResourceOwnerReferenceKey]; ok {
				if ownerRefs, ok := ownerRefIntf.([]interface{}); ok {
					for _, ownerRef := range ownerRefs {
						allowedResponse := impl.K8sUtil.ValidateForResource(resourceNamespace, ownerRef, func(namespace string, group string, kind string, resourceName string) bool {
							k8sRequest.ResourceIdentifier = k8s3.ResourceIdentifier{Name: resourceName, Namespace: resourceNamespace, GroupVersionKind: schema.GroupVersionKind{Group: group, Kind: kind}}
							return validateResourceAccess(token, clusterBean.ClusterName, *request, casbin.ActionGet)
						})
						if allowedResponse {
							filteredDataList = append(filteredDataList, dataRow)
							break
						}
					}
				}
			}
		}
		delete(dataRow, k8sCommonBean.K8sClusterResourceMetadataKey)
	}
	resourceList.Data = filteredDataList
	k8sServerVersion, err := impl.k8sCommonService.GetK8sServerVersion(clusterId)
	if err != nil {
		impl.logger.Errorw("error in getting k8s server version", "clusterId", clusterId, "err", err)
		// return nil, err
	} else {
		resourceList.ServerVersion = k8sServerVersion.String()
	}
	return resourceList, nil
}

func (impl *K8sApplicationServiceImpl) GetResourceList(ctx context.Context, token string, request *k8s.ResourceRequestBean, validateResourceAccess func(token string, clusterName string, request k8s.ResourceRequestBean, casbinAction string) bool) (*k8s3.ClusterResourceListMap, error) {
	var resourceList *k8s3.ClusterResourceListMap
	var err error
	if impl.k8sAppConfig.UseResourceListV2 {
		resourceList, err = impl.getResourceListV2(ctx, token, request, validateResourceAccess)
	} else {
		resourceList, err = impl.getResourceListV1(ctx, token, request, validateResourceAccess)
	}
	return resourceList, err
}

func (impl *K8sApplicationServiceImpl) getResourceListV1(ctx context.Context, token string, request *k8s.ResourceRequestBean, validateResourceAccess func(token string, clusterName string, request k8s.ResourceRequestBean, casbinAction string) bool) (*k8s3.ClusterResourceListMap, error) {
	resourceList := &k8s3.ClusterResourceListMap{}
	clusterId := request.ClusterId
	restConfig, err, clusterBean := impl.k8sCommonService.GetRestConfigByClusterId(ctx, clusterId)
	if err != nil {
		impl.logger.Errorw("error in getting rest config by cluster Id", "err", err, "clusterId", request.ClusterId)
		return resourceList, err
	}
	k8sRequest := request.K8sRequest
	// store the copy of requested resource identifier
	resourceIdentifierCloned := k8sRequest.ResourceIdentifier
	listOptions := &metav1.ListOptions{
		TypeMeta: metav1.TypeMeta{
			Kind:       resourceIdentifierCloned.GroupVersionKind.Kind,
			APIVersion: resourceIdentifierCloned.GroupVersionKind.GroupVersion().String(),
		},
	}
	if len(request.LabelSelector) > 0 {
		for _, labelSelector := range request.LabelSelector {
			_, err = labels.Parse(labelSelector)
			if err != nil {
				return resourceList, err
			}
		}
		labelSelectorString := strings.Join(request.LabelSelector, ",")
		listOptions.LabelSelector = labelSelectorString
	}

	if len(request.FieldSelector) > 0 {
		for _, fieldSelector := range request.FieldSelector {
			_, err = fields.ParseSelector(fieldSelector)
			if err != nil {
				return resourceList, err
			}
		}
		FieldSelectorString := strings.Join(request.FieldSelector, ",")
		listOptions.FieldSelector = FieldSelectorString
	}
	asTable := len(request.Filter) == 0
	resp, namespaced, err := impl.K8sUtil.GetResourceList(ctx, restConfig, resourceIdentifierCloned.GroupVersionKind, resourceIdentifierCloned.Namespace, asTable, listOptions)
	if err != nil {
		impl.logger.Errorw("error in getting resource list", "err", err, "request", request)
		return resourceList, err
	}

	for _, item := range resp.Resources.Items {
		item.GetKind()
	}

	resources := resp.Resources
	if len(request.Filter) > 0 {
		filteredResources := unstructured.UnstructuredList{}
		filteredItems := make([]interface{}, 0)
		// resource := unstructured.Unstructured{}
		for _, v := range resp.Resources.Items {
			celRequest := resourceFilter.CELRequest{
				Expression: request.Filter,
				ExpressionMetadata: resourceFilter.ExpressionMetadata{
					Params: []resourceFilter.ExpressionParam{
						{
							ParamName: "self",
							Value:     v.Object,
							Type:      resourceFilter.ParamTypeObject,
						},
					},
				},
			}
			pass, err := impl.celEvaluatorService.EvaluateCELRequest(celRequest)
			if err != nil || !pass {
				continue
			}

			filteredItems = append(filteredItems, interface{}(v.Object))
			// resource = v
		}
		items := map[string]interface{}{
			"items": filteredItems,
		}
		filteredResources.SetUnstructuredContent(items)
		filteredResources.DeepCopyObject()
		filteredResources.SetKind(resources.GetKind())
		filteredResources.SetAPIVersion(resources.GetAPIVersion())
		resources = filteredResources
		lst := convertToCore(resources)
		t := &metav1.Table{}
		if lst == nil {
			t, err = convertUnstructuredToTable(resources)
			if err != nil {
				impl.logger.Errorw("error in converting unstructured content to table", "err", err)
			}
		} else {
			t, err = impl.printers.GenerateTable(lst, printers.GenerateOptions{NoHeaders: false})
			if err != nil {
				impl.logger.Errorw("error in generating table", "err", err)
			}
		}

		if err != nil || t == nil {
			resources.Items = nil
		} else {
			m, err := runtime.DefaultUnstructuredConverter.ToUnstructured(t)
			if err != nil {
				impl.logger.Errorw("error in converting table to unstructured data", "err", err)
			}
			resources.SetUnstructuredContent(m)
		}
	}
	checkForResourceCallback := func(namespace, group, kind, resourceName string) bool {
		resourceIdentifier := resourceIdentifierCloned
		resourceIdentifier.Name = resourceName
		resourceIdentifier.Namespace = namespace
		if group != "" && kind != "" {
			resourceIdentifier.GroupVersionKind = schema.GroupVersionKind{Group: group, Kind: kind}
		}
		k8sRequest.ResourceIdentifier = resourceIdentifier
		return validateResourceAccess(token, clusterBean.ClusterName, *request, casbin.ActionGet)
	}

	resourceList, err = impl.K8sUtil.BuildK8sObjectListTableData(&resources, namespaced, request.K8sRequest.ResourceIdentifier.GroupVersionKind, false, checkForResourceCallback)
	if err != nil {
		impl.logger.Errorw("error on parsing for k8s resource", "err", err)
		return resourceList, err
	}
	k8sServerVersion, err := impl.k8sCommonService.GetK8sServerVersion(clusterId)
	if err != nil {
		impl.logger.Errorw("error in getting k8s server version", "clusterId", clusterId, "err", err)
		// return nil, err
	} else {
		resourceList.ServerVersion = k8sServerVersion.String()
	}
	return resourceList, nil
}

func convertToCore(uns unstructured.UnstructuredList) runtime.Object {
	kind := uns.GetObjectKind().GroupVersionKind().Kind

	switch kind {
	case "PodList":
		return convertToCoreList(uns, &apiV1.Pod{}, &apiV1.PodList{})
	case "PodTemplateList":
		return convertToCoreList(uns, &apiV1.PodTemplate{}, &apiV1.PodTemplateList{})
	case "PodDisruptionBudgetList":
		return convertToCoreList(uns, &policyV1beta1.PodDisruptionBudget{}, &policyV1beta1.PodDisruptionBudgetList{})
	case "ReplicationControllerList":
		return convertToCoreList(uns, &apiV1.ReplicationController{}, &apiV1.ReplicationControllerList{})
	case "ReplicaSetList":
		return convertToCoreList(uns, &appsV1.ReplicaSet{}, &appsV1.ReplicaSetList{})
	case "DaemonSetList":
		return convertToCoreList(uns, &appsV1.DaemonSet{}, &appsV1.DaemonSetList{})
	case "JobList":
		return convertToCoreList(uns, &batchv1.Job{}, &batchv1.JobList{})
	case "CronJobList":
		return convertToCoreList(uns, &batchv1.CronJob{}, &batchv1.CronJobList{})
	case "ServiceList":
		return convertToCoreList(uns, &apiV1.Service{}, &apiV1.ServiceList{})
	case "IngressList":
		return convertToCoreList(uns, &networkingV1beta1.Ingress{}, &networkingV1beta1.IngressList{})
	case "IngressClassList":
		return convertToCoreList(uns, &networkingV1beta1.IngressClass{}, &networkingV1beta1.IngressClassList{})
	case "StatefulSetList":
		return convertToCoreList(uns, &appsV1.StatefulSet{}, &appsV1.StatefulSetList{})
	case "EndpointsList":
		return convertToCoreList(uns, &apiV1.Endpoints{}, &apiV1.EndpointsList{})
	case "NodeList":
		return convertToCoreList(uns, &apiV1.Node{}, &apiV1.NodeList{})
	case "EventList":
		return convertToCoreList(uns, &apiV1.Event{}, &apiV1.EventList{})
	case "NamespaceList":
		return convertToCoreList(uns, &apiV1.Namespace{}, &apiV1.NamespaceList{})
	case "SecretList":
		return convertToCoreList(uns, &apiV1.Secret{}, &apiV1.SecretList{})
	case "ServiceAccountList":
		return convertToCoreList(uns, &apiV1.ServiceAccount{}, &apiV1.ServiceAccountList{})
	case "PersistentVolumeList":
		return convertToCoreList(uns, &apiV1.PersistentVolume{}, &apiV1.PersistentVolumeList{})
	case "PersistentVolumeClaimList":
		return convertToCoreList(uns, &apiV1.PersistentVolumeClaim{}, &apiV1.PersistentVolumeClaimList{})
	case "ComponentStatusList":
		return convertToCoreList(uns, &apiV1.ComponentStatus{}, &apiV1.ComponentStatusList{})
	case "DeploymentList":
		return convertToCoreList(uns, &appsV1.Deployment{}, &appsV1.DeploymentList{})
	case "HorizontalPodAutoscalerList":
		return convertToCoreList(uns, &autoscalingV2.HorizontalPodAutoscaler{}, &autoscalingV2.HorizontalPodAutoscalerList{})
	case "ConfigMapList":
		return convertToCoreList(uns, &apiV1.ConfigMap{}, &apiV1.ConfigMapList{})
	case "PodSecurityPolicyList":
		return convertToCoreList(uns, &policyV1beta1.PodSecurityPolicy{}, &policyV1beta1.PodSecurityPolicyList{})
	case "NetworkPolicyList":
		return convertToCoreList(uns, &networkingV1.NetworkPolicy{}, &networkingV1.NetworkPolicyList{})
	case "RoleBindingList":
		return convertToCoreList(uns, &rbac.RoleBinding{}, &rbac.RoleBindingList{})
	case "ClusterRoleBindingList":
		return convertToCoreList(uns, &rbac.ClusterRoleBinding{}, &rbac.ClusterRoleBindingList{})
	case "CertificateSigningRequestList":
		return convertToCoreList(uns, &certificatesV1.CertificateSigningRequest{}, &certificatesV1.CertificateSigningRequestList{})
	case "LeaseList":
		return convertToCoreList(uns, &coordinationV1.Lease{}, &coordinationV1.LeaseList{})
	case "StorageClassList":
		return convertToCoreList(uns, &storageV1beta1.StorageClass{}, &storageV1beta1.StorageClassList{})
	case "ControllerRevisionList":
		return convertToCoreList(uns, &apiV1.ResourceQuota{}, &apiV1.ResourceQuotaList{})
	case "PriorityClassList":
		return convertToCoreList(uns, &schedulingV1.PriorityClass{}, &schedulingV1.PriorityClassList{})
	case "RuntimeClassList":
		return convertToCoreList(uns, &nodeV1.RuntimeClass{}, &nodeV1.RuntimeClassList{})
	case "VolumeAttachmentList":
		return convertToCoreList(uns, &storageV1beta1.VolumeAttachment{}, &storageV1beta1.VolumeAttachmentList{})
	case "EndpointSliceList":
		return convertToCoreList(uns, &discoveryV1beta1.EndpointSlice{}, &discoveryV1beta1.EndpointSliceList{})
	case "CSINodeList":
		return convertToCoreList(uns, &storageV1beta1.CSINode{}, &storageV1beta1.CSINodeList{})
	case "CSIDriverList":
		return convertToCoreList(uns, &storageV1beta1.CSIDriver{}, &storageV1beta1.CSIDriverList{})
	case "CSIStorageCapacityList":
		return convertToCoreList(uns, &storageV1beta1.CSIStorageCapacity{}, &storageV1beta1.CSIStorageCapacityList{})
	case "MutatingWebhookConfigurationList":
		return convertToCoreList(uns, &admissionregistrationV1beta1.MutatingWebhookConfiguration{}, &admissionregistrationV1beta1.MutatingWebhookConfigurationList{})
	case "ValidatingWebhookConfigurationList":
		return convertToCoreList(uns, &admissionregistrationV1beta1.ValidatingWebhookConfiguration{}, &admissionregistrationV1beta1.ValidatingWebhookConfigurationList{})
	case "ValidatingAdmissionPolicyList":
		return convertToCoreList(uns, &admissionregistrationV1alpha1.ValidatingAdmissionPolicy{}, &admissionregistrationV1alpha1.ValidatingAdmissionPolicyList{})
	case "ValidatingAdmissionPolicyBindingList":
		return convertToCoreList(uns, &admissionregistrationV1alpha1.ValidatingAdmissionPolicyBinding{}, &admissionregistrationV1alpha1.ValidatingAdmissionPolicyBindingList{})
	case "FlowSchemaList":
		return convertToCoreList(uns, &flowcontrol.FlowSchema{}, &flowcontrol.FlowSchemaList{})
	case "PriorityLevelConfigurationList":
		return convertToCoreList(uns, &flowcontrol.PriorityLevelConfiguration{}, &flowcontrol.PriorityLevelConfigurationList{})
	case "StorageVersionList":
		return convertToCoreList(uns, &apiserverinternalV1alpha1.StorageVersion{}, &apiserverinternalV1alpha1.StorageVersionList{})
	case "ClusterCIDRList":
		return convertToCoreList(uns, &networkingv1alpha1.ClusterCIDR{}, &networkingv1alpha1.ClusterCIDRList{})
	case "ResourceClassList":
		return convertToCoreList(uns, &resourceV1alpha1.ResourceClass{}, &resourceV1alpha1.ResourceClassList{})
	case "ResourceClaimList":
		return convertToCoreList(uns, &resourceV1alpha1.ResourceClaim{}, &resourceV1alpha1.ResourceClaimList{})
	case "ResourceClaimTemplateList":
		return convertToCoreList(uns, &resourceV1alpha1.ResourceClaimTemplate{}, &resourceV1alpha1.ResourceClaimTemplateList{})
	case "PodSchedulingList":
		return convertToCoreList(uns, &resourceV1alpha1.PodScheduling{}, &resourceV1alpha1.PodSchedulingList{})
	default:
		return nil
	}

}

func convertUnstructuredToTable(uns unstructured.UnstructuredList) (*metav1.Table, error) {
	columnDefinitions := []metav1.TableColumnDefinition{
		{Name: "Name", Type: "string", Format: "name", Description: metav1.ObjectMeta{}.SwaggerDoc()["name"]},
		{Name: "Labels", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["labels"]},
		{Name: "Age", Type: "string", Description: metav1.ObjectMeta{}.SwaggerDoc()["creationTimestamp"]},
	}
	table := metav1.Table{
		TypeMeta: metav1.TypeMeta{
			Kind:       uns.GetKind(),
			APIVersion: uns.GetAPIVersion(),
		},
		ColumnDefinitions: columnDefinitions,
		Rows:              nil,
	}
	rows := make([]metav1.TableRow, 0)
	for _, item := range uns.Items {
		row := metav1.TableRow{
			Object: runtime.RawExtension{Object: &item},
		}
		row.Cells = append(row.Cells, item.GetName(), labels.FormatLabels(item.GetLabels()), translateTimestampSince(item.GetCreationTimestamp()))
		rows = append(rows, row)
	}
	table.Rows = rows
	return &table, nil
}

// translateTimestampSince returns the elapsed time since timestamp in
// human-readable approximation.
func translateTimestampSince(timestamp metav1.Time) string {
	if timestamp.IsZero() {
		return "<unknown>"
	}

	return duration.HumanDuration(time.Since(timestamp.Time))
}

//	convertToCoreList function takes in three parameters:
//
// - uns: an unstructured.UnstructuredList containing a list of unstructured objects
// - itemPtr: a pointer to an instance of a structured object representing an item in the list
// - listPtr: a pointer to an instance of a structured object representing the list
// The function returns a runtime.Object, which represents the structured list.
func convertToCoreList(uns unstructured.UnstructuredList, itemPtr, listPtr interface{}) runtime.Object {
	// Create a new slice to hold the items, based on the type of itemPtr
	items := reflect.New(reflect.SliceOf(reflect.TypeOf(itemPtr).Elem())).Interface()

	// Iterate through each item in the unstructured list
	for _, item := range uns.Items {
		// Create a new instance of the structured item
		obj := reflect.New(reflect.TypeOf(itemPtr).Elem()).Interface()

		// Convert the unstructured item to the structured item
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(item.UnstructuredContent(), obj); err != nil {
			continue
		}

		// Append the converted item to the items slice
		reflect.ValueOf(items).Elem().Set(reflect.Append(reflect.ValueOf(items).Elem(), reflect.ValueOf(obj).Elem()))
	}

	// Create a new instance of the structured list
	list := reflect.New(reflect.TypeOf(listPtr).Elem()).Interface()

	// Get the value of the list instance
	listValue := reflect.ValueOf(list).Elem()

	// Set the TypeMeta field of the list with the metadata from the unstructured list
	listValue.FieldByName("TypeMeta").Set(reflect.ValueOf(metav1.TypeMeta{
		Kind:       uns.GetKind(),
		APIVersion: uns.GetAPIVersion(),
	}))

	// Initialize the Items field of the list as an empty slice
	listValue.FieldByName("Items").Set(reflect.MakeSlice(reflect.SliceOf(reflect.TypeOf(itemPtr).Elem()), 0, 0))

	// Set the Items field of the list with the converted items slice
	listValue.FieldByName("Items").Set(reflect.ValueOf(items).Elem())

	// Return the list as a runtime.Object
	return list.(runtime.Object)
}

func (impl *K8sApplicationServiceImpl) ApplyResources(ctx context.Context, token string, request *k8s3.ApplyResourcesRequest, validateResourceAccess func(token string, clusterName string, request k8s.ResourceRequestBean, casbinAction string) bool) ([]*k8s3.ApplyResourcesResponse, error) {
	manifests, err := yamlUtil.SplitYAMLs([]byte(request.Manifest))
	if err != nil {
		impl.logger.Errorw("error in splitting yaml in manifest", "err", err)
		return nil, err
	}

	// getting rest config by clusterId
	clusterId := request.ClusterId
	restConfig, err, clusterBean := impl.k8sCommonService.GetRestConfigByClusterId(ctx, clusterId)
	if err != nil {
		impl.logger.Errorw("error in getting rest config by cluster", "clusterId", clusterId, "err", err)
		return nil, err
	}

	var response []*k8s3.ApplyResourcesResponse
	for _, manifest := range manifests {
		var namespace string
		manifestNamespace := manifest.GetNamespace()
		if len(manifestNamespace) > 0 {
			namespace = manifestNamespace
		} else {
			namespace = bean3.DEFAULT_NAMESPACE
		}
		manifestRes := &k8s3.ApplyResourcesResponse{
			Name: manifest.GetName(),
			Kind: manifest.GetKind(),
		}
		resourceRequestBean := k8s.ResourceRequestBean{
			ClusterId: clusterId,
			K8sRequest: &k8s3.K8sRequestBean{
				ResourceIdentifier: k8s3.ResourceIdentifier{
					Name:             manifest.GetName(),
					Namespace:        namespace,
					GroupVersionKind: manifest.GroupVersionKind(),
				},
			},
		}
		actionAllowed := validateResourceAccess(token, clusterBean.ClusterName, resourceRequestBean, casbin.ActionUpdate)
		if actionAllowed {
			resourceExists, err := impl.applyResourceFromManifest(ctx, manifest, restConfig, namespace, clusterId)
			manifestRes.IsUpdate = resourceExists
			if err != nil {
				manifestRes.Error = err.Error()
			}
		} else {
			manifestRes.Error = "permission-denied"
		}
		response = append(response, manifestRes)
	}

	return response, nil
}

func (impl *K8sApplicationServiceImpl) applyResourceFromManifest(ctx context.Context, manifest unstructured.Unstructured, restConfig *rest.Config, namespace string, clusterId int) (bool, error) {
	var isUpdateResource bool
	k8sRequestBean := &k8s3.K8sRequestBean{
		ResourceIdentifier: k8s3.ResourceIdentifier{
			Name:             manifest.GetName(),
			Namespace:        namespace,
			GroupVersionKind: manifest.GroupVersionKind(),
		},
	}
	jsonStrByteErr, err := json.Marshal(manifest.UnstructuredContent())
	if err != nil {
		impl.logger.Errorw("error in marshalling json", "err", err)
		return isUpdateResource, err
	}
	jsonStr := string(jsonStrByteErr)
	request := &k8s.ResourceRequestBean{
		K8sRequest: k8sRequestBean,
		ClusterId:  clusterId,
	}

	_, err = impl.k8sCommonService.GetResource(ctx, request)
	if err != nil {
		statusError, ok := err.(*errors2.StatusError)
		if !ok || statusError == nil || statusError.ErrStatus.Reason != metav1.StatusReasonNotFound {
			impl.logger.Errorw("error in getting resource", "err", err)
			return isUpdateResource, err
		}
		resourceIdentifier := k8sRequestBean.ResourceIdentifier
		// case of resource not found
		_, err = impl.K8sUtil.CreateResources(ctx, restConfig, jsonStr, resourceIdentifier.GroupVersionKind, resourceIdentifier.Namespace)
		if err != nil {
			impl.logger.Errorw("error in creating resource", "err", err)
			return isUpdateResource, err
		}
	} else {
		// case of resource update
		isUpdateResource = true
		resourceIdentifier := k8sRequestBean.ResourceIdentifier
		_, err = impl.K8sUtil.PatchResourceRequest(ctx, restConfig, types.StrategicMergePatchType, jsonStr, resourceIdentifier.Name, resourceIdentifier.Namespace, resourceIdentifier.GroupVersionKind)
		if err != nil {
			impl.logger.Errorw("error in updating resource", "err", err)
			return isUpdateResource, err
		}
	}

	return isUpdateResource, nil
}
func (impl *K8sApplicationServiceImpl) CreatePodEphemeralContainers(req *cluster.EphemeralContainerRequest) error {
	var clientSet *kubernetes.Clientset
	var v1Client *v1.CoreV1Client
	var err error
	if len(req.ExternalArgoApplicationName) > 0 {
		clientSet, v1Client, err = impl.k8sCommonService.GetCoreClientByClusterIdForExternalArgoApps(req)
		if err != nil {
			impl.logger.Errorw("error in getting coreV1 client by clusterId", "err", err, "req", req)
			return err
		}
	} else {
		clientSet, v1Client, err = impl.k8sCommonService.GetCoreClientByClusterId(req.ClusterId)
		if err != nil {
			impl.logger.Errorw("error in getting coreV1 client by clusterId", "clusterId", req.ClusterId, "err", err)
			return err
		}
	}
	compatible, err := impl.K8sServerVersionCheckForEphemeralContainers(clientSet)
	if err != nil {
		impl.logger.Errorw("error in checking kubernetes server version compatability for ephemeral containers", "clusterId", req.ClusterId, "err", err)
		return err
	}
	if !compatible {
		return errors.New("This feature is supported on and above Kubernetes v1.23 only.")
	}
	pod, err := impl.K8sUtil.GetPodByName(req.Namespace, req.PodName, v1Client)
	if err != nil {
		impl.logger.Errorw("error in getting pod", "clusterId", req.ClusterId, "namespace", req.Namespace, "podName", req.PodName, "err", err)
		return err
	}

	podJS, err := json.Marshal(pod)
	if err != nil {
		impl.logger.Errorw("error occurred in unMarshaling pod object", "podObject", pod, "err", err)
		return fmt.Errorf("error creating JSON for pod: %v", err)
	}
	debugPod, debugContainer, err := impl.generateDebugContainer(pod, *req)
	if err != nil {
		impl.logger.Errorw("error in generateDebugContainer", "request", req, "err", err)
		return err
	}

	debugJS, err := json.Marshal(debugPod)
	if err != nil {
		impl.logger.Errorw("error occurred in unMarshaling debugPod object", "debugPod", debugPod, "err", err)
		return fmt.Errorf("error creating JSON for pod: %v", err)
	}

	patch, err := strategicpatch.CreateTwoWayMergePatch(podJS, debugJS, pod)
	if err != nil {
		impl.logger.Errorw("error occurred in CreateTwoWayMergePatch", "podJS", podJS, "debugJS", debugJS, "pod", pod, "err", err)
		return fmt.Errorf("error creating patch to add debug container: %v", err)
	}

	_, err = v1Client.Pods(req.Namespace).Patch(context.Background(), pod.Name, types.StrategicMergePatchType, patch, metav1.PatchOptions{}, "ephemeralcontainers")
	if err != nil {
		if serr, ok := err.(*errors2.StatusError); ok && serr.Status().Reason == metav1.StatusReasonNotFound && serr.ErrStatus.Details.Name == "" {
			impl.logger.Errorw("error occurred while creating ephemeral containers", "err", err, "reason", "ephemeral containers are disabled for this cluster")
			return fmt.Errorf("ephemeral containers are disabled for this cluster (error from kubernetes server: %q)", err)
		}
		if runtime.IsNotRegisteredError(err) {
			patch, err := json.Marshal([]map[string]interface{}{{
				"op":    "add",
				"path":  "/ephemeralContainers/-",
				"value": debugContainer,
			}})
			if err != nil {
				impl.logger.Errorw("error occured while trying to create epehemral containers with legacy API", "err", err)
				return fmt.Errorf("error creating JSON 6902 patch for old /ephemeralcontainers API: %s", err)
			}
			// try with legacy API
			result := v1Client.RESTClient().Patch(types.JSONPatchType).
				Namespace(pod.Namespace).
				Resource("pods").
				Name(pod.Name).
				SubResource("ephemeralcontainers").
				Body(patch).
				Do(context.Background())
			return result.Error()
		}
		return err
	}

	if err == nil {
		debugContainerJs, err := json.Marshal(debugContainer)
		if err != nil {
			impl.logger.Errorw("error occurred in unMarshaling debugContainer object", "debugContainerJs", debugContainer, "err", err)
			return fmt.Errorf("error creating JSON for pod: %v", err)
		}
		req.AdvancedData = &cluster.EphemeralContainerAdvancedData{
			Manifest: string(debugContainerJs),
		}
		req.BasicData = &cluster.EphemeralContainerBasicData{
			ContainerName:       debugContainer.Name,
			TargetContainerName: debugContainer.TargetContainerName,
			Image:               debugContainer.Image,
		}
		err = impl.ephemeralContainerService.AuditEphemeralContainerAction(*req, repository.ActionCreate)
		if err != nil {
			impl.logger.Errorw("error in saving ephemeral container data", "err", err)
			return err
		}
		return nil
	}

	impl.logger.Errorw("error in creating ephemeral containers ", "err", err, "clusterId", req.ClusterId, "namespace", req.Namespace, "podName", req.PodName, "ephemeralContainerSpec", debugContainer)
	return err
}

func (impl *K8sApplicationServiceImpl) generateDebugContainer(pod *apiV1.Pod, req cluster.EphemeralContainerRequest) (*apiV1.Pod, *apiV1.EphemeralContainer, error) {
	copied := pod.DeepCopy()
	ephemeralContainer := &apiV1.EphemeralContainer{}
	if req.AdvancedData != nil {
		err := json.Unmarshal([]byte(req.AdvancedData.Manifest), ephemeralContainer)
		if err != nil {
			impl.logger.Errorw("error occurred in unMarshaling advanced ephemeral data", "err", err, "advancedData", req.AdvancedData.Manifest)
			return copied, ephemeralContainer, err
		}
		if ephemeralContainer.TargetContainerName == "" || ephemeralContainer.Name == "" || ephemeralContainer.Image == "" {
			return copied, ephemeralContainer, errors.New("containerName,targetContainerName and image cannot be empty")
		}
		if len(ephemeralContainer.Command) > 0 {
			return copied, ephemeralContainer, errors.New("Command field is not supported, please remove command and try again")
		}
	} else {
		ephemeralContainer = &apiV1.EphemeralContainer{
			EphemeralContainerCommon: apiV1.EphemeralContainerCommon{
				Name:                     req.BasicData.ContainerName,
				Env:                      nil,
				Image:                    req.BasicData.Image,
				ImagePullPolicy:          apiV1.PullIfNotPresent,
				Stdin:                    true,
				TerminationMessagePolicy: apiV1.TerminationMessageReadFile,
				TTY:                      true,
			},
			TargetContainerName: req.BasicData.TargetContainerName,
		}
	}
	ephemeralContainer.Name = ephemeralContainer.Name + "-" + util2.Generate(5)
	scriptCreateCommand := fmt.Sprintf("echo 'while true; do sleep 600; done;' > "+k8sObjectUtils.EphemeralContainerStartingShellScriptFileName, ephemeralContainer.Name)
	scriptRunCommand := fmt.Sprintf("sh "+k8sObjectUtils.EphemeralContainerStartingShellScriptFileName, ephemeralContainer.Name)
	ephemeralContainer.Command = []string{"sh", "-c", scriptCreateCommand + " && " + scriptRunCommand}
	copied.Spec.EphemeralContainers = append(copied.Spec.EphemeralContainers, *ephemeralContainer)
	ephemeralContainer = &copied.Spec.EphemeralContainers[len(copied.Spec.EphemeralContainers)-1]
	return copied, ephemeralContainer, nil

}

func (impl *K8sApplicationServiceImpl) TerminatePodEphemeralContainer(req cluster.EphemeralContainerRequest) (bool, error) {
	terminalReq := &terminal.TerminalSessionRequest{
		PodName:                     req.PodName,
		ClusterId:                   req.ClusterId,
		Namespace:                   req.Namespace,
		ContainerName:               req.BasicData.ContainerName,
		ExternalArgoApplicationName: req.ExternalArgoApplicationName,
	}
	container, err := impl.ephemeralContainerRepository.FindContainerByName(terminalReq.ClusterId, terminalReq.Namespace, terminalReq.PodName, terminalReq.ContainerName)
	if err != nil {
		impl.logger.Errorw("error in finding ephemeral container in the database", "err", err, "ClusterId", terminalReq.ClusterId, "Namespace", terminalReq.Namespace, "PodName", terminalReq.PodName, "ContainerName", terminalReq.ContainerName)
		return false, err
	}
	if container == nil {
		return false, errors.New("externally created ephemeral containers cannot be removed")
	}
	containerKillCommand := fmt.Sprintf("kill -16 $(pgrep -f '%s' -o)", fmt.Sprintf(k8sObjectUtils.EphemeralContainerStartingShellScriptFileName, terminalReq.ContainerName))
	cmds := []string{"sh", "-c", containerKillCommand}
	_, errBuf, err := impl.terminalSession.RunCmdInRemotePod(terminalReq, cmds)
	if err != nil {
		impl.logger.Errorw("failed to execute commands ", "err", err, "commands", cmds, "podName", req.PodName, "namespace", req.Namespace)
		return false, err
	}
	errBufString := errBuf.String()
	if errBufString != "" {
		impl.logger.Errorw("error response on executing commands ", "err", errBufString, "commands", cmds, "podName", req.Namespace, "namespace", req.Namespace)
		return false, err
	}

	if err == nil {

		err = impl.ephemeralContainerService.AuditEphemeralContainerAction(req, repository.ActionTerminate)
		if err != nil {
			impl.logger.Errorw("error in saving ephemeral container data", "err", err)
			return true, err
		}

	}

	return true, nil
}

func (impl *K8sApplicationServiceImpl) GetPodContainersList(clusterId int, namespace, podName string) (*k8s.PodContainerList, error) {
	_, v1Client, err := impl.k8sCommonService.GetCoreClientByClusterId(clusterId)
	if err != nil {
		impl.logger.Errorw("error in getting coreV1 client by clusterId", "clusterId", clusterId, "err", err)
		return nil, err
	}
	pod, err := impl.K8sUtil.GetPodByName(namespace, podName, v1Client)
	if err != nil {
		impl.logger.Errorw("error in getting pod", "clusterId", clusterId, "namespace", namespace, "podName", podName, "err", err)
		return nil, err
	}
	ephemeralContainerStatusMap := make(map[string]bool)
	for _, c := range pod.Status.EphemeralContainerStatuses {
		// c.state contains three states running,waiting and terminated
		// at any point of time only one state will be there
		if c.State.Running != nil {
			ephemeralContainerStatusMap[c.Name] = true
		}
	}
	containers := make([]string, len(pod.Spec.Containers))
	initContainers := make([]string, len(pod.Spec.InitContainers))
	ephemeralContainers := make([]string, 0, len(pod.Spec.EphemeralContainers))

	for i, c := range pod.Spec.Containers {
		containers[i] = c.Name
	}

	for _, ec := range pod.Spec.EphemeralContainers {
		if _, ok := ephemeralContainerStatusMap[ec.Name]; ok {
			ephemeralContainers = append(ephemeralContainers, ec.Name)
		}
	}

	for i, ic := range pod.Spec.InitContainers {
		initContainers[i] = ic.Name
	}

	return &k8s.PodContainerList{
		Containers:          containers,
		EphemeralContainers: ephemeralContainers,
		InitContainers:      initContainers,
	}, nil
}

func (impl *K8sApplicationServiceImpl) GetPodListByLabel(clusterId int, namespace, label string) ([]apiV1.Pod, error) {
	clientSet, _, err := impl.k8sCommonService.GetCoreClientByClusterId(clusterId)
	if err != nil {
		impl.logger.Errorw("error in getting coreV1 client by clusterId", "clusterId", clusterId, "err", err)
		return nil, err
	}
	pods, err := impl.K8sUtil.GetPodListByLabel(namespace, label, clientSet)
	if err != nil {
		impl.logger.Errorw("error in getting pods list", "clusterId", clusterId, "namespace", namespace, "label", label, "err", err)
		return nil, err
	}
	return pods, err
}

func (impl *K8sApplicationServiceImpl) RecreateResource(ctx context.Context, request *k8s.ResourceRequestBean) (*k8s3.ManifestResponse, error) {
	resourceIdentifier := &openapi.ResourceIdentifier{
		Name:      &request.K8sRequest.ResourceIdentifier.Name,
		Namespace: &request.K8sRequest.ResourceIdentifier.Namespace,
		Group:     &request.K8sRequest.ResourceIdentifier.GroupVersionKind.Group,
		Version:   &request.K8sRequest.ResourceIdentifier.GroupVersionKind.Version,
		Kind:      &request.K8sRequest.ResourceIdentifier.GroupVersionKind.Kind,
	}
	manifestRes, err := impl.helmAppService.GetDesiredManifest(ctx, request.AppIdentifier, resourceIdentifier)
	if err != nil {
		impl.logger.Errorw("error in getting desired manifest for validation", "err", err)
		return nil, err
	}
	manifest, manifestOk := manifestRes.GetManifestOk()
	if manifestOk == false || len(*manifest) == 0 {
		impl.logger.Debugw("invalid request, desired manifest not found", "err", err)
		return nil, fmt.Errorf("no manifest found for this request")
	}

	// getting rest config by clusterId
	restConfig, err, _ := impl.k8sCommonService.GetRestConfigByClusterId(ctx, request.AppIdentifier.ClusterId)
	if err != nil {
		impl.logger.Errorw("error in getting rest config by cluster Id", "err", err, "clusterId", request.AppIdentifier.ClusterId)
		return nil, err
	}
	resp, err := impl.K8sUtil.CreateResources(ctx, restConfig, *manifest, request.K8sRequest.ResourceIdentifier.GroupVersionKind, request.K8sRequest.ResourceIdentifier.Namespace)
	if err != nil {
		impl.logger.Errorw("error in creating resource", "err", err, "request", request)
		return nil, err
	}
	return resp, nil
}

func (impl *K8sApplicationServiceImpl) checkForDeploymentWindow(identifier *bean3.DevtronAppIdentifier, userid int32) error {

	if impl.deploymentWindowService == nil || identifier == nil {
		return nil
	}
	actionState, envState, err := impl.deploymentWindowService.GetStateForAppEnv(time.Now(), identifier.AppId, identifier.EnvId, userid)
	if err != nil {
		return fmt.Errorf("error in getting deployment window state %v", err)
	}
	if !actionState.IsActionAllowedWithBypass() {
		return deploymentWindow.GetActionBlockedError(actionState.GetErrorMessageForProfileAndState(envState), constants.HttpStatusUnprocessableEntity)
	}
	return nil
}

func (impl *K8sApplicationServiceImpl) GetScoopServiceProxyHandler(ctx context.Context, clusterId int) (*httputil.ReverseProxy, ScoopServiceClusterConfig, error) {
	// read scoop service metadata from config
	scoopConfig, ok := impl.scoopClusterServiceMap[clusterId]
	if !ok {
		return nil, scoopConfig, ScoopNotConfiguredErr
	}
	proxyHandler, err := impl.interClusterServiceCommunicationHandler.GetClusterServiceProxyHandler(ctx, NewClusterServiceKey(clusterId, scoopConfig.ServiceName, scoopConfig.Namespace, scoopConfig.Port))
	if err != nil {
		impl.logger.Errorw("error occurred while fetching scoop proxy handler", "clusterId", clusterId, "err", err)
		return nil, scoopConfig, err
	}
	return proxyHandler, scoopConfig, nil
}

func (impl *K8sApplicationServiceImpl) DeleteResourceWithAudit(ctx context.Context, request *k8s.ResourceRequestBean, userId int32) (*k8s3.ManifestResponse, error) {

	err := impl.checkForDeploymentWindow(request.DevtronAppIdentifier, userId)
	if err != nil {
		return nil, err
	}
	resp, err := impl.k8sCommonService.DeleteResource(ctx, request)
	if err != nil {
		if IsResourceNotFoundErr(err) {
			return nil, &utils.ApiError{Code: "404",
				HttpStatusCode:  http.StatusNotFound,
				InternalMessage: err.Error(),
				UserMessage:     k8s.ResourceNotFoundErr}
		}
		impl.logger.Errorw("error in deleting resource", "err", err)
		return nil, err
	}
	if request.AppIdentifier != nil {
		saveAuditLogsErr := impl.K8sResourceHistoryService.SaveHelmAppsResourceHistory(request.AppIdentifier, request.K8sRequest, userId, bean3.Delete)
		if saveAuditLogsErr != nil {
			impl.logger.Errorw("error in saving audit logs for delete resource request", "err", err)
		}
	}

	return resp, nil
}

func (impl *K8sApplicationServiceImpl) GetUrlsByBatchForIngress(ctx context.Context, resp []k8s.BatchResourceResponse) []interface{} {
	result := make([]interface{}, 0)
	for _, res := range resp {
		err := res.Err
		if err != nil {
			continue
		}
		urlRes := getUrls(res.ManifestResponse)
		result = append(result, urlRes)
	}
	return result
}

func getUrls(manifest *k8s3.ManifestResponse) bean3.Response {
	var res bean3.Response
	kind := manifest.Manifest.Object["kind"]
	if _, ok := manifest.Manifest.Object["metadata"]; ok {
		metadata := manifest.Manifest.Object["metadata"].(map[string]interface{})
		if metadata != nil {
			name := metadata["name"]
			if name != nil {
				res.Name = name.(string)
			}
		}
	}

	if kind != nil {
		res.Kind = kind.(string)
	}
	res.PointsTo = ""
	urls := make([]string, 0)
	if res.Kind == k8sCommonBean.IngressKind {
		if manifest.Manifest.Object["spec"] != nil {
			spec := manifest.Manifest.Object["spec"].(map[string]interface{})
			if spec["rules"] != nil {
				rules := spec["rules"].([]interface{})
				for _, rule := range rules {
					ruleMap := rule.(map[string]interface{})
					url := ""
					if ruleMap["host"] != nil {
						url = ruleMap["host"].(string)
					}
					var httpPaths []interface{}
					if ruleMap["http"] != nil && ruleMap["http"].(map[string]interface{})["paths"] != nil {
						httpPaths = ruleMap["http"].(map[string]interface{})["paths"].([]interface{})
					} else {
						continue
					}
					for _, httpPath := range httpPaths {
						path := httpPath.(map[string]interface{})["path"]
						if path != nil {
							url = url + path.(string)
						}
						urls = append(urls, url)
					}
				}
			}
		}
	}

	if manifest.Manifest.Object["status"] != nil {
		status := manifest.Manifest.Object["status"].(map[string]interface{})
		if status["loadBalancer"] != nil {
			loadBalancer := status["loadBalancer"].(map[string]interface{})
			if loadBalancer["ingress"] != nil {
				ingressArray := loadBalancer["ingress"].([]interface{})
				if len(ingressArray) > 0 {
					if hostname, ok := ingressArray[0].(map[string]interface{})["hostname"]; ok {
						res.PointsTo = hostname.(string)
					} else if ip, ok := ingressArray[0].(map[string]interface{})["ip"]; ok {
						res.PointsTo = ip.(string)
					}
				}
			}
		}
	}
	res.Urls = urls
	return res
}

func (impl *K8sApplicationServiceImpl) K8sServerVersionCheckForEphemeralContainers(clientSet *kubernetes.Clientset) (bool, error) {
	k8sServerVersion, err := impl.K8sUtil.GetK8sServerVersion(clientSet)
	if err != nil || k8sServerVersion == nil {
		impl.logger.Errorw("error occurred in getting k8sServerVersion", "err", err)
		return false, err
	}

	// ephemeral containers feature is introduced in version v1.23 of kubernetes, it is stable from version v1.25
	// https://kubernetes.io/docs/concepts/workloads/pods/ephemeral-containers/
	ephemeralRegex := impl.k8sAppConfig.EphemeralServerVersionRegex
	matched, err := util2.MatchRegexExpression(ephemeralRegex, k8sServerVersion.String())
	if err != nil {
		impl.logger.Errorw("error in matching ephemeral containers support version regex with k8sServerVersion", "err", err, "EphemeralServerVersionRegex", ephemeralRegex)
		return false, err
	}
	return matched, nil
}

func (impl *K8sApplicationServiceImpl) PortForwarding(ctx context.Context, clusterId int, serviceName string, namespace string, port string) (*httputil.ReverseProxy, error) {
	impl.logger.Infow("received request for port forwarding", "clusterId", clusterId, "serviceName", serviceName, "namespace", namespace, "port", port)
	proxyHandler, err := impl.interClusterServiceCommunicationHandler.GetClusterServiceProxyHandler(ctx, NewClusterServiceKey(clusterId, serviceName, namespace, port))
	return proxyHandler, err
}

func (impl *K8sApplicationServiceImpl) StartProxyServer(ctx context.Context, clusterId int) (*httputil.ReverseProxy, error) {
	proxyHandler, err := impl.interClusterServiceCommunicationHandler.GetK8sApiProxyHandler(ctx, clusterId)
	return proxyHandler, err
}

func (impl *K8sApplicationServiceImpl) GetClusterForK8sProxy(request *bean3.K8sProxyRequest) (*repository.Cluster, error) {
	clusterID, err := impl.getClusterIDFromIdentifier(request)
	if err != nil {
		impl.logger.Errorw("Error getting clusterId from identifier", "Error:", err)
		return nil, err
	}
	clusterFound, err := impl.clusterRepository.FindById(clusterID)
	if err != nil {
		impl.logger.Errorw("Error finding cluster from clusterId.", "clusterId", clusterID)
		return nil, err
	}
	return clusterFound, nil
}

func (impl *K8sApplicationServiceImpl) getClusterIDFromIdentifier(request *bean3.K8sProxyRequest) (int, error) {
	if request.ClusterId == 0 {
		if request.ClusterName != "" {
			clusterFound, err := impl.clusterRepository.FindOne(request.ClusterName)
			if err != nil {
				impl.logger.Errorw("Error finding clusterId from clusterName.", "clusterName", request.ClusterName)
				return 0, err
			}
			return clusterFound.Id, nil
		} else if request.EnvName != "" {
			environment, err := impl.environmentRepository.FindByName(request.EnvName)
			if err != nil {
				impl.logger.Errorw("Error finding clusterId from envName.", "envName", request.EnvName)
				return 0, err
			}
			return environment.ClusterId, nil
		} else if request.EnvId != 0 {
			environment, err := impl.environmentRepository.FindById(request.EnvId)
			if err != nil {
				impl.logger.Errorw("Error finding clusterId from envId.", "envId", request.EnvId)
				return 0, err
			}
			return environment.ClusterId, nil
		}
	}

	return request.ClusterId, nil
}

func (impl K8sApplicationServiceImpl) GetScoopPort(ctx context.Context, clusterId int) (int, ScoopServiceClusterConfig, error) {
	scoopConfig, ok := impl.scoopClusterServiceMap[clusterId]
	if !ok {
		return 0, scoopConfig, ScoopNotConfiguredErr
	}
	scoopPort, err := impl.interClusterServiceCommunicationHandler.GetClusterServiceProxyPort(ctx, NewClusterServiceKey(clusterId, scoopConfig.ServiceName, scoopConfig.Namespace, scoopConfig.Port))
	if err != nil {
		impl.logger.Errorw("error in getting a service proxy port for scoop", "clusterId", clusterId, "err", err)
		return 0, scoopConfig, err
	}
	return scoopPort, scoopConfig, nil
	// return 8081, ScoopServiceClusterConfig{PassKey: "abcd"}, nil
}
