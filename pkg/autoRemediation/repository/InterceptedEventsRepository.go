package repository

import (
	"github.com/devtron-labs/devtron/pkg/sql"
	"github.com/go-pg/pg"
	"go.uber.org/zap"
	"time"
)

type InterceptedEventExecution struct {
	tableName          struct{}  `sql:"intercepted_event_execution" pg:",discard_unknown_columns"`
	Id                 int       `sql:"id,pk"`
	ClusterId          int       `sql:"cluster_id"`
	Namespace          string    `sql:"namespace"`
	Message            string    `sql:"message"`
	MessageType        string    `sql:"message_type"`
	Event              string    `sql:"event"`
	InvolvedObject     string    `sql:"involved_object"`
	InterceptedAt      time.Time `sql:"intercepted_at"`
	TriggerId          int       `sql:"trigger_id"`
	TriggerExecutionId int       `sql:"trigger_execution_id"`
	Status             Status    `sql:"status"`
}
type Status string

const (
	Failure     Status = "Failure"
	Success     Status = "Success"
	Progressing Status = "Progressing"
)

type InterceptedEventsRepository interface {
	Save(interceptedEvents *InterceptedEventExecution, tx *pg.Tx) (*InterceptedEventExecution, error)
	GetAllInterceptedEvents() ([]*InterceptedEventExecution, error)
	// UpdateStatus(status string, interceptedEventId int) error
	FindAll(offset int, size int, sortOrder string, searchString string, from time.Time, to time.Time, watchers []string, clusters []string, namespaces []string) ([]*InterceptedEventExecution, error)
	sql.TransactionWrapper
}

type InterceptedEventsRepositoryImpl struct {
	dbConnection *pg.DB
	logger       *zap.SugaredLogger
	*sql.TransactionUtilImpl
}

func NewInterceptedEventsRepositoryImpl(dbConnection *pg.DB, logger *zap.SugaredLogger) *InterceptedEventsRepositoryImpl {
	TransactionUtilImpl := sql.NewTransactionUtilImpl(dbConnection)
	return &InterceptedEventsRepositoryImpl{
		dbConnection:        dbConnection,
		logger:              logger,
		TransactionUtilImpl: TransactionUtilImpl,
	}
}

func (impl InterceptedEventsRepositoryImpl) Save(interceptedEvents *InterceptedEventExecution, tx *pg.Tx) (*InterceptedEventExecution, error) {
	_, err := tx.Model(interceptedEvents).Insert()
	if err != nil {
		impl.logger.Error(err)
		return nil, err
	}
	return interceptedEvents, nil
}
func (impl InterceptedEventsRepositoryImpl) GetAllInterceptedEvents() ([]*InterceptedEventExecution, error) {
	var interceptedEvents []*InterceptedEventExecution
	err := impl.dbConnection.Model(&interceptedEvents).
		Select()
	if err != nil {
		return nil, err
	}
	return interceptedEvents, nil
}

//	func (impl InterceptedEventsRepositoryImpl) UpdateStatus(status string, interceptedEventId int)  error {
//		_, err := impl.dbConnection.Model(&InterceptedEvents{}).Where("id=?", interceptedEventId).Set("status=?", status).Update()
//		if err != nil {
//			return err
//		}
//		return  nil
//
// }
func (impl InterceptedEventsRepositoryImpl) FindAll(offset int, size int, sortOrder string, searchString string, from time.Time, to time.Time, watchers []string, clusters []string, namespaces []string) ([]*InterceptedEventExecution, error) {
	var interceptedEvents []*InterceptedEventExecution
	query := impl.dbConnection.Model(&interceptedEvents)
	if searchString != "" {
		query = query.Where("message LIKE ? or involved_object LIKE ?", "%"+searchString+"%", "%"+searchString+"%")
	}
	query = query.Where("intercepted_at BETWEEN ? AND ?", from, to)
	if len(clusters) > 0 {
		query = query.Where("cluster_name IN (?)", pg.In(clusters))
	}
	if len(namespaces) > 0 {
		query = query.Where("namespace IN (?)", pg.In(namespaces))
	}
	err := query.Order("intercepted_at ?", sortOrder).
		Offset(offset).
		Limit(size).
		Select()
	if err != nil {
		return nil, err
	}
	return interceptedEvents, nil
}
