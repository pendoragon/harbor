/*
   Copyright (c) 2016 VMware, Inc. All Rights Reserved.
   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package api

import (
	"bytes"
	// "encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/vmware/harbor/src/common/api"
	"github.com/vmware/harbor/src/common/dao"
	"github.com/vmware/harbor/src/common/models"
	"github.com/vmware/harbor/src/common/utils/log"

	"strconv"
)

// LabelAPIV1 handles request to /api/labels/{} /api/labels/{}/logs
type LabelAPIV1 struct {
	api.BaseAPI
	userID    int
	labelID   int64
	projectID int64
}

type createReqV1 struct {
	Name   string `json:"name"`
	Remark string `json:"remark"`
}

type updateReqV1 struct {
	Name   string   `json:"name"`
	Remark string   `json:"remark"`
	Repos  []string `json:"repos"`
}

// Prepare validates the URL and the user
func (l *LabelAPIV1) Prepare() {
	idStr := l.Ctx.Input.Param(":lid")
	if len(idStr) > 0 {
		var err error
		l.labelID, err = strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Errorf("Error parsing label id: %s, error: %v", idStr, err)
			l.CustomAbort(http.StatusBadRequest, "invalid label id")
		}
		exist, err := dao.LabelExists(l.labelID)
		if err != nil {
			log.Errorf("Error occurred in LabelExists, error: %v", err)
			l.CustomAbort(http.StatusInternalServerError, "Internal error.")
		}
		if !exist {
			l.CustomAbort(http.StatusNotFound, fmt.Sprintf("label does not exist, id: %v", l.labelID))
		}
	}
}

// Get ...
func (l *LabelAPIV1) Get() {
	idStr := l.Ctx.Input.Param(":lid")
	if len(idStr) <= 0 {
		l.CustomAbort(http.StatusBadRequest, "lid not found in URL")
	}

	labelID, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		log.Errorf("Error parsing label id: %s, error: %v", idStr, err)
		l.CustomAbort(http.StatusBadRequest, "invalid label id")
	}

	label, err := dao.GetLabelByID(labelID)

	labelV1 := models.LabelV1{
		LabelID:      label.LabelID,
		OwnerID:      label.OwnerID,
		ProjectID:    label.ProjectID,
		Name:         label.Name,
		Remark:       label.Remark,
		CreationTime: label.CreationTime,
		UpdateTime:   label.UpdateTime,
	}

	l.Data["json"] = labelV1
	l.ServeJSON()
}

// Post ...
func (l *LabelAPIV1) Post() {
	l.userID = l.ValidateUser()

	var req createReqV1
	var err error
	l.DecodeJSONReq(&req)
	log.Debugf("POST api/v1/labels, req: %v", req)

	idStr := l.Ctx.Input.Param(":pid")
	l.projectID = 0

	if len(idStr) > 0 {
		l.projectID, err = strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Errorf("Error parsing project id: %s, error: %v", idStr, err)
			l.CustomAbort(http.StatusBadRequest, "invalid project id")
		}

		exist, err := dao.ProjectExists(l.projectID)
		if err != nil {
			log.Errorf("Error occurred in ProjectExists, error: %v", err)
			l.CustomAbort(http.StatusInternalServerError, "Internal error.")
		}

		if !exist {
			l.CustomAbort(http.StatusNotFound, fmt.Sprintf("project does not exist, id: %v", l.projectID))
		}
	}

	label := models.Label{
		OwnerID:   l.userID,
		ProjectID: l.projectID,
		Name:      req.Name,
		Remark:    req.Remark,
	}

	labelID, err := dao.NewLabel(label)
	if err != nil {
		log.Errorf("Failed to new label, error: %v", err)
		dup, _ := regexp.MatchString(dupLabelPattern, err.Error())
		if dup {
			l.RenderError(http.StatusConflict, "project_id and name conflict")
		} else {
			l.RenderError(http.StatusInternalServerError, "Failed to new label")
		}
		return
	}

	log.Debugf("Add new label, id: %v", labelID)
	l.CustomAbort(http.StatusCreated, strconv.Itoa(int(labelID)))
}

// Put ...
func (l *LabelAPIV1) Put() {
	idStr := l.Ctx.Input.Param(":lid")
	labelId, _ := strconv.Atoi(idStr)

	l.userID = l.ValidateUser()

	var req updateReqV1
	var err error
	l.DecodeJSONReq(&req)
	log.Debugf("PUT api/v1/projects/:pid/labels/:lid, req: %v", req)

	var repos_buffer bytes.Buffer

	for i, repo := range req.Repos {
		log.Debugf("i: %v", i)
		repos_buffer.WriteString(repo)
		if i+1 < len(req.Repos) {
			repos_buffer.WriteString(",")
		}
	}

	if err != nil {
		log.Errorf("json.Marshal error: %v", err)
		l.CustomAbort(http.StatusInternalServerError, "json.Marshal error")
	}

	log.Debugf("PUT api/v1/projects/:pid/labels/:lid, repos_str: %v", repos_buffer.String())

	label := models.Label{
		LabelID:  int64(labelId),
		Name:     req.Name,
		Remark:   req.Remark,
		ReposStr: repos_buffer.String(),
	}

	err = dao.UpdateLabel(label)

	if err != nil {
		log.Errorf("UpdateLabel error: %v", err)
		l.CustomAbort(http.StatusInternalServerError, "UpdateLabel error")
	}

	go func() {
		for i, repo := range req.Repos {
			log.Debugf("SyncRepositoryLabelNamesV1, repos[%d]: %v", i, repo)
			dao.SyncRepositoryLabelNamesV1(repo)
		}
	}()
}

// Delete  ...
func (l *LabelAPIV1) Delete() {
	idStr := l.Ctx.Input.Param(":lid")
	labelId, _ := strconv.Atoi(idStr)

	log.Debugf("DELETE api/v1/projects/:pid/labels/:lid, labelId: %v", labelId)

	if err := dao.DeleteLabel(int64(labelId)); err != nil {
		log.Errorf("Failed to delete label, error: %v", err)
		l.RenderError(http.StatusInternalServerError, "Failed to delete label")
	}

	l.RenderNoContent()
}

// List ...
func (l *LabelAPIV1) List() {
	log.Infof("/api/v1/projects/:pid/labels")
	idStr := l.Ctx.Input.Param(":pid")
	var err error
	l.projectID = 0

	if len(idStr) > 0 {
		l.projectID, err = strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Errorf("Error parsing project id: %s, error: %v", idStr, err)
			l.CustomAbort(http.StatusBadRequest, "invalid project id")
		}

		exist, err := dao.ProjectExists(l.projectID)
		if err != nil {
			log.Errorf("Error occurred in ProjectExists, error: %v", err)
			l.CustomAbort(http.StatusInternalServerError, "Internal error.")
		}

		if !exist {
			l.CustomAbort(http.StatusNotFound, fmt.Sprintf("project does not exist, id: %v", l.projectID))
		}
	}

	labelName := l.GetString("label_name")
	labels, err := dao.GetLabelsByProjectID(l.projectID, labelName)
	if err != nil {
		log.Errorf("failed to get labels from project %d: %v", l.projectID, err)
		l.CustomAbort(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	}

	var lebelsV1 []models.LabelV1
	for i := 0; i < len(labels); i++ {
		labelV1 := models.LabelV1{
			LabelID:      labels[i].LabelID,
			OwnerID:      labels[i].OwnerID,
			ProjectID:    labels[i].ProjectID,
			Name:         labels[i].Name,
			Remark:       labels[i].Remark,
			CreationTime: labels[i].CreationTime,
			UpdateTime:   labels[i].UpdateTime,
		}
		log.Debugf("labels[%d].ReposStr: %v", i, labels[i].ReposStr)
		if len(labels[i].ReposStr) > 0 {
			labelV1.Repos = strings.Split(labels[i].ReposStr, ",")
			log.Debugf("labels[%d].Repos: %v", i, labels[i].Repos)
		}

		lebelsV1 = append(lebelsV1, labelV1)
	}

	l.Data["json"] = models.NewListResponse(len(lebelsV1), lebelsV1)
	l.ServeJSON()
}

// List repos by label names
// func (l *LabelAPIV1) ListReposByNames() {
// 	var req labelReqV1
// 	l.DecodeJSONReq(&req)
// 	log.Debugf("POST api/repos_by_labelnames, req: %v", req)

// 	repo_names, err := dao.GetReposByNames(req.Names)
// 	if err != nil {
// 		log.Errorf("dao.GetReposByNames error: %v", err)
// 		l.RenderError(http.StatusBadRequest, fmt.Sprintf("invalid GetReposByNames request: %v", err))
// 		return
// 	}

// 	log.Debugf("POST api/repos_by_labelnames, result: %v", repo_names)
// 	l.Data["json"] = repo_names
// 	l.ServeJSON()
// }
