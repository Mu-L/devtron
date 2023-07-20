package application

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/devtron-labs/devtron/api/connector"
	client "github.com/devtron-labs/devtron/api/helm-app"
	"github.com/devtron-labs/devtron/pkg/cluster"
	"github.com/devtron-labs/devtron/pkg/k8s"
	bean3 "github.com/devtron-labs/devtron/pkg/k8s/application/bean"
	"github.com/devtron-labs/devtron/pkg/kubernetesResourceAuditLogs"
	"github.com/devtron-labs/devtron/pkg/terminal"
	"github.com/devtron-labs/devtron/pkg/user/casbin"
	util3 "github.com/devtron-labs/devtron/pkg/util"
	k8s2 "github.com/devtron-labs/devtron/util/k8s"
	yamlUtil "github.com/devtron-labs/devtron/util/yaml"
	"github.com/gorilla/mux"
	"go.uber.org/zap"
	"io"
	errors2 "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"net/http"
	"strconv"
	"strings"
)

type K8sApplicationService interface {
	ValidatePodLogsRequestQuery(r *http.Request) (*k8s.ResourceRequestBean, error)
	ValidateTerminalRequestQuery(r *http.Request) (*terminal.TerminalSessionRequest, *k8s.ResourceRequestBean, error)
	DecodeDevtronAppId(applicationId string) (*bean3.DevtronAppIdentifier, error)
	GetPodLogs(ctx context.Context, request *k8s.ResourceRequestBean) (io.ReadCloser, error)
	ValidateResourceRequest(ctx context.Context, appIdentifier *client.AppIdentifier, request *k8s2.K8sRequestBean) (bool, error)
	ValidateClusterResourceRequest(ctx context.Context, clusterResourceRequest *k8s.ResourceRequestBean,
		rbacCallback func(clusterName string, resourceIdentifier k8s2.ResourceIdentifier) bool) (bool, error)
	ValidateClusterResourceBean(ctx context.Context, clusterId int, manifest unstructured.Unstructured, gvk schema.GroupVersionKind, rbacCallback func(clusterName string, resourceIdentifier k8s2.ResourceIdentifier) bool) bool
	GetResourceInfo(ctx context.Context) (*bean3.ResourceInfo, error)
	GetAllApiResources(ctx context.Context, clusterId int, isSuperAdmin bool, userId int32) (*k8s2.GetAllApiResourcesResponse, error)
	GetResourceList(ctx context.Context, token string, request *k8s.ResourceRequestBean, validateResourceAccess func(token string, clusterName string, request k8s.ResourceRequestBean, casbinAction string) bool) (*k8s2.ClusterResourceListMap, error)
	ApplyResources(ctx context.Context, token string, request *k8s2.ApplyResourcesRequest, resourceRbacHandler func(token string, clusterName string, request k8s.ResourceRequestBean, casbinAction string) bool) ([]*k8s2.ApplyResourcesResponse, error)
}
type K8sApplicationServiceImpl struct {
	logger                    *zap.SugaredLogger
	clusterService            cluster.ClusterService
	pump                      connector.Pump
	helmAppService            client.HelmAppService
	K8sUtil                   *k8s2.K8sUtil
	aCDAuthConfig             *util3.ACDAuthConfig
	K8sResourceHistoryService kubernetesResourceAuditLogs.K8sResourceHistoryService
	k8sCommonService          k8s.K8sCommonService
}

func NewK8sApplicationServiceImpl(Logger *zap.SugaredLogger, clusterService cluster.ClusterService, pump connector.Pump, helmAppService client.HelmAppService, K8sUtil *k8s2.K8sUtil, aCDAuthConfig *util3.ACDAuthConfig, K8sResourceHistoryService kubernetesResourceAuditLogs.K8sResourceHistoryService, k8sCommonService k8s.K8sCommonService) *K8sApplicationServiceImpl {

	return &K8sApplicationServiceImpl{
		logger:                    Logger,
		clusterService:            clusterService,
		pump:                      pump,
		helmAppService:            helmAppService,
		K8sUtil:                   K8sUtil,
		aCDAuthConfig:             aCDAuthConfig,
		K8sResourceHistoryService: K8sResourceHistoryService,
		k8sCommonService:          k8sCommonService,
	}
}

func (impl *K8sApplicationServiceImpl) ValidatePodLogsRequestQuery(r *http.Request) (*k8s.ResourceRequestBean, error) {
	v, vars := r.URL.Query(), mux.Vars(r)
	request := &k8s.ResourceRequestBean{}
	podName := vars["podName"]
	/*sinceSeconds, err := strconv.Atoi(v.Get("sinceSeconds"))
	if err != nil {
		sinceSeconds = 0
	}*/
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
	tailLines, err := strconv.Atoi(v.Get("tailLines"))
	if err != nil {
		tailLines = 0
	}
	k8sRequest := &k8s2.K8sRequestBean{
		ResourceIdentifier: k8s2.ResourceIdentifier{
			Name:             podName,
			GroupVersionKind: schema.GroupVersionKind{},
		},
		PodLogsRequest: k8s2.PodLogsRequest{
			//SinceTime:     sinceSeconds,
			TailLines:                  tailLines,
			Follow:                     follow,
			ContainerName:              containerName,
			IsPrevContainerLogsEnabled: isPrevLogs,
		},
	}
	request.K8sRequest = k8sRequest
	if appId != "" {
		// Validate App Type
		appType, err := strconv.Atoi(v.Get("appType"))
		if err != nil || !(appType == bean3.DevtronAppType || appType == bean3.HelmAppType) {
			impl.logger.Errorw("Invalid appType", "err", err, "appType", appType)
			return nil, err
		}
		request.AppType = appType
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
		k8sRequest := &k8s2.K8sRequestBean{
			ResourceIdentifier: k8s2.ResourceIdentifier{
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
	//getting rest config by clusterId
	restConfig, err := impl.k8sCommonService.GetRestConfigByClusterId(ctx, clusterId)
	if err != nil {
		impl.logger.Errorw("error in getting rest config by cluster Id", "err", err, "clusterId", clusterId)
		return nil, err
	}

	resourceIdentifier := request.K8sRequest.ResourceIdentifier
	podLogsRequest := request.K8sRequest.PodLogsRequest
	resp, err := impl.K8sUtil.GetPodLogs(ctx, restConfig, resourceIdentifier.Name, resourceIdentifier.Namespace, podLogsRequest.SinceTime, podLogsRequest.TailLines, podLogsRequest.Follow, podLogsRequest.ContainerName, podLogsRequest.IsPrevContainerLogsEnabled)
	if err != nil {
		impl.logger.Errorw("error in getting pod logs", "err", err, "clusterId", clusterId)
		return nil, err
	}
	return resp, nil
}

func (impl *K8sApplicationServiceImpl) ValidateClusterResourceRequest(ctx context.Context, clusterResourceRequest *k8s.ResourceRequestBean,
	rbacCallback func(clusterName string, resourceIdentifier k8s2.ResourceIdentifier) bool) (bool, error) {
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
	return impl.validateResourceManifest(clusterName, respManifest.Manifest, k8sRequest.ResourceIdentifier.GroupVersionKind, rbacCallback), nil
}

func (impl *K8sApplicationServiceImpl) validateResourceManifest(clusterName string, resourceManifest unstructured.Unstructured, gvk schema.GroupVersionKind, rbacCallback func(clusterName string, resourceIdentifier k8s2.ResourceIdentifier) bool) bool {
	validateCallback := func(namespace, group, kind, resourceName string) bool {
		resourceIdentifier := k8s2.ResourceIdentifier{
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

func (impl *K8sApplicationServiceImpl) ValidateClusterResourceBean(ctx context.Context, clusterId int, manifest unstructured.Unstructured, gvk schema.GroupVersionKind, rbacCallback func(clusterName string, resourceIdentifier k8s2.ResourceIdentifier) bool) bool {
	clusterBean, err := impl.clusterService.FindById(clusterId)
	if err != nil {
		impl.logger.Errorw("error in getting clusterBean by cluster Id", "clusterId", clusterId, "err", err)
		return false
	}
	return impl.validateResourceManifest(clusterBean.ClusterName, manifest, gvk, rbacCallback)
}

func (impl *K8sApplicationServiceImpl) ValidateResourceRequest(ctx context.Context, appIdentifier *client.AppIdentifier, request *k8s2.K8sRequestBean) (bool, error) {
	app, err := impl.helmAppService.GetApplicationDetail(ctx, appIdentifier)
	if err != nil {
		impl.logger.Errorw("error in getting app detail", "err", err, "appDetails", appIdentifier)
		return false, err
	}
	valid := false
	for _, node := range app.ResourceTreeResponse.Nodes {
		nodeDetails := k8s2.ResourceIdentifier{
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

func (impl *K8sApplicationServiceImpl) validateContainerNameIfReqd(valid bool, request *k8s2.K8sRequestBean, app *client.AppDetail) bool {
	if !valid {
		requestContainerName := request.PodLogsRequest.ContainerName
		podName := request.ResourceIdentifier.Name
		for _, pod := range app.ResourceTreeResponse.PodMetadata {
			if pod.Name == podName {

				//finding the container name in main Containers
				for _, container := range pod.Containers {
					if container == requestContainerName {
						return true
					}
				}

				//finding the container name in init containers
				for _, initContainer := range pod.InitContainers {
					if initContainer == requestContainerName {
						return true
					}
				}

				//finding the container name in ephemeral containers
				for _, ephemeralContainer := range pod.EphemeralContainers {
					if ephemeralContainer == requestContainerName {
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
		impl.logger.Errorw("error on getting resource from k8s, unable to fetch installer pod", "err", err)
		return nil, err
	}
	response := &bean3.ResourceInfo{PodName: pod.Name}
	return response, nil
}

func (impl *K8sApplicationServiceImpl) GetAllApiResources(ctx context.Context, clusterId int, isSuperAdmin bool, userId int32) (*k8s2.GetAllApiResourcesResponse, error) {
	impl.logger.Infow("getting all api-resources", "clusterId", clusterId)
	restConfig, err := impl.k8sCommonService.GetRestConfigByClusterId(ctx, clusterId)
	if err != nil {
		impl.logger.Errorw("error in getting cluster rest config", "clusterId", clusterId, "err", err)
		return nil, err
	}
	allApiResources, err := impl.K8sUtil.GetApiResources(restConfig, bean3.LIST_VERB)
	if err != nil {
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
	}
	if k8sEventIndex > -1 && v1EventIndex > -1 {
		allApiResources = append(allApiResources[:v1EventIndex], allApiResources[v1EventIndex+1:]...)
	}
	// FILTER ENDS

	// RBAC FILER STARTS
	allowedAll := isSuperAdmin
	filteredApiResources := make([]*k8s2.K8sApiResource, 0)
	if !isSuperAdmin {
		clusterBean, err := impl.clusterService.FindById(clusterId)
		if err != nil {
			impl.logger.Errorw("failed to find cluster for id", "err", err, "clusterId", clusterId)
			return nil, err
		}
		roles, err := impl.clusterService.FetchRolesFromGroup(userId)
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
			children, found := k8s2.KindVsChildrenGvk[kind]
			if found {
				// if rollout kind other than argo, then neglect only
				if kind != k8s2.K8sClusterResourceRolloutKind || groupName == k8s2.K8sClusterResourceRolloutGroup {
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
	response := &k8s2.GetAllApiResourcesResponse{
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

func (impl *K8sApplicationServiceImpl) GetResourceList(ctx context.Context, token string, request *k8s.ResourceRequestBean, validateResourceAccess func(token string, clusterName string, request k8s.ResourceRequestBean, casbinAction string) bool) (*k8s2.ClusterResourceListMap, error) {
	resourceList := &k8s2.ClusterResourceListMap{}
	clusterId := request.ClusterId
	clusterBean, err := impl.clusterService.FindById(clusterId)
	if err != nil {
		impl.logger.Errorw("error in getting cluster by cluster Id", "err", err, "clusterId", clusterId)
		return resourceList, err
	}
	restConfig, err := impl.k8sCommonService.GetRestConfigByClusterId(ctx, clusterId)
	if err != nil {
		impl.logger.Errorw("error in getting rest config by cluster Id", "err", err, "clusterId", request.ClusterId)
		return resourceList, err
	}
	k8sRequest := request.K8sRequest
	//store the copy of requested resource identifier
	resourceIdentifierCloned := k8sRequest.ResourceIdentifier
	resp, namespaced, err := impl.K8sUtil.GetResourceList(ctx, restConfig, resourceIdentifierCloned.GroupVersionKind, resourceIdentifierCloned.Name)
	if err != nil {
		impl.logger.Errorw("error in getting resource list", "err", err, "request", request)
		return resourceList, err
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
	resourceList, err = impl.K8sUtil.BuildK8sObjectListTableData(&resp.Resources, namespaced, request.K8sRequest.ResourceIdentifier.GroupVersionKind, checkForResourceCallback)
	if err != nil {
		impl.logger.Errorw("error on parsing for k8s resource", "err", err)
		return resourceList, err
	}
	return resourceList, nil
}

func (impl *K8sApplicationServiceImpl) ApplyResources(ctx context.Context, token string, request *k8s2.ApplyResourcesRequest, validateResourceAccess func(token string, clusterName string, request k8s.ResourceRequestBean, casbinAction string) bool) ([]*k8s2.ApplyResourcesResponse, error) {
	manifests, err := yamlUtil.SplitYAMLs([]byte(request.Manifest))
	if err != nil {
		impl.logger.Errorw("error in splitting yaml in manifest", "err", err)
		return nil, err
	}

	//getting rest config by clusterId
	clusterId := request.ClusterId
	clusterBean, err := impl.clusterService.FindById(clusterId)
	if err != nil {
		impl.logger.Errorw("error in getting clusterBean by cluster Id", "clusterId", clusterId, "err", err)
		return nil, err
	}
	restConfig, err := impl.k8sCommonService.GetRestConfigByClusterId(ctx, clusterId)
	if err != nil {
		impl.logger.Errorw("error in getting rest config by cluster", "clusterId", clusterId, "err", err)
		return nil, err
	}

	var response []*k8s2.ApplyResourcesResponse
	for _, manifest := range manifests {
		var namespace string
		manifestNamespace := manifest.GetNamespace()
		if len(manifestNamespace) > 0 {
			namespace = manifestNamespace
		} else {
			namespace = bean3.DEFAULT_NAMESPACE
		}
		manifestRes := &k8s2.ApplyResourcesResponse{
			Name: manifest.GetName(),
			Kind: manifest.GetKind(),
		}
		resourceRequestBean := k8s.ResourceRequestBean{
			ClusterId: clusterId,
			K8sRequest: &k8s2.K8sRequestBean{
				ResourceIdentifier: k8s2.ResourceIdentifier{
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
	k8sRequestBean := &k8s2.K8sRequestBean{
		ResourceIdentifier: k8s2.ResourceIdentifier{
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
