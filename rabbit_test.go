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
			ConnectionMax: 2,
			ChannelMax:    10,
			ChannelActive: 2,
			ChannelIdle:   5,
		})
}

func sendWithoutConfirm(t *testing.T) error {
	ch, err := rabbit.Get()
	if err != nil {
		return fmt.Errorf("Get channel error, %s", err.Error())
	}
	defer rabbit.Push(ch)
	t.Logf("Send messages without using confirm mode......, ChId %d", ch.ChId)

	qu, err := ch.Ch.QueueDeclare(queue, true, false, false, false, nil)
	if err != nil {
		return fmt.Errorf("Queue declare error, %s", err.Error())
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
		return fmt.Errorf("Send message error, %s", err.Error())
	}
	return nil
}

// go test -v -run TestSend
func TestSend(t *testing.T) {
	err := sendWithoutConfirm(t)
	if err != nil {
		t.Logf(err.Error())
	}
}

// go test -v -run TestSendConfirm
func TestSendConfirm(t *testing.T) {
	go func() {
		for {
			if err := sendWithoutConfirm(t); err != nil {
				t.Logf(err.Error())
			}
			time.Sleep(time.Second)
		}
	}()

	for {
		func() {
			ch, err := rabbit.Get()
			if err != nil {
				t.Logf("Get channel error, %s", err.Error())
			}
			defer rabbit.Push(ch)

			// confirm mode
			ch.Confirm(false)
			defer func() {
				if confirmed := <-ch.NotifyConfirm; confirmed.Ack {
					// code when messages is confirmed
					t.Logf("Confirmed tag %d, ChId %d", confirmed.DeliveryTag, ch.ChId)
				} else {
					// code when messages is nacked
					t.Logf("Nacked tag %d, ChId %d", confirmed.DeliveryTag, ch.ChId)
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
			t.Logf("Send messages with using confirm mode......, ChId %d", ch.ChId)
		}()
		time.Sleep(time.Second)
	}

}
