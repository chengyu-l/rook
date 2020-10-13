package master

import (
	"encoding/json"
	"fmt"
	"testing"
)

func Test_getStatus(t *testing.T) {

	contents := `
{
  "code": 0,
  "msg": "success",
  "data": {
    "dataNodeStatInfo": {
      "TotalGB": 22225,
      "UsedGB": 0,
      "IncreasedGB": 0,
      "UsedRatio": "0.000"
    },
    "metaNodeStatInfo": {
      "TotalGB": 48,
      "UsedGB": 0,
      "IncreasedGB": 0,
      "UsedRatio": "0.009"
    },
    "ZoneStatInfo": {
      "default": {
        "dataNodeStat": {
          "TotalGB": 22225.22,
          "UsedGB": 0.19,
          "AvailGB": 22225.03,
          "UsedRatio": 0,
          "TotalNodes": 3,
          "WritableNodes": 3
        },
        "metaNodeStat": {
          "TotalGB": 48,
          "UsedGB": 0.45,
          "AvailGB": 47.55,
          "UsedRatio": 0.01,
          "TotalNodes": 3,
          "WritableNodes": 3
        }
      }
    }
  }
}
`
	data := &responseData{}
	json.Unmarshal([]byte(contents), data)
	fmt.Println(data)
	//got, err := getStatus("")
	//if err != nil {
	//	fmt.Println(err)
	//	return
	//}
	//
	//fmt.Println(got)
}
