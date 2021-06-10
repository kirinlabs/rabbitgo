package rabbitgo

import (
	"fmt"
	"testing"
	"time"

	"github.com/streadway/amqp"
)

var rabbit *RabbitPool
var queue string

func init() {
	queue = "rabbitgo_pool"
	rabbit = New(fmt.Sprintf("amqp://%s:%s@%s:%d/%s", "guest", "guest", "127.0.0.1", 5672, ""),
		Config{
			ConnectionMax: 5,
			ChannelActive: 20,
			ChannelIdle:   5,
		})
}

// go test -v -run TestSend
func TestSend(t *testing.T) {
	ch, err := rabbit.Get()
	if err != nil {
		t.Logf("Get channel error, %s", err.Error())
	}
	defer rabbit.Push(ch)

	qu, err := ch.Ch.QueueDeclare(queue, true, false, false, false, nil)
	if err != nil {
		t.Logf("Queue declare error, %s", err.Error())
	}

	data := fmt.Sprintf("{\"code\":200,\"message\":\"success\",\"data\":\"%s\"}", time.Now().String())
	err = ch.Ch.Publish(
		"",
		qu.Name,
		false,
		false,
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "appliction/plain",
			Body:         []byte(data),
		},
	)
	if err != nil {
		t.Logf("Send message error, %s", err.Error())
	}
}

// go test -v -run TestSendConfirm
func TestSendConfirm(t *testing.T) {
	ch, err := rabbit.Get()
	if err != nil {
		t.Logf("Get channel error, %s", err.Error())
	}
	defer rabbit.Push(ch)

	ch.Ch.Confirm(false)
	confirms := make(chan amqp.Confirmation)
	ch.Ch.NotifyPublish(confirms)

	defer func() {
		if confirm := <-confirms; confirm.Ack {
			// code when messages is confirmed
			t.Logf("Confirmed tag %d", confirm.DeliveryTag)
		} else {
			// code when messages is nacked
			t.Logf("Nacked tag %d", confirm.DeliveryTag)
		}
	}()

	qu, err := ch.Ch.QueueDeclare(queue, true, false, false, false, nil)
	if err != nil {
		t.Logf("Queue declare error, %s", err.Error())
	}

	data := fmt.Sprintf("{\"code\":200,\"message\":\"success\",\"data\":\"%s\"}", time.Now().String())
	err = ch.Ch.Publish(
		"",
		qu.Name,
		false,
		false,
		amqp.Publishing{
			DeliveryMode: amqp.Persistent,
			ContentType:  "appliction/plain",
			Body:         []byte(data),
		},
	)
	if err != nil {
		t.Logf("Send message error, %s", err.Error())
	}
}
