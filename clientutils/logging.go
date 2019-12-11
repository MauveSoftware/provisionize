package clientutils

import (
	log "github.com/sirupsen/logrus"

	"github.com/MauveSoftware/provisionize/api/proto"
)

func LogServiceResult(service *proto.StatusUpdate, debug bool) {
	log.Println(service.ServiceName)

	if service.Failed {
		log.Println("Failed!")
	}

	if len(service.Message) != 0 {
		log.Println(service.Message)
	}

	if debug && len(service.DebugMessage) != 0 {
		log.Println("Debug:")
		log.Println(service.DebugMessage)
	}

	log.Println()
}
