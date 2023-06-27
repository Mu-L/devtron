package repository

import (
	"github.com/devtron-labs/devtron/pkg/sql"
	"github.com/go-pg/pg"
	"go.uber.org/zap"
)

type ManifestPushConfig struct {
	tableName           struct{} `sql:"manifest_push_config" pg:",discard_unknown_columns"`
	Id                  int      `sql:"id,pk"`
	AppId               int      `sql:"appId"`
	EnvId               int      `sql:"envId"`
	ContainerRegistryId int      `sql:"container_registry_id"`
	RepoUrl             string   `sql:"repo_url"`
	ChartName           string   `sql:"chart_name"`
	ChartBaseVersion    string   `sql:"chart_base_version"`
	StorageType         string   `sql:"storage_type"`
	Deleted             bool     `sql:"deleted, notnull"`
	sql.AuditLog
}

type ManifestPushConfigRepository interface {
	SaveConfig(manifestPushConfig *ManifestPushConfig) (*ManifestPushConfig, error)
}

type ManifestPushConfigRepositoryImpl struct {
	logger       *zap.SugaredLogger
	dbConnection *pg.DB
}

func NewManifestPushConfigRepository(logger *zap.SugaredLogger,
	dbConnection *pg.DB,
) *ManifestPushConfigRepositoryImpl {
	return &ManifestPushConfigRepositoryImpl{
		logger:       logger,
		dbConnection: dbConnection,
	}
}

func (impl ManifestPushConfigRepositoryImpl) SaveConfig(manifestPushConfig *ManifestPushConfig) (*ManifestPushConfig, error) {
	err := impl.dbConnection.Insert(manifestPushConfig)
	if err != nil {
		return manifestPushConfig, err
	}
	return manifestPushConfig, err
}
