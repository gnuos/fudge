# Fudge - Compact key/value store

Table of Contents
=================

* [Description](#description)
* [Usage](#usage)
* [Cookbook](#cookbook)
* [Disadvantages](#disadvantages)
* [Motivation](#motivation)


## Description

Package fudge is a fast and simple key/value store written using Go's standard library.

Fork from [pudge](https://github.com/recoilme/pudge)

It presents the following:
* Supporting very efficient lookup, insertions and deletions
* Performance is comparable to hash tables
* Ability to get the data in sorted order, which enables additional operations like range scan
* Select with limit/offset/from key, with ordering or by prefix
* Safe for use in goroutines
* Space efficient
* Very short and simple codebase
* Well tested, used in production

这个包对于我来说最大的好处是单文件存储并且能快速存取少量数据。


## Usage


```golang
package main

import (
	"log"

	"github.com/gnuos/fudge"
)

func main() {
	// Close all database on exit
	defer fudge.CloseAll()

	// Set (directories will be created)
	fudge.Set("../test/test", "Hello", "World")

	// Get (lazy open db if needed)
	var output string
	fudge.Get("../test/test", "Hello", &output)
	log.Println("Output:", output)

	ExampleSelect()
}

//ExampleSelect
func ExampleSelect() {
	cfg := &fudge.Config{ SyncInterval: 1 } // every second fsync
	db, err := fudge.Open("../test/db", cfg)
	if err != nil {
		log.Panic(err)
	}
	defer db.DeleteFile()
	type Point struct {
		X int
		Y int
	}
	for i := 100; i >= 0; i-- {
		p := &Point{X: i, Y: i}
		db.Set(i, p)
	}
	var point Point
	db.Get(8, &point)
	log.Printf("%v", point)
	// Output: {8 8}
	// Select 2 keys, from 7 in ascending order
	keys, _ := db.Keys(7, 2, 0, true)
	for _, key := range keys {
		var p Point
		db.Get(key, &p)
		log.Println(p)
	}
	// Output: {8 8}
	// Output: {9 9}
}

```

## Cookbook

 - Store data of any type. Fudge uses CBOR encoder/decoder internally. No limits on keys/values size.

```golang
fudge.Set("strings", "Hello", "World")
fudge.Set("numbers", 1, 42)

type User struct {
	Id int
	Name string
}
u := &User{Id: 1, Name: "name"}
fudge.Set("users", u.Id, u)

```
 - Fudge is stateless and safe for use in goroutines. You don't need to create/open files before use. Just write data to fudge, don't worry about state.

 - Fudge is parallel. Readers don't block readers, but a writer - does, but by the stateless nature of fudge it's safe to use multiples files for storages.

 - Default store system: like memcache + file storage. Fudge uses in-memory hashmap for keys, and writes values to files (no value data stored in memory). But you may use inmemory mode for values, with custom config:
```golang
cfg = fudge.DefaultConfig()
cfg.StoreMode = 2
db, err := fudge.Open(dbPrefix+"/"+group, cfg)
...
db.Counter(key, val)
```
In that case, all data is stored in memory and will be stored on disk only on Close. 


 - Don't forget to close all opened databases on shutdown/kill.
```golang
 	// Wait for interrupt signal to gracefully shutdown the server 
	quit := make(chan os.Signal)
	signal.Notify(quit, os.Interrupt, os.Kill)
	<-quit
	log.Println("Shutdown Server ...")
	if err := fudge.CloseAll(); err != nil {
		log.Println("Fudge Shutdown err:", err)
	}
 ```

 - example recovery function for gin framework
```golang
func globalRecover(c *gin.Context) {
	defer func(c *gin.Context) {

		if err := recover(); err != nil {
			if err := fudge.CloseAll(); err != nil {
				log.Println("Database Shutdown err:", err)
			}
			log.Println("Server recovery with err:", err)
			gin.RecoveryWithWriter(gin.DefaultErrorWriter)
		}
	}(c)
	c.Next()
}
	
```


 - Fudge has a primitive select/query engine.
 ```golang
 // Select 2 keys, from 7 in ascending order
	keys, _ := db.Keys(7, 2, 0, true)
// select keys from db where key>7 order by keys asc limit 2 offset 0
 ```

 - Fudge will work well on SSD or spined disks. Fudge doesn't eat memory or storage or your sandwich. No hidden compaction/rebalancing/resizing and so on tasks. No LSM Tree. No MMap. It's a very simple database with less than 500 LOC.


## Disadvantages

 - No transaction system. All operations are isolated, but you don't may batching them with automatic rollback.
 - Keys function (select/query engine) may be slow. Speed of query may vary from 10ms to 1sec per million keys. Fudge don't use BTree/Skiplist or Adaptive radix tree for store keys in ordered way on every insert. Ordering operation is "lazy" and run only if needed.
 - No fsync on every insert. Most of database fsync data by the timer too
 - Deleted data don't remove from physically (but upsert will try to reuse space). You may shrink database only with backup right now
```golang
fudge.BackupAll("backup")
```
 - Keys automatically convert to binary and ordered with binary comparator. It's simple for use, but ordering will not work correctly for negative numbers for example
 - Author of project don't work at Google or Facebook and his name not Howard Chu or Brad Fitzpatrick. But I'm open for issue or contributions.


## Motivation

Some databases very well for writing. Some of the databases very well for reading. But fudge is well balanced for both types of operations.
It has small api, and don't have hidden graveyards. It's just hashmap where values written in files.
And you may use one database for in-memory/persistent storage in a stateless stressfree way.


## Licence

MIT
