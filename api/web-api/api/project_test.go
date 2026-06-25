//go:build integration && cgo
// +build integration,cgo

package web_api

import (
	"context"
	"errors"
	"testing"

	web_config "github.com/rapidaai/api/web-api/config"
	internal_entity "github.com/rapidaai/api/web-api/internal/entity"
	internal_project_service "github.com/rapidaai/api/web-api/internal/service/project"
	internal_user_service "github.com/rapidaai/api/web-api/internal/service/user"
	app_config "github.com/rapidaai/config"
	external_clients "github.com/rapidaai/pkg/clients/external"
	external_emailer_template "github.com/rapidaai/pkg/clients/external/emailer/template"
	"github.com/rapidaai/pkg/commons"
	"github.com/rapidaai/pkg/connectors"
	pkg_errors "github.com/rapidaai/pkg/errors"
	gorm_models "github.com/rapidaai/pkg/models/gorm"
	"github.com/rapidaai/pkg/types"
	type_enums "github.com/rapidaai/pkg/types/enums"
	"github.com/rapidaai/protos"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type testPostgresConnector struct {
	db *gorm.DB
}

func (t *testPostgresConnector) Connect(ctx context.Context) error {
	return nil
}

func (t *testPostgresConnector) Name() string {
	return "test-postgres"
}

func (t *testPostgresConnector) IsConnected(ctx context.Context) bool {
	return true
}

func (t *testPostgresConnector) Disconnect(ctx context.Context) error {
	return nil
}

func (t *testPostgresConnector) Query(ctx context.Context, qry string, dest interface{}) error {
	return t.DB(ctx).Raw(qry).Scan(dest).Error
}

func (t *testPostgresConnector) DB(ctx context.Context) *gorm.DB {
	if tx, ok := connectors.PostgresTxFromContext(ctx); ok {
		return tx.WithContext(ctx)
	}
	return t.db.WithContext(ctx)
}

type testEmailer struct {
	err   error
	calls int
	to    external_clients.Contact
	args  map[string]string
}

func (t *testEmailer) EmailText(ctx context.Context, to external_clients.Contact, subject string, content string) error {
	return nil
}

func (t *testEmailer) EmailRichText(ctx context.Context, to external_clients.Contact, subject string, template external_emailer_template.TemplateName, args map[string]string) error {
	t.calls++
	t.to = to
	t.args = args
	return t.err
}

func (t *testEmailer) EmailTemplate(ctx context.Context, to external_clients.Contact, subject string, templateId string, args map[string]string) error {
	return nil
}

func newProjectAPITest(t *testing.T) (*webProjectGRPCApi, *gorm.DB, *testEmailer) {
	t.Helper()
	logger, err := commons.NewApplicationLogger()
	require.NoError(t, err)
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{Logger: logger2Discard()})
	require.NoError(t, err)
	require.NoError(t, db.Exec(`CREATE TABLE organizations (id integer primary key, created_date datetime, updated_date datetime, status text, created_by integer, updated_by integer, name text, description text, size text, industry text, contact text)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE projects (id integer primary key, created_date datetime, updated_date datetime, status text, created_by integer, updated_by integer, organization_id integer, name text, description text)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE user_auths (id integer primary key, created_date datetime, updated_date datetime, status text, created_by integer, updated_by integer, name text, email text, password text, source text)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE user_auth_tokens (id integer primary key, created_date datetime, updated_date datetime, status text, created_by integer, updated_by integer, user_auth_id integer, token_type text, token text, expire_at datetime)`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE user_organization_roles (id integer primary key, created_date datetime, updated_date datetime, status text, created_by integer, updated_by integer, user_auth_id integer, organization_id integer, role text, UNIQUE (user_auth_id, organization_id, status))`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE user_project_roles (id integer primary key, created_date datetime, updated_date datetime, status text, created_by integer, updated_by integer, user_auth_id integer, project_id integer, role text, UNIQUE (user_auth_id, project_id, status))`).Error)
	require.NoError(t, db.Create(&internal_entity.Organization{
		Audited: gorm_models.Audited{Id: 10},
		Mutable: gorm_models.Mutable{
			Status:    type_enums.RECORD_ACTIVE,
			CreatedBy: 1,
		},
		Name:        "Acme",
		Description: "Acme org",
		Industry:    "software",
		Contact:     "admin@example.com",
	}).Error)
	require.NoError(t, db.Create(&internal_entity.Organization{
		Audited: gorm_models.Audited{Id: 20},
		Mutable: gorm_models.Mutable{
			Status:    type_enums.RECORD_ACTIVE,
			CreatedBy: 1,
		},
		Name:        "Other",
		Description: "Other org",
		Industry:    "software",
		Contact:     "admin@example.com",
	}).Error)
	require.NoError(t, db.Create(&internal_entity.Project{
		Audited: gorm_models.Audited{Id: 100},
		Mutable: gorm_models.Mutable{
			Status:    type_enums.RECORD_ACTIVE,
			CreatedBy: 1,
		},
		OrganizationId: 10,
		Name:           "Alpha",
		Description:    "Alpha project",
	}).Error)
	require.NoError(t, db.Create(&internal_entity.Project{
		Audited: gorm_models.Audited{Id: 101},
		Mutable: gorm_models.Mutable{
			Status:    type_enums.RECORD_ACTIVE,
			CreatedBy: 1,
		},
		OrganizationId: 10,
		Name:           "Beta",
		Description:    "Beta project",
	}).Error)
	require.NoError(t, db.Create(&internal_entity.Project{
		Audited: gorm_models.Audited{Id: 200},
		Mutable: gorm_models.Mutable{
			Status:    type_enums.RECORD_ACTIVE,
			CreatedBy: 1,
		},
		OrganizationId: 20,
		Name:           "Cross",
		Description:    "Cross project",
	}).Error)
	require.NoError(t, db.Create(&internal_entity.Project{
		Audited: gorm_models.Audited{Id: 300},
		Mutable: gorm_models.Mutable{
			Status:    type_enums.RECORD_ARCHIEVE,
			CreatedBy: 1,
		},
		OrganizationId: 10,
		Name:           "Archived",
		Description:    "Archived project",
	}).Error)

	emailer := &testEmailer{}
	postgres := &testPostgresConnector{db: db}
	return &webProjectGRPCApi{
		webProjectApi: webProjectApi{
			cfg: &web_config.WebAppConfig{
				AppConfig: app_config.AppConfig{
					Ui: app_config.ServiceHostConfig{Host: "http://ui.test"},
				},
			},
			logger:         logger,
			postgres:       postgres,
			projectService: internal_project_service.NewProjectService(logger, postgres),
			userService:    internal_user_service.NewUserService(logger, postgres),
			emailerClient:  emailer,
		},
	}, db, emailer
}

func newOrganizationAPITest(t *testing.T) (*webOrganizationGRPCApi, *gorm.DB, *testEmailer) {
	t.Helper()
	projectApi, db, emailer := newProjectAPITest(t)
	return &webOrganizationGRPCApi{
		webOrganizationApi: webOrganizationApi{
			cfg:            projectApi.cfg,
			logger:         projectApi.logger,
			postgres:       projectApi.postgres,
			projectService: projectApi.projectService,
			userService:    projectApi.userService,
			emailerClient:  emailer,
		},
	}, db, emailer
}

func logger2Discard() logger.Interface {
	return logger.Discard.LogMode(logger.Silent)
}

func ownerContext(role string) context.Context {
	return context.WithValue(context.Background(), types.CTX_, &types.PlainAuthPrinciple{
		User: types.UserInfo{
			Id:    1,
			Name:  "Owner",
			Email: "owner@example.com",
		},
		OrganizationRole: &types.OrganizaitonRole{
			Id:               1,
			OrganizationId:   10,
			Role:             role,
			OrganizationName: "Acme",
		},
	})
}

func TestGetAllUserReturnsActiveAndInvitedOrganizationMembers(t *testing.T) {
	projectApi, db, _ := newProjectAPITest(t)
	api := &webAuthGRPCApi{
		webAuthApi: webAuthApi{
			logger:      projectApi.logger,
			postgres:    projectApi.postgres,
			userService: projectApi.userService,
		},
	}
	require.NoError(t, db.Create(&internal_entity.UserAuth{
		Audited: gorm_models.Audited{Id: 80},
		Mutable: gorm_models.Mutable{
			Status:    type_enums.RECORD_ACTIVE,
			CreatedBy: 1,
		},
		Name:     "Active",
		Email:    "active@example.com",
		Password: "hash",
		Source:   "direct",
	}).Error)
	require.NoError(t, db.Create(&internal_entity.UserAuth{
		Audited: gorm_models.Audited{Id: 81},
		Mutable: gorm_models.Mutable{
			Status:    type_enums.RECORD_INVITED,
			CreatedBy: 1,
		},
		Name:     "Invited",
		Email:    "invited@example.com",
		Password: "hash",
		Source:   "invited-by-other",
	}).Error)
	require.NoError(t, db.Create(&internal_entity.UserAuth{
		Audited: gorm_models.Audited{Id: 82},
		Mutable: gorm_models.Mutable{
			Status:    type_enums.RECORD_ACTIVE,
			CreatedBy: 1,
		},
		Name:     "Other",
		Email:    "other@example.com",
		Password: "hash",
		Source:   "direct",
	}).Error)
	require.NoError(t, db.Create(&internal_entity.UserOrganizationRole{
		Audited: gorm_models.Audited{Id: 83},
		Mutable: gorm_models.Mutable{
			Status:    type_enums.RECORD_ACTIVE,
			CreatedBy: 1,
		},
		UserAuthId:     80,
		OrganizationId: 10,
		Role:           type_enums.ORGANIZATION_ROLE_MEMBER.String(),
	}).Error)
	require.NoError(t, db.Create(&internal_entity.UserOrganizationRole{
		Audited: gorm_models.Audited{Id: 84},
		Mutable: gorm_models.Mutable{
			Status:    type_enums.RECORD_INVITED,
			CreatedBy: 1,
		},
		UserAuthId:     81,
		OrganizationId: 10,
		Role:           type_enums.ORGANIZATION_ROLE_MEMBER.String(),
	}).Error)
	require.NoError(t, db.Create(&internal_entity.UserOrganizationRole{
		Audited: gorm_models.Audited{Id: 85},
		Mutable: gorm_models.Mutable{
			Status:    type_enums.RECORD_ACTIVE,
			CreatedBy: 1,
		},
		UserAuthId:     82,
		OrganizationId: 20,
		Role:           type_enums.ORGANIZATION_ROLE_MEMBER.String(),
	}).Error)

	res, err := api.GetAllUser(ownerContext(type_enums.ORGANIZATION_ROLE_ADMIN.String()), &protos.GetAllUserRequest{
		Paginate: &protos.Paginate{Page: 1, PageSize: 10},
	})
	require.NoError(t, err)
	require.True(t, res.GetSuccess())
	require.EqualValues(t, 2, res.GetPaginated().GetTotalItem())

	members := map[string]string{}
	for _, member := range res.GetData() {
		members[member.GetEmail()] = member.GetStatus()
	}
	require.Equal(t, map[string]string{
		"active@example.com":  type_enums.RECORD_ACTIVE.String(),
		"invited@example.com": type_enums.RECORD_INVITED.String(),
	}, members)
}

func TestInviteUserToOrganizationRejectsAuthAndValidationFailures(t *testing.T) {
	api, db, _ := newOrganizationAPITest(t)

	res, err := api.InviteUserToOrganization(context.Background(), &protos.InviteUserToOrganizationRequest{
		Email:            "new@example.com",
		OrganizationRole: type_enums.ORGANIZATION_ROLE_MEMBER.String(),
		ProjectRoles: []*protos.ProjectRoleAssignment{
			{ProjectId: 100, ProjectRole: type_enums.PROJECT_ROLE_READER.String()},
		},
	})
	require.Error(t, err)
	require.Equal(t, pkg_errors.InviteUserToOrganizationUnauthenticated.ErrorMessage, err.Error())
	require.False(t, res.GetSuccess())
	require.Equal(t, pkg_errors.InviteUserToOrganizationUnauthenticated.HTTPStatusCodeInt32(), res.GetCode())
	require.EqualValues(t, pkg_errors.InviteUserToOrganizationUnauthenticated.Code, res.GetError().GetErrorCode())
	require.Equal(t, pkg_errors.InviteUserToOrganizationUnauthenticated.Error, res.GetError().GetErrorMessage())
	require.Equal(t, pkg_errors.InviteUserToOrganizationUnauthenticated.ErrorMessage, res.GetError().GetHumanMessage())

	tests := []struct {
		name          string
		ctx           context.Context
		req           *protos.InviteUserToOrganizationRequest
		platformError pkg_errors.PlatformError
		expectError   bool
	}{
		{
			name:          "non admin",
			ctx:           ownerContext(type_enums.ORGANIZATION_ROLE_MEMBER.String()),
			platformError: pkg_errors.InviteUserToOrganizationUnauthorized,
			expectError:   true,
			req: &protos.InviteUserToOrganizationRequest{
				Email:            "new@example.com",
				OrganizationRole: type_enums.ORGANIZATION_ROLE_MEMBER.String(),
				ProjectRoles: []*protos.ProjectRoleAssignment{
					{ProjectId: 100, ProjectRole: type_enums.PROJECT_ROLE_READER.String()},
				},
			},
		},
		{
			name:          "invalid raw email",
			ctx:           ownerContext(type_enums.ORGANIZATION_ROLE_OWNER.String()),
			platformError: pkg_errors.InviteUserToOrganizationInvalidEmail,
			req: &protos.InviteUserToOrganizationRequest{
				Email:            " new@example.com",
				OrganizationRole: type_enums.ORGANIZATION_ROLE_MEMBER.String(),
				ProjectRoles: []*protos.ProjectRoleAssignment{
					{ProjectId: 100, ProjectRole: type_enums.PROJECT_ROLE_READER.String()},
				},
			},
		},
		{
			name:          "invalid organization role",
			ctx:           ownerContext(type_enums.ORGANIZATION_ROLE_OWNER.String()),
			platformError: pkg_errors.InviteUserToOrganizationInvalidOrganizationRole,
			req: &protos.InviteUserToOrganizationRequest{
				Email:            "new@example.com",
				OrganizationRole: "Member",
				ProjectRoles: []*protos.ProjectRoleAssignment{
					{ProjectId: 100, ProjectRole: type_enums.PROJECT_ROLE_READER.String()},
				},
			},
		},
		{
			name:          "invalid project role",
			ctx:           ownerContext(type_enums.ORGANIZATION_ROLE_OWNER.String()),
			platformError: pkg_errors.InviteUserToOrganizationInvalidProjectRole,
			req: &protos.InviteUserToOrganizationRequest{
				Email:            "new@example.com",
				OrganizationRole: type_enums.ORGANIZATION_ROLE_MEMBER.String(),
				ProjectRoles: []*protos.ProjectRoleAssignment{
					{ProjectId: 100, ProjectRole: "Reader"},
				},
			},
		},
		{
			name:          "empty project id",
			ctx:           ownerContext(type_enums.ORGANIZATION_ROLE_OWNER.String()),
			platformError: pkg_errors.InviteUserToOrganizationInvalidProjects,
			req: &protos.InviteUserToOrganizationRequest{
				Email:            "new@example.com",
				OrganizationRole: type_enums.ORGANIZATION_ROLE_MEMBER.String(),
				ProjectRoles: []*protos.ProjectRoleAssignment{
					{ProjectId: 0, ProjectRole: type_enums.PROJECT_ROLE_READER.String()},
				},
			},
		},
		{
			name:          "duplicate project",
			ctx:           ownerContext(type_enums.ORGANIZATION_ROLE_OWNER.String()),
			platformError: pkg_errors.InviteUserToOrganizationDuplicateProject,
			req: &protos.InviteUserToOrganizationRequest{
				Email:            "new@example.com",
				OrganizationRole: type_enums.ORGANIZATION_ROLE_MEMBER.String(),
				ProjectRoles: []*protos.ProjectRoleAssignment{
					{ProjectId: 100, ProjectRole: type_enums.PROJECT_ROLE_READER.String()},
					{ProjectId: 100, ProjectRole: type_enums.PROJECT_ROLE_WRITER.String()},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := api.InviteUserToOrganization(tt.ctx, tt.req)
			if tt.expectError {
				require.Error(t, err)
				require.Equal(t, tt.platformError.ErrorMessage, err.Error())
			} else {
				require.NoError(t, err)
			}
			require.False(t, res.GetSuccess())
			require.Equal(t, tt.platformError.HTTPStatusCodeInt32(), res.GetCode())
			require.EqualValues(t, tt.platformError.Code, res.GetError().GetErrorCode())
			require.Equal(t, tt.platformError.Error, res.GetError().GetErrorMessage())
			require.Equal(t, tt.platformError.ErrorMessage, res.GetError().GetHumanMessage())
			var count int64
			require.NoError(t, db.Model(&internal_entity.UserProjectRole{}).Count(&count).Error)
			require.Zero(t, count)
		})
	}
}

func TestInviteUserToOrganizationRejectsInvalidProjectsBeforeWrites(t *testing.T) {
	api, db, _ := newOrganizationAPITest(t)

	for _, ids := range [][]uint64{{200}, {300}, {999}, {100, 200}} {
		projectRoles := []*protos.ProjectRoleAssignment{}
		for _, id := range ids {
			projectRoles = append(projectRoles, &protos.ProjectRoleAssignment{
				ProjectId:   id,
				ProjectRole: type_enums.PROJECT_ROLE_WRITER.String(),
			})
		}
		res, err := api.InviteUserToOrganization(ownerContext(type_enums.ORGANIZATION_ROLE_ADMIN.String()), &protos.InviteUserToOrganizationRequest{
			Email:            "new@example.com",
			OrganizationRole: type_enums.ORGANIZATION_ROLE_MEMBER.String(),
			ProjectRoles:     projectRoles,
		})
		require.NoError(t, err)
		require.False(t, res.GetSuccess())
		require.Equal(t, pkg_errors.InviteUserToOrganizationInvalidProjects.HTTPStatusCodeInt32(), res.GetCode())
		require.EqualValues(t, pkg_errors.InviteUserToOrganizationInvalidProjects.Code, res.GetError().GetErrorCode())
		require.Equal(t, pkg_errors.InviteUserToOrganizationInvalidProjects.Error, res.GetError().GetErrorMessage())
		require.Equal(t, pkg_errors.InviteUserToOrganizationInvalidProjects.ErrorMessage, res.GetError().GetHumanMessage())
		var count int64
		require.NoError(t, db.Model(&internal_entity.UserProjectRole{}).Count(&count).Error)
		require.Zero(t, count)
	}
}

func TestInviteUserToOrganizationInvitesNewUserWithOrganizationAndProjectRoles(t *testing.T) {
	api, db, emailer := newOrganizationAPITest(t)

	res, err := api.InviteUserToOrganization(ownerContext(type_enums.ORGANIZATION_ROLE_OWNER.String()), &protos.InviteUserToOrganizationRequest{
		Email:            "new@example.com",
		OrganizationRole: type_enums.ORGANIZATION_ROLE_ADMIN.String(),
		ProjectRoles: []*protos.ProjectRoleAssignment{
			{ProjectId: 100, ProjectRole: type_enums.PROJECT_ROLE_WRITER.String()},
			{ProjectId: 101, ProjectRole: type_enums.PROJECT_ROLE_READER.String()},
		},
	})
	require.NoError(t, err)
	require.True(t, res.GetSuccess())
	require.Equal(t, "new@example.com", res.GetData().GetEmail())
	require.Equal(t, type_enums.ORGANIZATION_ROLE_ADMIN.String(), res.GetData().GetRole())
	require.Equal(t, 1, emailer.calls)
	require.Equal(t, "new@example.com", emailer.to.Email)
	require.Equal(t, "Alpha,Beta", emailer.args["project_name"])

	var user internal_entity.UserAuth
	require.NoError(t, db.First(&user, "email = ?", "new@example.com").Error)
	var orgRole internal_entity.UserOrganizationRole
	require.NoError(t, db.First(&orgRole, "user_auth_id = ?", user.Id).Error)
	require.Equal(t, type_enums.ORGANIZATION_ROLE_ADMIN.String(), orgRole.Role)
	require.Equal(t, type_enums.RECORD_INVITED, orgRole.Status)
	var projectRoles []internal_entity.UserProjectRole
	require.NoError(t, db.Find(&projectRoles, "user_auth_id = ?", user.Id).Error)
	require.Len(t, projectRoles, 2)
	roles := map[uint64]string{}
	for _, projectRole := range projectRoles {
		roles[projectRole.ProjectId] = projectRole.Role
		require.Equal(t, type_enums.RECORD_INVITED, projectRole.Status)
	}
	require.Equal(t, map[uint64]string{
		100: type_enums.PROJECT_ROLE_WRITER.String(),
		101: type_enums.PROJECT_ROLE_READER.String(),
	}, roles)
}

func TestInviteUserToOrganizationRejectsExistingSameOrgUser(t *testing.T) {
	api, db, _ := newOrganizationAPITest(t)
	require.NoError(t, db.Create(&internal_entity.UserAuth{
		Audited: gorm_models.Audited{Id: 50},
		Mutable: gorm_models.Mutable{
			Status:    type_enums.RECORD_ACTIVE,
			CreatedBy: 1,
		},
		Name:     "Existing",
		Email:    "existing@example.com",
		Password: "hash",
		Source:   "direct",
	}).Error)
	require.NoError(t, db.Create(&internal_entity.UserOrganizationRole{
		Audited: gorm_models.Audited{Id: 51},
		Mutable: gorm_models.Mutable{
			Status:    type_enums.RECORD_ACTIVE,
			CreatedBy: 1,
		},
		UserAuthId:     50,
		OrganizationId: 10,
		Role:           type_enums.ORGANIZATION_ROLE_MEMBER.String(),
	}).Error)
	require.NoError(t, db.Create(&internal_entity.UserProjectRole{
		Audited: gorm_models.Audited{Id: 52},
		Mutable: gorm_models.Mutable{
			Status:    type_enums.RECORD_ACTIVE,
			CreatedBy: 1,
		},
		UserAuthId: 50,
		ProjectId:  100,
		Role:       type_enums.PROJECT_ROLE_READER.String(),
	}).Error)

	res, err := api.InviteUserToOrganization(ownerContext(type_enums.ORGANIZATION_ROLE_ADMIN.String()), &protos.InviteUserToOrganizationRequest{
		Email:            "existing@example.com",
		OrganizationRole: type_enums.ORGANIZATION_ROLE_ADMIN.String(),
		ProjectRoles: []*protos.ProjectRoleAssignment{
			{ProjectId: 100, ProjectRole: type_enums.PROJECT_ROLE_ADMIN.String()},
		},
	})
	require.NoError(t, err)
	require.False(t, res.GetSuccess())
	require.Equal(t, pkg_errors.InviteUserToOrganizationUserAlreadyInOrganization.HTTPStatusCodeInt32(), res.GetCode())
	require.EqualValues(t, pkg_errors.InviteUserToOrganizationUserAlreadyInOrganization.Code, res.GetError().GetErrorCode())
	require.Equal(t, pkg_errors.InviteUserToOrganizationUserAlreadyInOrganization.Error, res.GetError().GetErrorMessage())
	require.Equal(t, pkg_errors.InviteUserToOrganizationUserAlreadyInOrganization.ErrorMessage, res.GetError().GetHumanMessage())

	var orgRoles []internal_entity.UserOrganizationRole
	require.NoError(t, db.Find(&orgRoles, "user_auth_id = ? AND organization_id = ?", 50, 10).Error)
	require.Len(t, orgRoles, 1)
	require.Equal(t, type_enums.ORGANIZATION_ROLE_MEMBER.String(), orgRoles[0].Role)

	var projectRoles []internal_entity.UserProjectRole
	require.NoError(t, db.Find(&projectRoles, "user_auth_id = ? AND project_id = ?", 50, 100).Error)
	require.Len(t, projectRoles, 1)
	require.Equal(t, type_enums.PROJECT_ROLE_READER.String(), projectRoles[0].Role)
	require.Equal(t, type_enums.RECORD_ACTIVE, projectRoles[0].Status)
}

func TestInviteUserToOrganizationRejectsExistingInvitedSameOrgUser(t *testing.T) {
	api, db, _ := newOrganizationAPITest(t)
	require.NoError(t, db.Create(&internal_entity.UserAuth{
		Audited: gorm_models.Audited{Id: 60},
		Mutable: gorm_models.Mutable{
			Status:    type_enums.RECORD_INVITED,
			CreatedBy: 1,
		},
		Name:     "Invited",
		Email:    "invited@example.com",
		Password: "hash",
		Source:   "invited-by-other",
	}).Error)
	require.NoError(t, db.Create(&internal_entity.UserOrganizationRole{
		Audited: gorm_models.Audited{Id: 61},
		Mutable: gorm_models.Mutable{
			Status:    type_enums.RECORD_INVITED,
			CreatedBy: 1,
		},
		UserAuthId:     60,
		OrganizationId: 10,
		Role:           type_enums.ORGANIZATION_ROLE_MEMBER.String(),
	}).Error)

	res, err := api.InviteUserToOrganization(ownerContext(type_enums.ORGANIZATION_ROLE_OWNER.String()), &protos.InviteUserToOrganizationRequest{
		Email:            "invited@example.com",
		OrganizationRole: type_enums.ORGANIZATION_ROLE_MEMBER.String(),
		ProjectRoles: []*protos.ProjectRoleAssignment{
			{ProjectId: 100, ProjectRole: type_enums.PROJECT_ROLE_READER.String()},
		},
	})
	require.NoError(t, err)
	require.False(t, res.GetSuccess())
	require.Equal(t, pkg_errors.InviteUserToOrganizationUserAlreadyInOrganization.HTTPStatusCodeInt32(), res.GetCode())
	require.EqualValues(t, pkg_errors.InviteUserToOrganizationUserAlreadyInOrganization.Code, res.GetError().GetErrorCode())
	require.Equal(t, pkg_errors.InviteUserToOrganizationUserAlreadyInOrganization.Error, res.GetError().GetErrorMessage())
	require.Equal(t, pkg_errors.InviteUserToOrganizationUserAlreadyInOrganization.ErrorMessage, res.GetError().GetHumanMessage())

	var orgRoleCount int64
	require.NoError(t, db.Model(&internal_entity.UserOrganizationRole{}).Where("user_auth_id = ?", 60).Count(&orgRoleCount).Error)
	require.EqualValues(t, 1, orgRoleCount)

	var projectRoleCount int64
	require.NoError(t, db.Model(&internal_entity.UserProjectRole{}).Where("user_auth_id = ?", 60).Count(&projectRoleCount).Error)
	require.Zero(t, projectRoleCount)
}

func TestInviteUserToOrganizationRejectsExistingInvitedUserInAnotherOrganization(t *testing.T) {
	api, db, emailer := newOrganizationAPITest(t)
	require.NoError(t, db.Create(&internal_entity.UserAuth{
		Audited: gorm_models.Audited{Id: 70},
		Mutable: gorm_models.Mutable{
			Status:    type_enums.RECORD_INVITED,
			CreatedBy: 1,
		},
		Name:     "Invited",
		Email:    "cross-invited@example.com",
		Password: "hash",
		Source:   "invited-by-other",
	}).Error)
	require.NoError(t, db.Create(&internal_entity.UserOrganizationRole{
		Audited: gorm_models.Audited{Id: 71},
		Mutable: gorm_models.Mutable{
			Status:    type_enums.RECORD_INVITED,
			CreatedBy: 1,
		},
		UserAuthId:     70,
		OrganizationId: 20,
		Role:           type_enums.ORGANIZATION_ROLE_MEMBER.String(),
	}).Error)

	res, err := api.InviteUserToOrganization(ownerContext(type_enums.ORGANIZATION_ROLE_OWNER.String()), &protos.InviteUserToOrganizationRequest{
		Email:            "cross-invited@example.com",
		OrganizationRole: type_enums.ORGANIZATION_ROLE_MEMBER.String(),
		ProjectRoles: []*protos.ProjectRoleAssignment{
			{ProjectId: 100, ProjectRole: type_enums.PROJECT_ROLE_READER.String()},
		},
	})
	require.NoError(t, err)
	require.False(t, res.GetSuccess())
	require.Equal(t, pkg_errors.InviteUserToOrganizationUserInAnotherOrganization.HTTPStatusCodeInt32(), res.GetCode())
	require.EqualValues(t, pkg_errors.InviteUserToOrganizationUserInAnotherOrganization.Code, res.GetError().GetErrorCode())
	require.Equal(t, pkg_errors.InviteUserToOrganizationUserInAnotherOrganization.Error, res.GetError().GetErrorMessage())
	require.Equal(t, pkg_errors.InviteUserToOrganizationUserInAnotherOrganization.ErrorMessage, res.GetError().GetHumanMessage())
	require.Zero(t, emailer.calls)

	var projectRoleCount int64
	require.NoError(t, db.Model(&internal_entity.UserProjectRole{}).Where("user_auth_id = ?", 70).Count(&projectRoleCount).Error)
	require.Zero(t, projectRoleCount)
}

func TestInviteUserToOrganizationRestoresArchivedUserAsActive(t *testing.T) {
	api, db, emailer := newOrganizationAPITest(t)
	require.NoError(t, db.Create(&internal_entity.UserAuth{
		Audited: gorm_models.Audited{Id: 80},
		Mutable: gorm_models.Mutable{
			Status:    type_enums.RECORD_ARCHIEVE,
			CreatedBy: 1,
		},
		Name:     "Archived",
		Email:    "archived@example.com",
		Password: "hash",
		Source:   "direct",
	}).Error)
	require.NoError(t, db.Create(&internal_entity.UserOrganizationRole{
		Audited: gorm_models.Audited{Id: 81},
		Mutable: gorm_models.Mutable{
			Status:    type_enums.RECORD_ARCHIEVE,
			CreatedBy: 1,
		},
		UserAuthId:     80,
		OrganizationId: 10,
		Role:           type_enums.ORGANIZATION_ROLE_MEMBER.String(),
	}).Error)
	require.NoError(t, db.Create(&internal_entity.UserProjectRole{
		Audited: gorm_models.Audited{Id: 82},
		Mutable: gorm_models.Mutable{
			Status:    type_enums.RECORD_ARCHIEVE,
			CreatedBy: 1,
		},
		UserAuthId: 80,
		ProjectId:  100,
		Role:       type_enums.PROJECT_ROLE_READER.String(),
	}).Error)

	res, err := api.InviteUserToOrganization(ownerContext(type_enums.ORGANIZATION_ROLE_OWNER.String()), &protos.InviteUserToOrganizationRequest{
		Email:            "archived@example.com",
		OrganizationRole: type_enums.ORGANIZATION_ROLE_ADMIN.String(),
		ProjectRoles: []*protos.ProjectRoleAssignment{
			{ProjectId: 100, ProjectRole: type_enums.PROJECT_ROLE_WRITER.String()},
			{ProjectId: 101, ProjectRole: type_enums.PROJECT_ROLE_READER.String()},
		},
	})
	require.NoError(t, err)
	require.True(t, res.GetSuccess())
	require.Equal(t, type_enums.RECORD_ACTIVE.String(), res.GetData().GetStatus())
	require.Equal(t, 1, emailer.calls)
	require.Contains(t, emailer.args["invite_url"], "/auth/signin")

	var user internal_entity.UserAuth
	require.NoError(t, db.First(&user, "id = ?", 80).Error)
	require.Equal(t, type_enums.RECORD_ACTIVE, user.Status)

	var orgRoles []internal_entity.UserOrganizationRole
	require.NoError(t, db.Order("id ASC").Find(&orgRoles, "user_auth_id = ? AND organization_id = ?", 80, 10).Error)
	require.Len(t, orgRoles, 2)
	require.Equal(t, type_enums.RECORD_ARCHIEVE, orgRoles[0].Status)
	require.Equal(t, type_enums.RECORD_ACTIVE, orgRoles[1].Status)
	require.Equal(t, type_enums.ORGANIZATION_ROLE_ADMIN.String(), orgRoles[1].Role)

	var projectRoles []internal_entity.UserProjectRole
	require.NoError(t, db.Find(&projectRoles, "user_auth_id = ? AND status = ?", 80, type_enums.RECORD_ACTIVE.String()).Error)
	require.Len(t, projectRoles, 2)
	activeRoles := map[uint64]string{}
	for _, projectRole := range projectRoles {
		activeRoles[projectRole.ProjectId] = projectRole.Role
	}
	require.Equal(t, map[uint64]string{
		100: type_enums.PROJECT_ROLE_WRITER.String(),
		101: type_enums.PROJECT_ROLE_READER.String(),
	}, activeRoles)
}

func TestInviteUserToOrganizationProjectRoleFailureReturnsError(t *testing.T) {
	api, db, _ := newOrganizationAPITest(t)
	require.NoError(t, db.Exec(`DROP TABLE user_project_roles`).Error)

	res, err := api.InviteUserToOrganization(ownerContext(type_enums.ORGANIZATION_ROLE_OWNER.String()), &protos.InviteUserToOrganizationRequest{
		Email:            "new@example.com",
		OrganizationRole: type_enums.ORGANIZATION_ROLE_MEMBER.String(),
		ProjectRoles: []*protos.ProjectRoleAssignment{
			{ProjectId: 100, ProjectRole: type_enums.PROJECT_ROLE_READER.String()},
		},
	})
	require.Error(t, err)
	require.False(t, res.GetSuccess())
	require.Equal(t, pkg_errors.InviteUserToOrganizationCreateProjectRoles.HTTPStatusCodeInt32(), res.GetCode())
	require.EqualValues(t, pkg_errors.InviteUserToOrganizationCreateProjectRoles.Code, res.GetError().GetErrorCode())
	require.Equal(t, pkg_errors.InviteUserToOrganizationCreateProjectRoles.Error, res.GetError().GetErrorMessage())
	require.Equal(t, pkg_errors.InviteUserToOrganizationCreateProjectRoles.ErrorMessage, res.GetError().GetHumanMessage())

	var count int64
	require.NoError(t, db.Model(&internal_entity.UserAuth{}).Count(&count).Error)
	require.EqualValues(t, 1, count)
}

func TestInviteUserToOrganizationEmailFailureDoesNotRollback(t *testing.T) {
	api, db, emailer := newOrganizationAPITest(t)
	emailer.err = errors.New("email failed")

	res, err := api.InviteUserToOrganization(ownerContext(type_enums.ORGANIZATION_ROLE_OWNER.String()), &protos.InviteUserToOrganizationRequest{
		Email:            "new@example.com",
		OrganizationRole: type_enums.ORGANIZATION_ROLE_MEMBER.String(),
		ProjectRoles: []*protos.ProjectRoleAssignment{
			{ProjectId: 100, ProjectRole: type_enums.PROJECT_ROLE_READER.String()},
		},
	})
	require.NoError(t, err)
	require.True(t, res.GetSuccess())
	require.Equal(t, 1, emailer.calls)

	var count int64
	require.NoError(t, db.Model(&internal_entity.UserProjectRole{}).Count(&count).Error)
	require.EqualValues(t, 1, count)
}

func TestAddUserToProjectsAssignsExistingOrganizationUserProjectRoles(t *testing.T) {
	api, db, _ := newProjectAPITest(t)
	require.NoError(t, db.Create(&internal_entity.UserAuth{
		Audited: gorm_models.Audited{Id: 90},
		Mutable: gorm_models.Mutable{
			Status:    type_enums.RECORD_ACTIVE,
			CreatedBy: 1,
		},
		Name:     "Existing",
		Email:    "existing-projects@example.com",
		Password: "hash",
		Source:   "direct",
	}).Error)
	require.NoError(t, db.Create(&internal_entity.UserOrganizationRole{
		Audited: gorm_models.Audited{Id: 91},
		Mutable: gorm_models.Mutable{
			Status:    type_enums.RECORD_ACTIVE,
			CreatedBy: 1,
		},
		UserAuthId:     90,
		OrganizationId: 10,
		Role:           type_enums.ORGANIZATION_ROLE_MEMBER.String(),
	}).Error)

	res, err := api.AddUserToProjects(ownerContext(type_enums.ORGANIZATION_ROLE_ADMIN.String()), &protos.AddUserToProjectsRequest{
		UserId: 90,
		ProjectRoles: []*protos.ProjectRoleAssignment{
			{ProjectId: 100, ProjectRole: type_enums.PROJECT_ROLE_ADMIN.String()},
			{ProjectId: 101, ProjectRole: type_enums.PROJECT_ROLE_READER.String()},
		},
	})
	require.NoError(t, err)
	require.True(t, res.GetSuccess())
	require.Len(t, res.GetData(), 2)

	var projectRoles []internal_entity.UserProjectRole
	require.NoError(t, db.Find(&projectRoles, "user_auth_id = ?", 90).Error)
	require.Len(t, projectRoles, 2)
	roles := map[uint64]string{}
	for _, projectRole := range projectRoles {
		roles[projectRole.ProjectId] = projectRole.Role
		require.Equal(t, type_enums.RECORD_ACTIVE, projectRole.Status)
	}
	require.Equal(t, map[uint64]string{
		100: type_enums.PROJECT_ROLE_ADMIN.String(),
		101: type_enums.PROJECT_ROLE_READER.String(),
	}, roles)
}

func TestAddUserToProjectsRejectsExistingProjectMembership(t *testing.T) {
	api, db, _ := newProjectAPITest(t)
	require.NoError(t, db.Create(&internal_entity.UserAuth{
		Audited: gorm_models.Audited{Id: 88},
		Mutable: gorm_models.Mutable{
			Status:    type_enums.RECORD_ACTIVE,
			CreatedBy: 1,
		},
		Name:     "Existing Project Member",
		Email:    "existing-project-member@example.com",
		Password: "hash",
		Source:   "direct",
	}).Error)
	require.NoError(t, db.Create(&internal_entity.UserOrganizationRole{
		Audited: gorm_models.Audited{Id: 89},
		Mutable: gorm_models.Mutable{
			Status:    type_enums.RECORD_ACTIVE,
			CreatedBy: 1,
		},
		UserAuthId:     88,
		OrganizationId: 10,
		Role:           type_enums.ORGANIZATION_ROLE_MEMBER.String(),
	}).Error)
	require.NoError(t, db.Create(&internal_entity.UserProjectRole{
		Audited: gorm_models.Audited{Id: 87},
		Mutable: gorm_models.Mutable{
			Status:    type_enums.RECORD_ACTIVE,
			CreatedBy: 1,
		},
		UserAuthId: 88,
		ProjectId:  100,
		Role:       type_enums.PROJECT_ROLE_READER.String(),
	}).Error)

	res, err := api.AddUserToProjects(ownerContext(type_enums.ORGANIZATION_ROLE_ADMIN.String()), &protos.AddUserToProjectsRequest{
		UserId: 88,
		ProjectRoles: []*protos.ProjectRoleAssignment{
			{ProjectId: 100, ProjectRole: type_enums.PROJECT_ROLE_ADMIN.String()},
			{ProjectId: 101, ProjectRole: type_enums.PROJECT_ROLE_READER.String()},
		},
	})
	require.NoError(t, err)
	require.False(t, res.GetSuccess())
	require.Equal(t, pkg_errors.AddUserToProjectsUserAlreadyInProject.HTTPStatusCodeInt32(), res.GetCode())
	require.EqualValues(t, pkg_errors.AddUserToProjectsUserAlreadyInProject.Code, res.GetError().GetErrorCode())
	require.Equal(t, pkg_errors.AddUserToProjectsUserAlreadyInProject.Error, res.GetError().GetErrorMessage())
	require.Equal(t, pkg_errors.AddUserToProjectsUserAlreadyInProject.ErrorMessage, res.GetError().GetHumanMessage())

	var projectRole internal_entity.UserProjectRole
	require.NoError(t, db.First(&projectRole, "user_auth_id = ? AND project_id = ?", 88, 100).Error)
	require.Equal(t, type_enums.PROJECT_ROLE_READER.String(), projectRole.Role)
}

func TestAddUserToProjectsRejectsUserOutsideOrganization(t *testing.T) {
	api, db, _ := newProjectAPITest(t)
	require.NoError(t, db.Create(&internal_entity.UserAuth{
		Audited: gorm_models.Audited{Id: 92},
		Mutable: gorm_models.Mutable{
			Status:    type_enums.RECORD_ACTIVE,
			CreatedBy: 1,
		},
		Name:     "Outside",
		Email:    "outside@example.com",
		Password: "hash",
		Source:   "direct",
	}).Error)
	require.NoError(t, db.Create(&internal_entity.UserOrganizationRole{
		Audited: gorm_models.Audited{Id: 93},
		Mutable: gorm_models.Mutable{
			Status:    type_enums.RECORD_ACTIVE,
			CreatedBy: 1,
		},
		UserAuthId:     92,
		OrganizationId: 20,
		Role:           type_enums.ORGANIZATION_ROLE_MEMBER.String(),
	}).Error)

	res, err := api.AddUserToProjects(ownerContext(type_enums.ORGANIZATION_ROLE_OWNER.String()), &protos.AddUserToProjectsRequest{
		UserId: 92,
		ProjectRoles: []*protos.ProjectRoleAssignment{
			{ProjectId: 100, ProjectRole: type_enums.PROJECT_ROLE_READER.String()},
		},
	})
	require.NoError(t, err)
	require.False(t, res.GetSuccess())
	require.Equal(t, pkg_errors.AddUserToProjectsUserNotInOrganization.HTTPStatusCodeInt32(), res.GetCode())
	require.EqualValues(t, pkg_errors.AddUserToProjectsUserNotInOrganization.Code, res.GetError().GetErrorCode())
	require.Equal(t, pkg_errors.AddUserToProjectsUserNotInOrganization.Error, res.GetError().GetErrorMessage())
	require.Equal(t, pkg_errors.AddUserToProjectsUserNotInOrganization.ErrorMessage, res.GetError().GetHumanMessage())
}

func TestAddUserToProjectsRejectsArchivedOrganizationUser(t *testing.T) {
	api, db, _ := newProjectAPITest(t)
	require.NoError(t, db.Create(&internal_entity.UserAuth{
		Audited: gorm_models.Audited{Id: 96},
		Mutable: gorm_models.Mutable{
			Status:    type_enums.RECORD_ARCHIEVE,
			CreatedBy: 1,
		},
		Name:     "Archived Org User",
		Email:    "archived-org-user@example.com",
		Password: "hash",
		Source:   "direct",
	}).Error)
	require.NoError(t, db.Create(&internal_entity.UserOrganizationRole{
		Audited: gorm_models.Audited{Id: 97},
		Mutable: gorm_models.Mutable{
			Status:    type_enums.RECORD_ARCHIEVE,
			CreatedBy: 1,
		},
		UserAuthId:     96,
		OrganizationId: 10,
		Role:           type_enums.ORGANIZATION_ROLE_MEMBER.String(),
	}).Error)

	res, err := api.AddUserToProjects(ownerContext(type_enums.ORGANIZATION_ROLE_OWNER.String()), &protos.AddUserToProjectsRequest{
		UserId: 96,
		ProjectRoles: []*protos.ProjectRoleAssignment{
			{ProjectId: 100, ProjectRole: type_enums.PROJECT_ROLE_READER.String()},
		},
	})
	require.NoError(t, err)
	require.False(t, res.GetSuccess())
	require.Equal(t, pkg_errors.AddUserToProjectsUserNotInOrganization.HTTPStatusCodeInt32(), res.GetCode())
	require.EqualValues(t, pkg_errors.AddUserToProjectsUserNotInOrganization.Code, res.GetError().GetErrorCode())
	require.Equal(t, pkg_errors.AddUserToProjectsUserNotInOrganization.Error, res.GetError().GetErrorMessage())
	require.Equal(t, pkg_errors.AddUserToProjectsUserNotInOrganization.ErrorMessage, res.GetError().GetHumanMessage())

	var projectRoleCount int64
	require.NoError(t, db.Model(&internal_entity.UserProjectRole{}).Where("user_auth_id = ?", 96).Count(&projectRoleCount).Error)
	require.Zero(t, projectRoleCount)
}

func TestAddUserToProjectsRejectsInvalidAssignmentsBeforeWrites(t *testing.T) {
	api, db, _ := newProjectAPITest(t)
	require.NoError(t, db.Create(&internal_entity.UserAuth{
		Audited: gorm_models.Audited{Id: 94},
		Mutable: gorm_models.Mutable{
			Status:    type_enums.RECORD_ACTIVE,
			CreatedBy: 1,
		},
		Name:     "Invalid Assignments",
		Email:    "invalid-assignments@example.com",
		Password: "hash",
		Source:   "direct",
	}).Error)
	require.NoError(t, db.Create(&internal_entity.UserOrganizationRole{
		Audited: gorm_models.Audited{Id: 95},
		Mutable: gorm_models.Mutable{
			Status:    type_enums.RECORD_ACTIVE,
			CreatedBy: 1,
		},
		UserAuthId:     94,
		OrganizationId: 10,
		Role:           type_enums.ORGANIZATION_ROLE_MEMBER.String(),
	}).Error)

	tests := []struct {
		name          string
		req           *protos.AddUserToProjectsRequest
		platformError pkg_errors.PlatformError
	}{
		{
			name:          "zero user id",
			platformError: pkg_errors.AddUserToProjectsInvalidUser,
			req: &protos.AddUserToProjectsRequest{
				UserId: 0,
				ProjectRoles: []*protos.ProjectRoleAssignment{
					{ProjectId: 100, ProjectRole: type_enums.PROJECT_ROLE_READER.String()},
				},
			},
		},
		{
			name:          "missing project roles",
			platformError: pkg_errors.AddUserToProjectsMissingProjectRoles,
			req:           &protos.AddUserToProjectsRequest{UserId: 94},
		},
		{
			name:          "empty project id",
			platformError: pkg_errors.AddUserToProjectsInvalidProjects,
			req: &protos.AddUserToProjectsRequest{
				UserId: 94,
				ProjectRoles: []*protos.ProjectRoleAssignment{
					{ProjectId: 0, ProjectRole: type_enums.PROJECT_ROLE_READER.String()},
				},
			},
		},
		{
			name:          "duplicate project",
			platformError: pkg_errors.AddUserToProjectsDuplicateProject,
			req: &protos.AddUserToProjectsRequest{
				UserId: 94,
				ProjectRoles: []*protos.ProjectRoleAssignment{
					{ProjectId: 100, ProjectRole: type_enums.PROJECT_ROLE_READER.String()},
					{ProjectId: 100, ProjectRole: type_enums.PROJECT_ROLE_WRITER.String()},
				},
			},
		},
		{
			name:          "invalid project role",
			platformError: pkg_errors.AddUserToProjectsInvalidProjectRole,
			req: &protos.AddUserToProjectsRequest{
				UserId: 94,
				ProjectRoles: []*protos.ProjectRoleAssignment{
					{ProjectId: 100, ProjectRole: "Reader"},
				},
			},
		},
		{
			name:          "project outside organization",
			platformError: pkg_errors.AddUserToProjectsInvalidProjects,
			req: &protos.AddUserToProjectsRequest{
				UserId: 94,
				ProjectRoles: []*protos.ProjectRoleAssignment{
					{ProjectId: 200, ProjectRole: type_enums.PROJECT_ROLE_READER.String()},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := api.AddUserToProjects(ownerContext(type_enums.ORGANIZATION_ROLE_ADMIN.String()), tt.req)
			require.NoError(t, err)
			require.False(t, res.GetSuccess())
			require.Equal(t, tt.platformError.HTTPStatusCodeInt32(), res.GetCode())
			require.EqualValues(t, tt.platformError.Code, res.GetError().GetErrorCode())
			require.Equal(t, tt.platformError.Error, res.GetError().GetErrorMessage())
			require.Equal(t, tt.platformError.ErrorMessage, res.GetError().GetHumanMessage())
			var count int64
			require.NoError(t, db.Model(&internal_entity.UserProjectRole{}).Count(&count).Error)
			require.Zero(t, count)
		})
	}
}

func TestAddUserToProjectsProjectRoleFailureReturnsError(t *testing.T) {
	api, db, _ := newProjectAPITest(t)
	require.NoError(t, db.Create(&internal_entity.UserAuth{
		Audited: gorm_models.Audited{Id: 96},
		Mutable: gorm_models.Mutable{
			Status:    type_enums.RECORD_ACTIVE,
			CreatedBy: 1,
		},
		Name:     "Failure",
		Email:    "failure@example.com",
		Password: "hash",
		Source:   "direct",
	}).Error)
	require.NoError(t, db.Create(&internal_entity.UserOrganizationRole{
		Audited: gorm_models.Audited{Id: 97},
		Mutable: gorm_models.Mutable{
			Status:    type_enums.RECORD_ACTIVE,
			CreatedBy: 1,
		},
		UserAuthId:     96,
		OrganizationId: 10,
		Role:           type_enums.ORGANIZATION_ROLE_MEMBER.String(),
	}).Error)
	require.NoError(t, db.Exec(`DROP TABLE user_project_roles`).Error)

	res, err := api.AddUserToProjects(ownerContext(type_enums.ORGANIZATION_ROLE_OWNER.String()), &protos.AddUserToProjectsRequest{
		UserId: 96,
		ProjectRoles: []*protos.ProjectRoleAssignment{
			{ProjectId: 100, ProjectRole: type_enums.PROJECT_ROLE_READER.String()},
		},
	})
	require.Error(t, err)
	require.False(t, res.GetSuccess())
	require.Equal(t, pkg_errors.AddUserToProjectsCreateProjectRoles.HTTPStatusCodeInt32(), res.GetCode())
	require.EqualValues(t, pkg_errors.AddUserToProjectsCreateProjectRoles.Code, res.GetError().GetErrorCode())
	require.Equal(t, pkg_errors.AddUserToProjectsCreateProjectRoles.Error, res.GetError().GetErrorMessage())
	require.Equal(t, pkg_errors.AddUserToProjectsCreateProjectRoles.ErrorMessage, res.GetError().GetHumanMessage())
}

func TestDeleteUserFromOrganizationArchivesUserOrganizationAndProjectRoles(t *testing.T) {
	api, db, _ := newOrganizationAPITest(t)
	require.NoError(t, db.Create(&internal_entity.UserAuth{
		Audited: gorm_models.Audited{Id: 110},
		Mutable: gorm_models.Mutable{
			Status:    type_enums.RECORD_ACTIVE,
			CreatedBy: 1,
		},
		Name:     "Delete Org",
		Email:    "delete-org@example.com",
		Password: "hash",
		Source:   "direct",
	}).Error)
	require.NoError(t, db.Create(&internal_entity.UserOrganizationRole{
		Audited: gorm_models.Audited{Id: 111},
		Mutable: gorm_models.Mutable{
			Status:    type_enums.RECORD_ACTIVE,
			CreatedBy: 1,
		},
		UserAuthId:     110,
		OrganizationId: 10,
		Role:           type_enums.ORGANIZATION_ROLE_MEMBER.String(),
	}).Error)
	require.NoError(t, db.Create(&internal_entity.UserProjectRole{
		Audited: gorm_models.Audited{Id: 112},
		Mutable: gorm_models.Mutable{
			Status:    type_enums.RECORD_ACTIVE,
			CreatedBy: 1,
		},
		UserAuthId: 110,
		ProjectId:  100,
		Role:       type_enums.PROJECT_ROLE_READER.String(),
	}).Error)
	require.NoError(t, db.Create(&internal_entity.UserProjectRole{
		Audited: gorm_models.Audited{Id: 113},
		Mutable: gorm_models.Mutable{
			Status:    type_enums.RECORD_INVITED,
			CreatedBy: 1,
		},
		UserAuthId: 110,
		ProjectId:  101,
		Role:       type_enums.PROJECT_ROLE_WRITER.String(),
	}).Error)

	res, err := api.DeleteUserFromOrganization(ownerContext(type_enums.ORGANIZATION_ROLE_ADMIN.String()), &protos.DeleteUserFromOrganizationRequest{
		UserId: 110,
	})
	require.NoError(t, err)
	require.True(t, res.GetSuccess())
	require.EqualValues(t, 110, res.GetId())

	var user internal_entity.UserAuth
	require.NoError(t, db.First(&user, "id = ?", 110).Error)
	require.Equal(t, type_enums.RECORD_ARCHIEVE, user.Status)

	var orgRole internal_entity.UserOrganizationRole
	require.NoError(t, db.First(&orgRole, "user_auth_id = ? AND organization_id = ?", 110, 10).Error)
	require.Equal(t, type_enums.RECORD_ARCHIEVE, orgRole.Status)

	var projectRoles []internal_entity.UserProjectRole
	require.NoError(t, db.Find(&projectRoles, "user_auth_id = ?", 110).Error)
	require.Len(t, projectRoles, 2)
	for _, projectRole := range projectRoles {
		require.Equal(t, type_enums.RECORD_ARCHIEVE, projectRole.Status)
	}
}

func TestDeleteUserFromOrganizationRejectsAuthAndValidationFailures(t *testing.T) {
	api, db, _ := newOrganizationAPITest(t)
	require.NoError(t, db.Create(&internal_entity.UserAuth{
		Audited: gorm_models.Audited{Id: 120},
		Mutable: gorm_models.Mutable{
			Status:    type_enums.RECORD_ACTIVE,
			CreatedBy: 1,
		},
		Name:     "Outside",
		Email:    "delete-outside@example.com",
		Password: "hash",
		Source:   "direct",
	}).Error)
	require.NoError(t, db.Create(&internal_entity.UserOrganizationRole{
		Audited: gorm_models.Audited{Id: 121},
		Mutable: gorm_models.Mutable{
			Status:    type_enums.RECORD_ACTIVE,
			CreatedBy: 1,
		},
		UserAuthId:     120,
		OrganizationId: 20,
		Role:           type_enums.ORGANIZATION_ROLE_MEMBER.String(),
	}).Error)
	require.NoError(t, db.Create(&internal_entity.UserAuth{
		Audited: gorm_models.Audited{Id: 122},
		Mutable: gorm_models.Mutable{
			Status:    type_enums.RECORD_ACTIVE,
			CreatedBy: 1,
		},
		Name:     "Owner Target",
		Email:    "delete-owner@example.com",
		Password: "hash",
		Source:   "direct",
	}).Error)
	require.NoError(t, db.Create(&internal_entity.UserOrganizationRole{
		Audited: gorm_models.Audited{Id: 123},
		Mutable: gorm_models.Mutable{
			Status:    type_enums.RECORD_ACTIVE,
			CreatedBy: 1,
		},
		UserAuthId:     122,
		OrganizationId: 10,
		Role:           type_enums.ORGANIZATION_ROLE_OWNER.String(),
	}).Error)

	res, err := api.DeleteUserFromOrganization(context.Background(), &protos.DeleteUserFromOrganizationRequest{UserId: 120})
	require.Error(t, err)
	require.False(t, res.GetSuccess())
	require.Equal(t, pkg_errors.DeleteUserFromOrganizationUnauthenticated.HTTPStatusCodeInt32(), res.GetCode())

	tests := []struct {
		name          string
		ctx           context.Context
		req           *protos.DeleteUserFromOrganizationRequest
		platformError pkg_errors.PlatformError
		expectError   bool
	}{
		{
			name:          "non admin",
			ctx:           ownerContext(type_enums.ORGANIZATION_ROLE_MEMBER.String()),
			req:           &protos.DeleteUserFromOrganizationRequest{UserId: 120},
			platformError: pkg_errors.DeleteUserFromOrganizationUnauthorized,
			expectError:   true,
		},
		{
			name:          "zero user id",
			ctx:           ownerContext(type_enums.ORGANIZATION_ROLE_OWNER.String()),
			req:           &protos.DeleteUserFromOrganizationRequest{},
			platformError: pkg_errors.DeleteUserFromOrganizationInvalidUser,
		},
		{
			name:          "user outside organization",
			ctx:           ownerContext(type_enums.ORGANIZATION_ROLE_OWNER.String()),
			req:           &protos.DeleteUserFromOrganizationRequest{UserId: 120},
			platformError: pkg_errors.DeleteUserFromOrganizationUserNotInOrg,
		},
		{
			name:          "organization owner",
			ctx:           ownerContext(type_enums.ORGANIZATION_ROLE_ADMIN.String()),
			req:           &protos.DeleteUserFromOrganizationRequest{UserId: 122},
			platformError: pkg_errors.DeleteUserFromOrganizationOwner,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := api.DeleteUserFromOrganization(tt.ctx, tt.req)
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.False(t, res.GetSuccess())
			require.Equal(t, tt.platformError.HTTPStatusCodeInt32(), res.GetCode())
			require.EqualValues(t, tt.platformError.Code, res.GetError().GetErrorCode())
			require.Equal(t, tt.platformError.Error, res.GetError().GetErrorMessage())
			require.Equal(t, tt.platformError.ErrorMessage, res.GetError().GetHumanMessage())
		})
	}
}

func TestDeleteUserFromProjectArchivesOnlySelectedProjectRole(t *testing.T) {
	api, db, _ := newProjectAPITest(t)
	require.NoError(t, db.Create(&internal_entity.UserAuth{
		Audited: gorm_models.Audited{Id: 130},
		Mutable: gorm_models.Mutable{
			Status:    type_enums.RECORD_ACTIVE,
			CreatedBy: 1,
		},
		Name:     "Delete Project",
		Email:    "delete-project@example.com",
		Password: "hash",
		Source:   "direct",
	}).Error)
	require.NoError(t, db.Create(&internal_entity.UserOrganizationRole{
		Audited: gorm_models.Audited{Id: 131},
		Mutable: gorm_models.Mutable{
			Status:    type_enums.RECORD_ACTIVE,
			CreatedBy: 1,
		},
		UserAuthId:     130,
		OrganizationId: 10,
		Role:           type_enums.ORGANIZATION_ROLE_MEMBER.String(),
	}).Error)
	require.NoError(t, db.Create(&internal_entity.UserProjectRole{
		Audited: gorm_models.Audited{Id: 132},
		Mutable: gorm_models.Mutable{
			Status:    type_enums.RECORD_ACTIVE,
			CreatedBy: 1,
		},
		UserAuthId: 130,
		ProjectId:  100,
		Role:       type_enums.PROJECT_ROLE_READER.String(),
	}).Error)
	require.NoError(t, db.Create(&internal_entity.UserProjectRole{
		Audited: gorm_models.Audited{Id: 133},
		Mutable: gorm_models.Mutable{
			Status:    type_enums.RECORD_ACTIVE,
			CreatedBy: 1,
		},
		UserAuthId: 130,
		ProjectId:  101,
		Role:       type_enums.PROJECT_ROLE_WRITER.String(),
	}).Error)

	res, err := api.DeleteUserFromProject(ownerContext(type_enums.ORGANIZATION_ROLE_OWNER.String()), &protos.DeleteUserFromProjectRequest{
		UserId:    130,
		ProjectId: 100,
	})
	require.NoError(t, err)
	require.True(t, res.GetSuccess())
	require.EqualValues(t, 130, res.GetId())

	var user internal_entity.UserAuth
	require.NoError(t, db.First(&user, "id = ?", 130).Error)
	require.Equal(t, type_enums.RECORD_ACTIVE, user.Status)

	var orgRole internal_entity.UserOrganizationRole
	require.NoError(t, db.First(&orgRole, "user_auth_id = ? AND organization_id = ?", 130, 10).Error)
	require.Equal(t, type_enums.RECORD_ACTIVE, orgRole.Status)

	var deletedProjectRole internal_entity.UserProjectRole
	require.NoError(t, db.First(&deletedProjectRole, "user_auth_id = ? AND project_id = ?", 130, 100).Error)
	require.Equal(t, type_enums.RECORD_ARCHIEVE, deletedProjectRole.Status)

	var activeProjectRole internal_entity.UserProjectRole
	require.NoError(t, db.First(&activeProjectRole, "user_auth_id = ? AND project_id = ?", 130, 101).Error)
	require.Equal(t, type_enums.RECORD_ACTIVE, activeProjectRole.Status)
}

func TestDeleteUserFromProjectRejectsAuthAndValidationFailures(t *testing.T) {
	api, db, _ := newProjectAPITest(t)
	require.NoError(t, db.Create(&internal_entity.UserAuth{
		Audited: gorm_models.Audited{Id: 140},
		Mutable: gorm_models.Mutable{
			Status:    type_enums.RECORD_ACTIVE,
			CreatedBy: 1,
		},
		Name:     "Project Delete Validation",
		Email:    "project-delete-validation@example.com",
		Password: "hash",
		Source:   "direct",
	}).Error)
	require.NoError(t, db.Create(&internal_entity.UserOrganizationRole{
		Audited: gorm_models.Audited{Id: 141},
		Mutable: gorm_models.Mutable{
			Status:    type_enums.RECORD_ACTIVE,
			CreatedBy: 1,
		},
		UserAuthId:     140,
		OrganizationId: 10,
		Role:           type_enums.ORGANIZATION_ROLE_MEMBER.String(),
	}).Error)

	res, err := api.DeleteUserFromProject(context.Background(), &protos.DeleteUserFromProjectRequest{UserId: 140, ProjectId: 100})
	require.Error(t, err)
	require.False(t, res.GetSuccess())
	require.Equal(t, pkg_errors.DeleteUserFromProjectUnauthenticated.HTTPStatusCodeInt32(), res.GetCode())

	tests := []struct {
		name          string
		ctx           context.Context
		req           *protos.DeleteUserFromProjectRequest
		platformError pkg_errors.PlatformError
		expectError   bool
	}{
		{
			name:          "non admin",
			ctx:           ownerContext(type_enums.ORGANIZATION_ROLE_MEMBER.String()),
			req:           &protos.DeleteUserFromProjectRequest{UserId: 140, ProjectId: 100},
			platformError: pkg_errors.DeleteUserFromProjectUnauthorized,
			expectError:   true,
		},
		{
			name:          "zero user id",
			ctx:           ownerContext(type_enums.ORGANIZATION_ROLE_ADMIN.String()),
			req:           &protos.DeleteUserFromProjectRequest{ProjectId: 100},
			platformError: pkg_errors.DeleteUserFromProjectInvalidUser,
		},
		{
			name:          "zero project id",
			ctx:           ownerContext(type_enums.ORGANIZATION_ROLE_ADMIN.String()),
			req:           &protos.DeleteUserFromProjectRequest{UserId: 140},
			platformError: pkg_errors.DeleteUserFromProjectInvalidProject,
		},
		{
			name:          "project outside organization",
			ctx:           ownerContext(type_enums.ORGANIZATION_ROLE_ADMIN.String()),
			req:           &protos.DeleteUserFromProjectRequest{UserId: 140, ProjectId: 200},
			platformError: pkg_errors.DeleteUserFromProjectInvalidProject,
		},
		{
			name:          "user not in project",
			ctx:           ownerContext(type_enums.ORGANIZATION_ROLE_ADMIN.String()),
			req:           &protos.DeleteUserFromProjectRequest{UserId: 140, ProjectId: 100},
			platformError: pkg_errors.DeleteUserFromProjectUserNotInProject,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res, err := api.DeleteUserFromProject(tt.ctx, tt.req)
			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
			require.False(t, res.GetSuccess())
			require.Equal(t, tt.platformError.HTTPStatusCodeInt32(), res.GetCode())
			require.EqualValues(t, tt.platformError.Code, res.GetError().GetErrorCode())
			require.Equal(t, tt.platformError.Error, res.GetError().GetErrorMessage())
			require.Equal(t, tt.platformError.ErrorMessage, res.GetError().GetHumanMessage())
		})
	}
}
