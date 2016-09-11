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
	"net/http"

	"github.com/vmware/harbor/dao"
	"github.com/vmware/harbor/models"
	"github.com/vmware/harbor/utils/log"

	"strconv"
)

// RepoRemarkAPI handles request to /api/reporemarks
type RepoRemarkAPI struct {
	BaseAPI
}

type repoRemarkReq struct {
	RepoName string `json:"repo_name"`
	Remark   string `json:"remark"`
}

// GET ...
func (r *RepoRemarkAPI) Get() {
	repoName := r.GetString("repo_name")
	log.Debugf("GET api/reporemarks, repo_name: %v", repoName)
	if len(repoName) == 0 {
		r.CustomAbort(http.StatusBadRequest, "repo_name is needed")
	}

	repo_remark, err := dao.GetRepoRemark(repoName)
	if err != nil {
		log.Errorf("Failed to get repo remark, error: %v", err)
		r.RenderError(http.StatusInternalServerError, "Failed to get repo remark")
		return
	}

	log.Debugf("GET repo_remark: %v", repo_remark)
	r.CustomAbort(http.StatusOK, repo_remark)
}

// POST ...
func (r *RepoRemarkAPI) Post() {
	var req repoRemarkReq
	r.DecodeJSONReq(&req)
	log.Debugf("POST api/reporemarks, req: %v", req)

	repoRemark := models.RepoRemark{
		RepoName: req.RepoName,
		Remark:   req.Remark,
	}

	repoRemarkID, err := dao.UpsertRepoRemark(repoRemark)
	if err != nil {
		log.Errorf("Failed to upsert repo remark, error: %v", err)
		r.RenderError(http.StatusInternalServerError, "Failed to upsert repo remark")
		return
	}

	log.Debugf("Upsert repo remark, id: %v", repoRemarkID)
	r.CustomAbort(http.StatusCreated, strconv.Itoa(int(repoRemarkID)))
}
