// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

package builtin

import (
	"fmt"
	"lunex/internal/runtime"

	amqp "github.com/rabbitmq/amqp091-go"
)

func rabbitConnObj(conn *amqp.Connection) *runtime.Value {
	createChannel := runtime.FuncVal(&runtime.Function{
		Name: "createChannel",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			ch, err := conn.Channel()
			if err != nil {
				return runtime.Null, err
			}
			return rabbitChannelObj(ch), nil
		},
	})

	closeConn := runtime.FuncVal(&runtime.Function{
		Name: "close",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			return runtime.Undefined, conn.Close()
		},
	})

	isClosed := runtime.FuncVal(&runtime.Function{
		Name: "isClosed",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			return runtime.BoolVal(conn.IsClosed()), nil
		},
	})

	return runtime.ObjectVal(map[string]*runtime.Value{
		"createChannel": createChannel,
		"close":         closeConn,
		"isClosed":      isClosed,
	})
}

func rabbitChannelObj(ch *amqp.Channel) *runtime.Value {
	declareQueue := runtime.FuncVal(&runtime.Function{
		Name: "declareQueue",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.Null, fmt.Errorf("declareQueue(name, options?)")
			}
			name := args[0].ToString()
			durable := true
			autoDelete := false
			exclusive := false
			if len(args) > 1 && args[1].Tag == runtime.TypeObject {
				opts := args[1].ObjVal
				if v, ok := opts["durable"]; ok {
					durable = v.BoolVal
				}
				if v, ok := opts["autoDelete"]; ok {
					autoDelete = v.BoolVal
				}
				if v, ok := opts["exclusive"]; ok {
					exclusive = v.BoolVal
				}
			}
			q, err := ch.QueueDeclare(name, durable, autoDelete, exclusive, false, nil)
			if err != nil {
				return runtime.Null, err
			}
			return runtime.ObjectVal(map[string]*runtime.Value{
				"name":      runtime.StringVal(q.Name),
				"messages":  runtime.NumberVal(float64(q.Messages)),
				"consumers": runtime.NumberVal(float64(q.Consumers)),
			}), nil
		},
	})

	declareExchange := runtime.FuncVal(&runtime.Function{
		Name: "declareExchange",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.Null, fmt.Errorf("declareExchange(name, kind)")
			}
			name := args[0].ToString()
			kind := args[1].ToString()
			err := ch.ExchangeDeclare(name, kind, true, false, false, false, nil)
			if err != nil {
				return runtime.Null, err
			}
			return runtime.True, nil
		},
	})

	bindQueue := runtime.FuncVal(&runtime.Function{
		Name: "bindQueue",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 3 {
				return runtime.Null, fmt.Errorf("bindQueue(queue, exchange, routingKey)")
			}
			err := ch.QueueBind(args[0].ToString(), args[2].ToString(), args[1].ToString(), false, nil)
			return runtime.BoolVal(err == nil), err
		},
	})

	publish := runtime.FuncVal(&runtime.Function{
		Name: "publish",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.Null, fmt.Errorf("publish(queue, message, options?)")
			}
			queue := args[0].ToString()
			body := args[1].ToString()
			contentType := "text/plain"
			exchange := ""
			routingKey := queue
			if len(args) > 2 && args[2].Tag == runtime.TypeObject {
				opts := args[2].ObjVal
				if v, ok := opts["contentType"]; ok {
					contentType = v.ToString()
				}
				if v, ok := opts["exchange"]; ok {
					exchange = v.ToString()
					routingKey = queue
				}
				if v, ok := opts["routingKey"]; ok {
					routingKey = v.ToString()
				}
			}
			err := ch.Publish(exchange, routingKey, false, false, amqp.Publishing{
				ContentType: contentType,
				Body:        []byte(body),
			})
			return runtime.BoolVal(err == nil), err
		},
	})

	publishJSON := runtime.FuncVal(&runtime.Function{
		Name: "publishJSON",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.Null, fmt.Errorf("publishJSON(queue, object)")
			}
			queue := args[0].ToString()
			body := valueToJSON(args[1])
			err := ch.Publish("", queue, false, false, amqp.Publishing{
				ContentType: "application/json",
				Body:        []byte(body),
			})
			return runtime.BoolVal(err == nil), err
		},
	})

	consume := runtime.FuncVal(&runtime.Function{
		Name: "consume",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 || args[1].Tag != runtime.TypeFunction {
				return runtime.Null, fmt.Errorf("consume(queue, handler, options?)")
			}
			queue := args[0].ToString()
			handler := args[1]
			autoAck := true
			if len(args) > 2 && args[2].Tag == runtime.TypeObject {
				if v, ok := args[2].ObjVal["autoAck"]; ok {
					autoAck = v.BoolVal
				}
			}
			msgs, err := ch.Consume(queue, "", autoAck, false, false, false, nil)
			if err != nil {
				return runtime.Null, err
			}
			runtime.KeepAliveAdd()
			go func() {
				defer runtime.KeepAliveDone()
				for msg := range msgs {
					if runtime.CallFunction == nil {
						continue
					}
					msgObj := runtime.ObjectVal(map[string]*runtime.Value{
						"body":        runtime.StringVal(string(msg.Body)),
						"contentType": runtime.StringVal(msg.ContentType),
						"routingKey":  runtime.StringVal(msg.RoutingKey),
						"exchange":    runtime.StringVal(msg.Exchange),
						"ack": runtime.FuncVal(&runtime.Function{
							Name: "ack",
							Native: func(a []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
								return runtime.BoolVal(msg.Ack(false) == nil), nil
							},
						}),
						"nack": runtime.FuncVal(&runtime.Function{
							Name: "nack",
							Native: func(a []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
								requeue := true
								if len(a) > 0 {
									requeue = a[0].BoolVal
								}
								return runtime.BoolVal(msg.Nack(false, requeue) == nil), nil
							},
						}),
					})
					runtime.CallFunction(handler, []*runtime.Value{msgObj}, nil)
				}
			}()
			return runtime.ObjectVal(map[string]*runtime.Value{
				"cancel": runtime.FuncVal(&runtime.Function{
					Name: "cancel",
					Native: func(a []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
						return runtime.BoolVal(ch.Cancel("", false) == nil), nil
					},
				}),
			}), nil
		},
	})

	qos := runtime.FuncVal(&runtime.Function{
		Name: "qos",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			prefetch := 1
			if len(args) > 0 && args[0].Tag == runtime.TypeNumber {
				prefetch = int(args[0].NumVal)
			}
			err := ch.Qos(prefetch, 0, false)
			return runtime.BoolVal(err == nil), err
		},
	})

	closeChannel := runtime.FuncVal(&runtime.Function{
		Name: "close",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			return runtime.Undefined, ch.Close()
		},
	})

	return runtime.ObjectVal(map[string]*runtime.Value{
		"declareQueue":    declareQueue,
		"declareExchange": declareExchange,
		"bindQueue":       bindQueue,
		"publish":         publish,
		"publishJSON":     publishJSON,
		"consume":         consume,
		"qos":             qos,
		"close":           closeChannel,
	})
}

func RabbitMQModule() *runtime.Value {
	connect := runtime.FuncVal(&runtime.Function{
		Name: "connect",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			url := "amqp://guest:guest@localhost:5672/"
			if len(args) > 0 {
				url = args[0].ToString()
			}
			conn, err := amqp.Dial(url)
			if err != nil {
				return runtime.Null, fmt.Errorf("rabbitmq connect failed: %w", err)
			}
			return rabbitConnObj(conn), nil
		},
	})

	return runtime.ObjectVal(map[string]*runtime.Value{
		"connect": connect,
	})
}
