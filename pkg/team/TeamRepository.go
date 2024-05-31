/*
 * Copyright (c) 2020-2024. Devtron Inc.
 */

package team

import (
	"github.com/devtron-labs/devtron/pkg/sql"
	"github.com/go-pg/pg"
)

type Team struct {
	tableName struct{} `sql:"team"`
	Id        int      `sql:"id,pk"`
	Name      string   `sql:"name,notnull"`
	Active    bool     `sql:"active,notnull"`
	sql.AuditLog
}

type TeamRepository interface {
	Save(team *Team) error
	FindAllActive() ([]Team, error)
	FindOne(id int) (Team, error)
	FindByTeamName(name string) (Team, error)
	Update(team *Team) error
	MarkTeamDeleted(team *Team, tx *pg.Tx) error
	GetConnection() *pg.DB
	FindByIds(ids []*int) ([]*Team, error)
	FindAllActiveTeamNames() ([]string, error)
	FindAllActiveTeamIds() ([]int, error)
	FindByNames(teams []string) ([]*Team, error)
}
type TeamRepositoryImpl struct {
	dbConnection *pg.DB
}

func NewTeamRepositoryImpl(dbConnection *pg.DB) *TeamRepositoryImpl {
	return &TeamRepositoryImpl{dbConnection: dbConnection}
}

const UNASSIGNED_PROJECT = "unassigned"

func (impl TeamRepositoryImpl) Save(team *Team) error {
	err := impl.dbConnection.Insert(team)
	return err
}

func (impl TeamRepositoryImpl) FindAllActive() ([]Team, error) {
	var teams []Team
	err := impl.dbConnection.Model(&teams).Where("active = ?", true).Select()
	return teams, err
}

func (impl TeamRepositoryImpl) FindAllActiveTeamNames() ([]string, error) {
	var teamNames []string
	err := impl.dbConnection.Model((*Team)(nil)).
		Where("active = ?", true).Select(&teamNames)
	return teamNames, err
}

func (impl TeamRepositoryImpl) FindAllActiveTeamIds() ([]int, error) {
	var teamIds []int
	err := impl.dbConnection.Model((*Team)(nil)).Column("id").
		Where("active = ?", true).Select(&teamIds)
	return teamIds, err
}

func (impl TeamRepositoryImpl) FindOne(id int) (Team, error) {
	var team Team
	err := impl.dbConnection.Model(&team).
		Where("id = ?", id).
		Where("active = ?", true).Select()
	return team, err
}

func (impl TeamRepositoryImpl) FindByTeamName(name string) (Team, error) {
	var team Team
	err := impl.dbConnection.Model(&team).
		Where("name = ?", name).
		Where("active = ?", true).Select()
	return team, err
}

func (impl TeamRepositoryImpl) Update(team *Team) error {
	err := impl.dbConnection.Update(team)
	return err
}

func (impl TeamRepositoryImpl) MarkTeamDeleted(team *Team, tx *pg.Tx) error {
	team.Active = false
	err := tx.Update(team)
	return err
}

func (repo TeamRepositoryImpl) FindByIds(ids []*int) ([]*Team, error) {
	var objects []*Team
	err := repo.dbConnection.Model(&objects).Where("active = ?", true).Where("id in (?)", pg.In(ids)).Select()
	return objects, err
}

func (repo TeamRepositoryImpl) GetConnection() *pg.DB {
	return repo.dbConnection
}

func (repo TeamRepositoryImpl) FindByNames(teams []string) ([]*Team, error) {
	var objects []*Team
	err := repo.dbConnection.Model(&objects).Where("active = ?", true).Where("name in (?)", pg.In(teams)).Select()
	return objects, err
}

type TeamRbacObjects struct {
	AppName  string `json:"appName"`
	TeamName string `json:"teamName"`
	AppId    int    `json:"appId"`
}
