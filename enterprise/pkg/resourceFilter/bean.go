package resourceFilter

const (
	NoResourceFiltersFound               = "no active resource filters found"
	AppAndEnvSelectorRequiredMessage     = "both application and environment selectors are required"
	InvalidExpressions                   = "one or more expressions are invalid"
	AllProjectsValue                     = "0"
	AllProjectsInt                       = 0
	AllExistingAndFutureProdEnvsValue    = "0"
	AllExistingAndFutureProdEnvsInt      = 0
	AllExistingAndFutureNonProdEnvsValue = "-1"
	AllExistingAndFutureNonProdEnvsInt   = -1
)

type IdentifierType int

const (
	ProjectIdentifier     = 0
	AppIdentifier         = 1
	ClusterIdentifier     = 2
	EnvironmentIdentifier = 3
)

type FilterResponseBean struct {
	Id           int                `json:"id"`
	TargetObject FilterTargetObject `json:"targetObject" validate:"required"`
	Description  string             `json:"description" `
	Name         string             `json:"name" validate:"required"`
}

type FilterRequestResponseBean struct {
	*FilterResponseBean
	Conditions        []ResourceCondition `json:"conditions"`
	QualifierSelector QualifierSelector   `json:"qualifierSelector"`
}

type ResourceCondition struct {
	ConditionType ResourceConditionType `json:"conditionType"`
	Expression    string                `json:"expression"`
	ErrorMsg      string                `json:"errorMsg"`
}

func (condition ResourceCondition) IsFailCondition() bool {
	return condition.ConditionType == FAIL
}

type QualifierSelector struct {
	ApplicationSelectors []ApplicationSelector `json:"applicationSelectors"`
	EnvironmentSelectors []EnvironmentSelector `json:"environmentSelectors"`
}

type ApplicationSelector struct {
	ProjectName  string   `json:"projectName"`
	Applications []string `json:"applications"`
}

type EnvironmentSelector struct {
	ClusterName  string   `json:"clusterName"`
	Environments []string `json:"environments"`
}

type ExpressionMetadata struct {
	Params []ExpressionParam
}

type ExpressionParam struct {
	ParamName string          `json:"paramName"`
	Value     interface{}     `json:"value"`
	Type      ParamValuesType `json:"type"`
}

type ParamValuesType string

const (
	ParamTypeString  ParamValuesType = "string"
	ParamTypeObject  ParamValuesType = "object"
	ParamTypeInteger ParamValuesType = "integer"
)

type expressionResponse struct {
	allowConditionAvail bool
	allowResponse       bool
	blockConditionAvail bool
	blockResponse       bool
}

func (response expressionResponse) getFinalResponse() bool {
	if response.blockConditionAvail && response.blockResponse {
		return false
	}

	if response.allowConditionAvail && !response.allowResponse {
		return false
	}
	return true
}
