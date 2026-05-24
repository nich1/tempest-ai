package handlers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Health is a deep health check that exercises every dependency.
//
// @Summary      Health check
// @Description  Pings Postgres, Redis (Asynq), and MinIO; returns per-dependency status and queue depth.
// @Tags         health
// @Produce      json
// @Success      200 {object} map[string]interface{}
// @Failure      503 {object} map[string]interface{}
// @Router       /health [get]
func (d *Deps) Health(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	out := gin.H{}
	overall := true

	if err := d.DB.Ping(ctx); err != nil {
		out["postgres"] = gin.H{"ok": false, "error": err.Error()}
		overall = false
	} else {
		out["postgres"] = gin.H{"ok": true}
	}

	if err := d.Storage.HealthCheck(ctx); err != nil {
		out["minio"] = gin.H{"ok": false, "error": err.Error()}
		overall = false
	} else {
		out["minio"] = gin.H{"ok": true}
	}

	queues, err := d.Inspector.Queues()
	if err != nil {
		out["redis"] = gin.H{"ok": false, "error": err.Error()}
		overall = false
	} else {
		depths := gin.H{}
		for _, q := range queues {
			info, err := d.Inspector.GetQueueInfo(q)
			if err != nil {
				depths[q] = gin.H{"error": err.Error()}
				continue
			}
			depths[q] = gin.H{
				"pending":   info.Pending,
				"active":    info.Active,
				"retry":     info.Retry,
				"scheduled": info.Scheduled,
			}
		}
		out["redis"] = gin.H{"ok": true, "queues": depths}
	}

	status := http.StatusOK
	if !overall {
		status = http.StatusServiceUnavailable
	}
	out["status"] = "ok"
	if !overall {
		out["status"] = "degraded"
	}
	c.JSON(status, out)
}
