package storage

import (
	"time"

	"gopkg.in/mgo.v2/bson"
)

const CALLBACK_COLLECTION = "internal_callbacks"

type SlashCommand struct {
	Token          string `schema:"token"`
	TeamID         string `schema:"team_id" bson:"team_id"`
	TeamDomain     string `schema:"team_domain" bson:"team_domain"`
	EnterpriseID   string `schema:"enterprise_id" bson:"enterprise_id"`
	EnterpriseName string `schema:"enterprise_name" bson:"enterprise_name"`
	ChannelID      string `schema:"channel_id" bson:"channel_id"`
	ChannelName    string `schema:"channel_name" bson:"channel_name"`
	UserID         string `schema:"user_id" bson:"user_id"`
	UserName       string `schema:"user_name" bson:"user_name"`
	Command        string `schema:"command"`
	Text           string `schema:"text"`
	ResponseURL    string `schema:"response_url" bson:"response_url"`
	TriggerID      string `schema:"trigger_id" bson:"trigger_id"`
	Timestamp      int64  `schema:"timestamp"`

	Config *ChannelConfig `bson:"-"`
}

func (db *MongoDatabase) StoreCallback(callbackID string, command *SlashCommand) error {
	c := db.C(CALLBACK_COLLECTION)

	_, err := c.UpsertId(callbackID, command)

	return err
}

func (db *MongoDatabase) RemoveCallback(callbackID string) error {
	c := db.C(CALLBACK_COLLECTION)
	_, err := c.RemoveAll(bson.M{"_id": callbackID})
	return err
}

func (db *MongoDatabase) RemoveStaleCallbacks(timeout int64) error {
	c := db.C(CALLBACK_COLLECTION)
	_, err := c.RemoveAll(bson.M{"action_ts": bson.M{"$lt": time.Now().Unix() - timeout}})
	return err
}

func (db *MongoDatabase) GetCallback(callbackID string) (*SlashCommand, error) {
	c := db.C(CALLBACK_COLLECTION)
	result := SlashCommand{}

	err := c.Find(bson.M{"_id": callbackID}).One(&result)
	if err != nil {
		return nil, err
	}

	return &result, nil
}
