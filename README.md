# rabbitgo
基于golang的rabbitmq连接池

### 安装rabbitgo
go get github.com/kirinlabs/rabbitgo


### 如何使用rabbitgo?

#### 初始化全局连接池对象
```go
var rabbit *rabbitgo.RabbitPool

func init(){
    rabbit = rabbitgo.New(fmt.Sprintf("amqp://%s:%s@%s:%d/%s", "guest", "guest", "127.0.0.1", 5672, ""),
    Config{
        ConnectionMax: 5,
        ChannelActive: 20,
        ChannelIdle:   10,
    })
    //设置日志打印级别，默认rabbitgo.LOG_DEBUG
    rabbit.SetLevel(rabbitgo.LOG_ERROR)
}

```

#### Sender
```go
// 获取Channel对象
ch, err := rabbit.Get()
if err != nil {
    log.Printf("Get channel error, %s", err.Error())
    retrun err
}
// 重入channel池复用
defer rabbit.Push(ch)

queue, err := ch.Ch.QueueDeclare("test_queue", true, false, false, false, nil)
if err != nil {
    log.Printf("Queue declare error, %s", err.Error())
    return err
}

data := fmt.Sprintf("{\"code\":200,\"message\":\"success\",\"data\":\"%s\"}", time.Now().String())
err = ch.Ch.Publish(
        "",
        queue.Name,
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

#### Sender Confirm
```go
// 获取Channel对象
ch, err := rabbit.Get()
if err != nil {
    log.Printf("Get channel error, %s", err.Error())
    retrun err
}
// 重入channel池复用
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

queue, err := ch.Ch.QueueDeclare("test_queue", true, false, false, false, nil)
if err != nil {
    log.Printf("Queue declare error, %s", err.Error())
    return err
}

data := fmt.Sprintf("{\"code\":200,\"message\":\"success\",\"data\":\"%s\"}", time.Now().String())
err = ch.Ch.Publish(
        "",
        queue.Name,
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
