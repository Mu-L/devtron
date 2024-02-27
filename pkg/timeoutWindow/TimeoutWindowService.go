package timeoutWindow

import (
	"github.com/devtron-labs/devtron/pkg/sql"
	"github.com/devtron-labs/devtron/pkg/timeoutWindow/repository"
	"github.com/devtron-labs/devtron/pkg/timeoutWindow/repository/bean"
	"github.com/go-pg/pg"
	"go.uber.org/zap"
	"time"
)

type TimeoutWindowService interface {
	GetAllWithIds(ids []int) ([]*repository.TimeoutWindowConfiguration, error)
	UpdateTimeoutExpressionAndFormatForIds(tx *pg.Tx, timeoutExpression string, ids []int, expressionFormat bean.ExpressionFormat, loggedInUserId int32) error
	CreateWithTimeoutExpressionAndFormat(tx *pg.Tx, timeoutExpression string, count int, expressionFormat bean.ExpressionFormat, loggedInUserId int32) ([]*repository.TimeoutWindowConfiguration, error)
	CreateForConfigurationList(tx *pg.Tx, models []*repository.TimeoutWindowConfiguration) ([]*repository.TimeoutWindowConfiguration, error)
}

func (impl TimeWindowServiceImpl) CreateForConfigurationList(tx *pg.Tx, configurations []*repository.TimeoutWindowConfiguration) ([]*repository.TimeoutWindowConfiguration, error) {
	return impl.timeWindowRepository.CreateInBatch(tx, configurations)
}

type TimeWindowServiceImpl struct {
	logger               *zap.SugaredLogger
	timeWindowRepository repository.TimeWindowRepository
}

func NewTimeWindowServiceImpl(logger *zap.SugaredLogger,
	timeWindowRepository repository.TimeWindowRepository) *TimeWindowServiceImpl {
	timeoutWindowServiceImpl := &TimeWindowServiceImpl{
		logger:               logger,
		timeWindowRepository: timeWindowRepository,
	}
	return timeoutWindowServiceImpl
}

func (impl TimeWindowServiceImpl) GetAllWithIds(ids []int) ([]*repository.TimeoutWindowConfiguration, error) {
	timeWindows, err := impl.timeWindowRepository.GetWithIds(ids)
	if err != nil {
		impl.logger.Errorw("error in GetAllWithIds", "err", err, "timeWindowIds", ids)
		return nil, err
	}
	return timeWindows, err
}

func (impl TimeWindowServiceImpl) UpdateTimeoutExpressionAndFormatForIds(tx *pg.Tx, timeoutExpression string, ids []int, expressionFormat bean.ExpressionFormat, loggedInUserId int32) error {
	err := impl.timeWindowRepository.UpdateTimeoutExpressionAndFormatForIds(tx, timeoutExpression, ids, expressionFormat, loggedInUserId)
	if err != nil {
		impl.logger.Errorw("error in UpdateTimeoutExpressionForIds", "err", err, "timeoutExpression", timeoutExpression)
		return err
	}
	return err
}

func (impl TimeWindowServiceImpl) CreateWithTimeoutExpressionAndFormat(tx *pg.Tx, timeoutExpression string, count int, expressionFormat bean.ExpressionFormat, loggedInUserId int32) ([]*repository.TimeoutWindowConfiguration, error) {
	var models []*repository.TimeoutWindowConfiguration
	for i := 0; i < count; i++ {
		model := &repository.TimeoutWindowConfiguration{
			TimeoutWindowExpression: timeoutExpression,
			ExpressionFormat:        expressionFormat,
			AuditLog: sql.AuditLog{
				CreatedOn: time.Now(),
				CreatedBy: loggedInUserId,
				UpdatedOn: time.Now(),
				UpdatedBy: loggedInUserId,
			},
		}
		models = append(models, model)
	}
	// create in batch
	models, err := impl.timeWindowRepository.CreateInBatch(tx, models)
	if err != nil {
		impl.logger.Errorw("error in CreateWithTimeoutExpression", "err", err, "timeoutExpression", timeoutExpression, "countToBeCreated", count)
		return nil, err
	}
	return models, nil

}
