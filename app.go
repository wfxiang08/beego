// Copyright 2014 beego Author. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package beego

import (
	"fmt"
	"net"
	"net/http"
	"net/http/fcgi"
	"os"
	"time"

	"github.com/astaxie/beego/grace"
	"github.com/astaxie/beego/utils"
)

// 负责Server的配置和运行
// App defines beego application with a new PatternServeMux.
type App struct {
	// ServeHTTP(ResponseWriter, *Request)
	Handlers *ControllerRegistor
	Server   *http.Server
}

// NewApp returns a new beego application.
func NewApp() *App {
	cr := NewControllerRegister()

	// Controller和HttpServer
	app := &App{Handlers: cr, Server: &http.Server{}}
	return app
}

// Run beego application.
func (app *App) Run() {
	addr := HttpAddr

	if HttpPort != 0 {
		addr = fmt.Sprintf("%s:%d", HttpAddr, HttpPort)
	}

	var (
		err error
		l   net.Listener
	)
	endRunning := make(chan bool, 1)

	if UseFcgi {
		// 暂时不用管
		if UseStdIo {
			err = fcgi.Serve(nil, app.Handlers) // standard I/O
			if err == nil {
				BeeLogger.Info("Use FCGI via standard I/O")
			} else {
				BeeLogger.Info("Cannot use FCGI via standard I/O", err)
			}
		} else {
			if HttpPort == 0 {
				// remove the Socket file before start
				if utils.FileExists(addr) {
					os.Remove(addr)
				}
				l, err = net.Listen("unix", addr)
			} else {
				l, err = net.Listen("tcp", addr)
			}
			if err != nil {
				BeeLogger.Critical("Listen: ", err)
			}
			err = fcgi.Serve(l, app.Handlers)
		}
	} else {
		if Graceful {
			// http://beego.me/docs/module/grace.md
			// http://grisha.org/blog/2014/06/03/graceful-restart-in-golang/
			app.Server.Addr = addr
			app.Server.Handler = app.Handlers
			app.Server.ReadTimeout = time.Duration(HttpServerTimeOut) * time.Second
			app.Server.WriteTimeout = time.Duration(HttpServerTimeOut) * time.Second
			// 多个不同的Server在: Go/Java等环境中好理解，但是在: Python的单线程中如何理解呢?
			// Python + Gevent似乎也可以理解, 至少有异步，并发的概念
			//
			if EnableHttpTLS {
				go func() {
					time.Sleep(20 * time.Microsecond)
					if HttpsPort != 0 {
						addr = fmt.Sprintf("%s:%d", HttpAddr, HttpsPort)
						app.Server.Addr = addr
					}

					//
					// 同时创建多个不同的Server
					//
					server := grace.NewServer(addr, app.Handlers)
					server.Server = app.Server
					err := server.ListenAndServeTLS(HttpCertFile, HttpKeyFile)
					if err != nil {
						BeeLogger.Critical("ListenAndServeTLS: ", err, fmt.Sprintf("%d", os.Getpid()))
						time.Sleep(100 * time.Microsecond)
						endRunning <- true
					}
				}()
			}
			if EnableHttpListen {
				go func() {
					//
					// 通过grace Server来管理app.Server呢?
					//
					server := grace.NewServer(addr, app.Handlers)
					server.Server = app.Server
					if ListenTCP4 && HttpAddr == "" {
						server.Network = "tcp4"
					}
					err := server.ListenAndServe()
					if err != nil {
						BeeLogger.Critical("ListenAndServe: ", err, fmt.Sprintf("%d", os.Getpid()))
						time.Sleep(100 * time.Microsecond)
						endRunning <- true
					}
				}()
			}
		} else {
			// 最简单的模式:
			// 配置Server: Addr, Handler, Timeout
			app.Server.Addr = addr
			app.Server.Handler = app.Handlers // ServeHTTP(ResponseWriter, *Request)
			app.Server.ReadTimeout = time.Duration(HttpServerTimeOut) * time.Second
			app.Server.WriteTimeout = time.Duration(HttpServerTimeOut) * time.Second

			// TLS如何工作呢?
			// 设置证书？ 然后呢?
			if EnableHttpTLS {
				go func() {
					time.Sleep(20 * time.Microsecond)
					if HttpsPort != 0 {
						app.Server.Addr = fmt.Sprintf("%s:%d", HttpAddr, HttpsPort)
					}
					BeeLogger.Info("https server Running on %s", app.Server.Addr)
					err := app.Server.ListenAndServeTLS(HttpCertFile, HttpKeyFile)
					if err != nil {
						BeeLogger.Critical("ListenAndServeTLS: ", err)
						time.Sleep(100 * time.Microsecond)
						endRunning <- true
					}
				}()
			}

			if EnableHttpListen {
				go func() {
					app.Server.Addr = addr
					BeeLogger.Info("http server Running on %s", app.Server.Addr)
					if ListenTCP4 && HttpAddr == "" {
						// HttpAddr == "", 什么情况下会这样呢?
						ln, err := net.Listen("tcp4", app.Server.Addr)
						if err != nil {
							BeeLogger.Critical("ListenAndServe: ", err)
							time.Sleep(100 * time.Microsecond)
							endRunning <- true
							return
						}
						// 核心: 代码会在这里一直运行着
						err = app.Server.Serve(ln)
						if err != nil {
							BeeLogger.Critical("ListenAndServe: ", err)
							time.Sleep(100 * time.Microsecond)
							endRunning <- true
							return
						}
					} else {
						// 和上面的区别是什么呢?
						// 集成度更高?
						//
						err := app.Server.ListenAndServe()
						if err != nil {
							BeeLogger.Critical("ListenAndServe: ", err)
							time.Sleep(100 * time.Microsecond)
							endRunning <- true
						}
					}
				}()
			}
		}

	}
	<-endRunning
}
