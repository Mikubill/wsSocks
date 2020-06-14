package main

import (
	"encoding/hex"
	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
	"net"
	"net/http"
	"net/http/pprof"
	"net/url"
	"time"
)

var (
	isClient = false
	client   = Client{
		CreatedAt: time.Now(),
	}
	server = Server{
		Resolver: &websocket.Upgrader{
			ReadBufferSize:   32768, // Expected average message size
			WriteBufferSize:  32768,
			HandshakeTimeout: 10 * time.Second,
		},
		CreatedAt: time.Now(),
	}
	globalFlag = []cli.Flag{
		&cli.BoolFlag{
			Name:    "debug",
			Aliases: []string{"verbose"},
			Usage:   "log debug messages",
		},
		&cli.StringFlag{
			Name:  "auth",
			Value: "Mikubill-wSocks",
			Usage: "key for authentication, leave blank to disable",
		},
		&cli.BoolFlag{
			Name:    "stats",
			Aliases: []string{"stat"},
			Usage:   "log connection stats",
		},
	}
	app = cli.App{
		Name:    "wSocks",
		Version: "0.1",
		Usage:   "Socks5 Proxy based on Websocket",
		Commands: []*cli.Command{
			&clientCmd,
			&serverCmd,
			&certCmd,
			&benchCmd,
		},
	}
	clientCmd = cli.Command{
		Name:    "client",
		Aliases: []string{"c"},
		Usage:   "start websocket client",
		Flags: append(
			[]cli.Flag{
				&cli.StringFlag{
					Name:  "hash",
					Value: "auto",
					Usage: "algorithm for hash [mem|xx|mur|adler|crc]Hash",
				},
				&cli.StringFlag{
					Name:     "server",
					Aliases:  []string{"s"},
					Value:    "ws://localhost:2233/ws",
					Required: true,
					Usage:    "websocket server link",
				},
				&cli.StringFlag{
					Name:    "listen",
					Aliases: []string{"l"},
					Value:   "127.0.0.1:2333",
					Usage:   "local listening port",
				},
				&cli.BoolFlag{
					Name:  "insecure",
					Usage: "allow insecure connections",
				},
				&cli.StringFlag{
					Name:  "sni",
					Value: "",
					Usage: "server name indication, leave blank to disable",
				},
				&cli.IntFlag{
					Name:  "conn",
					Value: 4,
					Usage: "total websocket connection count",
				},
			},
			globalFlag...,
		),
		Action: func(c *cli.Context) (err error) {
			isClient = true

			if c.String("hash") != "auto" {

				hashWorker, err = hashSelector(c.String("hash"))
				if err != nil {
					return
				}
				hashFlag = c.String("hash")
			}

			client.ServerAddr, err = url.Parse(c.String("server"))
			if err != nil {
				return
			}

			client.ListenAddr, err = net.ResolveTCPAddr("tcp", c.String("listen"))
			if err != nil {
				return
			}

			if c.Bool("stats") {
				taskAdd(stats)
			}

			if c.Bool("debug") {
				log.SetLevel(logrus.DebugLevel)
			}

			key := []byte(c.String("auth"))
			authKey = make([]byte, hex.EncodedLen(len(key)))
			hex.Encode(authKey, key)
			taskAdd(timeUpdater)

			if srv := c.String("sni"); srv != "" {
				tlsConfig.ServerName = srv
			}
			tlsConfig.InsecureSkipVerify = c.Bool("insecure")

			client.Connections = c.Int("conn")
			err = client.Listen()
			return
		},
	}

	benchCmd = cli.Command{
		Name:    "benchmark",
		Aliases: []string{"b"},
		Usage:   "start benchmark client",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "server",
				Aliases:  []string{"s"},
				Value:    "ws://localhost:2233/ws",
				Required: true,
				Usage:    "websocket server link",
			},
			&cli.BoolFlag{
				Name:  "debug",
				Usage: "log debug messages",
			},
			&cli.StringFlag{
				Name:  "sni",
				Value: "",
				Usage: "server name indication, leave blank to disable",
			},
			&cli.BoolFlag{
				Name:  "insecure",
				Usage: "allow insecure connections",
			},
			&cli.IntFlag{
				Name:  "block",
				Value: 30000,
				Usage: "set benchmark blockSize",
			},
			&cli.IntFlag{
				Name:  "conn",
				Value: 4,
				Usage: "total websocket connection count",
			},
			&cli.StringFlag{
				Name:  "auth",
				Value: "Mikubill-wSocks",
				Usage: "key for authentication, leave blank to disable",
			},
		},
		Action: func(c *cli.Context) (err error) {
			isClient = true

			taskAdd(debug)
			serverAddr, err := url.Parse(c.String("server"))
			if err != nil {
				return
			}
			taskAdd(benchStats)

			key := []byte(c.String("auth"))
			authKey = make([]byte, hex.EncodedLen(len(key)))
			hex.Encode(authKey, key)
			taskAdd(timeUpdater)
			if srv := c.String("sni"); srv != "" {
				tlsConfig.ServerName = srv
			}
			if c.Bool("debug") {
				log.SetLevel(logrus.DebugLevel)
			}
			tlsConfig.InsecureSkipVerify = c.Bool("insecure")

			local := Benchmark{
				Connections: c.Int("conn"),
				Block:       c.Int("block"),
				ServerAddr:  serverAddr,
				CreatedAt:   time.Now(),
			}

			err = local.Bench()
			return
		},
	}

	serverCmd = cli.Command{
		Name:    "server",
		Aliases: []string{"s"},
		Usage:   "start websocket server",
		Flags: append(
			[]cli.Flag{
				&cli.StringFlag{
					Name:    "listen",
					Aliases: []string{"l"},
					Value:   "ws://127.0.0.1:2333/ws",
					Usage:   "websocket server path",
				},
				&cli.StringFlag{
					Name:  "cert",
					Value: "root.pem",
					Usage: "tls cert path, leave blank to self generate",
				},
				&cli.StringFlag{
					Name:  "key",
					Value: "root.key",
					Usage: "tls key path, leave blank to self generate",
				},
				&cli.StringFlag{
					Name:    "reverse",
					Value:   "",
					Aliases: []string{"r"},
					Usage:   "reverse proxy url, leave blank to disable",
				},
			},
			globalFlag...,
		),
		Action: func(c *cli.Context) (err error) {
			if c.Bool("stats") {
				taskAdd(stats)
			}
			if c.Bool("debug") {
				log.SetLevel(logrus.DebugLevel)
			}

			key := []byte(c.String("auth"))
			authKey = make([]byte, hex.EncodedLen(len(key)))
			hex.Encode(authKey, key)
			taskAdd(timeUpdater)
			if c.String("reverse") != "" {
				server.Reverse, err = url.Parse(c.String("reverse"))
				if err != nil {
					return
				}
			}

			server.ListenAddr, err = url.Parse(c.String("listen"))
			if err != nil {
				return
			}

			server.Cert, server.PrivateKey = c.String("cert"), c.String("key")
			err = server.Listen()
			return
		},
	}
	certCmd = cli.Command{
		Name:    "cert",
		Aliases: []string{"cert"},
		Usage:   "generate self signed key and cert(use ecdsa)",
		Flags: []cli.Flag{
			&cli.StringSliceFlag{
				Name:  "hosts",
				Value: nil,
				Usage: "certificate hosts",
			},
		},
		Action: func(c *cli.Context) (err error) {
			hosts := c.StringSlice("hosts")
			cert, err := Generate(hosts, "Acme Co", 365*24*time.Hour)
			if err != nil {
				log.Fatal(err)
			}

			if err := writeCert(cert, "root"); err != nil {
				log.Fatal(err)
			}

			return
		},
	}
)

func stats() {
	for {
		time.Sleep(5 * time.Second)
		log.Infof("stats: uploaded %s, downloaded %s",
			ByteCountSI(uploaded), ByteCountSI(downloaded))
	}
}

func debug() {
	mux := http.NewServeMux()
	mux.Handle("/debug/pprof/", http.HandlerFunc(pprof.Index))
	mux.Handle("/debug/pprof/cmdline", http.HandlerFunc(pprof.Cmdline))
	mux.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
	mux.Handle("/debug/pprof/symbol", http.HandlerFunc(pprof.Symbol))
	mux.Handle("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))

	_ = http.ListenAndServe("127.0.0.1:8090", mux)
}
