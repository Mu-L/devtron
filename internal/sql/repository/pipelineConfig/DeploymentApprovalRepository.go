package pipelineConfig

import (
	"errors"
	"fmt"
	"time"

	repository2 "github.com/devtron-labs/devtron/internal/sql/repository"
	"github.com/devtron-labs/devtron/pkg/auth/user/repository"
	"github.com/devtron-labs/devtron/pkg/sql"
	"github.com/go-pg/pg"
	"go.uber.org/zap"
)

type DeploymentApprovalRepository interface {
	FetchApprovalDataForArtifacts(artifactIds []int, pipelineId int) ([]*DeploymentApprovalRequest, error)
	FetchApprovalPendingArtifacts(pipelineId, limit, offset, requiredApproval int, searchString string) ([]*DeploymentApprovalRequest, int, error)
	FetchById(requestId int) (*DeploymentApprovalRequest, error)
	FetchWithPipelineAndArtifactDetails(requestId int) (*DeploymentApprovalRequest, error)
	Save(deploymentApprovalRequest *DeploymentApprovalRequest) error
	Update(deploymentApprovalRequest *DeploymentApprovalRequest) error
	SaveDeploymentUserData(userData *ResourceApprovalUserData) error
	ConsumeApprovalRequest(requestId int) error
	FetchLatestDeploymentByArtifactIds(pipelineId int, artifactIds []int) ([]*DeploymentApprovalRequest, error)
}

type DeploymentApprovalRepositoryImpl struct {
	dbConnection               *pg.DB
	logger                     *zap.SugaredLogger
	resourceApprovalRepository ResourceApprovalRepository
}

func NewDeploymentApprovalRepositoryImpl(dbConnection *pg.DB, logger *zap.SugaredLogger, resourceApprovalRepository ResourceApprovalRepository) *DeploymentApprovalRepositoryImpl {
	return &DeploymentApprovalRepositoryImpl{dbConnection: dbConnection, logger: logger, resourceApprovalRepository: resourceApprovalRepository}
}

type DeploymentApprovalRequest struct {
	tableName                   struct{} `sql:"deployment_approval_request" pg:",discard_unknown_columns"`
	Id                          int      `sql:"id,pk"`
	PipelineId                  int      `sql:"pipeline_id"`
	ArtifactId                  int      `sql:"ci_artifact_id"`
	Active                      bool     `sql:"active,notnull"` // user can cancel request anytime
	ArtifactDeploymentTriggered bool     `sql:"artifact_deployment_triggered"`
	Pipeline                    *Pipeline
	CiArtifact                  *repository2.CiArtifact
	UserEmail                   string                      `sql:"-"` // used for internal purpose
	DeploymentApprovalUserData  []*ResourceApprovalUserData `sql:"-"`
	sql.AuditLog
}

type ResourceApprovalUserData struct {
	tableName         struct{}                   `sql:"resource_approval_user_data" pg:",discard_unknown_columns"`
	Id                int                        `sql:"id,pk"`
	RequestType       repository2.RequestType    `sql:"request_type"`
	ApprovalRequestId int                        `sql:"approval_request_id"` // keep in mind foreign key constraint
	UserId            int32                      `sql:"user_id"`             // keep in mid foreign key constraint
	UserResponse      DeploymentApprovalResponse `sql:"user_response"`
	Comments          string                     `sql:"comments"`
	User              *repository.UserModel
	sql.AuditLog
}

func (impl *DeploymentApprovalRepositoryImpl) FetchApprovalPendingArtifacts(pipelineId, limit, offset, requiredApprovals int, searchString string) ([]*DeploymentApprovalRequest, int, error) {

	var requests []*DeploymentApprovalRequest

	searchString = "%" + searchString + "%"

	subQuery := "WITH approval_requests AS " +
		" (SELECT approval_request_id,count(approval_request_id) AS approval_count " +
		" FROM resource_approval_user_data " +
		fmt.Sprintf(" WHERE user_response is NULL AND request_type = %d", repository2.DEPLOYMENT_APPROVAL) +
		" GROUP BY approval_request_id ) " +
		" SELECT approval_request_id " +
		" FROM approval_requests WHERE approval_count >= %v "
	subQuery = fmt.Sprintf(subQuery, requiredApprovals)
	finalQuery := impl.dbConnection.Model(&requests).
		Column("deployment_approval_request.*", "CiArtifact").
		Join("JOIN ci_artifact ca ON ca.id = deployment_approval_request.ci_artifact_id").
		Where(fmt.Sprintf("deployment_approval_request.id NOT IN (%v)", subQuery)).
		Where("deployment_approval_request.pipeline_id = ?", pipelineId).
		Where("deployment_approval_request.active=true").
		Where("deployment_approval_request.artifact_deployment_triggered=false").
		Where("ci_artifact.image LIKE ? ", searchString)

	totalCount, err := finalQuery.Count()
	if err != nil {
		impl.logger.Errorw("error occurred while fetching total count", "pipelineId", pipelineId, "err", err)
		return nil, 0, err
	}

	requests = make([]*DeploymentApprovalRequest, 0)
	err = finalQuery.
		Limit(limit).
		Offset(offset).
		Select()

	if err != nil && err != pg.ErrNoRows {
		impl.logger.Errorw("error occurred while fetching artifacts", "pipelineId", pipelineId, "err", err)
		return nil, 0, err
	}

	return requests, totalCount, nil
}

func (impl *DeploymentApprovalRepositoryImpl) FetchLatestDeploymentByArtifactIds(pipelineId int, artifactIds []int) ([]*DeploymentApprovalRequest, error) {
	var requests []*DeploymentApprovalRequest
	if len(artifactIds) == 0 {
		return requests, nil
	}

	query := `with minimal_pcos as (select max(pco.id) as id from pipeline_config_override pco where pco.pipeline_id  = ? and pco.ci_artifact_id in (?) group by pco.ci_artifact_id)
	select pco.ci_artifact_id,pco.created_on from pipeline_config_override pco where pco.id in (select id from minimal_pcos); `

	_, err := impl.dbConnection.Query(&requests, query, pipelineId, pg.In(artifactIds))
	if err != nil && err != pg.ErrNoRows {
		impl.logger.Errorw("error occurred while fetching latest deployment by artifact ids", "pipelineId", pipelineId, "artifactIds", artifactIds, "err", err)
		return nil, err
	}

	return requests, nil
}

func (impl *DeploymentApprovalRepositoryImpl) FetchApprovalDataForArtifacts(artifactIds []int, pipelineId int) ([]*DeploymentApprovalRequest, error) {
	impl.logger.Debugw("fetching approval data for artifacts", "ids", artifactIds, "pipelineId", pipelineId)
	if len(artifactIds) == 0 {
		return []*DeploymentApprovalRequest{}, nil
	}
	var requests []*DeploymentApprovalRequest
	err := impl.dbConnection.
		Model(&requests).
		// Column("deployment_approval_request.*", /*"ResourceApprovalUserData", "ResourceApprovalUserData.User"*/).
		Where("ci_artifact_id in (?) ", pg.In(artifactIds)).
		Where("pipeline_id = ?", pipelineId).
		Where("artifact_deployment_triggered = ?", false).
		Where("active = ?", true).
		Select()
	if err != nil && err != pg.ErrNoRows {
		impl.logger.Errorw("error occurred while fetching artifacts", "pipelineId", pipelineId, "err", err)
		return nil, err
	}
	requestIdMap := make(map[int]*DeploymentApprovalRequest)
	var requestIds []int
	for _, request := range requests {
		requestId := request.Id
		requestIdMap[requestId] = request
		requestIds = append(requestIds, requestId)
	}
	if len(requestIds) > 0 {
		usersData, err := impl.resourceApprovalRepository.FetchApprovalDataForRequests(requestIds, repository2.DEPLOYMENT_APPROVAL)
		if err != nil {
			return requests, err
		}
		for _, userData := range usersData {
			approvalRequestId := userData.ApprovalRequestId
			deploymentApprovalRequest := requestIdMap[approvalRequestId]
			approvalUsers := deploymentApprovalRequest.DeploymentApprovalUserData
			approvalUsers = append(approvalUsers, userData)
			deploymentApprovalRequest.DeploymentApprovalUserData = approvalUsers
		}
	}
	return requests, nil
}

func (impl *DeploymentApprovalRepositoryImpl) FetchWithPipelineAndArtifactDetails(requestId int) (*DeploymentApprovalRequest, error) {
	request := &DeploymentApprovalRequest{Id: requestId}
	err := impl.dbConnection.
		Model(request).
		Column("deployment_approval_request.*", "Pipeline", "CiArtifact").
		Where("active = ?", true).WherePK().Select()
	if err != nil {
		impl.logger.Errorw("error occurred while fetching request data", "id", requestId, "err", err)
		return nil, err
	}
	return request, nil
}

func (impl *DeploymentApprovalRepositoryImpl) FetchById(requestId int) (*DeploymentApprovalRequest, error) {
	request := &DeploymentApprovalRequest{Id: requestId}
	err := impl.dbConnection.
		Model(request).Where("active = ?", true).WherePK().Select()
	if err != nil {
		impl.logger.Errorw("error occurred while fetching request data", "id", requestId, "err", err)
		return nil, err
	}
	return request, nil
}

func (impl *DeploymentApprovalRepositoryImpl) ConsumeApprovalRequest(requestId int) error {
	request, err := impl.FetchById(requestId)
	if err != nil && err != pg.ErrNoRows {
		impl.logger.Errorw("error occurred while fetching approval request", "requestId", requestId, "err", err)
		return err
	} else if err == pg.ErrNoRows {
		return errors.New("approval request not raised for this artifact")
	}
	request.ArtifactDeploymentTriggered = true
	return impl.Update(request)
}

func (impl *DeploymentApprovalRepositoryImpl) Save(deploymentApprovalRequest *DeploymentApprovalRequest) error {
	currentTime := time.Now()
	deploymentApprovalRequest.CreatedOn = currentTime
	deploymentApprovalRequest.UpdatedOn = currentTime
	return impl.dbConnection.Insert(deploymentApprovalRequest)
}

func (impl *DeploymentApprovalRepositoryImpl) Update(deploymentApprovalRequest *DeploymentApprovalRequest) error {
	deploymentApprovalRequest.UpdatedOn = time.Now()
	return impl.dbConnection.Update(deploymentApprovalRequest)
}

func (impl *DeploymentApprovalRepositoryImpl) SaveDeploymentUserData(userData *ResourceApprovalUserData) error {
	currentTime := time.Now()
	userData.CreatedOn = currentTime
	userData.UpdatedOn = currentTime
	return impl.dbConnection.Insert(userData)
}

func (request *DeploymentApprovalRequest) ConvertToApprovalMetadata() *UserApprovalMetadata {
	approvalMetadata := &UserApprovalMetadata{ApprovalRequestId: request.Id}
	requestedUserData := UserApprovalData{DataId: request.Id}
	requestedUserData.UserId = request.CreatedBy
	requestedUserData.UserEmail = request.UserEmail
	requestedUserData.UserActionTime = request.CreatedOn
	approvalMetadata.RequestedUserData = requestedUserData
	var userApprovalData []UserApprovalData
	for _, approvalUser := range request.DeploymentApprovalUserData {
		userApprovalData = append(userApprovalData, UserApprovalData{DataId: approvalUser.Id, UserId: approvalUser.UserId, UserEmail: approvalUser.User.EmailId, UserResponse: approvalUser.UserResponse, UserActionTime: approvalUser.CreatedOn})
	}
	approvalMetadata.ApprovalUsersData = userApprovalData
	return approvalMetadata
}

func (request *DeploymentApprovalRequest) GetApprovedCount() int {
	count := 0
	for _, approvalUser := range request.DeploymentApprovalUserData {
		if approvalUser.UserResponse == APPROVED {
			count++
		}
	}
	return count
}
