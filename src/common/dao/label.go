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

package dao

import (
	"fmt"
	"github.com/vmware/harbor/src/common/models"
	"github.com/vmware/harbor/src/common/utils/log"
	"strings"
	"time"
)

// NewLabel insert a label to the database.
func NewLabel(label models.Label) (int64, error) {
	log.Debugf("NewLabel: %v", label)
	o := GetOrmer()
	p, err := o.Raw("insert into label (owner_id, project_id, name, remark, creation_time, update_time, deleted) values (?, ?, ?, ?, ?, ?, ?)").Prepare()
	if err != nil {
		return 0, err
	}

	now := time.Now()
	r, err := p.Exec(label.OwnerID, label.ProjectID, label.Name, label.Remark, now, now, 0)
	if err != nil {
		return 0, err
	}

	labelID, err := r.LastInsertId()
	if err != nil {
		return 0, err
	}

	return labelID, err
}

// NewLabelHook insert a labelhook to the database.
func NewLabelHook(labelhook models.LabelHook) (int64, error) {
	log.Debugf("NewLabelHook: %v", labelhook)
	o := GetOrmer()
	p, err := o.Raw("insert into labelhook (label_id, label_name, repo_name, creation_time, update_time, deleted) values (?, ?, ?, ?, ?, ?)").Prepare()
	if err != nil {
		return 0, err
	}

	now := time.Now()
	r, err := p.Exec(labelhook.LabelID, labelhook.LabelName, labelhook.RepoName, now, now, 0)
	if err != nil {
		return 0, err
	}

	labelHookID, err := r.LastInsertId()
	if err != nil {
		return 0, err
	}

	// update label names cached in repositry table
	err = SyncRepositoryLabelNames(labelhook.RepoName)

	return labelHookID, err
}

// UpdateLabel update label's name remark and repos
func UpdateLabel(label models.Label) error {
	params := []interface{}{}

	sql := "update label set "
	if len(label.Name) > 0 {
		sql += "name = ? "
		params = append(params, label.Name)
	}

	if len(label.Remark) > 0 {
		if strings.Contains(sql, "=") {
			sql += ", remark = ? "
		} else {
			sql += "remark = ? "
		}

		params = append(params, label.Remark)
	}

	if len(label.ReposStr) > 0 {
		if strings.Contains(sql, "=") {
			sql += ", repos_str = ? "
		} else {
			sql += "repos_str = ? "
		}

		params = append(params, label.ReposStr)
	}

	if !strings.Contains(sql, "=") {
		log.Debugf("Nothing to do on update label")
		return nil
	}

	sql += "where label_id = ?"
	log.Debugf("sql: %v", sql)
	params = append(params, label.LabelID)

	o := GetOrmer()

	if _, err := o.Raw(sql, params).Exec(); err != nil {
		log.Errorf("Failed to update label, error: %v", err)
		return err
	}

	return nil
}

// UpsertRepoRemark insert a repo_remark to the database or update if exists.
func UpsertRepoRemark(repoRemark models.RepoRemark) (int64, error) {
	log.Debugf("UpsertRepoRemark: %v", repoRemark)
	o := GetOrmer()
	p, err := o.Raw("insert into repo_remark (repo_name, remark, creation_time, update_time, deleted) values (?, ?, ?, ?, ?) ON DUPLICATE KEY UPDATE remark = ?").Prepare()
	if err != nil {
		return 0, err
	}

	now := time.Now()
	r, err := p.Exec(repoRemark.RepoName, repoRemark.Remark, now, now, 0, repoRemark.Remark)
	if err != nil {
		return 0, err
	}

	repoRemarkID, err := r.LastInsertId()
	if err != nil {
		return 0, err
	}

	return repoRemarkID, err
}

// GetRepoRemark ...
func GetRepoRemark(repo_name string) (string, error) {
	log.Debugf("GetRepoRemark: %v", repo_name)
	o := GetOrmer()
	var repo_remarks []models.RepoRemark
	n, err := o.Raw(`select remark from repo_remark where repo_name = ?`, repo_name).QueryRows(&repo_remarks)
	log.Debugf("repo_remarks: %v", repo_remarks)

	if err != nil {
		return "", err
	}

	if n == 0 {
		return "", nil
	}

	return repo_remarks[0].Remark, nil
}

// Delete remove a label from the database.
func DeleteLabel(labelID int64) error {
	log.Debugf("DeleteLabel labelID: %v", labelID)
	o := GetOrmer()

	// get repo_names by label_id
	var repo_names []string
	sql := "select repo_name from labelhook where label_id = ?"

	if _, err := o.Raw(sql, labelID).QueryRows(&repo_names); err != nil {
		log.Errorf("Failed to query labelhook to get repo_name, error: %v", err)
		return err
	}

	log.Debugf("get repo_names by label_id, repo_names: %v", repo_names)

	// get repos_str by label_id
	var repos_str []string
	sql = "select repos_str from label where label_id = ?"

	if _, err := o.Raw(sql, labelID).QueryRows(&repos_str); err != nil {
		log.Errorf("Failed to query label to get repos_str, error: %v", err)
		return err
	}

	log.Debugf("get repo_names by label_id, repo_names: %v", repo_names)
	// delete label
	sql = "delete from label where label_id = ?"

	if _, err := o.Raw(sql, labelID).Exec(); err != nil {
		log.Errorf("Failed to delete label, error: %v", err)
		return err
	}

	// update label names cached in repositry table V1
	if len(repos_str) > 0 {
		go func() {
			repos := strings.Split(repos_str[0], ",")
			for _, repo := range repos {
				log.Debugf("SyncRepositoryLabelNamesV1, repo: %v", repo)
				SyncRepositoryLabelNamesV1(repo)
			}
		}()
	}

	if len(repo_names) == 0 {
		return nil
	}

	// update label names cached in repositry table
	go func() {
		SyncRepositoryLabelNames(repo_names[0])
	}()

	return nil
}

// Delete remove a labelhook from the database.
func DeleteLabelHook(labelHookID int64) error {
	log.Debugf("DeleteLabelHook labelHookID: %v", labelHookID)
	o := GetOrmer()

	// get repo_names by labelhook_id
	var repo_names []string
	sql := "select repo_name from labelhook where labelhook_id = ?"

	if _, err := o.Raw(sql, labelHookID).QueryRows(&repo_names); err != nil {
		log.Errorf("Failed to query labelhook to get repo_name, error: %v", err)
		return err
	}

	// delete labelhook
	sql = "delete from labelhook where labelhook_id = ?"

	if _, err := o.Raw(sql, labelHookID).Exec(); err != nil {
		log.Errorf("Failed to delete labelhook, error: %v", err)
		return err
	}

	if len(repo_names) == 0 {
		return nil
	}

	// update label names cached in repositry table
	return SyncRepositoryLabelNames(repo_names[0])
}

func SyncRepositoryLabelNames(repo_name string) error {
	log.Debugf("SyncRepositoryLabelNames, repo_name: %v", repo_name)
	o := GetOrmer()
	var label_names []string

	sql := "select label_name from labelhook where repo_name = ?"

	if _, err := o.Raw(sql, repo_name).QueryRows(&label_names); err != nil {
		log.Errorf("Failed to get label names by repo_name, error: %v", err)
		return err
	}

	label_names_str := strings.Join(label_names, ",")
	log.Debugf("label_names_str: %v", label_names_str)

	if err := UpdateRepositoryLabelNames(repo_name, label_names_str); err != nil {
		log.Errorf("Error occurred in UpdateRepositoryLabelNames: %v", err)
		return err
	}

	return nil
}

func SyncRepositoryLabelNamesV1(repo_name string) error {
	log.Debugf("SyncRepositoryLabelNamesV1, repo_name: %v", repo_name)
	o := GetOrmer()
	var label_names []string

	sql := "select name from label where repos_str like ?"

	if _, err := o.Raw(sql, "%"+repo_name+"%").QueryRows(&label_names); err != nil {
		log.Errorf("Failed to get label names by repo_name, error: %v", err)
		return err
	}

	label_names_str := strings.Join(label_names, ",")
	log.Debugf("label_names_str: %v", label_names_str)

	if err := UpdateRepositoryLabelNames(repo_name, label_names_str); err != nil {
		log.Errorf("Error occurred in UpdateRepositoryLabelNames: %v", err)
		return err
	}

	return nil
}

// LabelExists returns whether the label exists according to its name of ID.
func LabelExists(nameOrID interface{}) (bool, error) {
	o := GetOrmer()
	type dummy struct{}
	sql := `select label_id from label where deleted = 0 and `
	switch nameOrID.(type) {
	case int64:
		sql += `label_id = ?`
	case string:
		sql += `name = ?`
	default:
		return false, fmt.Errorf("Invalid nameOrId: %v", nameOrID)
	}

	var d []dummy
	num, err := o.Raw(sql, nameOrID).QueryRows(&d)
	if err != nil {
		return false, err
	}
	return num > 0, nil
}

// LabelHookExists returns whether the labelhook exists according to its name of ID.
func LabelHookExists(nameOrID interface{}) (bool, error) {
	log.Debugf("LabelHookExists: %v", nameOrID)
	o := GetOrmer()
	type dummy struct{}
	sql := `select labelhook_id from labelhook where deleted = 0 and `
	switch nameOrID.(type) {
	case int64:
		sql += `labelhook_id = ?`
	case string:
		sql += `name = ?`
	default:
		return false, fmt.Errorf("Invalid nameOrId: %v", nameOrID)
	}

	log.Debugf("LabelHookExists sql: %v", sql)
	var d []dummy
	num, err := o.Raw(sql, nameOrID).QueryRows(&d)
	if err != nil {
		log.Errorf("LabelHookExists sql error: %v", err)
		return false, err
	}
	return num > 0, nil
}

// GetLabelByID ...
func GetLabelByID(label_id int64) (*models.Label, error) {
	o := GetOrmer()

	if label_id < 0 {
		return nil, fmt.Errorf("Invalid label_id: %v", label_id)
	}

	sql := `select * from label l where l.deleted = 0 and l.label_id = ?`
	queryParam := make([]interface{}, 1)

	queryParam = append(queryParam, label_id)

	var labels []models.Label
	count, err := o.Raw(sql, queryParam).QueryRows(&labels)

	if err != nil {
		return nil, err
	}

	if count == 0 {
		return nil, nil
	}

	return &labels[0], nil
}

// GetLabelsByProjectID ...
func GetLabelsByProjectID(project_id int64, labelName string) ([]models.Label, error) {
	o := GetOrmer()

	sql := `select l.label_id, l.project_id, l.name, l.remark,
			l.repos_str, l.owner_id, l.creation_time, l.update_time
			from label l left join user u on l.owner_id = u.user_id
			where l.deleted = 0`
	queryParam := make([]interface{}, 1)

	if project_id > 0 {
		sql += " and l.project_id = ?"
		queryParam = append(queryParam, project_id)
	}

	if len(labelName) > 0 {
		labelName = "%" + labelName + "%"
		sql += " and name like ? "
		queryParam = append(queryParam, labelName)
	}

	var labels []models.Label
	count, err := o.Raw(sql, queryParam).QueryRows(&labels)

	if err != nil {
		return nil, err
	}

	if count == 0 {
		return make([]models.Label, 0), nil
	}

	return labels, nil
}

// GetLabelHooksByLabelID ...
func GetLabelHooksByLabelID(label_id int64) ([]models.LabelHook, error) {
	o := GetOrmer()

	sql := `select lh.labelhook_id, lh.label_id, lh.repo_name, l.name as label_name,
			lh.creation_time,lh.update_time
			from labelhook lh left join label l on lh.label_id = l.label_id
			where lh.deleted = 0 and lh.label_id = ?`
	queryParam := make([]interface{}, 1)
	queryParam = append(queryParam, label_id)

	var labelhooks []models.LabelHook
	count, err := o.Raw(sql, queryParam).QueryRows(&labelhooks)

	if err != nil {
		return nil, err
	}

	if count == 0 {
		return nil, nil
	}

	return labelhooks, nil
}

// GetLabelHooksByRepoName ...
func GetLabelHooksByRepoName(repo_name string) ([]models.LabelHook, error) {
	o := GetOrmer()
	sql := `select lh.labelhook_id, lh.label_id, lh.repo_name, l.name as label_name,
			lh.creation_time, lh.update_time
			from labelhook lh left join label l on lh.label_id = l.label_id
			where lh.deleted = 0 and lh.repo_name = ?`
	queryParam := make([]interface{}, 1)
	queryParam = append(queryParam, repo_name)

	var labelhooks []models.LabelHook
	count, err := o.Raw(sql, queryParam).QueryRows(&labelhooks)

	if err != nil {
		return nil, err
	}

	if count == 0 {
		return nil, nil
	}

	return labelhooks, nil
}

// GetReposByLabelNames ...
func GetReposByLabelNames(label_names []string) ([]string, error) {
	// construct mysql query string
	for i := 0; i < len(label_names); i++ {
		label_names[i] = "\"" + label_names[i] + "\""
	}
	label_names_str := strings.Join(label_names, ",")

	// select from mysql table WHERE field='$array'?
	// ref:
	// http://stackoverflow.com/a/2382847/3167471
	o := GetOrmer()
	sql := "select label_id from label where name in (" + label_names_str + ")"

	// make sure dose QueryRows do type casting when map query results to container?
	// answer is YES
	// issue: https://github.com/astaxie/beego/issues/2177
	var label_ids []string
	count, err := o.Raw(sql).QueryRows(&label_ids)

	if err != nil {
		return nil, err
	}

	if count == 0 {
		return nil, nil
	}

	label_ids_str := strings.Join(label_ids, ",")

	sql = "select distinct(repo_name) from labelhook where label_id in (" + label_ids_str + ")"

	var repo_names []string
	count, err = o.Raw(sql).QueryRows(&repo_names)

	if err != nil {
		return nil, err
	}

	if count == 0 {
		return nil, nil
	}

	return repo_names, nil
}
