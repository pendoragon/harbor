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

	"github.com/vmware/harbor/src/common/api"
	"github.com/vmware/harbor/src/common/dao"
	"github.com/vmware/harbor/src/common/models"
	"github.com/vmware/harbor/src/common/utils/log"

	"strconv"
)

// LabelHookAPI handles request to /api/labelhooks/{} /api/labelhooks/{}/logs
type LabelHookAPI struct {
	api.BaseAPI
	userID      int
	labelHookID int64
	labelID     int64
}

type labelHookReq struct {
	LabelID  int64  `json:"label_id"`
	RepoName string `json:"repo_name"`
}

const dupLabelHookPattern = `Duplicate entry .* for key 'label_id'`

// Prepare validates the URL and the user
func (lh *LabelHookAPI) Prepare() {
	idStr := lh.Ctx.Input.Param(":id")
	if len(idStr) > 0 {
		var err error
		lh.labelHookID, err = strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Errorf("Error parsing labelhook id: %s, error: %v", idStr, err)
			lh.CustomAbort(http.StatusBadRequest, "invalid labelhook id")
		}

		exist, err := dao.LabelHookExists(lh.labelHookID)
		if err != nil {
			log.Errorf("Error occurred in LabelExists, error: %v", err)
			lh.CustomAbort(http.StatusInternalServerError, "Internal error.")
		}

		if !exist {
			lh.CustomAbort(http.StatusNotFound, fmt.Sprintf("labelhook does not exist, id: %v", lh.labelHookID))
		}
	}
}

// POST, new labelhook.
func (lh *LabelHookAPI) Post() {
	lh.userID = lh.ValidateUser()

	var req labelHookReq
	lh.DecodeJSONReq(&req)
	log.Debugf("POST api/labelhooks/hook, req: %v", req)

	// check whether label_id is exists
	labelID := req.LabelID
	label_id_exist, err := dao.LabelExists(labelID)
	if err != nil {
		log.Errorf("Error happened checking labelhook existence in db, error: %v, labelhook id: %s", err, labelID)
		return
	}
	if !label_id_exist {
		lh.RenderError(http.StatusNotFound, "Error, label_id does not exist")
		return
	}

	label, err := dao.GetLabelByID(labelID)
	labelHook := models.LabelHook{
		LabelID:   req.LabelID,
		LabelName: label[0].Name,
		RepoName:  req.RepoName,
	}

	labelHookID, err := dao.NewLabelHook(labelHook)
	if err != nil {
		log.Errorf("Failed to new labelHook, error: %v", err)
		dup, _ := regexp.MatchString(dupLabelHookPattern, err.Error())
		if dup {
			lh.RenderError(http.StatusConflict, "label_id + repo_name conflict")
		} else {
			lh.RenderError(http.StatusInternalServerError, "Failed to new labelHook")
		}
		return
	}

	log.Debugf("Add new labelHook, id: %v", labelHookID)
	lh.CustomAbort(http.StatusCreated, strconv.Itoa(int(labelHookID)))
}

// Delete, delete labelhook  ...
func (lh *LabelHookAPI) Delete() {
	idStr := lh.Ctx.Input.Param(":id")
	id, _ := strconv.Atoi(idStr)

	log.Debugf("DELETE api/labelhooks, id: %v", id)

	if err := dao.DeleteLabelHook(int64(id)); err != nil {
		log.Errorf("Failed to delete labelhook, error: %v", err)
		lh.RenderError(http.StatusInternalServerError, "Failed to delete labelhook")
	}
}

// List ...
func (lh *LabelHookAPI) List() {
	log.Infof("/api/labelhooks/list")
	idStr := lh.Ctx.Input.Param(":lid")

	if !(len(idStr) > 0) {
		lh.CustomAbort(http.StatusBadRequest, "invalid label id")
	}

	var err error
	lh.labelID, err = strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		log.Errorf("Error parsing label id: %s, error: %v", idStr, err)
		lh.CustomAbort(http.StatusBadRequest, "invalid label id")
	}

	exist, err := dao.LabelExists(lh.labelID)
	if err != nil {
		log.Errorf("Error occurred in LabelExists, error: %v", err)
		lh.CustomAbort(http.StatusInternalServerError, "Internal error.")
	}

	if !exist {
		lh.CustomAbort(http.StatusNotFound, fmt.Sprintf("label does not exist, id: %v", lh.labelID))
	}

	labelhooks, err := dao.GetLabelHooksByLabelID(lh.labelID)
	if err != nil {
		log.Errorf("failed to get labelhooks from label %d: %v", lh.labelID, err)
		lh.CustomAbort(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	}

	lh.Data["json"] = labelhooks
	lh.ServeJSON()
}

// ListLabelHooksByRepoName ...
func (lh *LabelHookAPI) ListLabelHooksByRepoName() {
	repoName := lh.GetString("repo_name")
	log.Debugf("GET /api/labelhooks/list/by_reponame, repoName: %v", repoName)

	if !(len(repoName) > 0) {
		lh.CustomAbort(http.StatusBadRequest, "invalid repo_name")
	}

	labelhooks, err := dao.GetLabelHooksByRepoName(repoName)
	if err != nil {
		log.Errorf("failed to get labelnames from repo_name %d: %v", repoName, err)
		lh.CustomAbort(http.StatusInternalServerError, http.StatusText(http.StatusInternalServerError))
	}

	lh.Data["json"] = labelhooks
	lh.ServeJSON()
}
