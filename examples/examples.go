package main

import (
	"log"

	"fudge"
)

func main() {
	ExampleSet()
	ExampleGet()
	ExampleDelete()
	ExampleDeleteFile()
	ExampleOpen()
	ExampleInMemoryWithoutPersist()
}

// ExampleSet lazy
func ExampleSet() {
	_ = fudge.Set("../test/test", "Hello", "World")
	defer fudge.CloseAll()
}

// ExampleGet lazy
func ExampleGet() {
	var output string
	_ = fudge.Get("../test/test", "Hello", &output)
	log.Printf("Hello: %v", output)
	// Output: World
	defer fudge.CloseAll()
}

// ExampleDelete lazy
func ExampleDelete() {
	err := fudge.Delete("../test/test", "Hello")
	if err == fudge.ErrKeyNotFound {
		log.Println(err)
	}
}

// ExampleDeleteFile lazy
func ExampleDeleteFile() {
	err := fudge.DeleteFile("../test/test")
	if err != nil {
		log.Panic(err)
	}
}

// ExampleOpen complex example
func ExampleOpen() {
	cfg := &fudge.Config{
		SyncInterval: 0} //disable every second fsync
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
	point := new(Point)
	db.Get(8, point)
	log.Printf("Point %v", *point)
	// Output: {8 8}
	// Select 2 keys, from 7 in ascending order
	keys, _ := db.Keys(7, 2, 0, true)
	for _, key := range keys {
		p := new(Point)
		db.Get(key, p)
		log.Printf("Point %v", *p)
	}
}

// ExampleInMemoryWithoutPersist -if file is empty in storemode 2 - without persist
func ExampleInMemoryWithoutPersist() {
	cfg := &fudge.Config{StoreMode: 2} //in memory
	db, err := fudge.Open("", cfg)     // if file is empty in storemode 2 - without persist
	if err != nil {
		log.Panic(err)
	}
	defer db.Close() //remove from memory
	type Point struct {
		X int
		Y int
	}
	for i := 100; i >= 0; i-- {
		p := &Point{X: i, Y: i}
		db.Set(i, p)
	}
	point := new(Point)
	db.Get(89, point)
	log.Printf("Point %v", *point)
	keys, _ := db.Keys(77, 2, 0, true)
	for _, key := range keys {
		p := new(Point)
		db.Get(key, p)
		log.Printf("Point %v", *p)
	}
}
