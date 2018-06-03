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

func New(logger logging.Logger, output, name string, next proxy.Proxy) proxy.Proxy {
	return func(ctx context.Context, req *proxy.Request) (*proxy.Response, error) {
		in := new(bytes.Buffer)
		memviz.Map(in, req)
		logger.Debug("memviz: request captured")

		resp, err := next(ctx, req)
		logger.Debug("proxy executed")

		out := new(bytes.Buffer)
		memviz.Map(out, resp)
		logger.Debug("rmemviz: response captured. returning")
		go func(in, out *bytes.Buffer) {
			now := time.Now().UnixNano()
			if err := ioutil.WriteFile(path.Join(output, fmt.Sprintf("%s_in_%d.dot", name, now)), in.Bytes(), 0666); err != nil {
				logger.Error("memviz: witting the in:", err.Error())
			}
			if err := ioutil.WriteFile(path.Join(output, fmt.Sprintf("%s_out_%d.dot", name, now)), out.Bytes(), 0666); err != nil {
				logger.Error("memviz: witting the out:", err.Error())
			}
		}(in, out)
		return resp, err
	}
}

func ProxyFactory(logger logging.Logger, factory proxy.Factory, output string) proxy.FactoryFunc {
	return func(cfg *config.EndpointConfig) (proxy.Proxy, error) {
		p, err := factory.New(cfg)
		if err != nil {
			return p, err
		}
		return New(logger, output, "proxy_"+base64.URLEncoding.EncodeToString([]byte(cfg.Endpoint)), p), nil
	}
}

func BackendFactory(logger logging.Logger, factory proxy.BackendFactory, output string) proxy.BackendFactory {
	return func(backend *config.Backend) proxy.Proxy {
		return New(logger, output, "backend_"+base64.URLEncoding.EncodeToString([]byte(backend.URLPattern)), factory(backend))
	}
}
