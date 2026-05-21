package hr

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"google.golang.org/grpc"

	"web-gin-service/recruitment/pb"
	"web-gin-service/rpc"
)

type cursorJobClient struct {
	got *pb.ListHRJobsRequest
}

func (f *cursorJobClient) CreateJob(context.Context, *pb.CreateJobRequest, ...grpc.CallOption) (*pb.CreateJobResponse, error) {
	return nil, nil
}

func (f *cursorJobClient) UpdateJob(context.Context, *pb.UpdateJobRequest, ...grpc.CallOption) (*pb.CommonResponse, error) {
	return nil, nil
}

func (f *cursorJobClient) OfflineJob(context.Context, *pb.OfflineJobRequest, ...grpc.CallOption) (*pb.CommonResponse, error) {
	return nil, nil
}

func (f *cursorJobClient) OnlineJob(context.Context, *pb.OfflineJobRequest, ...grpc.CallOption) (*pb.CommonResponse, error) {
	return nil, nil
}

func (f *cursorJobClient) ListHRJobs(_ context.Context, req *pb.ListHRJobsRequest, _ ...grpc.CallOption) (*pb.ListJobsResponse, error) {
	copied := *req
	f.got = &copied
	return &pb.ListJobsResponse{Code: 0, Msg: "success"}, nil
}

func (f *cursorJobClient) ListPublicJobs(context.Context, *pb.ListPublicJobsRequest, ...grpc.CallOption) (*pb.ListJobsResponse, error) {
	return nil, nil
}

func (f *cursorJobClient) GetJobDetail(context.Context, *pb.GetJobDetailRequest, ...grpc.CallOption) (*pb.GetJobDetailResponse, error) {
	return nil, nil
}

func (f *cursorJobClient) ListJobOptions(context.Context, *pb.ListJobOptionsRequest, ...grpc.CallOption) (*pb.ListJobOptionsResponse, error) {
	return nil, nil
}

func (f *cursorJobClient) ListDepartmentLocations(context.Context, *pb.ListDepartmentLocationsRequest, ...grpc.CallOption) (*pb.ListDepartmentLocationsResponse, error) {
	return nil, nil
}

func TestListPassesCursor(t *testing.T) {
	gin.SetMode(gin.TestMode)
	fake := &cursorJobClient{}
	handler := NewJobHandler(&rpc.Clients{Job: fake})
	router := gin.New()
	router.GET("/hr/jobs", func(c *gin.Context) {
		c.Set("user_id", int64(9))
		handler.List(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/hr/jobs?page=2&page_size=15&cursor=job-cursor", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if fake.got == nil {
		t.Fatalf("expected ListHRJobs to be called")
	}
	if fake.got.HrId != 9 || fake.got.Cursor != "job-cursor" || fake.got.Page != 0 || fake.got.PageSize != 15 {
		t.Fatalf("unexpected request: %+v", fake.got)
	}
}
