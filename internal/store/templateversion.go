package store

import (
	"context"
	b64 "encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	api "github.com/flightctl/flightctl/api/v1alpha1"
	"github.com/flightctl/flightctl/internal/store/model"
	"github.com/flightctl/flightctl/internal/util"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type TemplateVersion interface {
	Create(ctx context.Context, orgId uuid.UUID, templateVersion *api.TemplateVersion) (*api.TemplateVersion, error)
	List(ctx context.Context, orgId uuid.UUID, listParams ListParams) (*api.TemplateVersionList, error)
	DeleteAll(ctx context.Context, orgId uuid.UUID, fleet *string) error
	Get(ctx context.Context, orgId uuid.UUID, fleet string, name string) (*api.TemplateVersion, error)
	Delete(ctx context.Context, orgId uuid.UUID, fleet string, name string) error
	InitialMigration() error
}

type TemplateVersionStore struct {
	db  *gorm.DB
	log logrus.FieldLogger
}

type TemplateVersionStoreCallback func(before *model.TemplateVersion, after *model.TemplateVersion)
type TemplateVersionStoreAllDeletedCallback func(orgId uuid.UUID)

// Make sure we conform to TemplateVersion interface
var _ TemplateVersion = (*TemplateVersionStore)(nil)

func NewTemplateVersion(db *gorm.DB, log logrus.FieldLogger) TemplateVersion {
	return &TemplateVersionStore{db: db, log: log}
}

func (s *TemplateVersionStore) InitialMigration() error {
	return s.db.AutoMigrate(&model.TemplateVersion{})
}

func (s *TemplateVersionStore) Create(ctx context.Context, orgId uuid.UUID, resource *api.TemplateVersion) (*api.TemplateVersion, error) {
	if resource == nil {
		return nil, fmt.Errorf("resource is nil")
	}
	templateVersion := model.NewTemplateVersionFromApiResource(resource)
	templateVersion.OrgID = orgId
	templateVersion.Owner = util.SetResourceOwner(model.FleetKind, resource.Spec.Fleet)
	templateVersion.Generation = util.Int64ToPtr(1)
	status := api.TemplateVersionStatus{Conditions: &[]api.Condition{}}
	api.SetStatusCondition(status.Conditions, api.Condition{Type: api.TemplateVersionReady, Status: api.ConditionStatusFalse})
	templateVersion.Status = model.MakeJSONField(status)
	apiResource := templateVersion.ToApiResource()

	err := s.db.Transaction(func(innerTx *gorm.DB) (err error) {
		fleet := model.Fleet{Resource: model.Resource{OrgID: orgId, Name: resource.Spec.Fleet}}
		result := innerTx.First(&fleet)
		if result.Error != nil {
			return result.Error
		}
		duplicateName := model.TemplateVersion{
			ResourceWithPrimaryKeyOwner: model.ResourceWithPrimaryKeyOwner{
				OrgID: orgId,
				Owner: templateVersion.Owner,
				Name:  *resource.Metadata.Name,
			},
		}
		result = innerTx.First(&duplicateName)
		if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return result.Error
		}
		if result.Error == nil {
			return gorm.ErrInvalidData
		}

		result = innerTx.Create(templateVersion)
		return result.Error
	})

	return &apiResource, err
}

func (s *TemplateVersionStore) List(ctx context.Context, orgId uuid.UUID, listParams ListParams) (*api.TemplateVersionList, error) {
	var templateVersions model.TemplateVersionList
	var nextContinue *string
	var numRemaining *int64

	query := BuildBaseListQuery(s.db.Model(&templateVersions), orgId, listParams)
	if listParams.Limit > 0 {
		// Request 1 more than the user asked for to see if we need to return "continue"
		query = AddPaginationToQuery(query, listParams.Limit+1, listParams.Continue)
	}
	result := query.Find(&templateVersions)

	// If we got more than the user requested, remove one record and calculate "continue"
	if listParams.Limit > 0 && len(templateVersions) > listParams.Limit {
		nextContinueStruct := Continue{
			Name:    templateVersions[len(templateVersions)-1].Name,
			Version: CurrentContinueVersion,
		}
		templateVersions = templateVersions[:len(templateVersions)-1]

		var numRemainingVal int64
		if listParams.Continue != nil {
			numRemainingVal = listParams.Continue.Count - int64(listParams.Limit)
			if numRemainingVal < 1 {
				numRemainingVal = 1
			}
		} else {
			countQuery := BuildBaseListQuery(s.db.Model(&templateVersions), orgId, listParams)
			numRemainingVal = CountRemainingItems(countQuery, nextContinueStruct.Name)
		}
		nextContinueStruct.Count = numRemainingVal
		contByte, _ := json.Marshal(nextContinueStruct)
		contStr := b64.StdEncoding.EncodeToString(contByte)
		nextContinue = &contStr
		numRemaining = &numRemainingVal
	}

	apiTemplateVersionList := templateVersions.ToApiResource(nextContinue, numRemaining)
	return &apiTemplateVersionList, result.Error
}

func (s *TemplateVersionStore) DeleteAll(ctx context.Context, orgId uuid.UUID, owner *string) error {
	condition := model.TemplateVersion{}
	if owner != nil {
		return s.db.Unscoped().Where("org_id = ? AND owner = ?", orgId, *owner).Delete(&condition).Error
	}
	return s.db.Unscoped().Where("org_id = ?", orgId).Delete(&condition).Error
}

func (s *TemplateVersionStore) Get(ctx context.Context, orgId uuid.UUID, fleet string, name string) (*api.TemplateVersion, error) {
	owner := util.SetResourceOwner(model.FleetKind, fleet)
	templateVersion := model.TemplateVersion{
		ResourceWithPrimaryKeyOwner: model.ResourceWithPrimaryKeyOwner{OrgID: orgId, Owner: owner, Name: name},
	}
	result := s.db.First(&templateVersion)
	if result.Error != nil {
		return nil, result.Error
	}

	apiTemplateVersion := templateVersion.ToApiResource()
	return &apiTemplateVersion, nil
}

func (s *TemplateVersionStore) Delete(ctx context.Context, orgId uuid.UUID, fleet string, name string) error {
	owner := util.SetResourceOwner(model.FleetKind, fleet)
	condition := model.TemplateVersion{
		ResourceWithPrimaryKeyOwner: model.ResourceWithPrimaryKeyOwner{OrgID: orgId, Owner: owner, Name: name},
	}
	result := s.db.Unscoped().Delete(&condition)
	return result.Error
}