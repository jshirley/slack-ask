package asker

import (
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type Storage struct {
	cxn        *mgo.Session
	database   string
	collection string
}

type ChannelConfig struct {
	ChannelID      string
	ChannelName    string
	Project        string
	Components     []string
	AssignEndpoint string
}

func NewStorage(hosts string, database string, collection string) (*Storage, error) {
	session, err := mgo.Dial(hosts)
	if err != nil {
		return nil, err
	}
	return &Storage{cxn: session, database: database, collection: collection}, nil
}

func (storage *Storage) CloseStorage() {
	storage.cxn.Close()
}

func (storage *Storage) SetChannelProject(channelID string, project string) error {
	channelConfig := &ChannelConfig{ChannelID: channelID, Project: project}
	c := storage.cxn.DB(storage.database).C(storage.collection)

	// TODO: Validate that project is even legit by talking to the JIRA API
	_, err := c.UpsertId(channelConfig.ChannelID, channelConfig)

	return err
}

func (storage *Storage) SetChannelConfig(channelConfig *ChannelConfig) error {
	c := storage.cxn.DB(storage.database).C(storage.collection)

	// TODO: Validate that project is even legit by talking to the JIRA API
	_, err := c.UpsertId(channelConfig.ChannelID, channelConfig)

	return err
}

func (storage *Storage) GetChannelConfig(channelID string) (*ChannelConfig, error) {
	c := storage.cxn.DB(storage.database).C(storage.collection)

	result := ChannelConfig{}
	err := c.Find(bson.M{"_id": channelID}).One(&result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
