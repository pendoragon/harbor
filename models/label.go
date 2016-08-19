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

package models

import (
	"time"
)

// Label holds the details of a label.
type Label struct {
	LabelID         int64     `orm:"pk;column(label_id)" json:"label_id"`
	OwnerID         int       `orm:"column(owner_id)" json:"owner_id"`
	ProjectID       int64     `orm:"column(project_id)" json:"project_id"`
	ProjectName     string    `orm:"column(project_name)" json:"project_name"`
	Name            string    `orm:"column(name)" json:"name"`
	Remark          string    `orm:"column(remark)" json:"remark"`
	CreationTime    time.Time `orm:"column(creation_time)" json:"creation_time"`
	CreationTimeStr string    `json:"creation_time_str"`
	UpdateTime      time.Time `orm:"column(update_time)" json:"update_time"`
	Deleted         int       `orm:"column(deleted)" json:"deleted"`
}

// LabelHook holds the relationship between label and image.
type LabelHook struct {
	LabelHookID     int64     `orm:"pk;column(labelhook_id)" json:"labelhook_id"`
	LabelID         int64     `orm:"column(label_id)" json:"label_id"`
	RepoName        string    `orm:"column(repo_name)" json:"repo_name"`
	CreationTime    time.Time `orm:"column(creation_time)" json:"creation_time"`
	CreationTimeStr string    `json:"creation_time_str"`
	UpdateTime      time.Time `orm:"column(update_time)" json:"update_time"`
	Deleted         int       `orm:"column(deleted)" json:"deleted"`
}
