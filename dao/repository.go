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
	"strings"

	"github.com/astaxie/beego/orm"
	"github.com/vmware/harbor/models"
	"github.com/vmware/harbor/utils/log"
)

// AddRepository adds a repo to the database.
func AddRepository(repo models.RepoRecord) error {
	o := GetOrmer()
	sql := "insert into repository (owner_id, project_id, manager, name, description, pull_count, star_count, creation_time, update_time) " +
		"select (select user_id as owner_id from user where username=?), " +
		"(select project_id as project_id from project where name=?), " +
		"(select manager as manager from project where name=?), ?, ?, ?, ?, NOW(), NULL "

	_, err := o.Raw(sql, repo.OwnerName, repo.ProjectName, repo.ProjectName, repo.Name, repo.Description, repo.PullCount, repo.StarCount).Exec()
	return err
}

// AddImageVulnerability adds a image vulnerability to the database.
func AddImageVulnerability(image_vulnerability models.ImageVulnerability) error {
	o := GetOrmer()
	sql := "insert into image_vulnerability (repo_name, tag, v_count, vulnerabilities, creation_time, update_time) values (?, ?, ?, ?, NOW(), NOW())"

	_, err := o.Raw(sql, image_vulnerability.RepoName, image_vulnerability.Tag, image_vulnerability.VulnerabilityCount, image_vulnerability.Vulnerabilities).Exec()
	return err
}

// GetImageVulnerability ...
func GetImageVulnerability(repo_name string, tag string) ([]*models.ImageVulnerability, error) {
	sql := `select * from image_vulnerability where repo_name = ? and tag = ?`
	vulnerabilities := []*models.ImageVulnerability{}
	_, err := GetOrmer().Raw(sql, repo_name, tag).QueryRows(&vulnerabilities)
	return vulnerabilities, err
}

// GetRepositoryByName ...
func GetRepositoryByName(name string) (*models.RepoRecord, error) {
	o := GetOrmer()
	r := models.RepoRecord{Name: name}
	err := o.Read(&r, "Name")
	if err == orm.ErrNoRows {
		return nil, nil
	}
	return &r, err
}

// GetAllRepositories ...
func GetAllRepositories() ([]models.RepoRecord, error) {
	o := GetOrmer()
	var repos []models.RepoRecord
	_, err := o.QueryTable("repository").All(&repos)
	return repos, err
}

// DeleteRepository ...
func DeleteRepository(name string) error {
	o := GetOrmer()
	_, err := o.QueryTable("repository").Filter("name", name).Delete()
	return err
}

// UpdateRepository ...
func UpdateRepository(repo models.RepoRecord) error {
	o := GetOrmer()
	_, err := o.Update(&repo)
	return err
}

// UpdateRepositoryLabelNames ...
func UpdateRepositoryLabelNames(name string, label_names string) (err error) {
	o := GetOrmer()
	num, err := o.QueryTable("repository").Filter("name", name).Update(
		orm.Params{
			"label_names": label_names,
		})
	if num == 0 {
		err = fmt.Errorf("Failed to update repository's labelnames with name: %s %s", name, err.Error())
	}
	return err
}

// IncreasePullCount ...
func IncreasePullCount(name string) (err error) {
	o := GetOrmer()
	num, err := o.QueryTable("repository").Filter("name", name).Update(
		orm.Params{
			"pull_count": orm.ColValue(orm.ColAdd, 1),
		})
	if num == 0 {
		err = fmt.Errorf("Failed to increase repository pull count with name: %s %s", name, err.Error())
	}
	return err
}

//RepositoryExists returns whether the repository exists according to its name.
func RepositoryExists(name string) bool {
	o := GetOrmer()
	return o.QueryTable("repository").Filter("name", name).Exist()
}

// GetRepositoryByProjectName ...
func GetRepositoryByProjectName(name string) ([]*models.RepoRecord, error) {
	sql := `select * from repository 
		where project_id = (
			select project_id from project
			where name = ?
		)`
	repos := []*models.RepoRecord{}
	_, err := GetOrmer().Raw(sql, name).QueryRows(&repos)
	return repos, err
}

// GetRepositoryWithConditions ...
func GetRepositoryWithConditions(project_ids []string, label_ids []string, repo_name string, page int64, page_size int64) (int, []*models.RepoRecord, error) {
	if page <= 0 || page_size <= 0 {
		return 0, nil, fmt.Errorf("page and page_size should be greater than 0")
	}

	labelhooks := []*models.LabelHook{}
	var pick_repo_names []string

	if len(label_ids) > 0 {
		label_ids_str := strings.Join(label_ids, ",")
		labelhook_sql := "select lb.repo_name from labelhook lb where lb.label_id in (" + label_ids_str + ")"
		_, err := GetOrmer().Raw(labelhook_sql).QueryRows(&labelhooks)

		if err != nil {
			return 0, nil, err
		}

		log.Debugf("get labelhooks: %v", labelhooks)
	}

	// construct pick_repo_names
	if len(labelhooks) > 0 {
		for _, lebelhook := range labelhooks {
			pick_repo_names = append(pick_repo_names, lebelhook.RepoName)
		}
		log.Debugf("pick_repo_names: %v", pick_repo_names)
	}

	sql := "select * from repository"

	if len(project_ids) > 0 {
		project_ids_str := strings.Join(project_ids, ",")
		sql += " where project_id in (" + project_ids_str + ")"
	}

	if len(project_ids) > 0 && repo_name != "" {
		sql += " and name like \"%" + repo_name + "%\""
	} else if repo_name != "" {
		sql += " where name like \"%" + repo_name + "%\""
	}

	if len(pick_repo_names) > 0 {
		for i := 0; i < len(pick_repo_names); i++ {
			pick_repo_names[i] = "\"" + pick_repo_names[i] + "\""
		}

		pick_repo_names_str := strings.Join(pick_repo_names, ",")
		if strings.Contains(sql, "where") {
			sql += " and name in (" + pick_repo_names_str + ")"
		} else {
			sql += " where name in (" + pick_repo_names_str + ")"
		}
	}

	// for count without limit
	sql_count := sql

	offset := (page - 1) * page_size

	sql += " limit ?,?"
	log.Debugf("sql: %v", sql)

	repos := []*models.RepoRecord{}
	_, err := GetOrmer().Raw(sql, offset, page_size).QueryRows(&repos)

	if err != nil {
		return 0, nil, err
	}

	// get total count
	// ref:
	// http://stackoverflow.com/questions/186588/which-is-fastest-select-sql-calc-found-rows-from-table-or-select-count
	sql_count = strings.Replace(sql_count, "*", "COUNT(*)", 1)
	log.Debugf("sql_count: %v", sql_count)
	var total []int
	_, err = GetOrmer().Raw(sql_count).QueryRows(&total)

	if err != nil {
		return 0, nil, err
	}

	log.Debugf("total: %v", total)

	return total[0], repos, err
}

// GetUnmarkedRepositoryByProjectIDAndLabelID ...
func GetUnmarkedRepositoryByProjectIDAndLabelID(project_id string, label_id string, repo_name string, start int64, limit int64) (int, []*models.RepoRecord, error) {
	if start <= 0 || limit <= 0 {
		return 0, nil, fmt.Errorf("start and limit should be greater than 0")
	}

	sql := "SELECT * FROM repository r WHERE r.project_id = ? and r.name NOT IN (SELECT lh.repo_name FROM labelhook lh WHERE lh.label_id = ?)"

	if repo_name != "" {
		sql += " and r.name like \"%" + repo_name + "%\""
	}

	// for count without limit
	sql_count := sql

	sql += " limit ?,?"
	log.Debugf("sql: %v", sql)

	repos := []*models.RepoRecord{}
	_, err := GetOrmer().Raw(sql, project_id, label_id, start-1, limit).QueryRows(&repos)

	if err != nil {
		return 0, nil, err
	}

	// get total count
	// ref:
	// http://stackoverflow.com/questions/186588/which-is-fastest-select-sql-calc-found-rows-from-table-or-select-count
	sql_count = strings.Replace(sql_count, "*", "COUNT(*)", 1)
	log.Debugf("sql_count: %v", sql_count)
	var total []int
	_, err = GetOrmer().Raw(sql_count, project_id, label_id).QueryRows(&total)

	if err != nil {
		return 0, nil, err
	}

	log.Debugf("total: %v", total)

	return total[0], repos, err
}

//GetTopRepos returns the most popular repositories
func GetTopRepos(count int) ([]models.TopRepo, error) {
	topRepos := []models.TopRepo{}

	repositories := []*models.RepoRecord{}
	if _, err := GetOrmer().QueryTable(&models.RepoRecord{}).
		OrderBy("-PullCount", "Name").Limit(count).All(&repositories); err != nil {
		return topRepos, err
	}

	for _, repository := range repositories {
		topRepos = append(topRepos, models.TopRepo{
			RepoName:    repository.Name,
			AccessCount: repository.PullCount,
		})
	}

	return topRepos, nil
}

// GetTotalOfRepositories ...
func GetTotalOfRepositories(name string) (int64, error) {
	qs := GetOrmer().QueryTable(&models.RepoRecord{})
	if len(name) != 0 {
		qs = qs.Filter("Name__contains", name)
	}
	return qs.Count()
}

// GetTotalOfPublicRepositories ...
func GetTotalOfPublicRepositories(name string) (int64, error) {
	params := []interface{}{}
	sql := `select count(*) from repository r 
		join project p 
		on r.project_id = p.project_id and p.public = 1 `
	if len(name) != 0 {
		sql += ` where r.name like ?`
		params = append(params, "%"+name+"%")
	}

	var total int64
	err := GetOrmer().Raw(sql, params).QueryRow(&total)
	return total, err
}

// GetTotalOfUserRelevantRepositories ...
func GetTotalOfUserRelevantRepositories(userID int, name string) (int64, error) {
	params := []interface{}{}
	sql := `select count(*) 
		from repository r 
		join (
			select p.project_id, p.public 
				from project p
				join project_member pm
				on p.project_id = pm.project_id
				where pm.user_id = ?
		) as pp 
		on r.project_id = pp.project_id `
	params = append(params, userID)
	if len(name) != 0 {
		sql += ` where r.name like ?`
		params = append(params, "%"+name+"%")
	}

	var total int64
	err := GetOrmer().Raw(sql, params).QueryRow(&total)
	return total, err
}
