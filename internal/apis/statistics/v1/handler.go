package v1

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/cloudwego/hertz/pkg/protocol/consts"
	"github.com/cloudwego/hertz/pkg/route"
	dapisv1 "github.com/softcdata/testudo-operator/pkg/apis/disaster/v1"
	listers "github.com/softcdata/testudo-operator/pkg/listers/disaster/v1"
	"github.com/softcdata/testudo-server/internal/common"
	"github.com/softcdata/testudo-server/internal/kube"
	transport "github.com/softcdata/testudo-server/internal/transport"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
)

type StatisticsHandler struct {
	*kube.KubeClient
	Rg               *route.RouterGroup
	Mw               []app.HandlerFunc
	StatsLister      listers.BackupRestoreStatisticsLister
	AppBackupLister  listers.AppBackupLister
	AppRestoreLister listers.AppRestoreLister
	InstanceLister   listers.DisasterInstanceLister
	PolicyLister     listers.DisasterPolicyLister
	StorageLister    listers.StorageRepositoryLister
}

const (
	appResourceOriginLabelKey         = "testudo.softcdata.com/app-resource-origin"
	appResourceOriginUser             = "user"
	appResourceOriginDisasterInstance = "disaster-instance"

	taskProgressTypeBackup      = "backup"
	taskProgressTypeRestore     = "restore"
	taskProgressScopeAll        = "all"
	taskProgressScopeDisaster   = "disaster"
	taskProgressScopeApp        = "app"
	taskProgressDefaultRange    = "7d"
	taskProgressDefaultTimezone = "Asia/Shanghai"
	taskProgressStatusCompleted = "completed"
	taskProgressStatusFailed    = "failed"
	taskProgressStatusCanceled  = "canceled"
	taskProgressStatusRunning   = "inProgress"
	taskProgressStatusUnknown   = "unknown"
)

func NewStatisticsHandler(kc *kube.KubeClient, rg *route.RouterGroup, mw ...app.HandlerFunc) *StatisticsHandler {
	return &StatisticsHandler{
		KubeClient:       kc,
		Rg:               rg,
		Mw:               mw,
		StatsLister:      kc.InformerFactory.Disaster().V1().BackupRestoreStatisticses().Lister(),
		AppBackupLister:  kc.InformerFactory.Disaster().V1().AppBackups().Lister(),
		AppRestoreLister: kc.InformerFactory.Disaster().V1().AppRestores().Lister(),
		InstanceLister:   kc.InformerFactory.Disaster().V1().DisasterInstances().Lister(),
		PolicyLister:     kc.InformerFactory.Disaster().V1().DisasterPolicies().Lister(),
		StorageLister:    kc.InformerFactory.Disaster().V1().StorageRepositories().Lister(),
	}
}

func (h *StatisticsHandler) GetBackupStatistics(c context.Context, ctx *app.RequestContext) {
	h.getStatistics(c, ctx, "AppBackup")
}

func (h *StatisticsHandler) GetRestoreStatistics(c context.Context, ctx *app.RequestContext) {
	h.getStatistics(c, ctx, "AppRestore")
}

// GetOperationStatistics 聚合所有 DisasterOperation 的统计数据
// 遵循与 GetBackupStatistics/GetRestoreStatistics 相同的统计规范：
// 通过 label testudo.softcdata.com/owner-kind=DisasterOperation 查询 BackupRestoreStatistics CR
func (h *StatisticsHandler) GetOperationStatistics(c context.Context, ctx *app.RequestContext) {
	h.getStatistics(c, ctx, "DisasterOperation")
}

func (h *StatisticsHandler) GetInstanceStatistics(c context.Context, ctx *app.RequestContext) {
	namespace := string(ctx.Query("namespace"))
	var items []*dapisv1.DisasterInstance
	var err error

	if namespace != "" {
		items, err = h.InstanceLister.DisasterInstances(namespace).List(labels.Everything())
	} else {
		items, err = h.InstanceLister.List(labels.Everything())
	}

	if err != nil {
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}

	result := InstanceStatisticsDTO{
		Total: int32(len(items)),
	}

	for _, item := range items {
		switch item.Status.FsmState {
		case dapisv1.FsmStateProtected:
			result.Protected++
		case dapisv1.FsmStatePaused:
			result.Paused++
		case dapisv1.FsmStateFailingOver:
			result.FailingOver++
		case dapisv1.FsmStateActive:
			result.Active++
		case dapisv1.FsmStateFailingBack:
			result.FailingBack++
		case dapisv1.FsmStateInitializing:
			result.Initializing++
		case dapisv1.FsmStatePending, "":
			result.Pending++
		case dapisv1.FsmStateFailed:
			result.Failed++
		default:
			// Optionally handle other states or treat as unknown
			result.Pending++
		}
	}

	transport.WriteSuccess(ctx, consts.StatusOK, result, nil)
}

func (h *StatisticsHandler) GetBackupSuccessRate(c context.Context, ctx *app.RequestContext) {
	namespace := string(ctx.Query("namespace"))
	period := string(ctx.Query("period"))
	startTimeStr := string(ctx.Query("startTime"))
	endTimeStr := string(ctx.Query("endTime"))

	startTime, endTime, effectivePeriod, err := parseTimeRange(period, startTimeStr, endTimeStr)
	if err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	selector := labels.Set{
		"testudo.softcdata.com/owner-kind": "AppBackup",
	}.AsSelector()

	var items []*dapisv1.BackupRestoreStatistics
	if namespace != "" {
		items, err = h.StatsLister.BackupRestoreStatisticses(namespace).List(selector)
	} else {
		items, err = h.StatsLister.List(selector)
	}

	if err != nil {
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}

	var completedCount, failedCount int32
	for _, item := range items {
		itemTime := item.CreationTimestamp.Time
		if !startTime.IsZero() && itemTime.Before(startTime) {
			continue
		}
		if !endTime.IsZero() && itemTime.After(endTime) {
			continue
		}

		completedCount += item.Status.Statistics.Completed
		failedCount += item.Status.Statistics.Failed
	}

	total := completedCount + failedCount
	successRate := 100.0
	if total > 0 {
		successRate = (float64(completedCount) / float64(total)) * 100.0
	}

	result := SuccessRateDTO{
		SuccessRate: successRate,
		Details: SuccessRateDetails{
			Completed:       completedCount,
			Failed:          failedCount,
			TotalExecutions: total,
		},
		Period: effectivePeriod,
	}

	transport.WriteSuccess(ctx, consts.StatusOK, result, nil)
}

func (h *StatisticsHandler) GetAutoBackupExecutionSummary(c context.Context, ctx *app.RequestContext) {
	namespace := string(ctx.Query("namespace"))
	period := string(ctx.Query("period"))
	if period == "" {
		// Backward compatibility for the original proposal wording. New callers should use period.
		period = string(ctx.Query("range"))
	}
	if period == "" {
		period = "7d"
	}

	startTime, endTime, err := parseAutoBackupSummaryPeriod(period)
	if err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	var appBackups []*dapisv1.AppBackup
	if namespace != "" {
		appBackups, err = h.AppBackupLister.AppBackups(namespace).List(labels.Everything())
	} else {
		appBackups, err = h.AppBackupLister.List(labels.Everything())
	}
	if err != nil {
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}

	var successCount, failedCount int32
	for _, appBackup := range appBackups {
		if !h.isAutoBackupPolicyAppBackup(appBackup) {
			continue
		}
		for _, record := range appBackup.Status.History {
			recordTime, ok := backupRecordExecutionTime(record)
			if !ok || recordTime.Before(startTime) || recordTime.After(endTime) {
				continue
			}
			switch backupRecordExecutionResult(record) {
			case "success":
				successCount++
			case "failed":
				failedCount++
			}
		}
	}

	total := successCount + failedCount
	result := AutoBackupExecutionSummaryDTO{
		Period: period,
		Range:  period,
		Total:  total,
		Success: AutoBackupExecutionBucket{
			Count:   successCount,
			Percent: percentOf(successCount, total),
		},
		Failed: AutoBackupExecutionBucket{
			Count:   failedCount,
			Percent: percentOf(failedCount, total),
		},
		Window: AutoBackupExecutionWindow{
			Start: common.NewLocalTime(metav1.NewTime(startTime)),
			End:   common.NewLocalTime(metav1.NewTime(endTime)),
		},
	}

	transport.WriteSuccess(ctx, consts.StatusOK, result, nil)
}

func (h *StatisticsHandler) GetTaskProgressTrend(c context.Context, ctx *app.RequestContext) {
	query, err := parseTaskProgressQuery(ctx)
	if err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	aggregator := newTaskProgressAggregator(query)
	switch query.taskType {
	case taskProgressTypeBackup:
		err = h.aggregateBackupTaskProgress(query, aggregator)
	case taskProgressTypeRestore:
		err = h.aggregateRestoreTaskProgress(query, aggregator)
	default:
		err = fmt.Errorf("invalid type: %s, must be one of backup|restore", query.taskType)
	}
	if err != nil {
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}

	result := aggregator.result()
	transport.WriteSuccess(ctx, consts.StatusOK, result, map[string]interface{}{
		"resourceType": "backupRestoreTaskProgress",
		"filters": map[string]string{
			"type":      query.taskType,
			"scope":     query.scope,
			"range":     query.period,
			"timezone":  query.timezone,
			"namespace": query.namespace,
			"cluster":   query.cluster,
		},
	})
}

func (h *StatisticsHandler) GetOperationStatisticsByTime(c context.Context, ctx *app.RequestContext) {
	namespace := string(ctx.Query("namespace"))
	period := string(ctx.Query("period"))
	startTimeStr := string(ctx.Query("startTime"))
	endTimeStr := string(ctx.Query("endTime"))

	startTime, endTime, effectivePeriod, err := parseTimeRange(period, startTimeStr, endTimeStr)
	if err != nil {
		transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
		return
	}

	selector := labels.Set{
		"testudo.softcdata.com/owner-kind": "DisasterOperation",
	}.AsSelector()

	var items []*dapisv1.BackupRestoreStatistics

	if namespace != "" {
		items, err = h.StatsLister.BackupRestoreStatisticses(namespace).List(selector)
	} else {
		items, err = h.StatsLister.List(selector)
	}

	if err != nil {
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}

	result := StatisticsDTO{
		Period: effectivePeriod,
	}
	if !startTime.IsZero() {
		t := common.NewLocalTime(metav1.NewTime(startTime))
		result.StartTime = &t
	}
	if !endTime.IsZero() {
		t := common.NewLocalTime(metav1.NewTime(endTime))
		result.EndTime = &t
	}

	for _, item := range items {
		itemTime := item.CreationTimestamp.Time
		if !startTime.IsZero() && itemTime.Before(startTime) {
			continue
		}
		if !endTime.IsZero() && itemTime.After(endTime) {
			continue
		}

		result.Total += item.Status.Statistics.Total
		result.InProgress += item.Status.Statistics.InProgress
		result.Completed += item.Status.Statistics.Completed
		result.Failed += item.Status.Statistics.Failed
		result.Canceled += item.Status.Statistics.Canceled
		result.Unknown += item.Status.Statistics.Unknown
	}

	transport.WriteSuccess(ctx, consts.StatusOK, result, nil)
}

func parseTimeRange(period, startTimeStr, endTimeStr string) (time.Time, time.Time, string, error) {
	var startTime, endTime time.Time
	var err error

	effectivePeriod := period
	if effectivePeriod == "" && startTimeStr == "" && endTimeStr == "" {
		effectivePeriod = "all"
	}

	now := time.Now()
	switch effectivePeriod {
	case "today":
		startTime = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		endTime = now
	case "week":
		offset := int(time.Monday - now.Weekday())
		if offset > 0 {
			offset = -6
		}
		startTime = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()).AddDate(0, 0, offset)
		endTime = now
	case "month":
		startTime = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
		endTime = now
	case "all", "":
		// No time filtering
		startTime = time.Time{}
		endTime = time.Time{}
	}

	if startTimeStr != "" {
		startTime, err = time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			return time.Time{}, time.Time{}, "", fmt.Errorf("invalid startTime format, must be RFC3339")
		}
		effectivePeriod = "custom"
	}
	if endTimeStr != "" {
		endTime, err = time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			return time.Time{}, time.Time{}, "", fmt.Errorf("invalid endTime format, must be RFC3339")
		}
		effectivePeriod = "custom"
	}

	return startTime, endTime, effectivePeriod, nil
}

func parseAutoBackupSummaryPeriod(period string) (time.Time, time.Time, error) {
	var days int
	switch period {
	case "7d":
		days = 7
	case "30d":
		days = 30
	case "90d":
		days = 90
	default:
		return time.Time{}, time.Time{}, fmt.Errorf("invalid period: %s, must be one of 7d|30d|90d", period)
	}
	endTime := time.Now()
	startTime := endTime.AddDate(0, 0, -days)
	return startTime, endTime, nil
}

func backupRecordExecutionTime(record dapisv1.BackupRecord) (time.Time, bool) {
	if record.CompletionTimestamp != nil && !record.CompletionTimestamp.IsZero() {
		return record.CompletionTimestamp.Time, true
	}
	if record.StartTimestamp != nil && !record.StartTimestamp.IsZero() {
		return record.StartTimestamp.Time, true
	}
	return time.Time{}, false
}

func backupRecordExecutionResult(record dapisv1.BackupRecord) string {
	switch strings.TrimSpace(record.ManagedStatus) {
	case dapisv1.LastBackupStatusCompleted:
		return "success"
	case dapisv1.LastBackupStatusFailed:
		return "failed"
	}
	switch strings.TrimSpace(record.Phase) {
	case "Completed":
		return "success"
	case "Failed", "PartiallyFailed", "FailedValidation":
		return "failed"
	default:
		return ""
	}
}

type taskProgressQuery struct {
	taskType  string
	scope     string
	period    string
	timezone  string
	namespace string
	cluster   string
	location  *time.Location
	startTime time.Time
	endTime   time.Time
	days      int
}

type taskProgressAggregator struct {
	query       taskProgressQuery
	summary     TaskProgressCountDTO
	buckets     []TaskProgressBucketDTO
	bucketIndex map[string]int
	sources     map[string]*TaskProgressSourceDTO
	sourceOrder []string
}

func parseTaskProgressQuery(ctx *app.RequestContext) (taskProgressQuery, error) {
	taskType := strings.TrimSpace(string(ctx.Query("type")))
	switch taskType {
	case taskProgressTypeBackup, taskProgressTypeRestore:
	case "":
		return taskProgressQuery{}, fmt.Errorf("type is required, must be one of backup|restore")
	default:
		return taskProgressQuery{}, fmt.Errorf("invalid type: %s, must be one of backup|restore", taskType)
	}

	scope := strings.TrimSpace(string(ctx.Query("scope")))
	if scope == "" {
		scope = taskProgressScopeAll
	}
	switch scope {
	case taskProgressScopeAll, taskProgressScopeDisaster, taskProgressScopeApp:
	default:
		return taskProgressQuery{}, fmt.Errorf("invalid scope: %s, must be one of all|disaster|app", scope)
	}

	period := strings.TrimSpace(string(ctx.Query("range")))
	if period == "" {
		period = taskProgressDefaultRange
	}
	days, err := taskProgressRangeDays(period)
	if err != nil {
		return taskProgressQuery{}, err
	}

	timezone := strings.TrimSpace(string(ctx.Query("timezone")))
	if timezone == "" {
		timezone = taskProgressDefaultTimezone
	}
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return taskProgressQuery{}, fmt.Errorf("invalid timezone: %s", timezone)
	}

	endTime := time.Now().In(location)
	endDay := time.Date(endTime.Year(), endTime.Month(), endTime.Day(), 0, 0, 0, 0, location)
	startTime := endDay.AddDate(0, 0, -(days - 1))

	return taskProgressQuery{
		taskType:  taskType,
		scope:     scope,
		period:    period,
		timezone:  timezone,
		namespace: strings.TrimSpace(string(ctx.Query("namespace"))),
		cluster:   strings.TrimSpace(string(ctx.Query("cluster"))),
		location:  location,
		startTime: startTime,
		endTime:   endTime,
		days:      days,
	}, nil
}

func taskProgressRangeDays(period string) (int, error) {
	switch period {
	case "7d":
		return 7, nil
	case "30d":
		return 30, nil
	case "90d":
		return 90, nil
	default:
		return 0, fmt.Errorf("invalid range: %s, must be one of 7d|30d|90d", period)
	}
}

func newTaskProgressAggregator(query taskProgressQuery) *taskProgressAggregator {
	aggregator := &taskProgressAggregator{
		query:       query,
		buckets:     make([]TaskProgressBucketDTO, 0, query.days),
		bucketIndex: make(map[string]int, query.days),
		sources:     make(map[string]*TaskProgressSourceDTO),
	}

	for i := 0; i < query.days; i++ {
		date := query.startTime.AddDate(0, 0, i).Format("2006-01-02")
		aggregator.bucketIndex[date] = i
		aggregator.buckets = append(aggregator.buckets, TaskProgressBucketDTO{Date: date})
	}

	for _, scope := range taskProgressIncludedSourceScopes(query.scope) {
		aggregator.ensureSource(scope)
	}

	return aggregator
}

func taskProgressIncludedSourceScopes(scope string) []string {
	switch scope {
	case taskProgressScopeApp:
		return []string{taskProgressScopeApp}
	case taskProgressScopeDisaster:
		return []string{taskProgressScopeDisaster}
	default:
		return []string{taskProgressScopeDisaster, taskProgressScopeApp}
	}
}

func (a *taskProgressAggregator) add(t time.Time, sourceScope, status string) {
	if t.IsZero() {
		return
	}
	localTime := t.In(a.query.location)
	if localTime.Before(a.query.startTime) || localTime.After(a.query.endTime) {
		return
	}

	date := localTime.Format("2006-01-02")
	index, ok := a.bucketIndex[date]
	if !ok {
		return
	}

	incrementTaskProgressCount(&a.summary, status)
	incrementTaskProgressBucket(&a.buckets[index], status)

	source := a.ensureSource(sourceScope)
	source.Total++
	switch status {
	case taskProgressStatusCompleted:
		source.Completed++
	case taskProgressStatusFailed:
		source.Failed++
	}
}

func (a *taskProgressAggregator) ensureSource(scope string) *TaskProgressSourceDTO {
	if source, ok := a.sources[scope]; ok {
		return source
	}

	source := &TaskProgressSourceDTO{
		Scope: scope,
		Label: taskProgressSourceLabel(a.query.taskType, scope),
	}
	a.sources[scope] = source
	a.sourceOrder = append(a.sourceOrder, scope)
	return source
}

func (a *taskProgressAggregator) result() TaskProgressTrendDTO {
	sources := make([]TaskProgressSourceDTO, 0, len(a.sourceOrder))
	for _, scope := range a.sourceOrder {
		if source, ok := a.sources[scope]; ok {
			sources = append(sources, *source)
		}
	}

	return TaskProgressTrendDTO{
		Type:      a.query.taskType,
		Scope:     a.query.scope,
		Range:     a.query.period,
		Timezone:  a.query.timezone,
		StartTime: common.NewLocalTime(metav1.NewTime(a.query.startTime)),
		EndTime:   common.NewLocalTime(metav1.NewTime(a.query.endTime)),
		Summary:   a.summary,
		Buckets:   a.buckets,
		Series:    taskProgressSeries(a.query.taskType),
		Sources:   sources,
	}
}

func incrementTaskProgressCount(count *TaskProgressCountDTO, status string) {
	count.Total++
	switch status {
	case taskProgressStatusCompleted:
		count.Completed++
	case taskProgressStatusFailed:
		count.Failed++
	case taskProgressStatusCanceled:
		count.Canceled++
	case taskProgressStatusRunning:
		count.InProgress++
	default:
		count.Unknown++
	}
}

func incrementTaskProgressBucket(bucket *TaskProgressBucketDTO, status string) {
	bucket.Total++
	switch status {
	case taskProgressStatusCompleted:
		bucket.Completed++
	case taskProgressStatusFailed:
		bucket.Failed++
	case taskProgressStatusCanceled:
		bucket.Canceled++
	case taskProgressStatusRunning:
		bucket.InProgress++
	default:
		bucket.Unknown++
	}
}

func taskProgressSeries(taskType string) []TaskProgressSeriesDTO {
	switch taskType {
	case taskProgressTypeRestore:
		return []TaskProgressSeriesDTO{
			{Key: taskProgressStatusCompleted, Label: "恢复成功"},
			{Key: taskProgressStatusFailed, Label: "恢复失败"},
		}
	default:
		return []TaskProgressSeriesDTO{
			{Key: taskProgressStatusCompleted, Label: "备份成功"},
			{Key: taskProgressStatusFailed, Label: "备份失败"},
		}
	}
}

func taskProgressSourceLabel(taskType, scope string) string {
	if taskType == taskProgressTypeRestore {
		if scope == taskProgressScopeDisaster {
			return "容灾恢复"
		}
		return "应用恢复"
	}
	if scope == taskProgressScopeDisaster {
		return "容灾备份"
	}
	return "应用备份"
}

func (h *StatisticsHandler) aggregateBackupTaskProgress(query taskProgressQuery, aggregator *taskProgressAggregator) error {
	var appBackups []*dapisv1.AppBackup
	var err error
	if query.namespace != "" {
		appBackups, err = h.AppBackupLister.AppBackups(query.namespace).List(labels.Everything())
	} else {
		appBackups, err = h.AppBackupLister.List(labels.Everything())
	}
	if err != nil {
		return err
	}

	for _, appBackup := range appBackups {
		if query.cluster != "" && appBackup.Spec.Cluster != query.cluster {
			continue
		}
		sourceScope := taskProgressScopeFromOrigin(h.inferAppBackupOrigin(appBackup.Namespace, appBackup.Name))
		if !taskProgressScopeMatches(query.scope, sourceScope) {
			continue
		}
		for _, record := range appBackup.Status.History {
			status := backupRecordProgressStatus(record)
			recordTime, ok := backupRecordProgressTime(record, status)
			if !ok {
				continue
			}
			aggregator.add(recordTime, sourceScope, status)
		}
	}
	return nil
}

func (h *StatisticsHandler) aggregateRestoreTaskProgress(query taskProgressQuery, aggregator *taskProgressAggregator) error {
	var appRestores []*dapisv1.AppRestore
	var err error
	if query.namespace != "" {
		appRestores, err = h.AppRestoreLister.AppRestores(query.namespace).List(labels.Everything())
	} else {
		appRestores, err = h.AppRestoreLister.List(labels.Everything())
	}
	if err != nil {
		return err
	}

	for _, appRestore := range appRestores {
		if query.cluster != "" && appRestore.Spec.Cluster != query.cluster {
			continue
		}
		sourceScope := taskProgressScopeFromOrigin(h.inferAppRestoreOrigin(appRestore.Namespace, appRestore.Name))
		if !taskProgressScopeMatches(query.scope, sourceScope) {
			continue
		}
		aggregator.add(restoreProgressTime(appRestore), sourceScope, restoreProgressStatus(appRestore.Status.Status))
	}
	return nil
}

func taskProgressScopeFromOrigin(origin string) string {
	if origin == appResourceOriginDisasterInstance {
		return taskProgressScopeDisaster
	}
	return taskProgressScopeApp
}

func taskProgressScopeMatches(filter, sourceScope string) bool {
	return filter == taskProgressScopeAll || filter == sourceScope
}

func backupRecordProgressStatus(record dapisv1.BackupRecord) string {
	switch strings.TrimSpace(record.ManagedStatus) {
	case dapisv1.LastBackupStatusCompleted:
		return taskProgressStatusCompleted
	case dapisv1.LastBackupStatusFailed:
		return taskProgressStatusFailed
	case dapisv1.LastBackupStatusCanceled:
		return taskProgressStatusCanceled
	case dapisv1.LastBackupStatusInProgress:
		return taskProgressStatusRunning
	case dapisv1.LastBackupStatusUnknown:
		return taskProgressStatusUnknown
	}

	switch strings.TrimSpace(record.Phase) {
	case "Completed":
		return taskProgressStatusCompleted
	case "Failed", "PartiallyFailed", "FailedValidation":
		return taskProgressStatusFailed
	case "Canceled", "Cancelled":
		return taskProgressStatusCanceled
	case "New", "InProgress":
		return taskProgressStatusRunning
	default:
		return taskProgressStatusUnknown
	}
}

func backupRecordProgressTime(record dapisv1.BackupRecord, status string) (time.Time, bool) {
	if status == taskProgressStatusRunning {
		if record.StartTimestamp != nil && !record.StartTimestamp.IsZero() {
			return record.StartTimestamp.Time, true
		}
		return time.Time{}, false
	}
	if record.CompletionTimestamp != nil && !record.CompletionTimestamp.IsZero() {
		return record.CompletionTimestamp.Time, true
	}
	if record.StartTimestamp != nil && !record.StartTimestamp.IsZero() {
		return record.StartTimestamp.Time, true
	}
	return time.Time{}, false
}

func restoreProgressStatus(status dapisv1.AppRestorePhase) string {
	switch status {
	case dapisv1.PhaseSucceeded:
		return taskProgressStatusCompleted
	case dapisv1.PhaseFailed:
		return taskProgressStatusFailed
	case dapisv1.PhaseCancelled:
		return taskProgressStatusCanceled
	case dapisv1.PhasePending, dapisv1.PhaseInitiating, dapisv1.PhaseRestoring, dapisv1.PhaseDeleting:
		return taskProgressStatusRunning
	default:
		return taskProgressStatusUnknown
	}
}

func restoreProgressTime(appRestore *dapisv1.AppRestore) time.Time {
	if appRestore.Status.RestoreStatus.CompletionTimestamp != nil && !appRestore.Status.RestoreStatus.CompletionTimestamp.IsZero() {
		return appRestore.Status.RestoreStatus.CompletionTimestamp.Time
	}
	if appRestore.Status.RestoreStatus.StartTimestamp != nil && !appRestore.Status.RestoreStatus.StartTimestamp.IsZero() {
		return appRestore.Status.RestoreStatus.StartTimestamp.Time
	}
	return appRestore.CreationTimestamp.Time
}

func percentOf(count, total int32) int32 {
	if total == 0 {
		return 0
	}
	return int32(math.Round(float64(count) * 100 / float64(total)))
}

func (h *StatisticsHandler) getStatistics(c context.Context, ctx *app.RequestContext, ownerKind string) {
	namespace := ctx.Query("namespace")
	originFilter := "all"
	if ownerKind == "AppBackup" || ownerKind == "AppRestore" {
		var err error
		originFilter, err = parseAppResourceOriginFilter(ctx.Query("origin"))
		if err != nil {
			transport.WriteError(ctx, transport.CodeBadRequest, err.Error(), nil)
			return
		}
	}

	// Build selector
	selector := labels.Set{
		"testudo.softcdata.com/owner-kind": ownerKind,
	}.AsSelector()

	var items []*dapisv1.BackupRestoreStatistics
	var err error

	if namespace != "" {
		items, err = h.StatsLister.BackupRestoreStatisticses(namespace).List(selector)
	} else {
		items, err = h.StatsLister.List(selector)
	}

	if err != nil {
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}

	// Aggregate
	result := StatisticsDTO{}
	for _, item := range items {
		if !h.matchStatisticsOrigin(item, ownerKind, originFilter) {
			continue
		}

		result.Total += item.Status.Statistics.Total
		result.InProgress += item.Status.Statistics.InProgress
		result.Completed += item.Status.Statistics.Completed
		result.Failed += item.Status.Statistics.Failed
		result.Canceled += item.Status.Statistics.Canceled
		result.Unknown += item.Status.Statistics.Unknown
	}

	transport.WriteSuccess(ctx, consts.StatusOK, result, nil)
}

func parseAppResourceOriginFilter(origin string) (string, error) {
	switch origin {
	case "", "user":
		return "user", nil
	case "instance":
		return "instance", nil
	case "all":
		return "all", nil
	default:
		return "", fmt.Errorf("invalid origin: %s, must be one of user|instance|all", origin)
	}
}

func (h *StatisticsHandler) matchStatisticsOrigin(item *dapisv1.BackupRestoreStatistics, ownerKind, originFilter string) bool {
	if originFilter == "all" {
		return true
	}

	var origin string
	switch ownerKind {
	case "AppBackup":
		origin = h.inferAppBackupOrigin(item.Spec.ScopeRef.Namespace, item.Spec.ScopeRef.Name)
	case "AppRestore":
		origin = h.inferAppRestoreOrigin(item.Spec.ScopeRef.Namespace, item.Spec.ScopeRef.Name)
	default:
		return true
	}

	if originFilter == "instance" {
		return origin == appResourceOriginDisasterInstance
	}
	return origin != appResourceOriginDisasterInstance
}

func (h *StatisticsHandler) inferAppBackupOrigin(namespace, name string) string {
	if namespace != "" && name != "" {
		if item, err := h.AppBackupLister.AppBackups(namespace).Get(name); err == nil {
			return inferAppResourceOrigin(item.Name, item.Labels, item.OwnerReferences, false)
		}
	}

	// 兼容历史数据：部分统计对象可能先于来源标签接入
	if strings.HasPrefix(name, "ds-") || strings.HasPrefix(name, "rs-") {
		return appResourceOriginDisasterInstance
	}

	return appResourceOriginUser
}

func (h *StatisticsHandler) isAutoBackupPolicyAppBackup(appBackup *dapisv1.AppBackup) bool {
	if appBackup == nil || strings.TrimSpace(appBackup.Spec.DisasterPolicy) == "" {
		return false
	}
	policy, err := h.PolicyLister.DisasterPolicies(appBackup.Namespace).Get(appBackup.Spec.DisasterPolicy)
	if err != nil {
		return false
	}
	return policy.Spec.Type == dapisv1.PolicyTypeAutoBackup
}

func (h *StatisticsHandler) inferAppRestoreOrigin(namespace, name string) string {
	if namespace != "" && name != "" {
		if item, err := h.AppRestoreLister.AppRestores(namespace).Get(name); err == nil {
			return inferAppResourceOrigin(item.Name, item.Labels, item.OwnerReferences, true)
		}
	}

	// 兼容历史数据：兜底识别系统恢复任务命名前缀
	if strings.HasPrefix(name, "rec-ds-") || strings.HasPrefix(name, "rec-rs-") || strings.HasPrefix(name, "ddr-") || strings.HasPrefix(name, "drr-") {
		return appResourceOriginDisasterInstance
	}

	return appResourceOriginUser
}

func inferAppResourceOrigin(resourceName string, resourceLabels map[string]string, ownerRefs []metav1.OwnerReference, includeDrillFallback bool) string {
	for _, ownerRef := range ownerRefs {
		if ownerRef.Controller != nil && *ownerRef.Controller {
			if ownerRef.Kind == "DataSync" || ownerRef.Kind == "ResourceSync" {
				return appResourceOriginDisasterInstance
			}
		}
	}

	if includeDrillFallback && isDrillManagedRestore(resourceName, resourceLabels) {
		return appResourceOriginDisasterInstance
	}

	if resourceLabels != nil {
		if value := resourceLabels[appResourceOriginLabelKey]; value != "" {
			return value
		}
	}

	return appResourceOriginUser
}

func isDrillManagedRestore(resourceName string, resourceLabels map[string]string) bool {
	if resourceLabels != nil {
		if _, ok := resourceLabels["testudo.softcdata.com/drill"]; ok {
			return true
		}
	}

	return strings.HasPrefix(resourceName, "ddr-") || strings.HasPrefix(resourceName, "drr-")
}

func (h *StatisticsHandler) GetStorageStatistics(c context.Context, ctx *app.RequestContext) {
	storageName := string(ctx.Query("storageName"))

	var items []*dapisv1.StorageRepository
	var err error

	if storageName != "" {
		// Single query by exact name (cluster-scoped conceptually, or namespace-less/any namespace, assuming one cluster-wide list)
		// Usually StorageRepository is cluster scoped, so we can just list all and filter, or fetch by name.
		itemsTmp, err := h.StorageLister.List(labels.Everything())
		if err != nil {
			transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
			return
		}
		for _, item := range itemsTmp {
			if item.Name == storageName {
				items = append(items, item)
				break
			}
		}
	} else {
		items, err = h.StorageLister.List(labels.Everything())
	}

	if err != nil {
		transport.WriteError(ctx, transport.CodeInternalServerError, err.Error(), nil)
		return
	}

	result := StorageUsageDTO{}

	for _, item := range items {
		result.TotalBackupCount += item.Status.TotalBackupCount
		result.UsedSpaceBytes += item.Status.UsedSpaceBytes
		result.QuotaBytes += item.Spec.QuotaBytes
	}

	if result.QuotaBytes > 0 {
		result.AvailableSpaceBytes = result.QuotaBytes - result.UsedSpaceBytes
		if result.AvailableSpaceBytes < 0 {
			result.AvailableSpaceBytes = 0
		}
		result.UsageRate = float64(result.UsedSpaceBytes) / float64(result.QuotaBytes)
	} else {
		result.AvailableSpaceBytes = 0
		result.UsageRate = 0
	}

	transport.WriteSuccess(ctx, consts.StatusOK, result, nil)
}
