package fudge

import (
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"sync"
	"testing"
	"time"
)

const (
	f = "test/1"
)

func nrandbin(n int) [][]byte {
	i := make([][]byte, n)
	for ind := range i {
		bin, _ := KeyToBinary(rand.Int())
		i[ind] = bin
	}
	return i
}

func TestConfig(t *testing.T) {
	_, err := Open("", nil)
	if err == nil {
		t.Error("Open empty must error")
	}
	db, err := Open(f, &Config{FileMode: 0777, DirMode: 0777})
	if err != nil {
		t.Error(err)
	}
	err = db.DeleteFile()
	if err != nil {
		t.Error(err)
	}
}

func TestOpen(t *testing.T) {
	db, err := Open(f, nil)
	if err != nil {
		t.Error(err)
	}
	err = db.Set(1, 1)
	if err != nil {
		t.Fatal(err)
	}
	db.Close()
	db, err = Open(f, nil)
	if err != nil {
		t.Fatal(err)
	}
	var v int
	err = db.Get(1, &v)
	if err != nil {
		t.Fatal(err)
	}
	if v != 1 {
		t.Fatal("not 1")
	}
	err = db.DeleteFile()
	if err != nil {
		t.Fatal(err)
	}
}

func TestSet(t *testing.T) {
	db, err := Open(f, nil)
	if err != nil {
		t.Error(err)
	}
	err = db.Set(1, 1)
	if err != nil {
		t.Error(err)
	}
	err = db.DeleteFile()
	if err != nil {
		t.Error(err)
	}
}

func TestGet(t *testing.T) {
	db, err := Open(f, nil)
	if err != nil {
		t.Error(err)
	}
	err = db.Set(1, 1)
	if err != nil {
		t.Error(err)
	}
	var val int
	err = db.Get(1, &val)
	if err != nil {
		t.Error(err)
		return
	}

	if val != 1 {
		t.Error("val != 1", val)
		return
	}
	db.Close()

	err = db.DeleteFile()
	if err != nil {
		t.Error(err)
	}
}

func TestKeys(t *testing.T) {
	f := "test/keys.db"
	db, err := Open(f, nil)
	if err != nil {
		t.Error(err)
	}
	defer db.Close()
	append := func(i int) {
		k := fmt.Appendf([]byte{}, "%02d", i)
		v := []byte("Val:" + strconv.Itoa(i))
		db.Set(k, &v)
	}
	for i := 22; i >= 1; i-- {
		append(i)
	}

	//ascending
	res, err := db.Keys(nil, 0, 0, true)
	if err != nil {
		t.Error(err)
	}
	var s = ""
	for _, r := range res {
		s += string(r)
	}
	if s != "01020304050607080910111213141516171819202122" {
		t.Error("not asc", s)
	}

	//descending
	resdesc, _ := db.Keys(nil, 0, 0, false)
	s = ""
	for _, r := range resdesc {
		s += string(r)
	}
	if s != "22212019181716151413121110090807060504030201" {
		t.Error("not desc", s)
	}

	//offset limit asc
	reslimit, _ := db.Keys(nil, 2, 2, true)

	s = ""
	for _, r := range reslimit {
		s += string(r)
	}
	if s != "0304" {
		t.Error("not off", s)
	}

	//offset limit desc
	reslimitdesc, _ := db.Keys(nil, 2, 2, false)

	s = ""
	for _, r := range reslimitdesc {
		s += string(r)
	}
	if s != "2019" {
		t.Error("not off desc", s)
	}

	//from byte asc
	resfromasc, _ := db.Keys([]byte("10"), 2, 2, true)
	s = ""
	for _, r := range resfromasc {
		s += string(r)
	}
	if s != "1314" {
		t.Error("not off asc", s)
	}

	//from byte desc
	resfromdesc, _ := db.Keys([]byte("10"), 2, 2, false)
	s = ""
	for _, r := range resfromdesc {
		s += string(r)
	}
	if s != "0706" {
		t.Error("not off desc", s)
	}

	//from byte desc
	resnotfound, _ := db.Keys([]byte("100"), 2, 2, false)
	s = ""
	for _, r := range resnotfound {
		s += string(r)
	}
	if s != "" {
		t.Error("resnotfound", s)
	}

	//from byte not eq
	resnoteq, _ := db.Keys([]byte("33"), 2, 2, false)
	s = ""
	for _, r := range resnoteq {
		s += string(r)
	}
	if s != "" {
		t.Error("resnoteq ", s)
	}

	//by prefix
	respref, _ := db.Keys([]byte("2*"), 4, 0, false)
	s = ""
	for _, r := range respref {
		s += string(r)
	}
	if s != "222120" {
		t.Error("respref", s)
	}

	//by prefix2
	respref2, _ := db.Keys([]byte("1*"), 2, 0, false)
	s = ""
	for _, r := range respref2 {
		s += string(r)
	}
	if s != "1918" {
		t.Error("respref2", s)
	}

	//by prefixasc
	resprefasc, err := db.Keys([]byte("1*"), 2, 0, true)
	s = ""
	for _, r := range resprefasc {
		s += string(r)
	}
	if s != "1011" {
		t.Error("resprefasc", s, err)
	}

	//by prefixasc2
	resprefasc2, err := db.Keys([]byte("1*"), 0, 0, true)
	s = ""
	for _, r := range resprefasc2 {
		s += string(r)
	}
	if s != "10111213141516171819" {
		t.Error("resprefasc2", s, err)
	}
	DeleteFile(f)
}

func TestLazyOpen(t *testing.T) {
	Set(f, 2, 42)
	CloseAll()

	var val int
	Get(f, 2, &val)
	if val != 42 {
		t.Error("not 42")
	}
	DeleteFile(f)
}

func TestAsync(t *testing.T) {
	len := 5000
	file := "test/async.db"
	DeleteFile(file)
	defer CloseAll()

	messages := make(chan int)
	readmessages := make(chan string)
	var wg sync.WaitGroup

	append := func(i int) {
		defer wg.Done()
		k := fmt.Sprintf("Key:%d", i)
		v := fmt.Sprintf("Val:%d", i)
		err := Set(file, k, &v)
		if err != nil {
			t.Error(err)
		}
		messages <- i
	}

	read := func(i int) {
		defer wg.Done()
		k := fmt.Sprintf("Key:%d", i)
		v := fmt.Sprintf("Val:%d", i)
		var b string
		Get(file, k, &b)
		if v != b {
			t.Error("not mutch", v, b)
		}
		readmessages <- fmt.Sprintf("read N:%d  content:%s", i, b)
	}

	for i := 1; i <= len; i++ {
		wg.Add(1)
		go append(i)
	}

	go func() {
		for i := range messages {
			_ = i
			//fmt.Println(i)
		}
	}()

	go func() {
		for i := range readmessages {
			_ = i
			//fmt.Println(i)
		}
	}()

	wg.Wait()

	for i := 1; i <= len; i++ {
		wg.Add(1)
		go read(i)
	}
	wg.Wait()
	DeleteFile(file)
}

func TestStoreMode(t *testing.T) {
	cfg := &Config{StoreMode: 2}
	db, err := Open("test/sm", cfg)
	if err != nil {
		t.Error(err)
	}
	err = db.Set(1, 2)
	if err != nil {
		t.Error(err)
	}
	var v int
	err = db.Get(1, &v)
	if err != nil {
		t.Error(err)
	}
	if v != 2 {
		t.Error("not 2")
	}
	db.Set(1, 42)
	db.Close()
	db, err = Open("test/sm", nil)
	if err != nil {
		t.Error(err)
	}
	var val int
	err = db.Get(1, &val)
	if err != nil {
		t.Error(err)
	}

	if val != 42 {
		t.Error("not 42")
	}
	DeleteFile("test/sm")
	CloseAll()
}

// run go test -bench=Store -benchmem
func BenchmarkStore(b *testing.B) {
	b.StopTimer()
	nums := nrandbin(b.N)
	DeleteFile(f)
	rm, err := Open(f, nil)
	if err != nil {
		b.Error("Open", err)
	}
	b.SetBytes(8)
	b.StartTimer()
	for _, n := range nums {
		err = rm.Set(n, n)
		if err != nil {
			b.Error("Set", err)
		}
	}
	b.StopTimer()
	err = DeleteFile(f)
	if err != nil {
		b.Error("DeleteFile", err)
	}
}

func BenchmarkLoad(b *testing.B) {
	b.StopTimer()
	nums := nrandbin(b.N)
	DeleteFile(f)
	rm, err := Open(f, nil)
	if err != nil {
		b.Error("Open", err)
	}
	for _, n := range nums {
		err = rm.Set(n, n)
		if err != nil {
			b.Error("Set", err)
		}
	}
	var wg sync.WaitGroup
	read := func(db *DB, key []byte) {
		defer wg.Done()
		var b []byte
		db.Get(key, &b)
	}
	b.StartTimer()
	for i := 0; b.Loop(); i++ {
		wg.Add(1)
		go read(rm, nums[i])
		//var v []byte
		//err := rm.Get(nums[i], &v)
		//if err != nil {
		//	log.Println("Get", err, nums[i], &v)
		//	break
		//}
	}
	wg.Wait()
	b.StopTimer()
	log.Println(rm.Count())
	DeleteFile(f)
	CloseAll()
}

func TestBackup(t *testing.T) {
	Set("test/1", 1, 2)
	Set("test/4", "4", "4")
	BackupAll("")
	DeleteFile("test/1")
	DeleteFile("test/4")

	var v1 int
	Get("backup/test/1", 1, &v1)
	if v1 != 2 {
		t.Error("not 2")
	}

	var v2 string
	Get("backup/test/4", "4", &v2)
	if v2 != "4" {
		t.Error("not 4")
	}

	DeleteFile("backup/test/1")
	DeleteFile("backup/test/4")
	CloseAll()
}

func TestMultipleOpen(t *testing.T) {
	for i := 1; i < 10000; i++ {
		Set("test/m", i, i)
	}
	Close("test/m")
	for i := 1; i < 100; i++ {
		go Open("test/m", nil)
	}
	time.Sleep(1 * time.Millisecond)
	DeleteFile("test/m")
}

func TestInMemory(t *testing.T) {
	DefaultConfig.StoreMode = 2

	for i := range 10 {
		fileName := fmt.Sprintf("test/inmemory%d", i)
		err := Set(fileName, i, i)
		if err != nil {
			t.Error(err)
		}
	}

	err := CloseAll()
	if err != nil {
		t.Error(err)
	}
	for i := range 10 {
		fileName := fmt.Sprintf("test/inmemory%d", i)
		c, e := Count(fileName)
		if c == 0 || e != nil {
			t.Error("no persist")
			break
		}
		DeleteFile(fileName)
	}
}

func TestInMemoryWithoutPersist(t *testing.T) {
	DefaultConfig.StoreMode = 2

	for i := range 10000 {
		err := Set("", i, i)
		if err != nil {
			t.Error(err)
		}
	}

	var v int
	Get("", 6, &v)
	if v != 6 {
		t.Error("v must be 6", v)
	}
	cnt, e := Count("")
	if cnt != 10000 {
		t.Error("count must be 10000", cnt, e)
	}
	for range 10000 {
		c, e := Count("")
		if c != 10000 || e != nil {
			t.Error("no persist", c, e)
			break
		}
	}
	noerr := DeleteFile("")
	if noerr != nil {
		t.Error("Delete empty file", noerr)
	}
	noerr = Close("")
	if noerr != nil {
		t.Error("Close empty file", noerr)
	}
	var n int
	notpresent := Get("", 8, &n)
	if n == 8 {
		t.Error("n  must be 0", n)
	}
	if notpresent != ErrKeyNotFound {
		t.Error("Must be Error: key not found error", notpresent)
	}
}

func Test42(t *testing.T) {
	DefaultConfig.StoreMode = 0
	f := "test/int64"
	for i := 1; i < 64; i++ {
		Set(f, int64(i), int64(i))
	}
	keys, err := Keys(f, int64(42), 100, 0, true)
	if err != nil {
		t.Error(err)
	}
	if len(keys) != 21 {
		t.Error("not 21", len(keys))
	}
	DeleteFile(f)
}

func TestSetsGets(t *testing.T) {
	f := "test/setsgets"
	DeleteFile(f)
	var pairs = make([]any, 0)
	for i := 1; i < 64; i++ {
		pairs = append(pairs, i)
		pairs = append(pairs, i+1)
	}
	err := Sets(f, pairs)
	if err != nil {
		t.Error("Sets err", err, pairs)
	}
	var v int

	err = Get(f, 63, &v)
	if err != nil || v != 64 {
		t.Error("Sets 64 err", err, v)
	}
	//Sets
	var pairsBin []any
	for i := range 100 {
		k := fmt.Appendf([]byte{}, "%04d", i)
		pairsBin = append(pairsBin, k)
		pairsBin = append(pairsBin, k)
	}
	err = Sets(f, pairsBin)
	if err != nil {
		t.Error("Sets err", err)
	}
	var s []byte
	err = Get(f, []byte("0063"), &s)
	if err != nil || string(s) != "0063" {
		t.Error("Sets err", err, s)
	}
	var keys []any
	for i := 2; i < 4; i++ {
		k := fmt.Appendf([]byte{}, "%04d", i)
		keys = append(keys, k)
	}
	err = Get(f, []byte("0068"), &s)
	if err != nil {
		t.Error("Sets err", err)
	}

	result := Gets(f, keys)
	if len(result) != 4 {
		t.Error("Sets err not 4")
	}
	DeleteFile(f)
}

func TestEmptyKeysByPrefix(t *testing.T) {
	// first pass with asc = false
	db, err := Open(f, nil)
	if err != nil {
		t.Error(err)
	}

	prefix, err := KeyToBinary("non-existant-prefix")
	if err != nil {
		t.Error(err)
	}

	keys, err := db.KeysByPrefix(prefix, 0, 0, false)
	if err != ErrKeyNotFound {
		t.Errorf("Error must be ErrKeyNotFound got: %s", err)
	}

	if len(keys) != 0 {
		t.Errorf("Wrong amount of keys for empty database: %d", len(keys))
	}

	db.Set("some-key", "some-value")

	keys, err = db.KeysByPrefix(prefix, 0, 0, false)
	if err != ErrKeyNotFound {
		t.Errorf("Error must be ErrKeyNotFound got: %s", err)
	}

	if len(keys) != 0 {
		t.Errorf("Wrong amount of keys: %d", len(keys))
	}

	err = db.DeleteFile()
	if err != nil {
		t.Fatal(err)
	}

	// second pass with asc = true
	db, err = Open(f, nil)
	if err != nil {
		t.Error(err)
	}

	prefix, err = KeyToBinary("non-existant-prefix")
	if err != nil {
		t.Error(err)
	}

	keys, err = db.KeysByPrefix(prefix, 0, 0, true)
	if err != ErrKeyNotFound {
		t.Errorf("Error must be ErrKeyNotFound got: %s", err)
	}

	if len(keys) != 0 {
		t.Errorf("Wrong amount of keys for empty database: %d", len(keys))
	}

	db.Set("some-key", "some-value")

	keys, err = db.KeysByPrefix(prefix, 0, 0, true)
	if err != ErrKeyNotFound {
		t.Errorf("Error must be ErrKeyNotFound got: %s", err)
	}

	if len(keys) != 0 {
		t.Errorf("Wrong amount of keys: %d", len(keys))
	}

	// third pass with non-zero offset
	db, err = Open(f, nil)
	if err != nil {
		t.Error(err)
	}

	prefix, err = KeyToBinary("non-existant-prefix")
	if err != nil {
		t.Error(err)
	}

	keys, err = db.KeysByPrefix(prefix, 0, 1, false)
	if err != ErrKeyNotFound {
		t.Errorf("Error must be ErrKeyNotFound got: %s", err)
	}

	if len(keys) != 0 {
		t.Errorf("Wrong amount of keys for empty database: %d", len(keys))
	}

	db.Set("some-key", "some-value")

	keys, err = db.KeysByPrefix(prefix, 0, 1, false)
	if err != ErrKeyNotFound {
		t.Errorf("Error must be ErrKeyNotFound got: %s", err)
	}

	if len(keys) != 0 {
		t.Errorf("Wrong amount of keys: %d", len(keys))
	}

	err = db.DeleteFile()
	if err != nil {
		t.Fatal(err)
	}
}
