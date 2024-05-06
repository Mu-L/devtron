package repository

import (
	"github.com/devtron-labs/devtron/pkg/sql"
	"github.com/devtron-labs/scoop/types"
	"github.com/go-pg/pg"
	"go.uber.org/zap"
	"time"
)

type K8sEventWatcher struct {
	tableName        struct{}          `sql:"k8s_event_watcher" pg:",discard_unknown_columns"`
	Id               int               `sql:"id,pk"`
	Name             string            `sql:"name,notnull"`
	Description      string            `sql:"description"`
	FilterExpression string            `sql:"filter_expression,notnull"`
	Gvks             string            `sql:"gvks"`
	SelectedActions  []types.EventType `sql:"selected_actions" pg:",array"`
	Active           bool              `sql:"active,notnull"`
	sql.AuditLog
}

type WatcherQueryParams struct {
	Offset      int    `json:"offset"`
	Size        int    `json:"size"`
	SortOrder   string `json:"sortOrder"`
	SortOrderBy string `json:"sortOrderBy"`
	Search      string `json:"Search"`
}

type K8sEventWatcherRepository interface {
	Save(watcher *K8sEventWatcher, tx *pg.Tx) (*K8sEventWatcher, error)
	Update(tx *pg.Tx, watcher *K8sEventWatcher, userId int32) error
	Delete(watcher *K8sEventWatcher) error
	GetWatcherById(id int) (*K8sEventWatcher, error)
	GetWatcherByIds(ids []int) ([]*K8sEventWatcher, error)
	DeleteWatcherById(id int) error
	FindAllWatchersByQueryName(params WatcherQueryParams) ([]*K8sEventWatcher, int, error)
	sql.TransactionWrapper
}
type K8sEventWatcherRepositoryImpl struct {
	dbConnection *pg.DB
	logger       *zap.SugaredLogger
	*sql.TransactionUtilImpl
}

func NewWatcherRepositoryImpl(dbConnection *pg.DB, logger *zap.SugaredLogger) *K8sEventWatcherRepositoryImpl {
	TransactionUtilImpl := sql.NewTransactionUtilImpl(dbConnection)
	return &K8sEventWatcherRepositoryImpl{
		dbConnection:        dbConnection,
		logger:              logger,
		TransactionUtilImpl: TransactionUtilImpl,
	}
}

func (impl K8sEventWatcherRepositoryImpl) Save(watcher *K8sEventWatcher, tx *pg.Tx) (*K8sEventWatcher, error) {
	_, err := tx.Model(watcher).Insert()
	if err != nil {
		return nil, err
	}
	return watcher, nil
}
func (impl K8sEventWatcherRepositoryImpl) Update(tx *pg.Tx, watcher *K8sEventWatcher, userId int32) error {
	_, err := tx.
		Model((*K8sEventWatcher)(nil)).
		Set("name = ?", watcher.Name).
		Set("description = ?", watcher.Description).
		Set("filter_expression = ?", watcher.FilterExpression).
		Set("gvks = ?", watcher.Gvks).
		Set("selected_actions = ?", watcher.SelectedActions).
		Set("updated_by = ?", userId).
		Set("updated_on = ?", time.Now()).
		Where("active = ?", true).
		Where("id = ?", watcher.Id).
		Update()
	return err
}

func (impl K8sEventWatcherRepositoryImpl) Delete(watcher *K8sEventWatcher) error {
	err := impl.dbConnection.Delete(&watcher)
	if err != nil {
		return err
	}
	return nil
}
func (impl K8sEventWatcherRepositoryImpl) GetWatcherById(id int) (*K8sEventWatcher, error) {
	var watcher K8sEventWatcher
	err := impl.dbConnection.Model(&watcher).Where("id = ? and active = ?", id, true).Select()
	if err != nil {
		return &K8sEventWatcher{}, err
	}
	return &watcher, nil
}

func (impl K8sEventWatcherRepositoryImpl) GetWatcherByIds(ids []int) ([]*K8sEventWatcher, error) {
	var watchers []*K8sEventWatcher
	err := impl.dbConnection.Model(&watchers).
		Where("id IN (?) and active = ?", pg.In(ids), true).
		Select()
	if err != nil {
		return nil, err
	}
	return watchers, nil
}

func (impl K8sEventWatcherRepositoryImpl) DeleteWatcherById(id int) error {
	_, err := impl.dbConnection.Model(&K8sEventWatcher{}).Set("active = ?", false).Where("id = ?", id).Update()
	if err != nil {
		return err
	}
	return nil
}

func (impl K8sEventWatcherRepositoryImpl) FindAllWatchersByQueryName(params WatcherQueryParams) ([]*K8sEventWatcher, int, error) {
	var watcher []*K8sEventWatcher
	query := impl.dbConnection.Model(&watcher)
	if params.Search != "" {
		query = query.Where("name ILIKE ? ", "%"+params.Search+"%")
	}
	if params.SortOrder == "desc" {
		query = query.Order("name desc")
	} else {
		query = query.Order("name asc")
	}
	total, err := query.Where("active = ?", true).Count()
	if err != nil {
		return []*K8sEventWatcher{}, 0, err
	}
	err = query.Offset(params.Offset).Limit(params.Size).Select()
	if err != nil {
		return []*K8sEventWatcher{}, 0, err
	}
	return watcher, total, nil
}
