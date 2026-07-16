package hr_tools

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	gogrpc "google.golang.org/grpc"

	"smart-recruit-platform-go/errs"
	"smart-recruit-proto/recruitment/pb"
)

type fakeAppListClient struct {
	byJob      map[int64][]*pb.JobApplication
	errorByJob map[int64]error
}

func (f *fakeAppListClient) ListJobApplications(_ context.Context, req *pb.ListJobApplicationsRequest, _ ...gogrpc.CallOption) (*pb.ListJobApplicationsResponse, error) {
	if err := f.errorByJob[req.GetJobId()]; err != nil {
		return nil, err
	}
	list := f.byJob[req.GetJobId()]
	return &pb.ListJobApplicationsResponse{Code: errs.OK, Total: int64(len(list)), List: list}, nil
}

type fakeSnapshotClient struct {
	resp *pb.GetApplicationSnapshotResponse
}

type boundedAppListClient struct {
	mu          sync.Mutex
	active      int
	maxActive   int
	callsByJob  map[int64]int
	errorByJob  map[int64]error
	rowsPerPage int
	totalPages  int
	block       bool
}

func (f *boundedAppListClient) ListJobApplications(ctx context.Context, req *pb.ListJobApplicationsRequest, _ ...gogrpc.CallOption) (*pb.ListJobApplicationsResponse, error) {
	f.mu.Lock()
	if f.callsByJob == nil {
		f.callsByJob = map[int64]int{}
	}
	f.callsByJob[req.GetJobId()]++
	f.active++
	if f.active > f.maxActive {
		f.maxActive = f.active
	}
	f.mu.Unlock()
	defer func() {
		f.mu.Lock()
		f.active--
		f.mu.Unlock()
	}()
	if f.block {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	time.Sleep(time.Millisecond)
	if err := f.errorByJob[req.GetJobId()]; err != nil {
		return nil, err
	}
	pages := f.totalPages
	if pages <= 0 {
		pages = 1
	}
	rows := f.rowsPerPage
	if rows <= 0 {
		rows = 2
	}
	if int(req.GetPage()) > pages {
		return &pb.ListJobApplicationsResponse{Code: errs.OK, Total: int64(pages * rows)}, nil
	}
	list := make([]*pb.JobApplication, 0, rows)
	for i := rows - 1; i >= 0; i-- {
		id := req.GetJobId()*1_000_000 + int64(req.GetPage())*1_000 + int64(i)
		list = append(list, &pb.JobApplication{ApplicationId: id, IsCurrent: 1})
	}
	return &pb.ListJobApplicationsResponse{Code: errs.OK, Total: int64(pages * rows), List: list}, nil
}

func (f *boundedAppListClient) snapshot() (int, map[int64]int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	calls := make(map[int64]int, len(f.callsByJob))
	for jobID, count := range f.callsByJob {
		calls[jobID] = count
	}
	return f.maxActive, calls
}

func (f *fakeSnapshotClient) GetApplicationSnapshot(context.Context, *pb.GetApplicationSnapshotRequest, ...gogrpc.CallOption) (*pb.GetApplicationSnapshotResponse, error) {
	if f.resp != nil {
		return f.resp, nil
	}
	return &pb.GetApplicationSnapshotResponse{Code: 404, Msg: "not found"}, nil
}

func TestSearchCandidatesAndStatusSummary(t *testing.T) {
	jobs := &fakeJobClient{listResp: &pb.ListJobsResponse{
		Code: errs.OK,
		List: []*pb.Job{
			{JobId: 10, Title: "Backend", Status: 1, ApplicationCount: 2},
			{JobId: 11, Title: "Frontend", Status: 1, ApplicationCount: 1},
		},
	}}
	apps := &fakeAppListClient{byJob: map[int64][]*pb.JobApplication{
		10: {
			{ApplicationId: 1001, RealName: "Ada Lovelace", Phone: "13800138000", Status: 0, IsCurrent: 1, AppliedAt: time.Now().Format("2006-01-02 15:04")},
			{ApplicationId: 1002, RealName: "Bob", Phone: "13900139000", Status: 2, IsCurrent: 1, AppliedAt: time.Now().AddDate(0, 0, -2).Format("2006-01-02 15:04")},
		},
		11: {
			{ApplicationId: 1101, RealName: "Carol", Phone: "13700137000", Status: 1, IsCurrent: 1, AppliedAt: time.Now().Format("2006-01-02 15:04")},
		},
	}}
	exec := &Executor{Jobs: jobs, Applications: apps}

	search, err := exec.Execute(context.Background(), 77, "search_candidates", map[string]any{"keyword": "Ada"})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if !jsonContains(search.Content, "Ada Lovelace") || !jsonContains(search.Content, "Backend") {
		t.Fatalf("search result = %s", search.Content)
	}

	summary, err := exec.Execute(context.Background(), 77, "get_application_status_summary", nil)
	if err != nil {
		t.Fatalf("summary: %v", err)
	}
	var payload struct {
		Total int64 `json:"total"`
	}
	if err := json.Unmarshal([]byte(summary.Content), &payload); err != nil {
		t.Fatalf("unmarshal summary: %v", err)
	}
	if payload.Total != 3 {
		t.Fatalf("total = %d, want 3", payload.Total)
	}

	today, err := exec.Execute(context.Background(), 77, "query_today_applications", nil)
	if err != nil {
		t.Fatalf("today: %v", err)
	}
	if !jsonContains(today.Content, "today_applications") {
		t.Fatalf("today = %s", today.Content)
	}
}

func TestProposeStatusUpdateDoesNotMutate(t *testing.T) {
	jobs := &fakeJobClient{listResp: &pb.ListJobsResponse{
		Code: errs.OK,
		List: []*pb.Job{{JobId: 10, Title: "Backend", Status: 1}},
	}}
	apps := &fakeAppListClient{byJob: map[int64][]*pb.JobApplication{
		10: {{ApplicationId: 1001, RealName: "Ada", Status: 0, IsCurrent: 1}},
	}}
	snaps := &fakeSnapshotClient{resp: &pb.GetApplicationSnapshotResponse{
		Code: errs.OK, ApplicationId: 1001, CandidateName: "Ada", JobTitle: "Backend", JobId: 10, JobHrId: 77,
	}}
	exec := &Executor{Jobs: jobs, Applications: apps, Snapshots: snaps}
	result, err := exec.Execute(context.Background(), 77, "propose_application_status_update", map[string]any{
		"application_id": 1001,
		"status":         2,
	})
	if err != nil {
		t.Fatalf("propose: %v", err)
	}
	if !jsonContains(result.Content, "requires_confirm") || !jsonContains(result.Content, "Ada") {
		t.Fatalf("result = %s", result.Content)
	}
}

func TestApplicationToolsReturnClassifiedBusinessErrors(t *testing.T) {
	exec := &Executor{}

	_, err := exec.Execute(context.Background(), 77, "list_applications_by_job", nil)
	assertToolErrorKind(t, err, "invalid_argument")

	_, err = exec.Execute(context.Background(), 77, "list_applications_by_status", nil)
	assertToolErrorKind(t, err, "invalid_argument")

	_, err = exec.Execute(context.Background(), 77, "search_candidates", map[string]any{"keyword": "  "})
	assertToolErrorKind(t, err, "invalid_argument")

	_, err = exec.Execute(context.Background(), 77, "get_candidate_detail", map[string]any{"application_id": 10})
	assertToolErrorKind(t, err, "downstream")

	_, err = exec.Execute(context.Background(), 77, "propose_application_status_update", map[string]any{"application_id": 10, "status": 9})
	assertToolErrorKind(t, err, "invalid_argument")
}

func TestAggregationAndCandidateSearchDoNotSwallowDownstreamErrors(t *testing.T) {
	jobs := &fakeJobClient{listResp: &pb.ListJobsResponse{Code: errs.OK, List: []*pb.Job{{JobId: 10, Title: "Backend", ApplicationCount: 99}}}}
	apps := &fakeAppListClient{errorByJob: map[int64]error{10: errors.New("recruitment unavailable")}}
	exec := &Executor{Jobs: jobs, Applications: apps}

	_, err := exec.Execute(context.Background(), 77, "query_total_applications", nil)
	assertToolErrorKind(t, err, "downstream")

	_, err = exec.Execute(context.Background(), 77, "search_candidates", map[string]any{"keyword": "Ada"})
	assertToolErrorKind(t, err, "downstream")
}

func TestApplicationAggregationBoundsConcurrencyOrderingAndRows(t *testing.T) {
	jobsList := make([]*pb.Job, 0, 101)
	for id := int64(101); id >= 1; id-- {
		jobsList = append(jobsList, &pb.Job{JobId: id, Title: fmt.Sprintf("job-%03d", id)})
	}
	jobs := &fakeJobClient{listResp: &pb.ListJobsResponse{Code: errs.OK, Total: int64(len(jobsList)), List: jobsList}}
	apps := &boundedAppListClient{rowsPerPage: 100, totalPages: 10}
	exec := &Executor{Jobs: jobs, Applications: apps}

	aggregation, err := exec.listAllApplications(context.Background(), 77, 0, true)
	if err != nil {
		t.Fatalf("listAllApplications returned error: %v", err)
	}
	if len(aggregation.Items) != applicationAggregationMaxRows {
		t.Fatalf("aggregated rows = %d, want %d", len(aggregation.Items), applicationAggregationMaxRows)
	}
	for i := 1; i < len(aggregation.Items); i++ {
		previous, current := aggregation.Items[i-1], aggregation.Items[i]
		if previous.JobID > current.JobID || previous.JobID == current.JobID && previous.App.GetApplicationId() > current.App.GetApplicationId() {
			t.Fatalf("aggregation is not ordered at %d: %#v then %#v", i, previous, current)
		}
	}
	maxActive, calls := apps.snapshot()
	if maxActive > applicationAggregationWorkers {
		t.Fatalf("max concurrency = %d, want <= %d", maxActive, applicationAggregationWorkers)
	}
	if len(calls) != applicationAggregationMaxJobs || calls[101] != 0 {
		t.Fatalf("jobs called = %d, job101=%d; want first %d sorted jobs only", len(calls), calls[101], applicationAggregationMaxJobs)
	}
	for jobID, count := range calls {
		if count > int(applicationAggregationMaxPagesPerJob) {
			t.Fatalf("job %d pages = %d, want <= %d", jobID, count, applicationAggregationMaxPagesPerJob)
		}
	}
}

func TestApplicationAggregationPartialAndAllFailureSemantics(t *testing.T) {
	jobs := &fakeJobClient{listResp: &pb.ListJobsResponse{Code: errs.OK, List: []*pb.Job{{JobId: 1}, {JobId: 2}, {JobId: 3}}}}
	apps := &boundedAppListClient{errorByJob: map[int64]error{1: errors.New("one"), 2: errors.New("two")}}
	exec := &Executor{Jobs: jobs, Applications: apps}

	result, err := exec.Execute(context.Background(), 77, "list_all_applications", nil)
	if err != nil {
		t.Fatalf("partial aggregation returned error: %v", err)
	}
	var partial struct {
		Partial        bool     `json:"partial"`
		FailedJobCount int      `json:"failed_job_count"`
		Warnings       []string `json:"warnings"`
		Total          int      `json:"total"`
	}
	if err := json.Unmarshal([]byte(result.Content), &partial); err != nil {
		t.Fatalf("unmarshal partial result: %v", err)
	}
	if !partial.Partial || partial.FailedJobCount != 2 || len(partial.Warnings) != 2 || partial.Total == 0 {
		t.Fatalf("partial payload = %#v", partial)
	}

	apps.errorByJob[3] = errors.New("three")
	_, err = exec.Execute(context.Background(), 77, "list_all_applications", nil)
	assertToolErrorKind(t, err, "downstream")
}

func TestApplicationAggregationHonorsCancellation(t *testing.T) {
	jobs := &fakeJobClient{listResp: &pb.ListJobsResponse{Code: errs.OK, List: []*pb.Job{{JobId: 1}, {JobId: 2}, {JobId: 3}, {JobId: 4}, {JobId: 5}}}}
	apps := &boundedAppListClient{block: true}
	exec := &Executor{Jobs: jobs, Applications: apps}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := exec.listAllApplications(ctx, 77, 0, true)
		done <- err
	}()
	time.Sleep(10 * time.Millisecond)
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("cancellation error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("aggregation did not stop promptly after cancellation")
	}
}

func TestApplicationAggregationBoundsWarnings(t *testing.T) {
	jobList := make([]*pb.Job, 0, 12)
	errorsByJob := make(map[int64]error, 11)
	for id := int64(1); id <= 12; id++ {
		jobList = append(jobList, &pb.Job{JobId: id})
		if id <= 11 {
			errorsByJob[id] = errors.New("sensitive downstream detail must not be copied")
		}
	}
	exec := &Executor{
		Jobs:         &fakeJobClient{listResp: &pb.ListJobsResponse{Code: errs.OK, List: jobList}},
		Applications: &boundedAppListClient{errorByJob: errorsByJob},
	}
	aggregation, err := exec.listAllApplications(context.Background(), 77, 0, true)
	if err != nil {
		t.Fatalf("partial aggregation returned error: %v", err)
	}
	if !aggregation.Partial || aggregation.FailedJobCount != 11 || len(aggregation.Warnings) != applicationAggregationMaxWarnings {
		t.Fatalf("aggregation metadata = %#v", aggregation)
	}
	for _, warning := range aggregation.Warnings {
		if strings.Contains(warning, "sensitive downstream detail") {
			t.Fatalf("warning leaked downstream error: %q", warning)
		}
	}
}

func TestPartialAggregationDoesNotTurnUnknownOwnershipIntoFalseDenial(t *testing.T) {
	jobs := &fakeJobClient{listResp: &pb.ListJobsResponse{Code: errs.OK, List: []*pb.Job{{JobId: 1}, {JobId: 2}}}}
	apps := &boundedAppListClient{errorByJob: map[int64]error{1: errors.New("unavailable")}}
	snapshot := &fakeSnapshotClient{resp: &pb.GetApplicationSnapshotResponse{Code: errs.OK, ApplicationId: 1001, JobHrId: 77}}
	exec := &Executor{Jobs: jobs, Applications: apps, Snapshots: snapshot}
	owned, err := exec.applicationOwnedByHR(context.Background(), 77, 1001)
	if err != nil || !owned {
		t.Fatalf("partial ownership fallback = owned:%v err:%v", owned, err)
	}

	exec.Snapshots = nil
	owned, err = exec.applicationOwnedByHR(context.Background(), 77, 1001)
	if err == nil || owned {
		t.Fatalf("partial ownership without fallback = owned:%v err:%v, want indeterminate error", owned, err)
	}
}

func TestTruncatedAggregationDoesNotTurnUnknownOwnershipIntoFalseDenial(t *testing.T) {
	jobs101 := make([]*pb.Job, 0, 101)
	for id := int64(1); id <= 101; id++ {
		jobs101 = append(jobs101, &pb.Job{JobId: id})
	}
	tests := []struct {
		name string
		jobs []*pb.Job
		apps *boundedAppListClient
	}{
		{name: "job limit", jobs: jobs101, apps: &boundedAppListClient{}},
		{name: "page limit", jobs: []*pb.Job{{JobId: 1}}, apps: &boundedAppListClient{rowsPerPage: 100, totalPages: 11}},
		{name: "row limit", jobs: []*pb.Job{{JobId: 1}, {JobId: 2}, {JobId: 3}, {JobId: 4}, {JobId: 5}, {JobId: 6}}, apps: &boundedAppListClient{rowsPerPage: 100, totalPages: 10}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			snapshot := &fakeSnapshotClient{resp: &pb.GetApplicationSnapshotResponse{Code: errs.OK, ApplicationId: 999999999, JobHrId: 77}}
			exec := &Executor{
				Jobs:         &fakeJobClient{listResp: &pb.ListJobsResponse{Code: errs.OK, Total: int64(len(tt.jobs)), List: tt.jobs}},
				Applications: tt.apps,
				Snapshots:    snapshot,
			}
			aggregation, err := exec.listAllApplications(context.Background(), 77, 0, false)
			if err != nil || !aggregation.Truncated {
				t.Fatalf("aggregation truncated=%v err=%v", aggregation.Truncated, err)
			}
			owned, err := exec.applicationOwnedByHR(context.Background(), 77, 999999999)
			if err != nil || !owned {
				t.Fatalf("truncated ownership fallback = owned:%v err:%v", owned, err)
			}
			exec.Snapshots = nil
			owned, err = exec.applicationOwnedByHR(context.Background(), 77, 999999999)
			if err == nil || owned {
				t.Fatalf("truncated ownership without snapshot = owned:%v err:%v", owned, err)
			}
		})
	}
}
