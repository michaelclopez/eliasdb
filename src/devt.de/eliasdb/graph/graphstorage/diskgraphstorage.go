/*
 * EliasDB
 *
 * Copyright 2016 Matthias Ladkau. All rights reserved.
 *
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at http://mozilla.org/MPL/2.0/.
 */

/*
Package graphstorage contains classes which model storage objects for graph data.

Graph storage which stores its data on disk.
*/
package graphstorage

import (
	"fmt"
	"os"
	"strings"

	"devt.de/common/datautil"
	"devt.de/common/fileutil"
	"devt.de/eliasdb/graph/util"
	"devt.de/eliasdb/storage"
)

/*
FilenameNameDB is the filename for the name storage file
*/
var FilenameNameDB = "names.pm"

/*
DiskGraphStorage data structure
*/
type DiskGraphStorage struct {
	name            string                     // Name of the graph storage
	mainDB          *datautil.PersistentMap    // Database storing names
	storagemanagers map[string]storage.Manager // Map of StorageManagers
}

/*
NewDiskGraphStorage creates a new DiskGraphStorage instance.
*/
func NewDiskGraphStorage(name string) (GraphStorage, error) {

	dgs := &DiskGraphStorage{name, nil, make(map[string]storage.Manager)}

	// Load the graph storage if the storage directory already exists if not try to create it

	if res, _ := fileutil.PathExists(name); !res {
		if err := os.Mkdir(name, 0770); err != nil {
			return nil, &util.GraphError{Type: util.ErrOpening, Detail: err.Error()}
		}

		// Create the graph storage files

		mainDB, err := datautil.NewPersistentMap(name + "/" + FilenameNameDB)
		if err != nil {
			return nil, &util.GraphError{Type: util.ErrOpening, Detail: err.Error()}
		}

		dgs.mainDB = mainDB

	} else {

		// Load graph storage files

		mainDB, err := datautil.LoadPersistentMap(name + "/" + FilenameNameDB)
		if err != nil {
			return nil, &util.GraphError{Type: util.ErrOpening, Detail: err.Error()}
		}

		dgs.mainDB = mainDB
	}

	return dgs, nil
}

/*
Name returns the name of the DiskGraphStorage instance.
*/
func (dgs *DiskGraphStorage) Name() string {
	return dgs.name
}

/*
MainDB returns the main database.
*/
func (dgs *DiskGraphStorage) MainDB() map[string]string {
	return dgs.mainDB.Data
}

/*
RollbackMain rollback the main database.
*/
func (dgs *DiskGraphStorage) RollbackMain() error {
	mainDB, err := datautil.LoadPersistentMap(dgs.name + "/" + FilenameNameDB)
	if err != nil {
		return &util.GraphError{Type: util.ErrOpening, Detail: err.Error()}
	}

	dgs.mainDB = mainDB

	return nil
}

/*
FlushMain writes the main database to the storage.
*/
func (dgs *DiskGraphStorage) FlushMain() error {
	if err := dgs.mainDB.Flush(); err != nil {
		return &util.GraphError{Type: util.ErrFlushing, Detail: err.Error()}
	}
	return nil
}

/*
StorageManager gets a storage manager with a certain name. A non-existing
StorageManager is not created automatically if the create flag is set to false.
*/
func (dgs *DiskGraphStorage) StorageManager(smname string, create bool) storage.Manager {

	sm, ok := dgs.storagemanagers[smname]

	filename := dgs.name + "/" + smname

	// Create storage manager object either if we may create or if the
	// database already exists

	if !ok && (create || storage.DataFileExist(filename)) {
		dsm := storage.NewDiskStorageManager(dgs.name+"/"+smname, false, false, false)
		sm = storage.NewCachedDiskStorageManager(dsm, 100000)
		dgs.storagemanagers[smname] = sm
	}

	return sm
}

/*
Close closes the storage.
*/
func (dgs *DiskGraphStorage) Close() error {

	var errors []string

	err := dgs.mainDB.Flush()
	if err != nil {
		errors = append(errors, err.Error())
	}

	for _, sm := range dgs.storagemanagers {
		err := sm.Close()
		if err != nil {
			errors = append(errors, err.Error())
		}
	}

	if len(errors) > 0 {
		details := fmt.Sprint(dgs.name, " :", strings.Join(errors, "; "))

		return &util.GraphError{Type: util.ErrClosing, Detail: details}
	}

	return nil
}
