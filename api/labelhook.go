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

// LabelHookAPI handles request to /api/labelhooks/{} /api/labelhooks/{}/logs
type LabelHookAPI struct {
	BaseAPI
	userID      int
	labelHookID int64
	projectID   int64
}

type labelHookReq struct {
	LabelID  int64  `json:"label_id"`
	RepoName string `json:"repo_name"`
	Tag      string `json:"tag"`
}

const dupLabelHookPattern = `Duplicate entry '\w+' for key 'name'`

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

	labelHook := models.LabelHook{
		LabelID:  req.LabelID,
		RepoName: req.RepoName,
		Tag:      req.Tag}

	labelHookID, err := dao.NewLabelHook(labelHook)
	if err != nil {
		log.Errorf("Failed to new labelHook, error: %v", err)
		dup, _ := regexp.MatchString(dupLabelHookPattern, err.Error())
		if dup {
			lh.RenderError(http.StatusConflict, "name conflict")
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
