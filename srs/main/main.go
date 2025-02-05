/*
The MIT License (MIT)

Copyright (c) 2013-2015 GOSRS(gosrs)

Permission is hereby granted, free of charge, to any person obtaining a copy of
this software and associated documentation files (the "Software"), to deal in
the Software without restriction, including without limitation the rights to
use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
the Software, and to permit persons to whom the Software is furnished to do so,
subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
*/
package main

import (
    "flag"
    "fmt"
    "go_srs/srs/app"
    "go_srs/srs/app/config"
	"net/http"
	"bytes"
    "io/ioutil"
    "math/rand"
	_ "net/http/pprof"
)

var (
	conf = flag.String("c", "./conf/srs.conf", "set conf `conf`")
)

func main() {
    flag.Parse()
    if err := config.GetInstance().Init(*conf); err != nil {
        fmt.Println(err)
        return
    }

	server := app.NewSrsServer()
	_ = server.StartProcess(config.GetInstance().ListenPort)
}

func handler(w http.ResponseWriter, r *http.Request) {
    err := r.ParseForm()
    if nil != err {
        w.Write([]byte(err.Error()))
        return
    }
    doSomeThingOne(10000)
    buff := genSomeBytes()
    b, err := ioutil.ReadAll(buff)
    if nil != err {
        w.Write([]byte(err.Error()))
        return
    }
    w.Write(b)
}

func doSomeThingOne(times int) {
    for i := 0; i < times; i++ {
        for j := 0; j < times; j++ {

        }
    }
}

func genSomeBytes() *bytes.Buffer {
    var buff bytes.Buffer
    for i := 1; i < 20000; i++ {
        buff.Write([]byte{'0' + byte(rand.Intn(10))})
    }
    return &buff
}
