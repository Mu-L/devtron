package application

import (
	"github.com/devtron-labs/devtron/api/restHandler/autoRemediation"
	"github.com/devtron-labs/devtron/pkg/terminal"
	"github.com/gorilla/mux"
)

type K8sApplicationRouter interface {
	InitK8sApplicationRouter(helmRouter *mux.Router)
}
type K8sApplicationRouterImpl struct {
	k8sApplicationRestHandler K8sApplicationRestHandler
	watcherRestHandler        autoRemediation.WatcherRestHandler
}

func NewK8sApplicationRouterImpl(k8sApplicationRestHandler K8sApplicationRestHandler, watcherRestHandler autoRemediation.WatcherRestHandler) *K8sApplicationRouterImpl {
	return &K8sApplicationRouterImpl{
		k8sApplicationRestHandler: k8sApplicationRestHandler,
		watcherRestHandler:        watcherRestHandler,
	}
}

func (impl *K8sApplicationRouterImpl) InitK8sApplicationRouter(k8sAppRouter *mux.Router) {
	k8sAppRouter.Path("/resource/rotate").Queries("appId", "{appId}").
		HandlerFunc(impl.k8sApplicationRestHandler.RotatePod).Methods("POST")

	k8sAppRouter.Path("/resource/urls").Queries("appId", "{appId}").
		HandlerFunc(impl.k8sApplicationRestHandler.GetHostUrlsByBatch).Methods("GET")

	k8sAppRouter.Path("/resource").
		HandlerFunc(impl.k8sApplicationRestHandler.GetResource).Methods("POST")

	k8sAppRouter.Path("/resource/create").
		HandlerFunc(impl.k8sApplicationRestHandler.CreateResource).Methods("POST")

	k8sAppRouter.Path("/resource").
		HandlerFunc(impl.k8sApplicationRestHandler.UpdateResource).Methods("PUT")

	k8sAppRouter.Path("/resource/delete").
		HandlerFunc(impl.k8sApplicationRestHandler.DeleteResource).Methods("POST")

	k8sAppRouter.Path("/events").
		HandlerFunc(impl.k8sApplicationRestHandler.ListEvents).Methods("POST")

	k8sAppRouter.Path("/pods/logs/{podName}").
		Queries("containerName", "{containerName}").
		Queries("follow", "{follow}").
		HandlerFunc(impl.k8sApplicationRestHandler.GetPodLogs).Methods("GET")

	k8sAppRouter.Path("/pods/logs/download/{podName}").
		Queries("containerName", "{containerName}").
		HandlerFunc(impl.k8sApplicationRestHandler.DownloadPodLogs).Methods("GET")

	k8sAppRouter.Path("/pod/exec/session/{identifier}/{namespace}/{pod}/{shell}/{container}").
		HandlerFunc(impl.k8sApplicationRestHandler.GetTerminalSession).Methods("GET")
	k8sAppRouter.PathPrefix("/pod/exec/sockjs/ws").Handler(terminal.CreateAttachHandler("/pod/exec/sockjs/ws"))

	/*k8sAppRouter.Path("/pod/exec/sockjs/ws/").
	Handler(terminal.CreateAttachHandler("/api/v1/applications/pod/exec/sockjs/ws/"))*/

	k8sAppRouter.Path("/resource/inception/info").
		HandlerFunc(impl.k8sApplicationRestHandler.GetResourceInfo).Methods("GET")

	k8sAppRouter.Path("/api-resources/{clusterId}").
		HandlerFunc(impl.k8sApplicationRestHandler.GetAllApiResources).Methods("GET")

	k8sAppRouter.Path("/resource/list").
		HandlerFunc(impl.k8sApplicationRestHandler.GetResourceList).Methods("POST")

	k8sAppRouter.Path("/resources/apply").
		HandlerFunc(impl.k8sApplicationRestHandler.ApplyResources).Methods("POST")

	//create/delete ephemeral containers API's
	k8sAppRouter.Path("/resources/ephemeralContainers").
		Queries("identifier", "{identifier}").
		HandlerFunc(impl.k8sApplicationRestHandler.CreateEphemeralContainer).Methods("POST")
	k8sAppRouter.Path("/resources/ephemeralContainers").
		Queries("identifier", "{identifier}").
		HandlerFunc(impl.k8sApplicationRestHandler.DeleteEphemeralContainer).Methods("DELETE")

	k8sAppRouter.Path("/api-resources/gvk/{clusterId}").
		HandlerFunc(impl.k8sApplicationRestHandler.GetAllApiResourceGVKWithoutAuthorization).Methods("GET")

	k8sAppRouter.Path("/watcher").HandlerFunc(impl.watcherRestHandler.SaveWatcher).Methods("POST")
	k8sAppRouter.Path("/watcher").Queries("search", "{search}").
		Queries("orderBy", "{orderBy}").
		Queries("order", "{order}").
		Queries("offset", "{offset}").
		Queries("size", "{size}").HandlerFunc(impl.watcherRestHandler.RetrieveWatchers).Methods("GET")
	k8sAppRouter.Path("/watcher/{identifier}").HandlerFunc(impl.watcherRestHandler.GetWatcherById).Methods("GET")
	k8sAppRouter.Path("/watcher/{identifier}").HandlerFunc(impl.watcherRestHandler.DeleteWatcherById).Methods("DELETE")
	//k8sAppRouter.Path("/watcher/events").HandlerFunc(impl.watcherRestHandler.RetrieveInterceptedEvents).Methods("GET")
	k8sAppRouter.Path("/watcher/{identifier}").HandlerFunc(impl.watcherRestHandler.UpdateWatcherById).Methods("PUT")

	//k8sAppRouter.Path("").
	//	Queries("watchers", "{watchers}").
	//	Queries("clusters", "{clusters}").
	//	Queries("namespaces", "{namespaces}").
	//	Queries("executionStatuses", "{executionStatuses}").
	//	Queries("from", "{from}").
	//	Queries("to", "{to}").
	//	Queries("offset", "{offset}").
	//	Queries("size", "{size}").
	//	Queries("searchString", "{searchString}").
	//	HandlerFunc(impl.watcherRestHandler.RetrieveWatchers).
	//	Methods("GET")
}
