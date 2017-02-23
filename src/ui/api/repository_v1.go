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
	"io/ioutil"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/docker/distribution/manifest/schema1"
	"github.com/docker/distribution/manifest/schema2"
	"github.com/vmware/harbor/src/common/api"
	"github.com/vmware/harbor/src/common/dao"
	"github.com/vmware/harbor/src/common/models"
	"github.com/vmware/harbor/src/common/utils"
	"github.com/vmware/harbor/src/common/utils/log"
	"github.com/vmware/harbor/src/common/utils/registry"
	registry_error "github.com/vmware/harbor/src/common/utils/registry/error"
	"github.com/vmware/harbor/src/ui/service/cache"
)

// RepositoryAPIV1 handles request to /api/v1/repos /api/v1/repos/:rid /api/v1/repos/:rid/tags /api/v1/repos/:rid/tags/:tag,
// the parm has to be put in the query string as the web framework can not parse the URL if it contains veriadic sectors.
type RepositoryAPIV1 struct {
	api.BaseAPI
	userID int
}

// Prepare validates the URL and the params
func (r *RepositoryAPIV1) Prepare() {
	idStr := r.Ctx.Input.Param(":rid")
	if len(idStr) > 0 {
		var err error
		repoId, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			log.Errorf("Error parsing repo id: %s, error: %v", idStr, err)
			r.CustomAbort(http.StatusBadRequest, "invalid repo id")
		}
		exist := dao.RepositoryIdExists(repoId)

		if !exist {
			r.CustomAbort(http.StatusNotFound, fmt.Sprintf("repo does not exist, id: %v", repoId))
		}
	}
}

// List ...
func (r *RepositoryAPIV1) List() {
	userId := r.ValidateUser()

	start, err := r.GetInt64("start", 0)
	limit, err := r.GetInt64("limit", 0)

	log.Debugf("List repos, err: %v", err)
	log.Debugf("List repos, start: %v", start)
	log.Debugf("List repos, limit: %v", limit)

	if err != nil || start < 0 || limit < 0 {
		r.CustomAbort(http.StatusBadRequest, "invalid start/limit")
	}

	name := r.GetString("name")
	projectIds := r.GetStrings("projectIds")
	labels := r.GetStrings("labels")

	log.Debugf("List repos, name: %v", name)
	log.Debugf("List repos, projectIds: %v", projectIds)
	log.Debugf("List repos, labels: %v", labels)

	// default value
	var page int64
	var pageSize int64
	page = 1
	pageSize = 20

	if limit > 0 {
		page = (start / limit) + 1
		pageSize = limit
	}

	total, repos, err := dao.GetRepositoryWithConditions(userId, projectIds, labels, name, page, pageSize)
	if err != nil {
		log.Errorf("failed to get repository: %v", err)
		r.CustomAbort(http.StatusInternalServerError, "failed to get repository with conditions")
	}

	var reposV1 []models.RepoRecordV1
	for i := 0; i < len(repos); i++ {
		repoV1 := models.RepoRecordV1{
			RepositoryID: repos[i].RepositoryID,
			Name:         repos[i].Name,
			ProjectID:    repos[i].ProjectID,
			Manager:      repos[i].Manager,
			Description:  repos[i].Description,
			PullCount:    repos[i].PullCount,
			StarCount:    repos[i].StarCount,
			TagCount:     repos[i].TagCount,
			LatestTag:    repos[i].LatestTag,
			LTagCTime:    repos[i].LTagCTime,
			Author:       repos[i].Author,
			LabelNames:   repos[i].LabelNames,
			CreationTime: repos[i].CreationTime,
		}

		repoV1.ProjectName = (strings.Split(repos[i].Name, "/"))[0]
		reposV1 = append(reposV1, repoV1)
	}

	r.Data["json"] = models.NewListResponse(total, reposV1)
	r.ServeJSON()
}

// Get handles GET /api/v1/repos/:rid
func (r *RepositoryAPIV1) Get() {
	idStr := r.Ctx.Input.Param(":rid")

	if len(idStr) < 0 {
		r.CustomAbort(http.StatusNotFound, fmt.Sprintf("repo does not exist, id: %v", idStr))
	}

	repoId, err := strconv.ParseInt(idStr, 10, 64)

	repoV1, err := dao.GetRepositoryByIdV1(repoId)
	if err != nil {
		log.Errorf("failed to get repository by id: %d, error: %v", repoId, err)
		r.CustomAbort(http.StatusInternalServerError, "")
	}

	r.Data["json"] = repoV1
	r.ServeJSON()
}

// Delete handles DELETE /api/v1/repos/:rid/tags:/:tag
func (r *RepositoryAPIV1) Delete() {
	idStr := r.Ctx.Input.Param(":rid")

	if len(idStr) < 0 {
		r.CustomAbort(http.StatusNotFound, fmt.Sprintf("repo does not exist, id: %v", idStr))
	}

	repoId, err := strconv.ParseInt(idStr, 10, 64)
	repoV1, err := dao.GetRepositoryByIdV1(repoId)

	if err != nil {
		log.Errorf("failed to get repository by id: %d, error: %v", repoId, err)
		r.CustomAbort(http.StatusInternalServerError, "")
	}

	repoName := repoV1.Name
	if len(repoName) == 0 {
		r.CustomAbort(http.StatusBadRequest, "repo name is nil")
	}

	projectName, _ := utils.ParseRepository(repoName)
	project, err := dao.GetProjectByName(projectName)
	if err != nil {
		log.Errorf("failed to get project %s: %v", projectName, err)
		r.CustomAbort(http.StatusInternalServerError, "")
	}

	if project == nil {
		r.CustomAbort(http.StatusNotFound, fmt.Sprintf("project %s not found", projectName))
	}

	if project.Public == 0 {
		userID := r.ValidateUser()
		if !hasProjectAdminRole(userID, project.ProjectID) {
			r.CustomAbort(http.StatusForbidden, "")
		}
	}

	endpoint := os.Getenv("REGISTRY_URL")

	// get tags and latest manifest
	rc, err := cache.NewRepositoryClient(endpoint, api.GetIsInsecure(), "admin", repoName,
		"repository", repoName, "pull", "push", "*")

	if err != nil {
		log.Errorf("error occurred while initializing repository client for %s: %v", repoName, err)
		r.CustomAbort(http.StatusInternalServerError, "internal error")
	}

	tags := []string{}
	tag := r.Ctx.Input.Param(":tag")
	if len(tag) == 0 {
		tagList, err := rc.ListTag()
		if err != nil {
			if regErr, ok := err.(*registry_error.Error); ok {
				r.CustomAbort(regErr.StatusCode, regErr.Detail)
			}

			log.Errorf("error occurred while listing tags of %s: %v", repoName, err)
			r.CustomAbort(http.StatusInternalServerError, "internal error")
		}

		// TODO remove the logic if the bug of registry is fixed
		if len(tagList) == 0 {
			r.CustomAbort(http.StatusNotFound, http.StatusText(http.StatusNotFound))
		}

		tags = append(tags, tagList...)
	} else {
		tags = append(tags, tag)
	}

	user, _, _ := r.Ctx.Request.BasicAuth()

	for _, t := range tags {
		if err := rc.DeleteTag(t); err != nil {
			if regErr, ok := err.(*registry_error.Error); ok {
				if regErr.StatusCode != http.StatusNotFound {
					r.CustomAbort(regErr.StatusCode, regErr.Detail)
				}
			} else {
				log.Errorf("error occurred while deleting tag %s:%s: %v", repoName, t, err)
				r.CustomAbort(http.StatusInternalServerError, "internal error")
			}
		}
		log.Infof("delete tag: %s:%s", repoName, t)
		go TriggerReplicationByRepository(repoName, []string{t}, models.RepOpDelete)

		go func(tag string) {
			if err := dao.AccessLog(user, projectName, repoName, tag, "delete"); err != nil {
				log.Errorf("failed to add access log: %v", err)
			}
		}(t)
	}

	exist, err := repositoryExist(repoName, rc)
	if err != nil {
		log.Errorf("failed to check the existence of repository %s: %v", repoName, err)
		r.CustomAbort(http.StatusInternalServerError, "")
	}
	if !exist {
		if err = dao.DeleteRepository(repoName); err != nil {
			log.Errorf("failed to delete repository %s: %v", repoName, err)
			r.CustomAbort(http.StatusInternalServerError, "")
		}
	} else {
		// Trigger sync repo latest manifest if delete a tag
		go TriggerSyncRepositoryLatestManifest(repoName)
	}

	go func() {
		log.Debug("refreshing catalog cache")
		if err := cache.RefreshCatalogCache(); err != nil {
			log.Errorf("error occurred while refresh catalog cache: %v", err)
		}
	}()

	r.RenderNoContent()
}

// GetTags handles GET /api/v1/repos/:rid/tags
func (r *RepositoryAPIV1) GetTags() {
	idStr := r.Ctx.Input.Param(":rid")

	if len(idStr) < 0 {
		r.CustomAbort(http.StatusNotFound, fmt.Sprintf("repo does not exist, id: %v", idStr))
	}

	repoId, err := strconv.ParseInt(idStr, 10, 64)
	repoV1, err := dao.GetRepositoryByIdV1(repoId)

	if err != nil {
		log.Errorf("failed to get repository by id: %d, error: %v", repoId, err)
		r.CustomAbort(http.StatusInternalServerError, "")
	}

	repoName := repoV1.Name
	if len(repoName) == 0 {
		r.CustomAbort(http.StatusBadRequest, "repo name is nil")
	}

	projectName, _ := utils.ParseRepository(repoName)
	project, err := dao.GetProjectByName(projectName)
	if err != nil {
		log.Errorf("failed to get project %s: %v", projectName, err)
		r.CustomAbort(http.StatusInternalServerError, "")
	}

	if project == nil {
		r.CustomAbort(http.StatusNotFound, fmt.Sprintf("project %s not found", projectName))
	}

	if project.Public == 0 {
		userID := r.ValidateUser()
		if !checkProjectPermission(userID, project.ProjectID) {
			r.CustomAbort(http.StatusForbidden, "")
		}
	}

	endpoint := os.Getenv("REGISTRY_URL")

	// get tags and latest manifest
	rc, err := cache.NewRepositoryClient(endpoint, api.GetIsInsecure(), "admin", repoName,
		"repository", repoName, "pull", "push", "*")

	if err != nil {
		log.Errorf("error occurred while initializing repository client for %s: %v", repoName, err)
		r.CustomAbort(http.StatusInternalServerError, "internal error")
	}

	tags := []string{}

	ts, err := rc.ListTag()
	if err != nil {
		regErr, ok := err.(*registry_error.Error)
		if !ok {
			log.Errorf("error occurred while listing tags of %s: %v", repoName, err)
			r.CustomAbort(http.StatusInternalServerError, "internal error")
		}
		// TODO remove the logic if the bug of registry is fixed
		// It's a workaround for a bug of registry: when listing tags of
		// a repository which is being pushed, a "NAME_UNKNOWN" error will
		// been returned, while the catalog API can list this repository.
		if regErr.StatusCode != http.StatusNotFound {
			r.CustomAbort(regErr.StatusCode, regErr.Detail)
		}
	}

	tags = append(tags, ts...)

	sort.Strings(tags)

	r.Data["json"] = models.NewListResponse(len(tags), tags)
	r.ServeJSON()
}

// GetManifests handles GET /api/v1/repos/:rid/tags/:tag
func (r *RepositoryAPIV1) GetManifests() {
	idStr := r.Ctx.Input.Param(":rid")

	if len(idStr) < 0 {
		r.CustomAbort(http.StatusNotFound, fmt.Sprintf("repo does not exist, id: %v", idStr))
	}

	repoId, err := strconv.ParseInt(idStr, 10, 64)
	repoV1, err := dao.GetRepositoryByIdV1(repoId)

	if err != nil {
		log.Errorf("failed to get repository by id: %d, error: %v", repoId, err)
		r.CustomAbort(http.StatusInternalServerError, "")
	}

	repoName := repoV1.Name
	if len(repoName) == 0 {
		r.CustomAbort(http.StatusBadRequest, "repo name is nil")
	}

	tag := r.Ctx.Input.Param(":tag")

	log.Errorf("GetManifests repoName: %v, tag: %v", repoName, tag)

	if len(repoName) == 0 || len(tag) == 0 {
		r.CustomAbort(http.StatusBadRequest, "repo_name or tag is nil")
	}

	version := r.GetString("version")
	if len(version) == 0 {
		version = "v2"
	}

	if version != "v1" && version != "v2" {
		r.CustomAbort(http.StatusBadRequest, "version should be v1 or v2")
	}

	projectName, _ := utils.ParseRepository(repoName)
	project, err := dao.GetProjectByName(projectName)
	if err != nil {
		log.Errorf("failed to get project %s: %v", projectName, err)
		r.CustomAbort(http.StatusInternalServerError, "")
	}

	if project == nil {
		r.CustomAbort(http.StatusNotFound, fmt.Sprintf("project %s not found", projectName))
	}

	if project.Public == 0 {
		userID := r.ValidateUser()
		if !checkProjectPermission(userID, project.ProjectID) {
			r.CustomAbort(http.StatusForbidden, "")
		}
	}

	endpoint := os.Getenv("REGISTRY_URL")

	// get tags and latest manifest
	rc, err := cache.NewRepositoryClient(endpoint, api.GetIsInsecure(), "admin", repoName,
		"repository", repoName, "pull", "push", "*")

	if err != nil {
		log.Errorf("error occurred while initializing repository client for %s: %v", repoName, err)
		r.CustomAbort(http.StatusInternalServerError, "internal error")
	}

	result := struct {
		Manifest interface{} `json:"manifest"`
		Config   interface{} `json:"config,omitempty" `
		VStatus  int         `json:"v_status"` // vulnerabilities analysis status
		VCount   int         `json:"v_count"`  // vulnerabilities count
		Vs       string      `json:"vs"`       // vulnerabilities string
	}{}

	mediaTypes := []string{}
	switch version {
	case "v1":
		mediaTypes = append(mediaTypes, schema1.MediaTypeManifest)
	case "v2":
		mediaTypes = append(mediaTypes, schema2.MediaTypeManifest)
	}

	_, mediaType, payload, err := rc.PullManifest(tag, mediaTypes)
	if err != nil {
		if regErr, ok := err.(*registry_error.Error); ok {
			r.CustomAbort(regErr.StatusCode, regErr.Detail)
		}

		log.Errorf("error occurred while getting manifest of %s:%s: %v", repoName, tag, err)
		r.CustomAbort(http.StatusInternalServerError, "internal error")
	}

	manifest, _, err := registry.UnMarshal(mediaType, payload)
	if err != nil {
		log.Errorf("an error occurred while parsing manifest of %s:%s: %v", repoName, tag, err)
		r.CustomAbort(http.StatusInternalServerError, "")
	}

	result.Manifest = manifest

	deserializedmanifest, ok := manifest.(*schema2.DeserializedManifest)
	if ok {
		_, data, err := rc.PullBlob(deserializedmanifest.Target().Digest.String())
		if err != nil {
			log.Errorf("failed to get config of manifest %s:%s: %v", repoName, tag, err)
			r.CustomAbort(http.StatusInternalServerError, "")
		}

		b, err := ioutil.ReadAll(data)
		if err != nil {
			log.Errorf("failed to read config of manifest %s:%s: %v", repoName, tag, err)
			r.CustomAbort(http.StatusInternalServerError, "")
		}

		result.Config = string(b)
	}

	// get image
	vulnerabilities, err := dao.GetImageVulnerability(repoName, tag)
	log.Debugf("get vulnerabilities: %v", vulnerabilities)

	// do ... while (0)
	for ok := true; ok; ok = false {
		if err != nil || len(vulnerabilities) <= 0 {
			log.Errorf("failed to get vulnerabilities: %v", err)
			result.VStatus = 404
			result.VCount = 0
			result.Vs = ""
			break
		}
		result.VStatus = 200
		result.VCount = vulnerabilities[0].VulnerabilityCount
		result.Vs = vulnerabilities[0].Vulnerabilities
	}

	r.Data["json"] = result
	r.ServeJSON()
}
