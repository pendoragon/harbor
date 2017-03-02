/*
Copyright 2017 caicloud authors. All rights reserved.
*/

package models

import (
	"time"
)

// Job holds the details of a Job.
type Job struct {
	JobID        int64     `orm:"pk;column(job_id)" json:"jobId"`
	Type         string    `orm:"column(type)" json:"type"`
	Message      string    `orm:"column(message)" json:"message"`
	CreationTime time.Time `orm:"column(creation_time)" json:"creationTime"`
}
