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

package main

import (
	"github.com/vmware/harbor/src/ui/api"
	"github.com/vmware/harbor/src/ui/controllers"
	"github.com/vmware/harbor/src/ui/service"
	"github.com/vmware/harbor/src/ui/service/token"

	"github.com/astaxie/beego"
)

func initRouters() {

	beego.SetStaticPath("static/resources", "static/resources")
	beego.SetStaticPath("static/vendors", "static/vendors")

	//Page Controllers:
	beego.Router("/", &controllers.IndexController{})
	beego.Router("/dashboard", &controllers.DashboardController{})
	beego.Router("/project", &controllers.ProjectController{})
	beego.Router("/repository", &controllers.RepositoryController{})
	beego.Router("/sign_up", &controllers.SignUpController{})
	beego.Router("/add_new", &controllers.AddNewController{})
	beego.Router("/account_setting", &controllers.AccountSettingController{})
	beego.Router("/change_password", &controllers.ChangePasswordController{})
	beego.Router("/admin_option", &controllers.AdminOptionController{})
	beego.Router("/forgot_password", &controllers.ForgotPasswordController{})
	beego.Router("/reset_password", &controllers.ResetPasswordController{})
	beego.Router("/search", &controllers.SearchController{})

	beego.Router("/login", &controllers.CommonController{}, "post:Login")
	beego.Router("/log_out", &controllers.CommonController{}, "get:LogOut")
	beego.Router("/reset", &controllers.CommonController{}, "post:ResetPassword")
	beego.Router("/userExists", &controllers.CommonController{}, "post:UserExists")
	beego.Router("/sendEmail", &controllers.CommonController{}, "get:SendEmail")
	beego.Router("/language", &controllers.CommonController{}, "get:SwitchLanguage")

	beego.Router("/optional_menu", &controllers.OptionalMenuController{})
	beego.Router("/navigation_header", &controllers.NavigationHeaderController{})
	beego.Router("/navigation_detail", &controllers.NavigationDetailController{})
	beego.Router("/sign_in", &controllers.SignInController{})

	//API:
	beego.Router("/api/search", &api.SearchAPI{})
	beego.Router("/api/projects/:pid([0-9]+)/members/?:mid", &api.ProjectMemberAPI{})
	beego.Router("/api/projects/", &api.ProjectAPI{}, "get:List;post:Post")
	beego.Router("/api/projects/:id", &api.ProjectAPI{})
	beego.Router("/api/projects/:id/publicity", &api.ProjectAPI{}, "put:ToggleProjectPublic")
	beego.Router("/api/statistics", &api.StatisticAPI{})
	beego.Router("/api/projects/:id([0-9]+)/logs/filter", &api.ProjectAPI{}, "post:FilterAccessLog")

	beego.Router("/api/labels/?:id", &api.LabelAPI{})
	beego.Router("/api/labels/list/?:pid", &api.LabelAPI{}, "get:List")
	beego.Router("/api/repos_by_labelnames", &api.LabelAPI{}, "post:ListReposByLabelNames")
	beego.Router("/api/labelhooks/?:id", &api.LabelHookAPI{})
	beego.Router("/api/labelhooks/list/?:lid", &api.LabelHookAPI{}, "get:List")
	beego.Router("/api/labelhooks/list/by_reponame", &api.LabelHookAPI{}, "get:ListLabelHooksByRepoName")
	beego.Router("/api/reporemarks", &api.RepoRemarkAPI{})
	beego.Router("/api/users/?:id", &api.UserAPI{})
	beego.Router("/api/users/:id([0-9]+)/password", &api.UserAPI{}, "put:ChangePassword")
	beego.Router("/api/internal/syncregistry", &api.InternalAPI{}, "post:SyncRegistry")
	beego.Router("/api/repositories", &api.RepositoryAPI{})
	beego.Router("/api/repositories/conditions", &api.RepositoryAPI{}, "post:GetRepositoryWithConditions")
	beego.Router("/api/repositories/tags", &api.RepositoryAPI{}, "get:GetTags")
	beego.Router("/api/repositories/manifests", &api.RepositoryAPI{}, "get:GetManifests")
	beego.Router("/api/repositories/vulnerabilities", &api.RepositoryAPI{}, "get:GetVulnerabilities")
	beego.Router("/api/repositories/unmarked", &api.RepositoryAPI{}, "post:GetUnmarkedRepos")
	beego.Router("/api/repositories/list", &api.RepositoryAPI{}, "get:List")
	beego.Router("/api/repositories/analysis", &api.RepoAnalysisAPI{})
	beego.Router("/api/jobs/replication/", &api.RepJobAPI{}, "get:List")
	beego.Router("/api/jobs/replication/:id([0-9]+)", &api.RepJobAPI{})
	beego.Router("/api/jobs/replication/:id([0-9]+)/log", &api.RepJobAPI{}, "get:GetLog")
	beego.Router("/api/policies/replication/:id([0-9]+)", &api.RepPolicyAPI{})
	beego.Router("/api/policies/replication", &api.RepPolicyAPI{}, "get:List")
	beego.Router("/api/policies/replication", &api.RepPolicyAPI{}, "post:Post")
	beego.Router("/api/policies/replication/:id([0-9]+)/enablement", &api.RepPolicyAPI{}, "put:UpdateEnablement")
	beego.Router("/api/targets/", &api.TargetAPI{}, "get:List")
	beego.Router("/api/targets/", &api.TargetAPI{}, "post:Post")
	beego.Router("/api/targets/:id([0-9]+)", &api.TargetAPI{})
	beego.Router("/api/targets/:id([0-9]+)/policies/", &api.TargetAPI{}, "get:ListPolicies")
	beego.Router("/api/targets/ping", &api.TargetAPI{}, "post:Ping")
	beego.Router("/api/users/:id/sysadmin", &api.UserAPI{}, "put:ToggleUserAdminRole")
	beego.Router("/api/repositories/top", &api.RepositoryAPI{}, "get:GetTopRepos")
	beego.Router("/api/logs", &api.LogAPI{})
	//external service that hosted on harbor process:
	beego.Router("/service/notifications", &service.NotificationHandler{})
	beego.Router("/service/token", &token.Handler{})
}

func initV1Routers() {
	// projects
	beego.Router("/api/v1/projects", &api.ProjectAPI{}, "get:ListV1;post:Post")
	beego.Router("/api/v1/projects/:id", &api.ProjectAPI{}, "get:GetV1;put:Put;delete:Delete")

	// labels
	beego.Router("/api/v1/projects/:pid/labels", &api.LabelAPIV1{}, "get:List;post:Post")
	beego.Router("/api/v1/projects/:pid/labels/:lid", &api.LabelAPIV1{})

	// repos
	beego.Router("/api/v1/repos", &api.RepositoryAPIV1{}, "get:List;post:UploadImages")
	beego.Router("/api/v1/repos/:rid", &api.RepositoryAPIV1{}, "get:Get;delete:Delete")
	beego.Router("/api/v1/repos/:rid/tags", &api.RepositoryAPIV1{}, "get:GetTags")
	beego.Router("/api/v1/repos/:rid/tags/:tag", &api.RepositoryAPIV1{}, "get:GetManifests;delete:Delete")

	// jobs
	beego.Router("/api/v1/jobs", &api.JobAPIV1{}, "post:Post")
	beego.Router("/api/v1/jobs/:jid", &api.JobAPIV1{}, "get:GetJob")
}
