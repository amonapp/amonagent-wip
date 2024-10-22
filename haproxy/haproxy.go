package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
)

var haproxyType = map[string]string{
	"0": "frontend",
	"1": "backend",
	"2": "server",
	"3": "listen",
}

//CSV format: https://cbonte.github.io/haproxy-dconv/configuration-1.5.html#9.1
const (
	HF_PXNAME         = 0  // 0. pxname [LFBS]: proxy name
	HF_SVNAME         = 1  // 1. svname [LFBS]: service name (FRONTEND for frontend, BACKEND for backend, any name for server/listener)
	HF_QCUR           = 2  //2. qcur [..BS]: current queued requests. For the backend this reports the number queued without a server assigned.
	HF_QMAX           = 3  //3. qmax [..BS]: max value of qcur
	HF_SCUR           = 4  // 4. scur [LFBS]: current sessions
	HF_SMAX           = 5  //5. smax [ LFBS]: max sessions
	HF_SLIM           = 6  //6. slim [LFBS]: configured session limit
	HF_STOT           = 7  //7. stot [LFBS]: cumulative number of connections
	HF_BIN            = 8  //8. bin [LFBS]: bytes in
	HF_BOUT           = 9  //9. bout [LFBS]: bytes out
	HF_DREQ           = 10 //10. dreq [LFB.]: requests denied because of security concerns.
	HF_DRESP          = 11 //11. dresp [LFBS]: responses denied because of security concerns.
	HF_EREQ           = 12 //12. ereq [LF..]: request errors. Some of the possible causes are:
	HF_ECON           = 13 //13. econ [..BS]: number of requests that encountered an error trying to
	HF_ERESP          = 14 //14. eresp [..BS]: response errors. srv_abrt will be counted here also.  Some other errors are: - write error on the client socket (won't be counted for the server stat) - failure applying filters to the response.
	HF_WRETR          = 15 //15. wretr [..BS]: number of times a connection to a server was retried.
	HF_WREDIS         = 16 //16. wredis [..BS]: number of times a request was redispatched to another server. The server value counts the number of times that server was switched away from.
	HF_STATUS         = 17 //17. status [LFBS]: status (UP/DOWN/NOLB/MAINT/MAINT(via)...)
	HF_WEIGHT         = 18 //18. weight [..BS]: total weight (backend), server weight (server)
	HF_ACT            = 19 //19. act [..BS]: number of active servers (backend), server is active (server)
	HF_BCK            = 20 //20. bck [..BS]: number of backup servers (backend), server is backup (server)
	HF_CHKFAIL        = 21 //21. chkfail [...S]: number of failed checks. (Only counts checks failed when the server is up.)
	HF_CHKDOWN        = 22 //22. chkdown [..BS]: number of UP->DOWN transitions. The backend counter counts transitions to the whole backend being down, rather than the sum of the counters for each server.
	HF_LASTCHG        = 23 //23. lastchg [..BS]: number of seconds since the last UP<->DOWN transition
	HF_DOWNTIME       = 24 //24. downtime [..BS]: total downtime (in seconds). The value for the backend is the downtime for the whole backend, not the sum of the server downtime.
	HF_QLIMIT         = 25 //25. qlimit [...S]: configured maxqueue for the server, or nothing in the value is 0 (default, meaning no limit)
	HF_PID            = 26 //26. pid [LFBS]: process id (0 for first instance, 1 for second, ...)
	HF_IID            = 27 //27. iid [LFBS]: unique proxy id
	HF_SID            = 28 //28. sid [L..S]: server id (unique inside a proxy)
	HF_THROTTLE       = 29 //29. throttle [...S]: current throttle percentage for the server, when slowstart is active, or no value if not in slowstart.
	HF_LBTOT          = 30 //30. lbtot [..BS]: total number of times a server was selected, either for new sessions, or when re-dispatching. The server counter is the number of times that server was selected.
	HF_TRACKED        = 31 //31. tracked [...S]: id of proxy/server if tracking is enabled.
	HF_TYPE           = 32 //32. type [LFBS]: (0                                                                                                                                                                                                  = frontend, 1 = backend, 2 = server, 3 = socket/listener)
	HF_RATE           = 33 //33. rate [.FBS]: number of sessions per second over last elapsed second
	HF_RATE_LIM       = 34 //34. rate_lim [.F..]: configured limit on new sessions per second
	HF_RATE_MAX       = 35 //35. rate_max [.FBS]: max number of new sessions per second
	HF_CHECK_STATUS   = 36 //36. check_status [...S]: status of last health check, one of:
	HF_CHECK_CODE     = 37 //37. check_code [...S]: layer5-7 code, if available
	HF_CHECK_DURATION = 38 //38. check_duration [...S]: time in ms took to finish last health check
	HF_HRSP_1xx       = 39 //39. hrsp_1xx [.FBS]: http responses with 1xx code
	HF_HRSP_2xx       = 40 //40. hrsp_2xx [.FBS]: http responses with 2xx code
	HF_HRSP_3xx       = 41 //41. hrsp_3xx [.FBS]: http responses with 3xx code
	HF_HRSP_4xx       = 42 //42. hrsp_4xx [.FBS]: http responses with 4xx code
	HF_HRSP_5xx       = 43 //43. hrsp_5xx [.FBS]: http responses with 5xx code
	HF_HRSP_OTHER     = 44 //44. hrsp_other [.FBS]: http responses with other codes (protocol error)
	HF_HANAFAIL       = 45 //45. hanafail [...S]: failed health checks details
	HF_REQ_RATE       = 46 //46. req_rate [.F..]: HTTP requests per second over last elapsed second
	HF_REQ_RATE_MAX   = 47 //47. req_rate_max [.F..]: max number of HTTP requests per second observed
	HF_REQ_TOT        = 48 //48. req_tot [.F..]: total number of HTTP requests received
	HF_CLI_ABRT       = 49 //49. cli_abrt [..BS]: number of data transfers aborted by the client
	HF_SRV_ABRT       = 50 //50. srv_abrt [..BS]: number of data transfers aborted by the server (inc. in eresp)
	HF_COMP_IN        = 51 //51. comp_in [.FB.]: number of HTTP response bytes fed to the compressor
	HF_COMP_OUT       = 52 //52. comp_out [.FB.]: number of HTTP response bytes emitted by the compressor
	HF_COMP_BYP       = 53 //53. comp_byp [.FB.]: number of bytes that bypassed the HTTP compressor (CPU/BW limit)
	HF_COMP_RSP       = 54 //54. comp_rsp [.FB.]: number of HTTP responses that were compressed
	HF_LASTSESS       = 55 //55. lastsess [..BS]: number of seconds since last session assigned to server/backend
	HF_LAST_CHK       = 56 //56. last_chk [...S]: last health check contents or textual error
	HF_LAST_AGT       = 57 //57. last_agt [...S]: last agent check contents or textual error
	HF_QTIME          = 58 //58. qtime [..BS]:
	HF_CTIME          = 59 //59. ctime [..BS]:
	HF_RTIME          = 60 //60. rtime [..BS]: (0 for TCP)
	HF_TTIME          = 61 //61. ttime [..BS]: the average total session time in ms over the 1024 last requests
)

// ParseCSVResult - XXX
func ParseCSVResult(r io.Reader, host string) error {
	csv := csv.NewReader(r)
	result, err := csv.ReadAll()
	if err != nil {
		return fmt.Errorf("Unable to Parse HAProxy CSV: %s", err.Error())
	}

	gauges := make(map[string]interface{})
	counters := make(map[string]interface{})
	for _, row := range result {
		HostName := row[HF_SVNAME]
		ProxyName := row[HF_PXNAME]
		// ServerType := row[HF_TYPE]

		Key := fmt.Sprintf("%s.%s", ProxyName, HostName)

		for field, v := range row {
			switch field {
			case HF_QCUR:
				ival, err := strconv.ParseUint(v, 10, 64)
				if err == nil {
					GaugeKey := fmt.Sprintf("queue.current.%s", Key)
					gauges[GaugeKey] = ival
				}
			case HF_QLIMIT:
				ival, err := strconv.ParseUint(v, 10, 64)
				if err == nil {
					GaugeKey := fmt.Sprintf("queue.max.%s", Key)
					gauges[GaugeKey] = ival
				}
			case HF_SCUR:
				ival, err := strconv.ParseUint(v, 10, 64)
				if err == nil {
					GaugeKey := fmt.Sprintf("session.current.%s", Key)
					gauges[GaugeKey] = ival
				}
			case HF_SMAX:
				ival, err := strconv.ParseUint(v, 10, 64)
				if err == nil {
					GaugeKey := fmt.Sprintf("session.max.%s", Key)
					gauges[GaugeKey] = ival
				}

			case HF_REQ_RATE:
				ival, err := strconv.ParseUint(v, 10, 64)
				if err == nil {
					GaugeKey := fmt.Sprintf("requests.rate.%s", Key)
					gauges[GaugeKey] = ival
				}

			case HF_REQ_RATE_MAX:
				ival, err := strconv.ParseUint(v, 10, 64)
				if err == nil {
					GaugeKey := fmt.Sprintf("requests.rate_peak.%s", Key)
					gauges[GaugeKey] = ival
				}

			case HF_RTIME:
				ival, err := strconv.ParseUint(v, 10, 64)
				if err == nil {
					GaugeKey := fmt.Sprintf("response.time.%s", Key)
					gauges[GaugeKey] = ival
				}

			case HF_TTIME:
				ival, err := strconv.ParseUint(v, 10, 64)
				if err == nil {
					GaugeKey := fmt.Sprintf("session_time.%s", Key)
					gauges[GaugeKey] = ival
				}

			case HF_BIN:
				ival, err := strconv.ParseUint(v, 10, 64)
				if err == nil {
					GaugeKey := fmt.Sprintf("bytes.in.%s", Key)
					gauges[GaugeKey] = ival
				}
			case HF_BOUT:
				ival, err := strconv.ParseUint(v, 10, 64)
				if err == nil {
					GaugeKey := fmt.Sprintf("bytes.out.%s", Key)
					gauges[GaugeKey] = ival
				}

			case HF_HRSP_1xx:
				ival, err := strconv.ParseUint(v, 10, 64)
				if err == nil {
					CounterKey := fmt.Sprintf("http_response.1xx.%s", Key)
					counters[CounterKey] = ival
				}
			case HF_HRSP_2xx:
				ival, err := strconv.ParseUint(v, 10, 64)
				if err == nil {
					CounterKey := fmt.Sprintf("http_response.2xx.%s", Key)
					counters[CounterKey] = ival
				}
			case HF_HRSP_3xx:
				ival, err := strconv.ParseUint(v, 10, 64)
				if err == nil {
					CounterKey := fmt.Sprintf("http_response.3xx.%s", Key)
					counters[CounterKey] = ival
				}
			case HF_HRSP_4xx:
				ival, err := strconv.ParseUint(v, 10, 64)
				if err == nil {
					CounterKey := fmt.Sprintf("http_response.4xx.%s", Key)
					counters[CounterKey] = ival
				}
			case HF_HRSP_5xx:
				ival, err := strconv.ParseUint(v, 10, 64)
				if err == nil {
					CounterKey := fmt.Sprintf("http_response.5xx.%s", Key)
					counters[CounterKey] = ival
				}

			case HF_ECON:
				ival, err := strconv.ParseUint(v, 10, 64)
				if err == nil {
					CounterKey := fmt.Sprintf("error_connection_count.%s", Key)
					counters[CounterKey] = ival
				}

			case HF_DREQ:
				ival, err := strconv.ParseUint(v, 10, 64)
				if err == nil {
					CounterKey := fmt.Sprintf("denied_request_count.%s", Key)
					counters[CounterKey] = ival
				}

			case HF_DRESP:
				ival, err := strconv.ParseUint(v, 10, 64)
				if err == nil {
					CounterKey := fmt.Sprintf("denied_response_count.%s", Key)
					counters[CounterKey] = ival
				}

			case HF_EREQ:
				ival, err := strconv.ParseUint(v, 10, 64)
				if err == nil {
					CounterKey := fmt.Sprintf("error_request_count.%s", Key)
					counters[CounterKey] = ival
				}

			case HF_ERESP:
				ival, err := strconv.ParseUint(v, 10, 64)
				if err == nil {
					CounterKey := fmt.Sprintf("error_response_count.%s", Key)
					counters[CounterKey] = ival
				}

			case HF_WRETR:
				ival, err := strconv.ParseUint(v, 10, 64)
				if err == nil {
					CounterKey := fmt.Sprintf("conn_retry_count.%s", Key)
					counters[CounterKey] = ival
				}

			case HF_WREDIS:
				ival, err := strconv.ParseUint(v, 10, 64)
				if err == nil {
					CounterKey := fmt.Sprintf("redispatch_count.%s", Key)
					counters[CounterKey] = ival
				}

			case HF_DOWNTIME:
				ival, err := strconv.ParseUint(v, 10, 64)
				if err == nil {
					CounterKey := fmt.Sprintf("downtime_seconds.%s", Key)
					counters[CounterKey] = ival
				}

			case HF_CHKFAIL:
				ival, err := strconv.ParseUint(v, 10, 64)
				if err == nil {
					CounterKey := fmt.Sprintf("failed_check_count.%s", Key)
					counters[CounterKey] = ival
				}

			case HF_LBTOT:
				ival, err := strconv.ParseUint(v, 10, 64)
				if err == nil {
					CounterKey := fmt.Sprintf("selection_count.%s", Key)
					counters[CounterKey] = ival
				}

			}
		}

	}
	return nil
}

// Collect - XXX
func Collect() error {
	addr := "http://127.0.0.1:1936"
	client := &http.Client{}

	u, err := url.Parse(addr)
	if err != nil {
		return fmt.Errorf("Unable parse server address '%s': %s", addr, err)
	}

	req, err := http.NewRequest("GET", fmt.Sprintf("%s://%s%s/;csv", u.Scheme, u.Host, u.Path), nil)
	if u.User != nil {
		p, _ := u.User.Password()
		req.SetBasicAuth(u.User.Username(), p)
	}

	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("Unable to connect to haproxy server '%s': %s", addr, err)
	}

	if res.StatusCode != 200 {
		return fmt.Errorf("Unable to get valid stat result from '%s': %s", addr, err)
	}

	ParseCSVResult(res.Body, u.Host)

	return nil
}

func main() {
	f := Collect()
	fmt.Println(f)
}
