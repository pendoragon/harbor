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
	"github.com/vmware/harbor/models"
	"github.com/vmware/harbor/utils/log"
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
	p, err := o.Raw("insert into labelhook (label_id, repo_name, creation_time, update_time, deleted) values (?, ?, ?, ?, ?)").Prepare()
	if err != nil {
		return 0, err
	}

	now := time.Now()
	r, err := p.Exec(labelhook.LabelID, labelhook.RepoName, now, now, 0)
	if err != nil {
		return 0, err
	}

	labelHookID, err := r.LastInsertId()
	if err != nil {
		return 0, err
	}

	return labelHookID, err
}

// Delete remove a label from the database.
func DeleteLabel(labelID int64) error {
	log.Debugf("DeleteLabel labelID: %v", labelID)
	o := GetOrmer()

	sql := "delete from label where label_id = ?"

	if _, err := o.Raw(sql, labelID).Exec(); err != nil {
		log.Errorf("Failed to delete label, error: %v", err)
		return err
	}

	return nil
}

// Delete remove a labelhook from the database.
func DeleteLabelHook(labelHookID int64) error {
	log.Debugf("DeleteLabelHook labelHookID: %v", labelHookID)
	o := GetOrmer()

	sql := "delete from labelhook where labelhook_id = ?"

	if _, err := o.Raw(sql, labelHookID).Exec(); err != nil {
		log.Errorf("Failed to delete labelhook, error: %v", err)
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

// GetLabelsByProjectID ...
func GetLabelsByProjectID(project_id int64, labelName string) ([]models.Label, error) {
	o := GetOrmer()

	sql := `select l.label_id, l.project_id, l.name, l.remark,
			l.owner_id, l.creation_time, l.update_time
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
		return nil, nil
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
			lh.creation_time,lh.update_time
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
