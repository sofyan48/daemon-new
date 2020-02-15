package transmiter

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/service/kinesis"
	entity "github.com/sofyan48/otp/src/entity/http/v1"
	"github.com/sofyan48/otp/src/util/helper/libaws"
	"github.com/sofyan48/otp/src/util/helper/provider"
	"github.com/sofyan48/otp/src/util/helper/request"
)

// Transmiter ...
type Transmiter struct {
	AwsLibs   libaws.AwsInterface
	Provider  provider.ProvidersInterface
	Requester request.RequesterInterface
}

// GetTransmiter ...
func GetTransmiter() *Transmiter {
	return &Transmiter{
		AwsLibs:   libaws.AwsHAndler(),
		Provider:  provider.ProvidersHandler(),
		Requester: request.RequesterHandler(),
	}
}

// ConsumerTrans ...
func (trs *Transmiter) ConsumerTrans(wg *sync.WaitGroup) {
	fmt.Println("SMS Consumer Exec")
	shardIterator, err := trs.AwsLibs.GetShardIterator()
	if err != nil {
		log.Println(err)
	}

	describeInput := trs.AwsLibs.GetDescribeInput()
	describeInput.SetStreamName("notification")
	describeInput.SetExclusiveStartShardId(os.Getenv("KINESIS_SHARD_ID"))
	for {
		err := trs.AwsLibs.WaitUntil(describeInput)
		if err != nil {
			log.Println("error Wait: ", err)
		}
		go func() {
			msgInput := &kinesis.GetRecordsInput{}
			msgInput.SetShardIterator(shardIterator)

			data, err := trs.AwsLibs.Consumer(msgInput)
			if err != nil {
				log.Println(err)
			}
			selectionType := &entity.DataReceiveSelection{}
			for _, i := range data.Records {
				json.Unmarshal([]byte(string(i.Data)), selectionType)
				switch selectionType.Type {
				case "sms":
					itemSMS := &entity.DynamoItem{}
					json.Unmarshal([]byte(string(i.Data)), itemSMS)
					trs.intercepActionShardSMS(itemSMS)
				case "email":
					itemEmail := &entity.DynamoItemEmail{}
					json.Unmarshal([]byte(string(i.Data)), itemEmail)
					trs.intercepActionShardEmail(itemEmail)
				}

			}
			shardIterator = *data.NextShardIterator
			return
		}()
		time.Sleep(3 * time.Second)
	}

}

// updateDynamoTransmitt ...
func (trs *Transmiter) updateDynamoTransmitt(ID, status, data string, history *entity.HistoryItem) (string, error) {
	result, err := trs.AwsLibs.UpdateDynamo(ID, status, data, history)
	return result.GoString(), err
}

// TransferToShardReceiver ...
func (trs *Transmiter) TransferToShardReceiver(historyString string) {}

func checkEnvironment() bool {
	envi := os.Getenv("APP_ENVIRONMENT")
	return envi == "development" || envi == "staging"
}