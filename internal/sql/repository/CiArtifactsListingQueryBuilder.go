package repository

import (
	"fmt"
	"github.com/devtron-labs/devtron/api/bean"
	"github.com/devtron-labs/devtron/internal/sql/repository/helper"
	"github.com/devtron-labs/devtron/pkg/policyGovernance/artifactPromotion/constants"
)

const EmptyLikeRegex = "%%"

func BuildQueryForParentTypeCIOrWebhook(listingFilterOpts bean.ArtifactsListFilterOptions, isApprovalNode bool) string {
	commonPaginatedQueryPart := fmt.Sprintf(" cia.image LIKE '%v'", listingFilterOpts.SearchString)
	orderByClause := " ORDER BY cia.id DESC"
	limitOffsetQueryPart := fmt.Sprintf(" LIMIT %v OFFSET %v", listingFilterOpts.Limit, listingFilterOpts.Offset)
	finalQuery := ""
	commonApprovalNodeSubQueryPart := fmt.Sprintf("cia.id NOT IN "+
		" ( "+
		" SELECT DISTINCT dar.ci_artifact_id "+
		" FROM deployment_approval_request dar "+
		" WHERE dar.pipeline_id = %v "+
		" AND dar.active=true "+
		" AND dar.artifact_deployment_triggered = false"+
		" ) AND ", listingFilterOpts.PipelineId)

	if listingFilterOpts.ParentStageType == bean.CI_WORKFLOW_TYPE {
		selectQuery := " SELECT cia.* "
		remainingQuery := " FROM ci_artifact cia" +
			" INNER JOIN ci_pipeline cp ON (cp.id=cia.pipeline_id or (cp.id=cia.component_id and cia.data_source='post_ci' ) )" +
			" INNER JOIN pipeline p ON (p.ci_pipeline_id = cp.id and p.id=%v )" +
			" WHERE "
		remainingQuery = fmt.Sprintf(remainingQuery, listingFilterOpts.PipelineId)
		if isApprovalNode {
			remainingQuery += commonApprovalNodeSubQueryPart
		} else if len(listingFilterOpts.ExcludeArtifactIds) > 0 {
			remainingQuery += fmt.Sprintf("cia.id NOT IN (%s) AND ", helper.GetCommaSepratedString(listingFilterOpts.ExcludeArtifactIds))
		}

		countQuery := " SELECT count(cia.id)  as total_count"
		totalCountQuery := countQuery + remainingQuery + commonPaginatedQueryPart
		selectQuery = fmt.Sprintf("%s,(%s) ", selectQuery, totalCountQuery)
		finalQuery = selectQuery + remainingQuery + commonPaginatedQueryPart + orderByClause + limitOffsetQueryPart
	} else if listingFilterOpts.ParentStageType == bean.WEBHOOK_WORKFLOW_TYPE {
		selectQuery := " SELECT cia.* "
		remainingQuery := " FROM ci_artifact cia " +
			" WHERE cia.external_ci_pipeline_id = %v AND "
		remainingQuery = fmt.Sprintf(remainingQuery, listingFilterOpts.ParentId)
		if isApprovalNode {
			remainingQuery += commonApprovalNodeSubQueryPart
		} else if len(listingFilterOpts.ExcludeArtifactIds) > 0 {
			remainingQuery += fmt.Sprintf("cia.id NOT IN (%s) AND ", helper.GetCommaSepratedString(listingFilterOpts.ExcludeArtifactIds))
		}

		countQuery := " SELECT count(cia.id)  as total_count"
		totalCountQuery := countQuery + remainingQuery + commonPaginatedQueryPart
		selectQuery = fmt.Sprintf("%s,(%s) ", selectQuery, totalCountQuery)
		finalQuery = selectQuery + remainingQuery + commonPaginatedQueryPart + orderByClause + limitOffsetQueryPart

	}
	return finalQuery
}

func BuildQueryForArtifactsForCdStage(listingFilterOptions bean.ArtifactsListFilterOptions, isApprovalNode bool) string {
	if listingFilterOptions.UseCdStageQueryV2 {
		return buildQueryForArtifactsForCdStageV2(listingFilterOptions, isApprovalNode)
	}

	// TODO: revisit this condition (cd_workflow.pipeline_id= %v and cd_workflow_runner.workflow_type = '%v' )
	// TODO: remove below code
	commonQuery := " from ci_artifact LEFT JOIN cd_workflow ON ci_artifact.id = cd_workflow.ci_artifact_id" +
		" LEFT JOIN cd_workflow_runner ON cd_workflow_runner.cd_workflow_id=cd_workflow.id " +
		" Where (((cd_workflow_runner.id in (select MAX(cd_workflow_runner.id) OVER (PARTITION BY cd_workflow.ci_artifact_id) FROM cd_workflow_runner inner join cd_workflow on cd_workflow.id=cd_workflow_runner.cd_workflow_id))" +
		" AND ((cd_workflow.pipeline_id= %v and cd_workflow_runner.workflow_type = '%v' ) OR (cd_workflow.pipeline_id = %v AND cd_workflow_runner.workflow_type = '%v' AND cd_workflow_runner.status IN ('Healthy','Succeeded') )))" +
		" OR (ci_artifact.component_id = %v  and ci_artifact.data_source= '%v' ))" +
		" AND (ci_artifact.image LIKE '%v' )"

	commonQuery = fmt.Sprintf(commonQuery, listingFilterOptions.PipelineId, listingFilterOptions.StageType, listingFilterOptions.ParentId, listingFilterOptions.ParentStageType, listingFilterOptions.ParentId, listingFilterOptions.PluginStage, listingFilterOptions.SearchString)
	if isApprovalNode {
		commonQuery = commonQuery + fmt.Sprintf(" AND ( cd_workflow.ci_artifact_id NOT IN (SELECT DISTINCT dar.ci_artifact_id FROM deployment_approval_request dar WHERE dar.pipeline_id = %v AND dar.active=true AND dar.artifact_deployment_triggered = false))", listingFilterOptions.PipelineId)
	} else if len(listingFilterOptions.ExcludeArtifactIds) > 0 {
		commonQuery = commonQuery + fmt.Sprintf(" AND ( cd_workflow.ci_artifact_id NOT IN (%v))", helper.GetCommaSepratedString(listingFilterOptions.ExcludeArtifactIds))
	}

	totalCountQuery := "SELECT COUNT(DISTINCT ci_artifact.id) as total_count " + commonQuery
	selectQuery := fmt.Sprintf("SELECT DISTINCT(ci_artifact.id) , (%v) ", totalCountQuery)
	// GroupByQuery := " GROUP BY cia.id "
	limitOffSetQuery := fmt.Sprintf(" order by ci_artifact.id desc LIMIT %v OFFSET %v", listingFilterOptions.Limit, listingFilterOptions.Offset)

	// finalQuery := selectQuery + commonQuery + GroupByQuery + limitOffSetQuery
	finalQuery := selectQuery + commonQuery + limitOffSetQuery
	return finalQuery
}

func buildQueryForArtifactsForCdStageV2(listingFilterOptions bean.ArtifactsListFilterOptions, isApprovalNode bool) string {
	whereCondition := fmt.Sprintf(" WHERE ( id IN ("+
		" SELECT DISTINCT(cd_workflow.ci_artifact_id) as ci_artifact_id "+
		" FROM cd_workflow_runner"+
		" INNER JOIN cd_workflow ON cd_workflow.id = cd_workflow_runner.cd_workflow_id "+
		" AND (cd_workflow.pipeline_id = %d OR cd_workflow.pipeline_id = %d)"+
		"    WHERE ("+
		"            (cd_workflow.pipeline_id = %d AND cd_workflow_runner.workflow_type = '%s')"+
		"            OR"+
		"            (cd_workflow.pipeline_id = %d"+
		"                AND cd_workflow_runner.workflow_type = '%s'"+
		"                AND cd_workflow_runner.status IN ('Healthy','Succeeded')"+
		"           )"+
		"      )   ) ", listingFilterOptions.PipelineId, listingFilterOptions.ParentId, listingFilterOptions.PipelineId, listingFilterOptions.StageType, listingFilterOptions.ParentId, listingFilterOptions.ParentStageType)

	// plugin artifacts
	whereCondition = fmt.Sprintf(" %s OR (ci_artifact.component_id = %d  AND ci_artifact.data_source= '%s' )", whereCondition, listingFilterOptions.ParentId, listingFilterOptions.PluginStage)

	// TODO: move constant. handle approval node
	// promoted artifacts
	// destination pipeline-id and artifact-id are indexed
	if listingFilterOptions.ParentStageType != bean.CD_WORKFLOW_TYPE_PRE && listingFilterOptions.StageType != bean.CD_WORKFLOW_TYPE_POST {
		whereCondition = fmt.Sprintf(" %s OR id in (select artifact_id from artifact_promotion_approval_request where status=%d and destination_pipeline_id = %d ) )", whereCondition, constants.PROMOTED, listingFilterOptions.PipelineId)
	}

	if listingFilterOptions.SearchString != EmptyLikeRegex {
		whereCondition = whereCondition + fmt.Sprintf(" AND ci_artifact.image LIKE '%s' ", listingFilterOptions.SearchString)
	}
	if isApprovalNode {
		whereCondition = whereCondition + fmt.Sprintf(" AND ( ci_artifact.id NOT IN (SELECT DISTINCT dar.ci_artifact_id FROM deployment_approval_request dar WHERE dar.pipeline_id = %d AND dar.active=true AND dar.artifact_deployment_triggered = false))", listingFilterOptions.PipelineId)
	} else if len(listingFilterOptions.ExcludeArtifactIds) > 0 {
		whereCondition = whereCondition + fmt.Sprintf(" AND ( ci_artifact.id NOT IN (%s))", helper.GetCommaSepratedString(listingFilterOptions.ExcludeArtifactIds))
	}

	selectQuery := fmt.Sprintf(" SELECT ci_artifact.* ,COUNT(id) OVER() AS total_count " +
		" FROM ci_artifact")
	ordeyByAndPaginated := fmt.Sprintf(" ORDER BY id DESC LIMIT %d OFFSET %d ", listingFilterOptions.Limit, listingFilterOptions.Offset)
	finalQuery := selectQuery + whereCondition + ordeyByAndPaginated
	return finalQuery
}

func BuildQueryForArtifactsForRollback(listingFilterOptions bean.ArtifactsListFilterOptions) string {
	commonQuery := " FROM cd_workflow_runner cdwr " +
		" INNER JOIN cd_workflow cdw ON cdw.id=cdwr.cd_workflow_id " +
		" INNER JOIN ci_artifact cia ON cia.id=cdw.ci_artifact_id " +
		" WHERE cdw.pipeline_id=%v AND cdwr.workflow_type = '%v' "

	commonQuery = fmt.Sprintf(commonQuery, listingFilterOptions.PipelineId, listingFilterOptions.StageType)
	if listingFilterOptions.SearchString != EmptyLikeRegex {
		commonQuery += fmt.Sprintf(" AND cia.image LIKE '%v' ", listingFilterOptions.SearchString)
	}
	if len(listingFilterOptions.ExcludeWfrIds) > 0 {
		commonQuery = fmt.Sprintf(" %s AND cdwr.id NOT IN (%s)", commonQuery, helper.GetCommaSepratedString(listingFilterOptions.ExcludeWfrIds))
	}
	totalCountQuery := " SELECT COUNT(cia.id) as total_count " + commonQuery
	orderByQuery := " ORDER BY cdwr.id DESC "
	limitOffsetQuery := fmt.Sprintf("LIMIT %v OFFSET %v", listingFilterOptions.Limit, listingFilterOptions.Offset)
	finalQuery := fmt.Sprintf(" SELECT cdwr.id as cd_workflow_runner_id,cdwr.triggered_by,cdwr.started_on,cia.*,(%s) ", totalCountQuery) + commonQuery + orderByQuery + limitOffsetQuery
	return finalQuery
}

func BuildApprovedOnlyArtifactsWithFilter(listingFilterOpts bean.ArtifactsListFilterOptions) string {
	withQuery := "WITH " +
		" approved_images AS " +
		" ( " +
		" SELECT approval_request_id,count(approval_request_id) AS approval_count " +
		" FROM request_approval_user_data daud " +
		fmt.Sprintf(" WHERE user_response is NULL AND request_type=%d", DEPLOYMENT_APPROVAL) +
		" GROUP BY approval_request_id " +
		" ) "
	countQuery := " SELECT count(cia.created_on) as total_count"

	commonQueryPart := " FROM deployment_approval_request dar " +
		" INNER JOIN approved_images ai ON ai.approval_request_id=dar.id AND ai.approval_count >= %v " +
		" INNER JOIN ci_artifact cia ON cia.id = dar.ci_artifact_id " +
		" WHERE dar.active=true AND dar.artifact_deployment_triggered = false AND dar.pipeline_id = %v "

	if listingFilterOpts.SearchString != EmptyLikeRegex {
		commonQueryPart += fmt.Sprintf(" AND cia.image LIKE '%v' ", listingFilterOpts.SearchString)
	}
	commonQueryPart = fmt.Sprintf(commonQueryPart, listingFilterOpts.ApproversCount, listingFilterOpts.PipelineId)
	if len(listingFilterOpts.ExcludeArtifactIds) > 0 {
		commonQueryPart += fmt.Sprintf(" AND cia.id NOT IN (%s) ", helper.GetCommaSepratedString(listingFilterOpts.ExcludeArtifactIds))
	}

	orderByClause := " ORDER BY cia.created_on "
	limitOffsetQueryPart := fmt.Sprintf(" LIMIT %v OFFSET %v ", listingFilterOpts.Limit, listingFilterOpts.Offset)
	totalCountQuery := withQuery + countQuery + commonQueryPart
	selectQuery := fmt.Sprintf(" SELECT cia.*,(%s)", totalCountQuery)
	finalQuery := withQuery + selectQuery + commonQueryPart + orderByClause + limitOffsetQueryPart
	return finalQuery
}

func BuildQueryForApprovedArtifactsForRollback(listingFilterOpts bean.ArtifactsListFilterOptions) string {
	subQuery := "WITH approved_requests AS " +
		" (SELECT approval_request_id,count(approval_request_id) AS approval_count " +
		" FROM request_approval_user_data " +
		fmt.Sprintf(" WHERE user_response is NULL AND request_type = %d", DEPLOYMENT_APPROVAL) +
		" GROUP BY approval_request_id ) " +
		" SELECT approval_request_id " +
		" FROM approved_requests WHERE approval_count >= %v "
	subQuery = fmt.Sprintf(subQuery, listingFilterOpts.ApproversCount)
	commonQuery := " FROM cd_workflow_runner cdwr " +
		"   INNER JOIN cd_workflow cdw ON cdw.id=cdwr.cd_workflow_id" +
		"	INNER JOIN ci_artifact cia ON cia.id=cdw.ci_artifact_id" +
		"	INNER JOIN deployment_approval_request dar ON dar.ci_artifact_id = cdw.ci_artifact_id" +
		"   WHERE dar.id IN (%s) AND cdw.pipeline_id = %v" +
		"   AND cdwr.workflow_type = '%v'"
	if listingFilterOpts.SearchString != EmptyLikeRegex {
		commonQuery += fmt.Sprintf(" AND cia.image LIKE '%v' ", listingFilterOpts.SearchString)
	}

	commonQuery = fmt.Sprintf(commonQuery, subQuery, listingFilterOpts.PipelineId, listingFilterOpts.StageType)
	if len(listingFilterOpts.ExcludeWfrIds) > 0 {
		commonQuery = fmt.Sprintf(" %s AND cdwr.id NOT IN (%s)", commonQuery, helper.GetCommaSepratedString(listingFilterOpts.ExcludeWfrIds))
	}

	totalCountQuery := " SELECT COUNT(cia.id) as total_count " + commonQuery
	orderByQuery := " ORDER BY cdwr.id DESC "
	limitOffsetQuery := fmt.Sprintf("LIMIT %v OFFSET %v ", listingFilterOpts.Limit, listingFilterOpts.Offset)
	finalQuery := fmt.Sprintf(" SELECT cdwr.id as cd_workflow_runner_id,cdwr.triggered_by,cdwr.started_on,cia.*,(%s) ", totalCountQuery) + commonQuery + orderByQuery + limitOffsetQuery
	return finalQuery
}
