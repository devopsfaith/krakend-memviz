package memviz

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"path"
	"time"

	"github.com/bradleyjkemp/memviz"
	"github.com/devopsfaith/krakend/config"
	"github.com/devopsfaith/krakend/logging"
	"github.com/devopsfaith/krakend/proxy"
)

// New returns a proxy middleware ready to start dumping all the requests and responses
// it processes.
//
// The dot files will be stored in the path defined by the putput argument, using the name argument
// as a prefix.
//
// Requests named "xxx" will be stored in the output filder as xxx_in_{timestamp}.dot
// Responses named "xxx" will be stored in the output filder as xxx_out_{timestamp}.dot
func New(logger logging.Logger, output, name string) proxy.Middleware {
	return func(next ...proxy.Proxy) proxy.Proxy {
		switch len(next) {
		case 0:
			panic(proxy.ErrNotEnoughProxies)
		case 1:
		default:
			panic(proxy.ErrTooManyProxies)
		}
		return func(ctx context.Context, req *proxy.Request) (*proxy.Response, error) {
			in := new(bytes.Buffer)
			memviz.Map(in, req)
			logger.Debug("memviz: request captured")

			resp, err := next[0](ctx, req)
			logger.Debug("proxy executed")

			out := new(bytes.Buffer)
			memviz.Map(out, resp)
			logger.Debug("memviz: response captured. returning")

			go func(in, out *bytes.Buffer) {
				now := time.Now().UnixNano()
				preffix := path.Join(output, name)
				if err := ioutil.WriteFile(fmt.Sprintf("%s_in_%d.dot", preffix, now), in.Bytes(), 0666); err != nil {
					logger.Error("memviz: witting the in:", err.Error())
				}
				if err := ioutil.WriteFile(fmt.Sprintf("%s_out_%d.dot", preffix, now), out.Bytes(), 0666); err != nil {
					logger.Error("memviz: witting the out:", err.Error())
				}
			}(in, out)

			return resp, err
		}
	}
}

// ProxyFactory returns a proxy.FactoryFunc over the received proxy.FactoryFunc with a memviz middleware wrapping
// the generated pipe
func ProxyFactory(logger logging.Logger, factory proxy.Factory, output string) proxy.FactoryFunc {
	return func(cfg *config.EndpointConfig) (proxy.Proxy, error) {
		p, err := factory.New(cfg)
		if err != nil {
			return p, err
		}

		name := "proxy_" + base64.URLEncoding.EncodeToString([]byte(cfg.Endpoint))
		return New(logger, output, name)(p), nil
	}
}

// BackendFactory returns a proxy.BackendFactory over the received proxy.BackendFactory with a memviz middleware wrapping
// the generated backend
func BackendFactory(logger logging.Logger, factory proxy.BackendFactory, output string) proxy.BackendFactory {
	return func(backend *config.Backend) proxy.Proxy {
		name := "backend_" + base64.URLEncoding.EncodeToString([]byte(backend.URLPattern))
		return New(logger, output, name)(factory(backend))
	}
}
