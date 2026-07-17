package handler

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"

	"smart-recruit-gateway/rpc"
	"smart-recruit-proto/recruitment/pb"
)

type PublicHandler struct {
	clients *rpc.Clients
}

func NewPublicHandler(clients *rpc.Clients) *PublicHandler {
	return &PublicHandler{clients: clients}
}

func (h *PublicHandler) ListJobs(c *gin.Context) {
	page, pageSize := pagination(c)
	cursor, hasCursor := c.GetQuery("cursor")
	if hasCursor {
		if err := validateCursor(cursor); err != nil {
			BadRequest(c, "cursor 参数格式错误")
			return
		}
		page = 0
	}
	resp, err := h.clients.Job.ListPublicJobs(c.Request.Context(), &pb.ListPublicJobsRequest{
		Page:          page,
		PageSize:      pageSize,
		Keyword:       c.Query("keyword"),
		Cursor:        cursor,
		DepartmentIds: parseIDList(c, "department_ids"),
		LocationIds:   parseIDList(c, "location_ids"),
	})
	if err != nil {
		Internal(c, err)
		return
	}
	ProtoResponse(c, resp)
}

// JobOptions returns the public department tree and location catalog for job-board facets.
func (h *PublicHandler) JobOptions(c *gin.Context) {
	resp, err := h.clients.Job.ListJobOptions(c.Request.Context(), &pb.ListJobOptionsRequest{})
	if err != nil {
		Internal(c, err)
		return
	}
	// Public surface: only active taxonomy nodes.
	if resp != nil {
		resp.DepartmentTree = filterActiveDepartments(resp.DepartmentTree)
		resp.Locations = filterActiveLocations(resp.Locations)
	}
	ProtoResponse(c, resp)
}

func (h *PublicHandler) JobDetail(c *gin.Context) {
	jobID, err := strconv.ParseInt(c.Param("job_id"), 10, 64)
	if err != nil {
		BadRequest(c, "岗位 ID 不合法")
		return
	}
	resp, err := h.clients.Job.GetJobDetail(c.Request.Context(), &pb.GetJobDetailRequest{JobId: jobID})
	if err != nil {
		Internal(c, err)
		return
	}
	if resp.Job == nil {
		BadRequest(c, resp.Msg)
		return
	}
	From(c, resp.Code, resp.Msg, resp.Job)
}

func pagination(c *gin.Context) (int32, int32) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}
	return int32(page), int32(pageSize)
}

// parseIDList accepts repeated query keys and comma-separated values.
// Examples: ?location_ids=1&location_ids=2  or  ?location_ids=1,2,3
func parseIDList(c *gin.Context, key string) []int64 {
	raw := c.QueryArray(key)
	if len(raw) == 0 {
		if single := c.Query(key); single != "" {
			raw = []string{single}
		}
	}
	if len(raw) == 0 {
		return nil
	}
	out := make([]int64, 0, len(raw))
	seen := make(map[int64]struct{})
	for _, part := range raw {
		for _, token := range splitCSV(part) {
			id, err := strconv.ParseInt(token, 10, 64)
			if err != nil || id <= 0 {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			out = append(out, id)
		}
	}
	return out
}

func splitCSV(s string) []string {
	parts := make([]string, 0, 4)
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			token := trimSpaceASCII(s[start:i])
			if token != "" {
				parts = append(parts, token)
			}
			start = i + 1
		}
	}
	return parts
}

func trimSpaceASCII(s string) string {
	i, j := 0, len(s)
	for i < j && (s[i] == ' ' || s[i] == '\t') {
		i++
	}
	for j > i && (s[j-1] == ' ' || s[j-1] == '\t') {
		j--
	}
	return s[i:j]
}

func filterActiveDepartments(nodes []*pb.DepartmentNode) []*pb.DepartmentNode {
	if len(nodes) == 0 {
		return nodes
	}
	out := make([]*pb.DepartmentNode, 0, len(nodes))
	for _, n := range nodes {
		if n == nil {
			continue
		}
		children := filterActiveDepartments(n.Children)
		// Keep active nodes; also keep inactive parents that still have active children
		// so hierarchy remains navigable. Prefer is_active == 1 for leaf inclusion.
		if n.IsActive == 1 {
			clone := *n
			clone.Children = children
			out = append(out, &clone)
			continue
		}
		if len(children) > 0 {
			clone := *n
			clone.Children = children
			out = append(out, &clone)
		}
	}
	return out
}

func filterActiveLocations(locs []*pb.LocationOption) []*pb.LocationOption {
	if len(locs) == 0 {
		return locs
	}
	out := make([]*pb.LocationOption, 0, len(locs))
	for _, loc := range locs {
		if loc != nil && loc.IsActive == 1 {
			out = append(out, loc)
		}
	}
	return out
}

// validateCursor validates the cursor format as a basic sanity check.
// Valid cursors are non-empty strings up to 256 characters containing only
// printable ASCII characters. This is a defense-in-depth check; the actual
// cursor parsing is done server-side.
func validateCursor(cursor string) error {
	if len(cursor) == 0 || len(cursor) > 256 {
		return errors.New("cursor length out of range")
	}
	for _, b := range cursor {
		if b < 0x20 || b > 0x7e {
			return errors.New("cursor contains invalid characters")
		}
	}
	return nil
}
