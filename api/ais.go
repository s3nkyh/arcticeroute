package api

import (
	"encoding/json"
	"fmt"
	"log"

	aisstream "github.com/aisstream/ais-message-models/golang/aisStream"
	"github.com/gorilla/websocket"
	"github.com/s3nkyh/arcticeroute/models"
)

func Get10Ships() []models.Ship {
	url := "wss://stream.aisstream.io/v0/stream"
	ws, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		log.Fatalln(err)
	} else {
		log.Println("Connected to WebSocket server")
	}
	defer ws.Close()

	subMsg := aisstream.SubscriptionMessage{
		APIKey:        "f4f956742f6bddcdc52ccd035a66057aa80e4f3e",
		BoundingBoxes: [][][]float64{{{65.0, 30.0}, {90.0, 180.0}}},
	}

	subMsgBytes, _ := json.Marshal(subMsg)
	if err := ws.WriteMessage(websocket.TextMessage, subMsgBytes); err != nil {
		log.Fatalln(err)
	}

	ships := make([]models.Ship, 0, 10)

	for len(ships) < 10 {
		_, message, err := ws.ReadMessage()
		if err != nil {
			log.Println("Read error:", err)
			continue
		}

		var packet aisstream.AisStreamMessage
		err = json.Unmarshal(message, &packet)
		if err != nil {
			log.Println("Unmarshal error:", err)
			continue
		}

		if packet.MessageType == aisstream.POSITION_REPORT {
			shipName := "Unknown"
			if name, ok := packet.MetaData["ShipName"]; ok && name != "" {
				shipName = name.(string)
			}
			ship := models.Ship{
				MMSI:      packet.Message.PositionReport.UserID,
				Name:      shipName,
				Latitude:  packet.Message.PositionReport.Latitude,
				Longitude: packet.Message.PositionReport.Longitude,
			}

			ships = append(ships, ship)
			fmt.Printf("Собрано кораблей: %d/10\n", len(ships))
		}
	}

	return ships
}
