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
	"fmt"
	"net/http"
	"regexp"

	"github.com/vmware/harbor/dao"
	"github.com/vmware/harbor/models"
	"github.com/vmware/harbor/utils/log"

	"strconv"
)

// LabelAPI handles request to /api/labels/{} /api/labels/{}/logs
type LabelAPI struct {
	BaseAPI
	userID    int
	labelID   int64
	projectID int64
}

type labelReq struct {
	ProjectID   int64  `json:"project_id"`
	ProjectName string `json:"project_name"`
	LabelName   string `json:"label_name"`
	LabelRemark string `json:"label_remark"`
}

const labelNameMaxLen int = 50
const labelNameMinLen int = 2
const labelRemarkMaxLen int = 100
const labelRemarkMinLen int = 2
const dupLabelPattern = `Duplicate entry '\w+' for key 'name'`

// Prepare validates the URL and the user
func (l *LabelAPI) Prepare() {
	idStr := l.Ctx.Input.Param(":id")
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

// Post ...
func (l *LabelAPI) Post() {
	l.userID = l.ValidateUser()

	var req labelReq
	l.DecodeJSONReq(&req)
	log.Debugf("POST api/labels, req: %v", req)

	err := validateLabelReq(req)
	if err != nil {
		log.Errorf("Invalid label request, error: %v", err)
		l.RenderError(http.StatusBadRequest, fmt.Sprintf("invalid request: %v", err))
		return
	}

	labelName := req.LabelName
	label_exist, err := dao.LabelExists(labelName)
	if err != nil {
		log.Errorf("Error happened checking label existence in db, error: %v, label name: %s", err, labelName)
		return
	}
	if label_exist {
		l.RenderError(http.StatusConflict, "label exist")
		return
	}

	// check whether project_id is exists
	projectID := req.ProjectID
	project_id_exist, err := dao.ProjectExists(projectID)
	if err != nil {
		log.Errorf("Error happened checking project existence in db, error: %v, project id: %s", err, projectID)
		return
	}
	if !project_id_exist {
		l.RenderError(http.StatusNotFound, "Error, project_id does not exist")
		return
	}

	// check whether project_name is exists
	projectName := req.ProjectName
	project_name_exist, err := dao.ProjectExists(projectName)
	if err != nil {
		log.Errorf("Error happened checking project existence in db, error: %v, project name: %s", err, projectName)
		return
	}
	if !project_name_exist {
		l.RenderError(http.StatusNotFound, "Error, project_name does not exist")
		return
	}

	label := models.Label{
		OwnerID:     l.userID,
		ProjectID:   req.ProjectID,
		ProjectName: req.ProjectName,
		Name:        req.LabelName,
		Remark:      req.LabelRemark}

	labelID, err := dao.NewLabel(label)
	if err != nil {
		log.Errorf("Failed to new label, error: %v", err)
		dup, _ := regexp.MatchString(dupLabelPattern, err.Error())
		if dup {
			l.RenderError(http.StatusConflict, "")
		} else {
			l.RenderError(http.StatusInternalServerError, "Failed to new label")
		}
		return
	}

	log.Debugf("Add new label, id: %v", labelID)
	l.CustomAbort(http.StatusCreated, strconv.Itoa(int(labelID)))
}

// Delete  ...
func (l *LabelAPI) Delete() {
	idStr := l.Ctx.Input.Param(":id")
	id, _ := strconv.Atoi(idStr)

	log.Debugf("DELETE api/labels, id: %v", id)

	if err := dao.DeleteLabel(int64(id)); err != nil {
		log.Errorf("Failed to delete label, error: %v", err)
		l.RenderError(http.StatusInternalServerError, "Failed to delete label")
	}
}

// List ...
func (l *LabelAPI) List() {
	log.Infof("/api/labels/list")
	idStr := l.Ctx.Input.Param(":id")

	if !(len(idStr) > 0) {
		l.CustomAbort(http.StatusBadRequest, "invalid project id")
	}

	var err error
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

	labels, err := dao.GetLabelsByProjectID(l.projectID)
	if err != nil {
		log.Errorf("failed to get labels from project %d: %v", l.projectID, err)
		l.CustomAbort(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	}

	l.Data["json"] = labels
	l.ServeJSON()
}

func validateLabelReq(req labelReq) error {
	log.Debugf("validateLabelReq, LabelName: %s, LabelRemark: %s", req.LabelName, req.LabelRemark)

	if isIllegalLength(req.LabelName, labelNameMinLen, labelNameMaxLen) {
		return fmt.Errorf("Label name is illegal in length. (greater than %v or less than %v)", labelNameMinLen, labelNameMaxLen)
	}

	if isIllegalLength(req.LabelRemark, labelRemarkMinLen, labelRemarkMaxLen) {
		return fmt.Errorf("Label remark is illegal in length. (greater than %v or less than %v)", labelRemarkMinLen, labelRemarkMaxLen)
	}

	return nil
}
