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
	"net/http"
	"os"
	"sort"

	klar_clair "github.com/optiopay/klar/clair"
	klar_docker "github.com/optiopay/klar/docker"
	"github.com/vmware/harbor/src/common/api"
	"github.com/vmware/harbor/src/common/dao"
	"github.com/vmware/harbor/src/common/models"
	"github.com/vmware/harbor/src/common/utils/log"
	// "strconv"
)

// RepoAnalysisAPI handles request to /api/repo/analysis
type RepoAnalysisAPI struct {
	api.BaseAPI
}

type repoAnalysisReq struct {
	RepoName string `json:"repo_name"`
}

type VulnerabilityList []klar_clair.Vulnerability

type Serverity string

const (
	// Unknown is either a security problem that has not been
	// assigned to a priority yet or a priority that our system
	// did not recognize
	Unknown Serverity = "Unknown"
	// Negligible is technically a security problem, but is
	// only theoretical in nature, requires a very special
	// situation, has almost no install base, or does no real
	// damage. These tend not to get backport from upstreams,
	// and will likely not be included in security updates unless
	// there is an easy fix and some other issue causes an update.
	Negligible Serverity = "Negligible"
	// Low is a security problem, but is hard to
	// exploit due to environment, requires a user-assisted
	// attack, a small install base, or does very little damage.
	// These tend to be included in security updates only when
	// higher priority issues require an update, or if many
	// low priority issues have built up.
	Low Serverity = "Low"
	// Medium is a real security problem, and is exploitable
	// for many people. Includes network daemon denial of service
	// attacks, cross-site scripting, and gaining user privileges.
	// Updates should be made soon for this priority of issue.
	Medium Serverity = "Medium"
	// High is a real problem, exploitable for many people in a default
	// installation. Includes serious remote denial of services,
	// local root privilege escalations, or data loss.
	High Serverity = "High"
	// Critical is a world-burning problem, exploitable for nearly all people
	// in a default installation of Linux. Includes remote root
	// privilege escalations, or massive data loss.
	Critical Serverity = "Critical"
	// Defcon1 is a Critical problem which has been manually highlighted by
	// the team. It requires an immediate attention.
	Defcon1 Serverity = "Defcon1"
)

var SeverityWeight map[Serverity]int

func init() {
	// init serverity weight map
	SeverityWeight = make(map[Serverity]int)
	weight := 0
	SeverityWeight[Unknown] = weight
	weight++
	SeverityWeight[Negligible] = weight
	weight++
	SeverityWeight[Low] = weight
	weight++
	SeverityWeight[Medium] = weight
	weight++
	SeverityWeight[High] = weight
	weight++
	SeverityWeight[Critical] = weight
	weight++
	SeverityWeight[Defcon1] = weight
}

// TriggerRepositoryAnalysis
func TriggerRepositoryAnalysis(repo string, tag string, username string, password string) ([]klar_clair.Vulnerability, error) {
	log.Debugf("TriggerRepositoryAnalysis, repo: %v, tag: %v, username: %v, password: %v", repo, tag, username, password)

	if len(repo) == 0 || len(tag) == 0 {
		return nil, fmt.Errorf("invalid parameter, repo/tag is required")
	}

	if len(username) == 0 || len(password) == 0 {
		username = "admin"
		password = os.Getenv("HARBOR_ADMIN_PASSWORD")
	}

	imageName := os.Getenv("HARBOR_REG_URL") + "/" + repo + ":" + tag

	log.Debugf("TriggerRepositoryAnalysis, imageName: %v, tag: %v, username: %v, password: %v", imageName, tag, username, password)

	image, err := klar_docker.NewImage(imageName, username, password)
	if err != nil {
		log.Errorf("new images err: %v", err)
		// return vulnerabilities, err
		return nil, fmt.Errorf("new images error: %v", err)
	}

	err = image.Pull()
	if err != nil {
		log.Errorf("get image layer info err: %v", err)
		// return vulnerabilities, err
		return nil, fmt.Errorf("get image layer info err: %v", err)
	}

	// AnalysisImage analysis image by Clair server
	var vulnerabilities []klar_clair.Vulnerability

	clairServerAddr := os.Getenv("CLAIR_SERVER_IP")
	if clairServerAddr == "" {
		clairServerAddr = "http://clair:6060"
	}

	log.Infof("clairServerAddr: %s", clairServerAddr)

	clairClient := klar_clair.NewClair(clairServerAddr)
	vulnerabilities = clairClient.Analyse(image)
	sort.Sort(VulnerabilityList(vulnerabilities))

	log.Infof("vulnerabilities got %d by Clair\n", len(vulnerabilities))

	return vulnerabilities, nil
}

// TriggerRepositoryAnalysisAndSaveResult
func TriggerRepositoryAnalysisAndSaveResult(repo string, tag string) error {
	log.Debugf("TriggerRepositoryAnalysisAndSaveResult, repo: %v, tag: %v", repo, tag)

	// username := r.GetSession("username").(string)
	// password := r.GetSession("password").(string)

	vulnerabilities, err := TriggerRepositoryAnalysis(repo, tag, "", "")

	if err != nil {
		return fmt.Errorf("repo analysis error: %v", err)
	}

	vulnerabilities_bytes, err := json.Marshal(&vulnerabilities)

	if err != nil {
		log.Errorf("json.Marshal error: %v", err)
		return fmt.Errorf("json.Marshal error: %v", err)
	}

	ImageVulnerability := models.ImageVulnerability{
		RepoName:           repo,
		Tag:                tag,
		VulnerabilityCount: len(vulnerabilities),
		Vulnerabilities:    string(vulnerabilities_bytes),
	}

	err = dao.AddImageVulnerability(ImageVulnerability)

	if err != nil {
		log.Errorf("add image vulnerability in to DB error: %v", err)
		return fmt.Errorf("add image vulnerability error: %v", err)
	}

	return nil
}

// GET ...
func (r *RepoAnalysisAPI) Get() {
	r.ValidateUser()

	repoName := r.GetString("repo_name")
	tag := r.GetString("tag")

	username := r.GetSession("username").(string)
	password := r.GetSession("password").(string)

	vulnerabilities, err := TriggerRepositoryAnalysis(repoName, tag, username, password)
	if err != nil {
		r.CustomAbort(http.StatusInternalServerError, "image analysis falied")
	}

	r.Data["json"] = vulnerabilities
	r.ServeJSON()
}

// High level is on the top
func compareSeverity(severity1 Serverity, severity2 Serverity) bool {
	return SeverityWeight[severity1] > SeverityWeight[severity2]
}

// Len realize function of interface sort
func (list VulnerabilityList) Len() int {
	return len(list)
}

// Less realize function of interface sort
func (list VulnerabilityList) Less(i, j int) bool {
	return compareSeverity(Serverity(list[i].Severity), Serverity(list[j].Severity))
}

// Swap realize function of interface sort
func (list VulnerabilityList) Swap(i, j int) {
	var temp klar_clair.Vulnerability = list[i]
	list[i] = list[j]
	list[j] = temp
}
