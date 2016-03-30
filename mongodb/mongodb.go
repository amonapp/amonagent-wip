package main

import (
	"encoding/json"
	"fmt"
	"net/url"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/amonapp/amonagent/plugins/mongodb"
	"github.com/mitchellh/mapstructure"

	// MongoDB Driver

	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var localhost = &url.URL{Host: "127.0.0.1:27017"}

// Server - XXX
type Server struct {
	URL        *url.URL
	Session    *mgo.Session
	lastResult *mongodb.ServerStatus
}

// TableSizeData - XXX
type TableSizeData struct {
	Headers []string      `json:"headers"`
	Data    []interface{} `json:"data"`
}

// SlowQueriesData - XXX
type SlowQueriesData struct {
	Headers []string      `json:"headers"`
	Data    []interface{} `json:"data"`
}

func (p PerformanceStruct) String() string {
	s, _ := json.Marshal(p)
	return string(s)
}

// PerformanceStruct - XXX
type PerformanceStruct struct {
	TableSizeData   `json:"tables_size"`
	SlowQueriesData `json:"slow_queries"`
	Gauges          map[string]interface{} `json:"gauges"`
}

// DefaultStats - XXX
var DefaultStats = map[string]string{
	"operations.inserts_per_sec":  "Insert",
	"operations.queries_per_sec":  "Query",
	"operations.updates_per_sec":  "Update",
	"operations.deletes_per_sec":  "Delete",
	"operations.getmores_per_sec": "GetMore",
	"operations.commands_per_sec": "Command",
	"operations.flushes_per_sec":  "Flushes",
	"memory.vsize_megabytes":      "Virtual",
	"memory.resident_megabytes":   "Resident",
	"queued.reads":                "QueuedReaders",
	"queued.writes":               "QueuedWriters",
	"active.reads":                "ActiveReaders",
	"active.writes":               "ActiveWriters",
	"net.bytes_in":                "NetIn",
	"net.bytes_out":               "NetOut",
	"open_connections":            "NumConnections",
}

// DefaultReplStats - XXX
var DefaultReplStats = map[string]string{
	"replica.inserts_per_sec":  "InsertR",
	"replica.queries_per_sec":  "QueryR",
	"replica.updates_per_sec":  "UpdateR",
	"replica.deletes_per_sec":  "DeleteR",
	"replica.getmores_per_sec": "GetMoreR",
	"replica.commands_per_sec": "CommandR",
}

// MmapStats - XXX
var MmapStats = map[string]string{
	"mapped_megabytes":               "Mapped",
	"non-mapped_megabytes":           "NonMapped",
	"operations.page_faults_per_sec": "Faults",
}

// WiredTigerStats - XXX
var WiredTigerStats = map[string]string{
	"percent_cache_dirty": "CacheDirtyPercent",
	"percent_cache_used":  "CacheUsedPercent",
}

// CollectionStats - XXX
// COLLECTION_ROWS = ['count','ns','avgObjSize', 'totalIndexSize', 'indexSizes', 'size']
type CollectionStats struct {
	Count          int64            `json:"number_of_documents"`
	Ns             string           `json:"ns"`
	AvgObjSize     int64            `json:"avgObjSize"`
	TotalIndexSize int64            `json:"total_index_size"`
	StorageSize    int64            `json:"storage_size"`
	IndexSizes     map[string]int64 `json:"index_sizes"`
	Size           int64            `json:"size"`
}

// SlowQueriesStats - XXX
type SlowQueriesStats struct {
	Millis int64 `json:"millis"`
	// Ns     string `json:"ns"`
	// Op     string `json:"op"`
	// Query  string `json:"query"`
	Ts string `json:"ts"`
}

// CollectSlowQueries - XXX
func CollectSlowQueries(server *Server, perf *PerformanceStruct) error {
	// SlowQueriesHeaders := []string{"millis", "ns", "op", "query", "ts"}
	// SlowQueriesData := SlowQueriesData{Headers: SlowQueriesHeaders}

	db := strings.Replace(server.URL.Path, "/", "", -1) // remove slash from Path
	result := []bson.M{}

	params := bson.M{"millis": bson.M{"$gt": 10}}
	c := server.Session.DB(db).C("system.profile")
	err := c.Find(params).All(&result)
	if err != nil {
		return err
	}
	for _, r := range result {
		// var SlowQueriesStats SlowQueriesStats
		// decodeError := mapstructure.Decode(result, &SlowQueriesStats)
		// if decodeError != nil {
		// 	fmt.Print("Can't decode slow queries stats", decodeError.Error())
		// }
		for key, value := range r {
			fmt.Println("Key:", key, "Value:", value)
			if key == "command" {
				fmt.Print(value)
			}
		}
		fmt.Println("-----")
		// SlowQueriesData.Data = append(SlowQueriesData.Data, SlowQueriesStats)
	}

	// perf.SlowQueriesData = SlowQueriesData

	return nil
}

// CollectCollectionSize - XXX
func CollectCollectionSize(server *Server, perf *PerformanceStruct) error {
	TableSizeHeaders := []string{"count", "ns", "avgObjSize", "totalIndexSize", "storageSize", "indexSizes", "size"}
	TableSizeData := TableSizeData{Headers: TableSizeHeaders}

	db := strings.Replace(server.URL.Path, "/", "", -1) // remove slash from Path
	collections, err := server.Session.DB(db).CollectionNames()
	if err != nil {
		return err
	}
	for _, col := range collections {

		result := bson.M{}
		err := server.Session.DB(db).Run(bson.D{{"collstats", col}}, &result)

		if err != nil {
			fmt.Print("Can't get stats for collection", err.Error())
		}
		var CollectionResult CollectionStats
		decodeError := mapstructure.Decode(result, &CollectionResult)
		if decodeError != nil {
			fmt.Print("Can't decode collection stats", decodeError.Error())
		}

		TableSizeData.Data = append(TableSizeData.Data, CollectionResult)
	}

	perf.TableSizeData = TableSizeData

	return nil
}

// GetSession - XXX
func GetSession(server *Server) error {
	if server.Session == nil {
		dialInfo := &mgo.DialInfo{
			Addrs:    []string{server.URL.Host},
			Database: server.URL.Path,
		}
		dialInfo.Timeout = 10 * time.Second
		if server.URL.User != nil {
			password, _ := server.URL.User.Password()
			dialInfo.Username = server.URL.User.Username()
			dialInfo.Password = password
		}

		session, connectionError := mgo.DialWithInfo(dialInfo)
		if connectionError != nil {
			return fmt.Errorf("Unable to connect to URL (%s), %s\n", server.URL.Host, connectionError.Error())
		}
		server.Session = session
		server.lastResult = nil

		server.Session.SetMode(mgo.Eventual, true)
		server.Session.SetSocketTimeout(0)
	}

	return nil
}

// CollectGauges - XXX
func CollectGauges(server *Server, perf *PerformanceStruct) error {
	db := strings.Replace(server.URL.Path, "/", "", -1) // remove slash from Path
	result := &mongodb.ServerStatus{}
	err := server.Session.DB(db).Run(bson.D{{"serverStatus", 1}, {"recordStats", 0}}, result)
	if err != nil {
		return err
	}
	defer func() {
		server.lastResult = result
	}()

	result.SampleTime = time.Now()

	if server.lastResult != nil && result != nil {
		duration := result.SampleTime.Sub(server.lastResult.SampleTime)
		durationInSeconds := int64(duration.Seconds())
		if durationInSeconds == 0 {
			durationInSeconds = 1
		}

		data := mongodb.NewStatLine(*server.lastResult, *result, server.URL.Host, true, durationInSeconds)

		statLine := reflect.ValueOf(data).Elem()
		storageEngine := statLine.FieldByName("StorageEngine").Interface()
		// nodeType := statLine.FieldByName("NodeType").Interface()

		gauges := make(map[string]interface{})
		for key, value := range DefaultStats {
			val := statLine.FieldByName(value).Interface()
			gauges[key] = val
		}

		if storageEngine == "mmapv1" {
			for key, value := range MmapStats {
				val := statLine.FieldByName(value).Interface()
				gauges[key] = val
			}
		} else if storageEngine == "wiredTiger" {
			for key, value := range WiredTigerStats {
				val := statLine.FieldByName(value).Interface()
				percentVal := fmt.Sprintf("%.1f", val.(float64)*100)
				floatVal, _ := strconv.ParseFloat(percentVal, 64)
				gauges[key] = floatVal
			}
		}

		perf.Gauges = gauges

	}

	return nil
}

func main() {
	s := "mongodb://127.0.0.1:27017/amon"

	url, err := url.Parse(s)
	if err != nil {
		panic(err)
	}

	server := Server{URL: url}
	GetSession(&server)
	PerformanceStruct := PerformanceStruct{}
	// f := CollectGauges(&server, &PerformanceStruct)
	// time.Sleep(time.Duration(1) * time.Second)
	// f = CollectGauges(&server, &PerformanceStruct)
	// fmt.Print(f)

	CollectCollectionSize(&server, &PerformanceStruct)
	CollectSlowQueries(&server, &PerformanceStruct)
	// fmt.Print(PerformanceStruct)
	if server.Session != nil {
		defer server.Session.Close()
	}

}
