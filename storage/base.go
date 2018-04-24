package storage

import (
	"time"

	"gopkg.in/mgo.v2"
)

type MongoCollection struct {
	*mgo.Collection
}

type Collection interface {
	Find(query interface{}) *mgo.Query
	Count() (n int, err error)
	FindId(id interface{}) *mgo.Query
	Insert(docs ...interface{}) error
	Remove(selector interface{}) error
	Update(selector interface{}, update interface{}) error
	Upsert(selector interface{}, update interface{}) (info *mgo.ChangeInfo, err error)
	UpsertId(id interface{}, update interface{}) (info *mgo.ChangeInfo, err error)
	EnsureIndex(index mgo.Index) error
	RemoveAll(selector interface{}) (info *mgo.ChangeInfo, err error)
}

type MongoDatabase struct {
	*mgo.Database
}

func (d MongoDatabase) C(name string) Collection {
	return &MongoCollection{Collection: d.Database.C(name)}
}

type DataLayer interface {
	C(name string) Collection
	SetChannelProject(channelID string, project string) error
	SetChannelConfig(config *ChannelConfig) error
	GetChannelConfig(channelID string) (*ChannelConfig, error)
	StoreCallback(callbackID string, command *SlashCommand) error
	RemoveCallback(callbackID string) error
	RemoveStaleCallbacks(timeout int64) error
	GetCallback(callbackID string) (*SlashCommand, error)
}

// Session is an interface to access to the Session struct.
type Session interface {
	DB(name string) DataLayer
	SetSafe(safe *mgo.Safe)
	SetSyncTimeout(d time.Duration)
	SetSocketTimeout(d time.Duration)
	Close()
	Copy() Session
}

// MongoSession is currently a Mongo session.
type MongoSession struct {
	*mgo.Session
}

// DB shadows *mgo.DB to returns a DataLayer interface instead of *mgo.Database.
func (s MongoSession) DB(name string) DataLayer {
	return &MongoDatabase{Database: s.Session.DB(name)}
}

// Copy mocks mgo.Session.Copy()
func (s MongoSession) Copy() Session {
	return MongoSession{s.Session.Copy()}
}

// NewSession returns a new Mongo Session.
func NewSession(hosts string) Session {
	mgoSession, err := mgo.Dial(hosts)
	if err != nil {
		panic(err)
	}
	session := MongoSession{mgoSession}
	session.SetSafe(&mgo.Safe{})
	session.SetSyncTimeout(3 * time.Second)
	session.SetSocketTimeout(3 * time.Second)

	return session
}
