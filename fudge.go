package fudge

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

var (
	dbs struct {
		sync.RWMutex
		dbs map[string]*DB
	}

	// ErrKeyNotFound - key not found
	ErrKeyNotFound = errors.New("error: key not found")
)

// DB represent database
type DB struct {
	sync.RWMutex
	name         string
	fk           *os.File
	fv           *os.File
	keys         [][]byte
	vals         map[string]*Cmd
	cancelSyncer context.CancelFunc
	storemode    int
}

// Cmd represent keys and vals addresses
type Cmd struct {
	Seek    uint32
	Size    uint32
	KeySeek uint32
	Val     []byte
}

// Config fo db
// Default FileMode = 0644
// Default DirMode = 0755
// Default SyncInterval = 0 sec, 0 - disable sync (os will sync, typically 30 sec or so)
// If StroreMode==2 && file == "" - pure inmemory mode
type Config struct {
	FileMode     int // 0644
	DirMode      int // 0755
	SyncInterval int // in seconds
	StoreMode    int // 0 - file first, 2 - memory first(with persist on close), 2 - with empty file - memory without persist
}

func init() {
	dbs.dbs = make(map[string]*DB)
}

func newDB(f string, cfg *Config) (*DB, error) {
	var err error
	// create
	db := new(DB)
	db.Lock()
	defer db.Unlock()
	// init
	db.name = f
	db.keys = make([][]byte, 0)
	db.vals = make(map[string]*Cmd)
	db.storemode = cfg.StoreMode

	// Apply default values
	if cfg.FileMode == 0 {
		cfg.FileMode = DefaultConfig.FileMode
	}
	if cfg.DirMode == 0 {
		cfg.DirMode = DefaultConfig.DirMode
	}
	if db.storemode == 2 && db.name == "" {
		return db, nil
	}
	_, err = os.Stat(f)
	if err != nil {
		// file not exists - create dirs if any
		if os.IsNotExist(err) {
			if filepath.Dir(f) != "." {
				err = os.MkdirAll(filepath.Dir(f), os.FileMode(cfg.DirMode))
				if err != nil {
					return nil, err
				}
			}
		} else {
			return nil, err
		}
	}
	db.fv, err = os.OpenFile(f, os.O_CREATE|os.O_RDWR, os.FileMode(cfg.FileMode))
	if err != nil {
		return nil, err
	}
	db.fk, err = os.OpenFile(f+".idx", os.O_CREATE|os.O_RDWR, os.FileMode(cfg.FileMode))
	if err != nil {
		return nil, err
	}
	//read keys
	buf := new(bytes.Buffer)
	b, err := io.ReadAll(db.fk)
	if err != nil {
		return nil, err
	}
	buf.Write(b)
	var readSeek uint32
	for buf.Len() > 0 {
		_ = uint8(buf.Next(1)[0]) //format version
		t := uint8(buf.Next(1)[0])
		seek := binary.BigEndian.Uint32(buf.Next(4))
		size := binary.BigEndian.Uint32(buf.Next(4))
		_ = buf.Next(4) //time
		sizeKey := int(binary.BigEndian.Uint16(buf.Next(2)))
		key := buf.Next(sizeKey)
		strkey := string(key)
		cmd := &Cmd{
			Seek:    seek,
			Size:    size,
			KeySeek: readSeek,
		}
		if db.storemode == 2 {
			cmd.Val = make([]byte, size)
			_, _ = db.fv.ReadAt(cmd.Val, int64(seek))
		}
		readSeek += uint32(16 + sizeKey)
		switch t {
		case 0:
			if _, exists := db.vals[strkey]; !exists {
				//write new key at keys store
				db.appendKey(key)
			}
			db.vals[strkey] = cmd
		case 1:
			delete(db.vals, strkey)
			db.deleteFromKeys(key)
		}
	}

	if cfg.SyncInterval > 0 {
		db.backgroundManager(cfg.SyncInterval)
	}
	return db, err
}

// backgroundManager runs continuously in the background and performs various
// operations such as syncing to disk.
func (db *DB) backgroundManager(interval int) {
	ctx, cancel := context.WithCancel(context.Background())
	db.cancelSyncer = cancel
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			default:
				db.Lock()
				_ = db.fk.Sync()
				_ = db.fv.Sync()
				db.Unlock()
				time.Sleep(time.Duration(interval) * time.Second)
			}
		}
	}()
}

// appendKey insert key in slice
func (db *DB) appendKey(b []byte) {
	//log.Println("append")
	db.keys = append(db.keys, b)
}

// deleteFromKeys delete key from slice keys
func (db *DB) deleteFromKeys(b []byte) {
	found := db.found(b, true)
	if found < len(db.keys) {
		if bytes.Equal(db.keys[found], b) {
			db.keys = append(db.keys[:found], db.keys[found+1:]...)
		}
	}
}

func (db *DB) sort() {
	if !sort.SliceIsSorted(db.keys, db.lessBinary) {
		//log.Println("sort")
		sort.Slice(db.keys, db.lessBinary)
	}
}

func (db *DB) lessBinary(i, j int) bool {
	return bytes.Compare(db.keys[i], db.keys[j]) <= 0
}

// found return binary search result with sort order
func (db *DB) found(b []byte, _ bool) int {
	db.sort()
	//if asc {
	return sort.Search(len(db.keys), func(i int) bool {
		return bytes.Compare(db.keys[i], b) >= 0
	})
	//}
	//return sort.Search(len(db.keys), func(i int) bool {
	//	return bytes.Compare(db.keys[i], b) <= 0
	//})
}

func writeKeyVal(fk, fv *os.File, readKey, writeVal []byte, exists bool, oldCmd *Cmd) (cmd *Cmd, err error) {

	var seek, newSeek int64
	cmd = &Cmd{Size: uint32(len(writeVal))}
	if exists {
		// key exists
		cmd.Seek = oldCmd.Seek
		cmd.KeySeek = oldCmd.KeySeek
		if oldCmd.Size >= uint32(len(writeVal)) {
			//write at old seek new value
			_, _, err = writeAtPos(fv, writeVal, int64(oldCmd.Seek))
		} else {
			//write at new seek (at the end of file)
			seek, _, err = writeAtPos(fv, writeVal, int64(-1))
			cmd.Seek = uint32(seek)
		}
		if err == nil {
			// if no error - store key at KeySeek
			newSeek, err = writeKey(fk, 0, cmd.Seek, cmd.Size, []byte(readKey), int64(cmd.KeySeek))
			cmd.KeySeek = uint32(newSeek)
		}
	} else {
		// new key
		// write value at the end of file
		seek, _, err = writeAtPos(fv, writeVal, int64(-1))
		cmd.Seek = uint32(seek)
		if err == nil {
			newSeek, err = writeKey(fk, 0, cmd.Seek, cmd.Size, []byte(readKey), -1)
			cmd.KeySeek = uint32(newSeek)
		}
	}
	return cmd, err
}

// if pos<0 store at the end of file
func writeAtPos(f *os.File, b []byte, pos int64) (seek int64, n int, err error) {
	seek = pos
	if pos < 0 {
		seek, err = f.Seek(0, 2)
		if err != nil {
			return seek, 0, err
		}
	}
	n, err = f.WriteAt(b, seek)
	if err != nil {
		return seek, n, err
	}
	return seek, n, err
}

// writeKey create buffer and store key with val address and size
func writeKey(fk *os.File, t uint8, seek, size uint32, key []byte, keySeek int64) (newSeek int64, err error) {
	//get buf from pool
	buf := new(bytes.Buffer)
	buf.Reset()
	buf.Grow(16 + len(key))

	//encode
	_ = binary.Write(buf, binary.BigEndian, uint8(0))                  //1byte version
	_ = binary.Write(buf, binary.BigEndian, t)                         //1byte command code(0-set,1-delete)
	_ = binary.Write(buf, binary.BigEndian, seek)                      //4byte seek
	_ = binary.Write(buf, binary.BigEndian, size)                      //4byte size
	_ = binary.Write(buf, binary.BigEndian, uint32(time.Now().Unix())) //4byte timestamp
	_ = binary.Write(buf, binary.BigEndian, uint16(len(key)))          //2byte key size
	_, _ = buf.Write(key)                                              //key

	if keySeek < 0 {
		newSeek, _, err = writeAtPos(fk, buf.Bytes(), int64(-1))
	} else {
		newSeek, _, err = writeAtPos(fk, buf.Bytes(), int64(keySeek))
	}

	return newSeek, err
}

// findKey return index of first key in ascending mode
// findKey return index of last key in descending mode
// findKey return 0 or len-1 in case of nil key
func (db *DB) findKey(key any, asc bool) (int, error) {
	if key == nil {
		db.sort()
		if asc {
			return 0, ErrKeyNotFound
		}
		return len(db.keys) - 1, ErrKeyNotFound
	}
	k, err := KeyToBinary(key)
	if err != nil {
		return -1, err
	}
	found := db.found(k, asc)
	//log.Println("found", found)
	// check found
	if found >= len(db.keys) {
		return -1, ErrKeyNotFound
	}
	if !bytes.Equal(db.keys[found], k) {
		return -1, ErrKeyNotFound
	}
	return found, nil
}

// startFrom return is a start from b in binary
func startFrom(a, b []byte) bool {
	if a == nil || b == nil {
		return false
	}
	if len(a) < len(b) {
		return false
	}
	return bytes.Equal(a[:len(b)], b)
}

func (db *DB) foundPref(b []byte, asc bool) int {
	db.sort()
	if asc {
		return sort.Search(len(db.keys), func(i int) bool {
			return bytes.Compare(db.keys[i], b) >= 0
		})
	}
	var j int
	for j = len(db.keys) - 1; j >= 0; j-- {
		if startFrom(db.keys[j], b) {
			break
		}
	}
	return j
}

func checkInterval(find, limit, offset, excludeFrom, len int, asc bool) (int, int) {
	end := 0
	start := find

	if asc {
		start += (offset + excludeFrom)
		if limit == 0 {
			end = len - excludeFrom
		} else {
			end = (start + limit - 1)
		}
	} else {
		start -= (offset + excludeFrom)
		if limit == 0 {
			end = 0
		} else {
			end = start - limit + 1
		}
	}

	if end < 0 {
		end = 0
	}
	if end >= len {
		end = len - 1
	}

	return start, end
}
