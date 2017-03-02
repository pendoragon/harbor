/*
Copyright 2017 caicloud authors. All rights reserved.
*/

package dao

import (
	"time"

	"github.com/vmware/harbor/src/common/models"
	"github.com/vmware/harbor/src/common/utils/log"
)

// JobIdExists returns whether the job exists according to its id.
func JobIdExists(jobId int64) bool {
	o := GetOrmer()
	return o.QueryTable("job").Filter("job_id", jobId).Exist()
}

// CreateJob insert a job to the database.
func CreateJob(job models.Job) (int64, error) {

	o := GetOrmer()
	p, err := o.Raw("insert into job (type, message, creation_time) values (?, ?, ?)").Prepare()
	if err != nil {
		return 0, err
	}

	now := time.Now()
	r, err := p.Exec(job.Type, job.Message, now)
	if err != nil {
		return 0, err
	}

	jobID, err := r.LastInsertId()
	if err != nil {
		return 0, err
	}

	return jobID, err
}

// UpdateJobStatus ...
func UpdateJobStatusById(jobId int64, message string) error {
	o := GetOrmer()

	sql := `update job set message = ? where job_id = ?`
	params := make([]interface{}, 2)
	params = append(params, message)
	params = append(params, jobId)

	if _, err := o.Raw(sql, params).Exec(); err != nil {
		log.Errorf("Failed to update job message, error: %v", err)
		return err
	}

	return nil
}

// GetJobById ...
func GetJobById(jobId int64) (*models.Job, error) {
	o := GetOrmer()

	sql := `select j.job_id, j.type, j.message, j.creation_time
            from job j where j.job_id = ?`
	params := make([]interface{}, 1)
	params = append(params, jobId)

	j := []models.Job{}
	count, err := o.Raw(sql, params).QueryRows(&j)

	if err != nil {
		return nil, err
	}

	if count == 0 {
		return nil, nil
	}

	return &j[0], nil
}
