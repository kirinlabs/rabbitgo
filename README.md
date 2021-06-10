# rabbitgo
Rabbitmq connection pool for golang

### Installation
go get github.com/kirinlabs/rabbitgo


### How do we use rabbitgo?

#### Create request object use http.DefaultTransport
```go
var rabbit *rabbitgo.RabbitPool

func init(){
    rabbit = New(fmt.Sprintf("amqp://%s:%s@%s:%d/%s", "guest", "guest", "127.0.0.1", 5672, ""),
    Config{
        ConnectionMax: 5,
        ChannelActive: 20,
        ChannelIdle:   5,
    })
}

```

#### Sender
```go
ch, err := rabbit.Get()
if err != nil {
    log.Printf("Get channel error, %s", err.Error())
    retrun err
}
defer rabbit.Push(ch)

queue, err := ch.Ch.QueueDeclare("test_queue", true, false, false, false, nil)
if err != nil {
    log.Printf("Queue declare error, %s", err.Error())
    return err
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
    })
if err != nil {
    log.Printf("Send message error, %s", err.Error())
    return err
}
```
