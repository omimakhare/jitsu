package queue

import (
	"encoding/json"
	"fmt"
	"github.com/gomodule/redigo/redis"
	"github.com/jitsucom/jitsu/server/meta"
	"github.com/jitsucom/jitsu/server/metrics"
)

const (
	eventsQueueKeyPrefix      = "events_queue:destination#"
	defaultWaitTimeoutSeconds = 30
)

//redis key [variables] - description
//** Events queue**
//events_queue:destination#$destinationID - list with event JSON's

//Redis is a queue implementation based on Redis
//it is used blocking pop (BLPOP) command for getting elements from queue
type Redis struct {
	identifier                string
	queueKey                  string
	serializationModelBuilder func() interface{}

	pool         *meta.RedisPool
	errorMetrics *meta.ErrorMetrics

	closed chan struct{}
}

func NewRedis(identifier string, redisPool *meta.RedisPool, serializationModelBuilder func() interface{}) Queue {
	return &Redis{
		identifier:                identifier,
		queueKey:                  eventsQueueKeyPrefix + identifier,
		serializationModelBuilder: serializationModelBuilder,
		pool:                      redisPool,
		errorMetrics:              meta.NewErrorMetrics(metrics.EventsRedisErrors),
		closed:                    make(chan struct{}),
	}
}

func (r *Redis) Push(v interface{}) error {
	select {
	case <-r.closed:
		return ErrQueueClosed
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Errorf("error serializing %v into json: %v", v, err)
		}
		return r.rpush(string(b))
	}
}

func (r *Redis) Pop() (interface{}, error) {
	for {
		select {
		case <-r.closed:
			return nil, ErrQueueClosed
		default:
			value, err := r.blpop()
			if err != nil {
				if err == ErrQueueEmpty {
					continue
				}

				return nil, err
			}

			model := r.serializationModelBuilder()
			if err := json.Unmarshal([]byte(value), model); err != nil {
				return nil, fmt.Errorf("error deserializing %v into %T: %v", value, model, err)
			}

			return model, nil
		}
	}
}

func (r *Redis) Size() int64 {
	conn := r.pool.Get()
	defer conn.Close()

	size, err := redis.Int64(conn.Do("LLEN", r.queueKey))
	if err != nil {
		if err == redis.ErrNil {
			return 0
		}

		r.errorMetrics.NoticeError(err)
		return -1
	}

	return size
}

func (r *Redis) Close() error {
	close(r.closed)
	return r.pool.Close()
}

func (r *Redis) blpop() (string, error) {
	conn := r.pool.Get()
	defer conn.Close()

	v, err := redis.Values(conn.Do("BLPOP", r.queueKey, defaultWaitTimeoutSeconds))
	if err != nil {
		if err == redis.ErrNil {
			return "", ErrQueueEmpty
		}

		r.errorMetrics.NoticeError(err)
		return "", err
	}

	if len(v) != 2 {
		return "", fmt.Errorf("malformed redis response: %v", v)
	}

	return redis.String(v[1], nil)
}

func (r *Redis) rpush(value string) error {
	conn := r.pool.Get()
	defer conn.Close()

	_, err := conn.Do("RPUSH", r.queueKey, value)
	if err != nil {
		if err == redis.ErrNil {
			return nil
		}

		r.errorMetrics.NoticeError(err)
		return err
	}

	return nil
}
