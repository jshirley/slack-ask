package storage

import (
	"gopkg.in/mgo.v2/bson"
)

type ChannelConfig struct {
	ChannelID      string
	ChannelName    string
	Project        string
	Components     []string
	AssignEndpoint string
}

const CONFIG_COLLECTION = "channel_configs"

func (db *MongoDatabase) SetChannelProject(channelID string, project string) error {
	channelConfig := &ChannelConfig{ChannelID: channelID, Project: project}
	c := db.C(CONFIG_COLLECTION)

	// TODO: Validate that project is even legit by talking to the JIRA API
	_, err := c.UpsertId(channelConfig.ChannelID, channelConfig)

	return err
}

func (db *MongoDatabase) SetChannelConfig(channelConfig *ChannelConfig) error {
	c := db.C(CONFIG_COLLECTION)

	// TODO: Validate that project is even legit by talking to the JIRA API
	_, err := c.UpsertId(channelConfig.ChannelID, channelConfig)

	return err
}

func (db *MongoDatabase) GetChannelConfig(channelID string) (*ChannelConfig, error) {
	c := db.C(CONFIG_COLLECTION)

	result := ChannelConfig{}
	err := c.Find(bson.M{"_id": channelID}).One(&result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}
