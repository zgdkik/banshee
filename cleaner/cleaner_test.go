// Copyright 2015 Eleme Inc. All rights reserved.

package cleaner

import (
	"github.com/eleme/banshee/models"
	"github.com/eleme/banshee/storage"
	"github.com/eleme/banshee/storage/indexdb"
	"github.com/eleme/banshee/storage/statedb"
	"github.com/eleme/banshee/util/assert"
	"os"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	numGrid := uint32(288)
	gridLen := uint32(300)
	period := numGrid * gridLen
	// Open storage
	dbFileName := "db-test"
	dbOptions := &storage.Options{numGrid, gridLen}
	db, _ := storage.Open(dbFileName, dbOptions)
	defer os.RemoveAll(dbFileName)
	defer db.Close()
	// Create cleaner
	c := New(db, period)
	assert.Ok(t, c.expiration == expirationNumToPeriod*period)
	assert.Ok(t, c.metricExpiration == metricExpirationNumToPeriod*period)
	assert.Ok(t, c.interval*periodNumToInterval == period)
}

func TestClean(t *testing.T) {
	numGrid := uint32(288)
	gridLen := uint32(300)
	period := numGrid * gridLen
	// Open storage
	dbFileName := "db-test"
	dbOptions := &storage.Options{numGrid, gridLen}
	db, _ := storage.Open(dbFileName, dbOptions)
	defer os.RemoveAll(dbFileName)
	defer db.Close()
	// Create cleaner
	c := New(db, period)
	// Add outdated data.
	// Case fully cleaned: 3 days no new data
	m1 := &models.Metric{Name: "fully-case", Stamp: uint32(time.Now().Unix() - 3*3600*24 - 1)}
	// Case outdated metrics cleaned.
	m2 := &models.Metric{Name: "metric-case", Stamp: uint32(time.Now().Unix() - 7*3600*24 - 100)}
	m3 := &models.Metric{Name: m2.Name, Stamp: uint32(time.Now().Unix() - 60)}
	i1 := &models.Index{Name: m1.Name, Stamp: m1.Stamp}
	i2 := &models.Index{Name: m2.Name, Stamp: m2.Stamp}
	i3 := &models.Index{Name: m3.Name, Stamp: m3.Stamp}
	// Put metrics.
	db.Metric.Put(m1)
	db.Metric.Put(m2)
	db.Metric.Put(m3)
	// Pit States.
	db.State.Put(m1, &models.State{})
	db.State.Put(m2, &models.State{})
	db.State.Put(m3, &models.State{})
	// Put indexes.
	db.Index.Put(i1)
	db.Index.Put(i2)
	db.Index.Put(i3)
	c.clean()
	// m1 should be fully cleaned
	var err error
	_, err = db.Index.Get(m1.Name)
	assert.Ok(t, err == indexdb.ErrNotFound)
	_, err = db.State.Get(m1)
	assert.Ok(t, err == statedb.ErrNotFound)
	l, err := db.Metric.Get(m1.Name, 0, uint32(time.Now().Unix()))
	assert.Ok(t, len(l) == 0)
	// m2 should be cleaned and m3 shouldn't be cleaned
	l, err = db.Metric.Get(m2.Name, m2.Stamp, uint32(time.Now().Unix()))
	assert.Ok(t, len(l) == 1)
	assert.Ok(t, l[0].Name == m2.Name)
	assert.Ok(t, l[0].Stamp == m3.Stamp && l[0].Stamp != m2.Stamp)
	// m2/m3's state and index shouldn't be cleaned
	i, err := db.Index.Get(m2.Name)
	assert.Ok(t, err == nil && i.Name == m2.Name)
	s, err := db.State.Get(m2)
	assert.Ok(t, err == nil && s != nil)
}
