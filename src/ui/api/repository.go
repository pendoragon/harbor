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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"sort"
	"strings"

	"github.com/docker/distribution/manifest/schema1"
	"github.com/docker/distribution/manifest/schema2"
	"github.com/vmware/harbor/src/common/dao"
	"github.com/vmware/harbor/src/common/models"
	"github.com/vmware/harbor/src/ui/service/cache"
	svc_utils "github.com/vmware/harbor/src/ui/service/utils"
	"github.com/vmware/harbor/src/common/utils/log"
    "github.com/vmware/harbor/src/common/api"
	"github.com/vmware/harbor/src/common/utils/registry"

	registry_error "github.com/vmware/harbor/src/common/utils/registry/error"

	"github.com/vmware/harbor/src/common/utils"
	"github.com/vmware/harbor/src/common/utils/registry/auth"
)

// RepositoryAPI handles request to /api/repositories /api/repositories/tags /api/repositories/manifests, the parm has to be put
// in the query string as the web framework can not parse the URL if it contains veriadic sectors.
type RepositoryAPI struct {
	api.BaseAPI
	userID int
}

type repositoryReq struct {
	ProjectIDs []string `json:"project_ids"`
	LabelIDs   []string `json:"label_ids"`
	RepoName   string   `json:"repo_name"`
	Page       int64    `json:"page"`
	PageSize   int64    `json:"page_size"`
}

type repositoryRes struct {
	Total int                  `json:"total"`
	Repos []*models.RepoRecord `json:"repos"`
}

type getUnmarkedRepositoryReq struct {
	ProjectID string `json:"project_id"`
	LabelID   string `json:"label_id"`
	RepoName  string `json:"repo_name"`
	Start     int64  `json:"start"`
	Limit     int64  `json:"limit"`
}

// Get ...
func (ra *RepositoryAPI) Get() {
	projectID, err := ra.GetInt64("project_id")
	if err != nil || projectID <= 0 {
		ra.CustomAbort(http.StatusBadRequest, "invalid project_id")
	}

	page, pageSize := ra.GetPaginationParams()

	project, err := dao.GetProjectByID(projectID)
	if err != nil {
		log.Errorf("failed to get project %d: %v", projectID, err)
		ra.CustomAbort(http.StatusInternalServerError, "")
	}

	if project == nil {
		ra.CustomAbort(http.StatusNotFound, fmt.Sprintf("project %d not found", projectID))
	}

	if project.Public == 0 {
		var userID int

		if svc_utils.VerifySecret(ra.Ctx.Request) {
			userID = 1
		} else {
			userID = ra.ValidateUser()
		}

		if !checkProjectPermission(userID, projectID) {
			ra.CustomAbort(http.StatusForbidden, "")
		}
	}

	repositories, err := getReposByProject(project.Name, ra.GetString("q"))
	if err != nil {
		log.Errorf("failed to get repository: %v", err)
		ra.CustomAbort(http.StatusInternalServerError, "")
	}

	total := int64(len(repositories))

	if (page-1)*pageSize > total {
		repositories = []string{}
	} else {
		repositories = repositories[(page-1)*pageSize:]
	}

	if page*pageSize <= total {
		repositories = repositories[:pageSize]
	}

	ra.SetPaginationHeader(total, page, pageSize)

	ra.Data["json"] = repositories
	ra.ServeJSON()
}

// GetUnmarkedRepos
func (ra *RepositoryAPI) GetUnmarkedRepos() {
	var req getUnmarkedRepositoryReq
	ra.DecodeJSONReq(&req)

	log.Debugf("GetUnmarkedRepos req: %v", req)

	total, repositories, err := dao.GetUnmarkedRepositoryByProjectIDAndLabelID(req.ProjectID, req.LabelID, req.RepoName, req.Start, req.Limit)
	if err != nil {
		log.Errorf("failed to get repository: %v", err)
		ra.CustomAbort(http.StatusInternalServerError, "failed to get unmarked repository")
	}

	log.Debugf("total: %v", total)

	repository_res := repositoryRes{
		Total: total,
		Repos: repositories,
	}

	ra.Data["json"] = repository_res
	ra.ServeJSON()
}

// GetRepositoryWithConditions ...
func (ra *RepositoryAPI) GetRepositoryWithConditions() {
	ra.userID = ra.ValidateUser()

	var req repositoryReq
	ra.DecodeJSONReq(&req)

	total, repositories, err := dao.GetRepositoryWithConditions(ra.userID, req.ProjectIDs, req.LabelIDs, req.RepoName, req.Page, req.PageSize)
	if err != nil {
		log.Errorf("failed to get repository: %v", err)
		ra.CustomAbort(http.StatusInternalServerError, "failed to get repository with conditions")
	}

	log.Debugf("total: %v", total)

	repository_res := repositoryRes{
		Total: total,
		Repos: repositories,
	}

	ra.Data["json"] = repository_res
	ra.ServeJSON()
}

// List ...
func (ra *RepositoryAPI) List() {
	repoList, err := cache.GetRepoFromCache()
	if err != nil {
		log.Errorf("Failed to get repo from cache, error: %v", err)
		ra.RenderError(http.StatusInternalServerError, "internal sever error")
	}

	repoName := ra.GetString("repo_name")
	var resp []string

	if len(repoName) > 0 {
		for _, r := range repoList {
			if strings.Contains(r, "/") && strings.Contains(r[strings.LastIndex(r, "/")+1:], repoName) {
				resp = append(resp, r)
			}
		}
		ra.Data["json"] = resp
	} else {
		ra.Data["json"] = repoList
	}
	ra.ServeJSON()
}

// Delete ...
func (ra *RepositoryAPI) Delete() {
	repoName := ra.GetString("repo_name")
	if len(repoName) == 0 {
		ra.CustomAbort(http.StatusBadRequest, "repo_name is nil")
	}

	projectName, _ := utils.ParseRepository(repoName)
	project, err := dao.GetProjectByName(projectName)
	if err != nil {
		log.Errorf("failed to get project %s: %v", projectName, err)
		ra.CustomAbort(http.StatusInternalServerError, "")
	}

	if project == nil {
		ra.CustomAbort(http.StatusNotFound, fmt.Sprintf("project %s not found", projectName))
	}

	if project.Public == 0 {
		userID := ra.ValidateUser()
		if !hasProjectAdminRole(userID, project.ProjectID) {
			ra.CustomAbort(http.StatusForbidden, "")
		}
	}

	rc, err := ra.initRepositoryClient(repoName)
	if err != nil {
		log.Errorf("error occurred while initializing repository client for %s: %v", repoName, err)
		ra.CustomAbort(http.StatusInternalServerError, "internal error")
	}

	tags := []string{}
	tag := ra.GetString("tag")
	if len(tag) == 0 {
		tagList, err := rc.ListTag()
		if err != nil {
			if regErr, ok := err.(*registry_error.Error); ok {
				ra.CustomAbort(regErr.StatusCode, regErr.Detail)
			}

			log.Errorf("error occurred while listing tags of %s: %v", repoName, err)
			ra.CustomAbort(http.StatusInternalServerError, "internal error")
		}

		// TODO remove the logic if the bug of registry is fixed
		if len(tagList) == 0 {
			ra.CustomAbort(http.StatusNotFound, http.StatusText(http.StatusNotFound))
		}

		tags = append(tags, tagList...)
	} else {
		tags = append(tags, tag)
	}

	user, _, ok := ra.Ctx.Request.BasicAuth()
	if !ok {
		user, err = ra.getUsername()
		if err != nil {
			log.Errorf("failed to get user: %v", err)
		}
	}

	for _, t := range tags {
		if err := rc.DeleteTag(t); err != nil {
			if regErr, ok := err.(*registry_error.Error); ok {
				if regErr.StatusCode != http.StatusNotFound {
					ra.CustomAbort(regErr.StatusCode, regErr.Detail)
				}
			} else {
				log.Errorf("error occurred while deleting tag %s:%s: %v", repoName, t, err)
				ra.CustomAbort(http.StatusInternalServerError, "internal error")
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
		ra.CustomAbort(http.StatusInternalServerError, "")
	}
	if !exist {
		if err = dao.DeleteRepository(repoName); err != nil {
			log.Errorf("failed to delete repository %s: %v", repoName, err)
			ra.CustomAbort(http.StatusInternalServerError, "")
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

}

type tag struct {
	Name string   `json:"name"`
	Tags []string `json:"tags"`
}

// GetTags handles GET /api/repositories/tags
func (ra *RepositoryAPI) GetTags() {
	repoName := ra.GetString("repo_name")
	if len(repoName) == 0 {
		ra.CustomAbort(http.StatusBadRequest, "repo_name is nil")
	}

	projectName, _ := utils.ParseRepository(repoName)
	project, err := dao.GetProjectByName(projectName)
	if err != nil {
		log.Errorf("failed to get project %s: %v", projectName, err)
		ra.CustomAbort(http.StatusInternalServerError, "")
	}

	if project == nil {
		ra.CustomAbort(http.StatusNotFound, fmt.Sprintf("project %s not found", projectName))
	}

	if project.Public == 0 {
		userID := ra.ValidateUser()
		if !checkProjectPermission(userID, project.ProjectID) {
			ra.CustomAbort(http.StatusForbidden, "")
		}
	}

	rc, err := ra.initRepositoryClient(repoName)
	if err != nil {
		log.Errorf("error occurred while initializing repository client for %s: %v", repoName, err)
		ra.CustomAbort(http.StatusInternalServerError, "internal error")
	}

	tags := []string{}

	ts, err := rc.ListTag()
	if err != nil {
		regErr, ok := err.(*registry_error.Error)
		if !ok {
			log.Errorf("error occurred while listing tags of %s: %v", repoName, err)
			ra.CustomAbort(http.StatusInternalServerError, "internal error")
		}
		// TODO remove the logic if the bug of registry is fixed
		// It's a workaround for a bug of registry: when listing tags of
		// a repository which is being pushed, a "NAME_UNKNOWN" error will
		// been returned, while the catalog API can list this repository.
		if regErr.StatusCode != http.StatusNotFound {
			ra.CustomAbort(regErr.StatusCode, regErr.Detail)
		}
	}

	tags = append(tags, ts...)

	sort.Strings(tags)

	ra.Data["json"] = tags
	ra.ServeJSON()
}

// GetManifests handles GET /api/repositories/manifests
func (ra *RepositoryAPI) GetManifests() {
	repoName := ra.GetString("repo_name")
	tag := ra.GetString("tag")

	if len(repoName) == 0 || len(tag) == 0 {
		ra.CustomAbort(http.StatusBadRequest, "repo_name or tag is nil")
	}

	version := ra.GetString("version")
	if len(version) == 0 {
		version = "v2"
	}

	if version != "v1" && version != "v2" {
		ra.CustomAbort(http.StatusBadRequest, "version should be v1 or v2")
	}

	projectName, _ := utils.ParseRepository(repoName)
	project, err := dao.GetProjectByName(projectName)
	if err != nil {
		log.Errorf("failed to get project %s: %v", projectName, err)
		ra.CustomAbort(http.StatusInternalServerError, "")
	}

	if project == nil {
		ra.CustomAbort(http.StatusNotFound, fmt.Sprintf("project %s not found", projectName))
	}

	if project.Public == 0 {
		userID := ra.ValidateUser()
		if !checkProjectPermission(userID, project.ProjectID) {
			ra.CustomAbort(http.StatusForbidden, "")
		}
	}

	rc, err := ra.initRepositoryClient(repoName)
	if err != nil {
		log.Errorf("error occurred while initializing repository client for %s: %v", repoName, err)
		ra.CustomAbort(http.StatusInternalServerError, "internal error")
	}

	result := struct {
		Manifest interface{} `json:"manifest"`
		Config   interface{} `json:"config,omitempty" `
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
			ra.CustomAbort(regErr.StatusCode, regErr.Detail)
		}

		log.Errorf("error occurred while getting manifest of %s:%s: %v", repoName, tag, err)
		ra.CustomAbort(http.StatusInternalServerError, "internal error")
	}

	manifest, _, err := registry.UnMarshal(mediaType, payload)
	if err != nil {
		log.Errorf("an error occurred while parsing manifest of %s:%s: %v", repoName, tag, err)
		ra.CustomAbort(http.StatusInternalServerError, "")
	}

	result.Manifest = manifest

	deserializedmanifest, ok := manifest.(*schema2.DeserializedManifest)
	if ok {
		_, data, err := rc.PullBlob(deserializedmanifest.Target().Digest.String())
		if err != nil {
			log.Errorf("failed to get config of manifest %s:%s: %v", repoName, tag, err)
			ra.CustomAbort(http.StatusInternalServerError, "")
		}

		b, err := ioutil.ReadAll(data)
		if err != nil {
			log.Errorf("failed to read config of manifest %s:%s: %v", repoName, tag, err)
			ra.CustomAbort(http.StatusInternalServerError, "")
		}

		result.Config = string(b)
	}

	ra.Data["json"] = result
	ra.ServeJSON()
}

func (ra *RepositoryAPI) GetVulnerabilities() {
	repoName := ra.GetString("repo_name")
	if len(repoName) == 0 {
		ra.CustomAbort(http.StatusBadRequest, "repo_name is nil")
	}

	tag := ra.GetString("tag")
	if len(repoName) == 0 {
		ra.CustomAbort(http.StatusBadRequest, "tag is nil")
	}

	vulnerabilities, err := dao.GetImageVulnerability(repoName, tag)
	log.Debugf("get vulnerabilities: %v", vulnerabilities)

	if err != nil {
		log.Errorf("failed to get vulnerabilities: %v", err)
		ra.CustomAbort(http.StatusInternalServerError, "failed to get vulnerabilities")
	}

	if len(vulnerabilities) <= 0 {
		ra.CustomAbort(http.StatusOK, "")
	}

	log.Debugf("get vulnerabilities[0]: %v", vulnerabilities[0])
	ra.CustomAbort(http.StatusOK, vulnerabilities[0].Vulnerabilities)
}

func (ra *RepositoryAPI) initRepositoryClient(repoName string) (r *registry.Repository, err error) {
	endpoint := os.Getenv("REGISTRY_URL")

	username, password, ok := ra.Ctx.Request.BasicAuth()
	if ok {
		return newRepositoryClient(endpoint, api.GetIsInsecure(), username, password,
			repoName, "repository", repoName, "pull", "push", "*")
	}

	username, err = ra.getUsername()
	if err != nil {
		return nil, err
	}

	return cache.NewRepositoryClient(endpoint, api.GetIsInsecure(), username, repoName,
		"repository", repoName, "pull", "push", "*")
}

func (ra *RepositoryAPI) getUsername() (string, error) {
	// get username from session
	sessionUsername := ra.GetSession("username")
	if sessionUsername != nil {
		username, ok := sessionUsername.(string)
		if ok {
			return username, nil
		}
	}

	// if username does not exist in session, try to get userId from sessiion
	// and then get username from DB according to the userId
	sessionUserID := ra.GetSession("userId")
	if sessionUserID != nil {
		userID, ok := sessionUserID.(int)
		if ok {
			u := models.User{
				UserID: userID,
			}
			user, err := dao.GetUser(u)
			if err != nil {
				return "", err
			}

			return user.Username, nil
		}
	}

	return "", nil
}

//GetTopRepos handles request GET /api/repositories/top
func (ra *RepositoryAPI) GetTopRepos() {
	count, err := ra.GetInt("count", 10)
	if err != nil || count <= 0 {
		ra.CustomAbort(http.StatusBadRequest, "invalid count")
	}

	repos, err := dao.GetTopRepos(count)
	if err != nil {
		log.Errorf("failed to get top repos: %v", err)
		ra.CustomAbort(http.StatusInternalServerError, "internal server error")
	}
	ra.Data["json"] = repos
	ra.ServeJSON()
}

func newRepositoryClient(endpoint string, insecure bool, username, password, repository, scopeType, scopeName string,
	scopeActions ...string) (*registry.Repository, error) {

	credential := auth.NewBasicAuthCredential(username, password)
	authorizer := auth.NewStandardTokenAuthorizer(credential, insecure, scopeType, scopeName, scopeActions...)

	store, err := auth.NewAuthorizerStore(endpoint, insecure, authorizer)
	if err != nil {
		return nil, err
	}

	client, err := registry.NewRepositoryWithModifiers(repository, endpoint, insecure, store)
	if err != nil {
		return nil, err
	}
	return client, nil
}

// TriggerSyncRepositoryLatestManifest
func TriggerSyncRepositoryLatestManifest(repo_name string) error {
	log.Debugf("TriggerSyncRepositoryLatestManifest, repo_name: %v", repo_name)

	endpoint := os.Getenv("REGISTRY_URL")

	// get tags and latest manifest
	rc, err := newRepositoryClient(endpoint, getIsInsecure(), "admin", os.Getenv("HARBOR_ADMIN_PASSWORD"),
		repo_name, "repository", repo_name, "pull", "push", "*")

	if err != nil {
		log.Errorf("error occurred while initializing repository client for %s: %v", repo_name, err)
		return nil
	}

	tags := []string{}

	ts, err := rc.ListTag()
	if err != nil {
		regErr, ok := err.(*registry_error.Error)
		if !ok {
			log.Errorf("error occurred while listing tags of %s: %v", repo_name, err)
			return nil
		}
		// TODO remove the logic if the bug of registry is fixed
		// It's a workaround for a bug of registry: when listing tags of
		// a repository which is being pushed, a "NAME_UNKNOWN" error will
		// been returned, while the catalog API can list this repository.
		if regErr.StatusCode != http.StatusNotFound {
			log.Errorf("regErr.StatusCode != http.StatusNotFound")
			return nil
		}
	}

	tags = append(tags, ts...)
	log.Debugf("get tags: %v", tags)

	sort.Strings(tags)
	log.Debugf("get tags after sort: %v", tags)

	// get manifest of latest tag
	mediaTypes := []string{}
	mediaTypes = append(mediaTypes, schema1.MediaTypeManifest)

	latest_tag := tags[len(tags)-1]

	_, mediaType, payload, err := rc.PullManifest(latest_tag, mediaTypes)
	log.Debugf("mediaType: %v", mediaType)
	if err != nil {
		if regErr, ok := err.(*registry_error.Error); ok {
			log.Errorf("registry_error: %v", regErr)
			return nil
		}

		log.Errorf("error occurred while getting manifest of %s:%s: %v", repo_name, latest_tag, err)
		return nil
	}

	signed_manifest_v1 := new(schema1.SignedManifest)
	err = signed_manifest_v1.UnmarshalJSON(payload)
	if err != nil {
		log.Errorf("UnmarshalJSON to get signed_manifest_v1 error, repo_name: %s, tag: %s", repo_name, latest_tag)
		return nil
	}

	if len(signed_manifest_v1.History) <= 0 {
		log.Errorf("signed_manifest_v1 has no history, repo_name: %s, tag: %s", repo_name, latest_tag)
		return nil
	}

	// convert string to json
	type V1Compatibility struct {
		Architecture    string      `json:"architecture"`
		Config          interface{} `json:"config"`
		Container       string      `json:"container"`
		ContainerConfig interface{} `json:"container_config"`
		Created         string      `json:"created"`
		DockerVersion   string      `json:"docker_version"`
		ID              string      `json:"id"`
		OS              string      `json:"os"`
		Parent          string      `json:"parent"`
		Throwaway       bool        `json:"throwaway"`
	}

	var v1_compatibility V1Compatibility

	history0_v1_compatibility_str := signed_manifest_v1.History[0].V1Compatibility
	err = json.Unmarshal([]byte(history0_v1_compatibility_str), &v1_compatibility)
	if err != nil {
		fmt.Println("error:", err)
	}

	log.Debugf("v1_compatibility.Created: %v", v1_compatibility.Created)

	if err := dao.UpdateRepositoryLatestManifest(repo_name, latest_tag, v1_compatibility.Created, len(tags), "N/A"); err != nil {
		log.Errorf("Error occurred in UpdateRepositoryLatestManifest: %v", err)
		return err
	}

	return nil
}
