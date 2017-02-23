/*
Copyright 2017 caicloud authors. All rights reserved.
*/

package api

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/vmware/harbor/src/common/api"
	"github.com/vmware/harbor/src/common/dao"
	"github.com/vmware/harbor/src/common/models"
	"github.com/vmware/harbor/src/common/utils/log"
)

// JobAPIV1 handles request to /api/v1/jobs /api/v1/jobs/:jid,

type JobAPIV1 struct {
	api.BaseAPI
}

type createJobReq struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// Prepare validates the URL and the params
func (j *JobAPIV1) Prepare() {
	idStr := j.Ctx.Input.Param(":jid")
	if len(idStr) > 0 {
		var err error
		jobId, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Errorf("Error parsing job id: %s, error: %v", idStr, err)
			j.CustomAbort(http.StatusBadRequest, "invalid job id")
		}
		exist := dao.JobIdExists(jobId)

		if !exist {
			j.CustomAbort(http.StatusNotFound, fmt.Sprintf("job does not exist, id: %v", jobId))
		}
	}
}

// GetJob
func (j *JobAPIV1) GetJob() {
	idStr := j.Ctx.Input.Param(":jid")
	jobId, err := strconv.ParseInt(idStr, 10, 64)

	job, err := dao.GetJobById(jobId)
	if err != nil {
		log.Errorf("GetJobById error: %v", err)
		j.CustomAbort(http.StatusInternalServerError, fmt.Sprintf("GetJobById error: %v", err))
	}

	j.Data["json"] = job
	j.ServeJSON()
}

// Post
func (j *JobAPIV1) Post() {
	var req createJobReq
	j.DecodeJSONReq(&req)

	job := models.Job{
		Type:    req.Type,
		Message: req.Message,
	}

	jobId, err := dao.CreateJob(job)
	if err != nil {
		log.Errorf("CreateJob error: %v", err)
		j.CustomAbort(http.StatusInternalServerError, fmt.Sprintf("CreateJob error: %v", err))
	}

	// return job id
	j.RenderError(http.StatusCreated, strconv.FormatInt(jobId, 10))
}
