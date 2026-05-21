package candidate

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

type cursorApplicationClient struct {
	got *pb.ListMyApplicationsRequest
}

func (f *cursorApplicationClient) ApplyJob(context.Context, *pb.ApplyJobRequest, ...grpc.CallOption) (*pb.CommonResponse, error) {
	return nil, nil
}

func (f *cursorApplicationClient) ListMyApplications(_ context.Context, req *pb.ListMyApplicationsRequest, _ ...grpc.CallOption) (*pb.ListMyApplicationsResponse, error) {
	copied := *req
	f.got = &copied
	return &pb.ListMyApplicationsResponse{Code: 0, Msg: "success"}, nil
}

func (f *cursorApplicationClient) ListJobApplications(context.Context, *pb.ListJobApplicationsRequest, ...grpc.CallOption) (*pb.ListJobApplicationsResponse, error) {
	return nil, nil
}

func (f *cursorApplicationClient) UpdateApplicationStatus(context.Context, *pb.UpdateApplicationStatusRequest, ...grpc.CallOption) (*pb.CommonResponse, error) {
	return nil, nil
}

func TestMinePassesCursor(t *testing.T) {
	gin.SetMode(gin.TestMode)
	fake := &cursorApplicationClient{}
	handler := NewApplyHandler(&rpc.Clients{Application: fake})
	router := gin.New()
	router.GET("/candidate/applications", func(c *gin.Context) {
		c.Set("user_id", int64(8))
		handler.Mine(c)
	})

	req := httptest.NewRequest(http.MethodGet, "/candidate/applications?page=2&page_size=12&cursor=app-cursor", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if fake.got == nil {
		t.Fatalf("expected ListMyApplications to be called")
	}
	if fake.got.UserId != 8 || fake.got.Cursor != "app-cursor" || fake.got.Page != 0 || fake.got.PageSize != 12 {
		t.Fatalf("unexpected request: %+v", fake.got)
	}
}
