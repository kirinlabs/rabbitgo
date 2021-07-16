package rabbitgo

import (
	"log"
	"math/rand"
	"os"
	"sync"
	"time"

	"github.com/streadway/amqp"
)

const (
	chIdOffset           = 100000
	chTempIdOffset       = 10000000
	defaultConnectionMax = 5
	defaultChannelMax    = (2 << 9) - 1
	defaultChannelActive = 10
	defaultChannelIdle   = 1
	defaultHealth        = 30 * time.Second
	defaultTimeout       = 60 * time.Second
	defaultHeartbeat     = 15 * time.Second
)

type Config struct {
	ConnectionMax int
	ChannelMax    int
	ChannelActive int
	ChannelIdle   int
	Health        time.Duration
	Timeout       time.Duration
	Heartbeat     time.Duration
}

type Connection struct {
	tempId int64
	conn   *amqp.Connection
	active map[int64]int64
}

type Channel struct {
	ChId          int64
	T             time.Time
	Ch            *amqp.Channel
	NotifyConfirm chan amqp.Confirmation
	close         bool
}

func (c *Channel) Confirm(noWait bool) error {
	if noWait {
		if c.NotifyConfirm != nil {
			close(c.NotifyConfirm)
		}
	}

	if err := c.Ch.Confirm(noWait); err != nil {
		return err
	}

	if !noWait && c.NotifyConfirm == nil {
		c.NotifyConfirm = c.Ch.NotifyPublish(make(chan amqp.Confirmation, 1))
	}

	return nil
}

// Rabbit connection pool
type RabbitPool struct {
	url           string
	connectionMax int
	channelMax    int
	channelActive int
	channelIdle   int
	health        time.Duration
	heartbeat     time.Duration
	timeout       time.Duration
	m             sync.Mutex
	connections   map[int]*Connection
	channels      map[int64]*Channel
	idleChannel   []int64
	busyChannel   map[int64]int64
	logger        *Logger
}

func New(url string, config Config) *RabbitPool {
	r := &RabbitPool{
		url:           url,
		connectionMax: config.ConnectionMax,
		channelMax:    config.ChannelMax,
		channelActive: config.ChannelActive,
		channelIdle:   config.ChannelIdle,
		health:        config.Health,
		timeout:       config.Timeout,
		heartbeat:     config.Heartbeat,
		logger:        newLogger(os.Stdout),
	}

	r.defaultInit()
	r.connectPool()
	r.channelPool()

	go r.healthCheck()

	return r
}

// Set log level
func (r *RabbitPool) SetLevel(level int) {
	r.logger.SetLevel(level)
}

// Initialize the connection pool
func (r *RabbitPool) connectPool() {
	r.connections = make(map[int]*Connection)
	for i := 1; i <= r.connectionMax; i++ {
		conn, err := r.connect()
		if err != nil {
			log.Panicf("Failed to connect to rabbit server: %s", err.Error())
		}
		conn.tempId = int64(i * chTempIdOffset)
		r.connections[i] = conn
	}
	r.logger.Infof("Rabbit connections size: %d", len(r.connections))
}

func (r *RabbitPool) connect() (*Connection, error) {
	conn, err := amqp.DialConfig(
		r.url, amqp.Config{
			ChannelMax: r.channelMax,
			Heartbeat:  r.heartbeat,
		},
	)
	if err != nil {
		return nil, err
	}

	return &Connection{conn: conn, active: make(map[int64]int64)}, err
}

// Initialize the channel pool
func (r *RabbitPool) channelPool() {
	r.channels = make(map[int64]*Channel)
	var wg sync.WaitGroup
	for connId, _ := range r.connections {
		wg.Add(1)
		go func(connId int) {
			defer wg.Done()
			for i := 1; i <= r.channelActive; i++ {
				chId := int64(connId*chIdOffset + i)
				ch, err := r.createChannel(connId)
				if err != nil {
					continue
				}
				ch.ChId = chId
				r.m.Lock()
				r.channels[chId] = ch
				r.idleChannel = append(r.idleChannel, chId)
				r.connections[connId].active[chId] = chId
				r.m.Unlock()
			}
			return
		}(connId)
	}
	wg.Wait()
	r.logger.Infof("Rabbit channels size: %d", len(r.channels))
}

func (r *RabbitPool) createChannel(connId int) (*Channel, error) {
	rc := &Channel{
		ChId:  0,
		T:     time.Now(),
		Ch:    nil,
		close: false,
	}

	if r.connections[connId].conn.IsClosed() {
		r.connections[connId].conn.Close()
		conn, err := r.connect()
		if err != nil {
			return nil, err
		}
		r.connections[connId] = conn
	}

	ch, err := r.connections[connId].conn.Channel()
	if err != nil {
		//r.logger.Errorf("Create Channel Error: connId %d, create channel error, %s", connId, err.Error())
		return nil, err
	}

	rc.Ch = ch

	return rc, nil
}

// Randomly return an available channel
func (r *RabbitPool) Get() (*Channel, error) {
	rand.Seed(time.Now().UnixNano())

	r.m.Lock()
	defer r.m.Unlock()
	idle := len(r.idleChannel)
	if idle > 0 {
		intn := rand.Intn(idle)
		chId := r.idleChannel[intn]
		r.idleChannel = append(r.idleChannel[:intn], r.idleChannel[intn+1:]...)
		if r.channels[chId].close {
			connId := r.connId(chId)
			ch, err := r.createChannel(connId)
			if err != nil {
				r.logger.Errorf("Create channel: connId %d, error %s", connId, err.Error())
				return nil, err
			}
			ch.ChId = chId
			r.channels[chId] = ch
			r.connections[connId].active[chId] = chId
		}
		r.channels[chId].T = time.Now()
		r.busyChannel[chId] = chId

		return r.channels[chId], nil
	}

	connId := rand.Intn(r.connectionMax) + 1
	ch, err := r.createChannel(connId)
	if err != nil {
		return nil, err
	}
	r.connections[connId].tempId += 1
	ch.ChId = r.connections[connId].tempId

	r.logger.Warnf("Channel is exhausted, create temporary channel......")

	return ch, nil
}

// Put back into the connection pool
func (r *RabbitPool) Push(ch *Channel) {
	if ch == nil {
		return
	}

	if ch.ChId >= chTempIdOffset {
		ch.Ch.Close()
		return
	}

	r.m.Lock()
	defer r.m.Unlock()
	if inSlice(ch.ChId, r.idleChannel) {
		return
	}
	r.channels[ch.ChId].T = time.Now()
	r.idleChannel = append(r.idleChannel, ch.ChId)
	delete(r.busyChannel, ch.ChId)
}

func (r *RabbitPool) connId(chId int64) int {
	if chId >= chTempIdOffset {
		return int(chId / chTempIdOffset)
	}
	return int(chId / chIdOffset)
}

func (r *RabbitPool) reconnect(connId int) error {
	//pc, _, line, _ := runtime.Caller(1)

	r.m.Lock()
	defer r.m.Unlock()

	if r.connections[connId].conn.IsClosed() {
		r.connections[connId].conn.Close()
		//r.logger.Errorf("Reconnect: connId %d, Function: %s, Line %d", connId, runtime.FuncForPC(pc).Name(), line)
		conn, err := r.connect()
		if err != nil {
			r.logger.Errorf("Reconnect: connId %d, error %s", connId, err.Error())
			return err
		}
		r.connections[connId] = conn
		for i := 1; i <= r.channelActive; i++ {
			chId := int64(connId*chIdOffset + i)
			ch, err := r.createChannel(connId)
			if err != nil {
				r.logger.Errorf("Reconnect: connId %d, create channel error %s", connId, err.Error())
				return err
			}
			ch.ChId = chId
			r.channels[chId] = ch
			if !inSlice(chId, r.idleChannel) {
				r.idleChannel = append(r.idleChannel, chId)
			}
			delete(r.busyChannel, chId)
			r.connections[connId].active[chId] = chId
		}
	}
	return nil
}

func (r *RabbitPool) healthCheck() {
	ticker := time.NewTicker(r.health)
	for {
		<-ticker.C

		r.logger.Infof("Rabbit Pool: Connections %d, Channels %d", len(r.connections), len(r.channels))
		for cid, c := range r.connections {
			if c.conn.IsClosed() {
				c.conn.Close()
				r.reconnect(cid)
			}
		}

		r.m.Lock()
		for chId, ch := range r.channels {
			connId := r.connId(chId)
			if time.Since(ch.T).Seconds() > r.timeout.Seconds() && !ch.close && len(r.connections[connId].active) > r.channelIdle {
				if _, ok := r.busyChannel[chId]; !ok {
					if err := ch.Ch.Close(); err == nil {
						ch.close = true
						delete(r.connections[connId].active, chId)
					}
				}
			}
		}
		r.m.Unlock()
	}
}

func (r *RabbitPool) defaultInit() {
	if r.connectionMax <= 0 {
		r.connectionMax = defaultConnectionMax
	}

	if r.channelMax <= 0 {
		r.channelMax = defaultChannelMax
	}

	if r.channelActive <= 0 {
		r.channelActive = defaultChannelActive
	}

	if r.channelIdle <= 0 {
		r.channelIdle = defaultChannelIdle
	}

	if r.health <= time.Second {
		r.health = defaultHealth
	}
	if r.timeout <= time.Second {
		r.timeout = defaultTimeout
	}

	if r.heartbeat <= time.Second {
		r.heartbeat = defaultHeartbeat
	}

	r.busyChannel = make(map[int64]int64)
}
