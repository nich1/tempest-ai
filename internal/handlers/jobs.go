package handlers

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/nich1/tempest-ai/internal/jobs"
	"github.com/nich1/tempest-ai/internal/llm"
	"github.com/nich1/tempest-ai/internal/logging"
	"github.com/nich1/tempest-ai/internal/middleware"
	"github.com/nich1/tempest-ai/internal/models"
	"github.com/nich1/tempest-ai/internal/schema"
	"github.com/nich1/tempest-ai/internal/storage"
)

// CreateJob validates schemas + inputs, persists the job, then enqueues
// it for the consumers.
//
// @Summary      Submit job
// @Description  Submit an LLM extraction job. Schemas, inputs, and prompts are validated up-front.
// @Tags         jobs
// @Accept       json
// @Produce      json
// @Param        body body models.CreateJobRequest true "Job request"
// @Success      202 {object} models.JobDTO
// @Failure      400 {object} models.ErrorResponse
// @Failure      401 {object} models.ErrorResponse
// @Router       /jobs [post]
func (d *Deps) CreateJob(c *gin.Context) {
	sess, ok := middleware.SessionFrom(c)
	if !ok {
		errResp(c, http.StatusUnauthorized, "authentication required")
		return
	}

	var req models.CreateJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errResp(c, http.StatusBadRequest, "invalid request body")
		return
	}

	inputSchema, err := schema.Parse(req.InputSchema)
	if err != nil {
		errResp(c, http.StatusBadRequest, "input_schema: "+err.Error())
		return
	}
	if err := inputSchema.Validate(schema.KindInput); err != nil {
		errResp(c, http.StatusBadRequest, "input_schema: "+err.Error())
		return
	}
	outputSchema, err := schema.Parse(req.OutputSchema)
	if err != nil {
		errResp(c, http.StatusBadRequest, "output_schema: "+err.Error())
		return
	}
	if err := outputSchema.Validate(schema.KindOutput); err != nil {
		errResp(c, http.StatusBadRequest, "output_schema: "+err.Error())
		return
	}
	if err := inputSchema.ValidateInputs(req.Inputs); err != nil {
		errResp(c, http.StatusBadRequest, "inputs: "+err.Error())
		return
	}

	provider := req.Provider
	if provider == "" {
		provider = d.Factory.DefaultSpec()
	}
	providerName, _, err := llm.ParseSpec(provider)
	if err != nil {
		errResp(c, http.StatusBadRequest, "provider: "+err.Error())
		return
	}
	if !d.Factory.IsAvailable(providerName) {
		errResp(c, http.StatusBadRequest, "provider not available")
		return
	}

	var (
		fileSize    int64
		fileType    string
	)
	if req.FileBlobKey != "" {
		info, err := d.Storage.Stat(c.Request.Context(), req.FileBlobKey)
		if err != nil {
			if errors.Is(err, storage.ErrObjectNotFound) {
				errResp(c, http.StatusBadRequest, "file_blob_key references missing object")
				return
			}
			internalErr(c, "jobs.stat_file_failed", err)
			return
		}
		if info.Size > d.Cfg.API.MaxFileSizeBytes {
			errResp(c, http.StatusBadRequest, "file exceeds MAX_FILE_SIZE_BYTES")
			return
		}
		fileSize = info.Size
		fileType = info.ContentType
	}

	row, err := d.Jobs.Create(c.Request.Context(), jobs.CreateParams{
		UserID:              sess.UserID,
		InputSchema:         req.InputSchema,
		OutputSchema:        req.OutputSchema,
		Inputs:              req.Inputs,
		Prompt:              req.Prompt,
		SystemPrompt:        req.SystemPrompt,
		FileBlobKey:         req.FileBlobKey,
		FileBlobSize:        fileSize,
		FileBlobContentType: fileType,
		Provider:            provider,
	})
	if err != nil {
		internalErr(c, "jobs.create_failed", err)
		return
	}

	jobID := bytesToUUID(row.ID)
	taskID, err := d.Queue.EnqueueProcessJob(c.Request.Context(), jobID, middleware.RequestIDFrom(c))
	if err != nil {
		internalErr(c, "jobs.enqueue_failed", err)
		return
	}

	logging.FromContext(c.Request.Context()).Info("job.created",
		slog.String("job_id", jobID.String()),
		slog.String("task_id", taskID),
		slog.String("provider", provider),
	)

	c.JSON(http.StatusAccepted, models.JobFromRow(row))
}

// GetJob returns a single job by ID, scoped to the calling user.
//
// @Summary      Get job
// @Tags         jobs
// @Produce      json
// @Param        id path string true "Job ID"
// @Success      200 {object} models.JobDTO
// @Failure      401 {object} models.ErrorResponse
// @Failure      404 {object} models.ErrorResponse
// @Router       /jobs/{id} [get]
func (d *Deps) GetJob(c *gin.Context) {
	sess, ok := middleware.SessionFrom(c)
	if !ok {
		errResp(c, http.StatusUnauthorized, "authentication required")
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		errResp(c, http.StatusBadRequest, "invalid job id")
		return
	}
	row, err := d.Jobs.GetByIDForUser(c.Request.Context(), id, sess.UserID)
	if err != nil {
		if errors.Is(err, jobs.ErrNotFound) {
			errResp(c, http.StatusNotFound, "job not found")
			return
		}
		internalErr(c, "jobs.get_failed", err)
		return
	}
	c.JSON(http.StatusOK, models.JobFromRow(row))
}

// ListJobs returns the current user's jobs (most recent first).
//
// @Summary      List jobs
// @Tags         jobs
// @Produce      json
// @Param        limit query int false "Max items (1-100)"
// @Param        offset query int false "Offset"
// @Success      200 {object} models.JobListResponse
// @Failure      401 {object} models.ErrorResponse
// @Router       /jobs [get]
func (d *Deps) ListJobs(c *gin.Context) {
	sess, ok := middleware.SessionFrom(c)
	if !ok {
		errResp(c, http.StatusUnauthorized, "authentication required")
		return
	}
	limit := parseIntDefault(c.Query("limit"), 20)
	offset := parseIntDefault(c.Query("offset"), 0)
	if limit < 1 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	rows, err := d.Jobs.ListForUser(c.Request.Context(), sess.UserID, limit, offset)
	if err != nil {
		internalErr(c, "jobs.list_failed", err)
		return
	}
	count, err := d.Jobs.CountForUser(c.Request.Context(), sess.UserID)
	if err != nil {
		internalErr(c, "jobs.count_failed", err)
		return
	}
	out := make([]models.JobDTO, 0, len(rows))
	for _, r := range rows {
		out = append(out, models.JobFromRow(r))
	}
	c.JSON(http.StatusOK, models.JobListResponse{
		Jobs:       out,
		TotalCount: count,
		Limit:      limit,
		Offset:     offset,
	})
}

// FileUploadURL mints a presigned PUT URL the client can use to upload
// the input file directly to MinIO.
//
// @Summary      Mint upload URL
// @Tags         jobs
// @Accept       json
// @Produce      json
// @Param        body body models.FileUploadURLRequest true "File metadata"
// @Success      200 {object} models.FileUploadURLResponse
// @Failure      400 {object} models.ErrorResponse
// @Failure      401 {object} models.ErrorResponse
// @Router       /jobs/file-upload-url [post]
func (d *Deps) FileUploadURL(c *gin.Context) {
	if _, ok := middleware.SessionFrom(c); !ok {
		errResp(c, http.StatusUnauthorized, "authentication required")
		return
	}
	var req models.FileUploadURLRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		errResp(c, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.SizeBytes > d.Cfg.API.MaxFileSizeBytes {
		errResp(c, http.StatusBadRequest, "size exceeds MAX_FILE_SIZE_BYTES")
		return
	}

	key := storage.NewBlobKey()
	url, err := d.Storage.PresignPutURL(c.Request.Context(), key, req.ContentType, req.SizeBytes, d.Cfg.API.PresignedPutTTL)
	if err != nil {
		internalErr(c, "jobs.presign_failed", err)
		return
	}
	c.JSON(http.StatusOK, models.FileUploadURLResponse{
		UploadURL: url,
		BlobKey:   key,
		MaxSize:   d.Cfg.API.MaxFileSizeBytes,
		ExpiresIn: int(d.Cfg.API.PresignedPutTTL.Seconds()),
	})
}

func parseIntDefault(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}
