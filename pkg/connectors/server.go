package connectors

import (
	"bytes"
	"context"
	"github.com/PxyUp/fitter/pkg/builder"
	"github.com/PxyUp/fitter/pkg/config"
	"github.com/PxyUp/fitter/pkg/http_client"
	"github.com/PxyUp/fitter/pkg/limitter"
	"github.com/PxyUp/fitter/pkg/logger"
	"github.com/PxyUp/fitter/pkg/utils"
	"golang.org/x/sync/semaphore"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

const (
	timeout                 = 60 * time.Second
	defaultConcurrentWorker = 1000
)

type apiConnector struct {
	url    string
	logger logger.Logger
	client *http.Client
	cfg    *config.ServerConnectorConfig
}

var (
	sem *semaphore.Weighted

	ctx = context.Background()
)

func init() {
	defaultConcurrentRequest := defaultConcurrentWorker
	if value, ok := os.LookupEnv("FITTER_HTTP_WORKER"); ok {
		intValue, err := strconv.ParseInt(value, 10, 32)
		if err == nil && intValue > 0 {
			defaultConcurrentRequest = int(intValue)
		}
	}
	sem = semaphore.NewWeighted(int64(defaultConcurrentRequest))
}

func NewAPI(url string, cfg *config.ServerConnectorConfig, client *http.Client) *apiConnector {
	return &apiConnector{
		client: client,
		url:    url,
		cfg:    cfg,
		logger: logger.Null,
	}
}

func (api *apiConnector) WithLogger(logger logger.Logger) *apiConnector {
	api.logger = logger
	return api
}

func (api *apiConnector) GetWithHeaders(parsedValue builder.Interfacable, index *uint32, input builder.Interfacable) (http.Header, []byte, error) {
	return api.get(parsedValue, index, input)
}

func (api *apiConnector) get(parsedValue builder.Interfacable, index *uint32, input builder.Interfacable) (http.Header, []byte, error) {
	formattedBody := utils.Format(api.cfg.Body, parsedValue, index, input)
	if api.cfg.JsonRawBody != nil && len(api.cfg.JsonRawBody) > 0 {
		formattedBody = utils.Format(string(api.cfg.JsonRawBody), parsedValue, index, input)
	}

	formattedURL := utils.Format(api.url, parsedValue, index, input)

	if formattedURL == "" {
		return nil, nil, errEmpty
	}

	err := sem.Acquire(ctx, 1)
	if err != nil {
		api.logger.Errorw("unable to acquire semaphore", "method", api.cfg.Method, "url", formattedURL, "error", err.Error())
		return nil, nil, err
	}

	defer sem.Release(1)

	req, err := http.NewRequest(api.cfg.Method, formattedURL, bytes.NewBufferString(formattedBody))

	if err != nil {
		api.logger.Errorw("unable to create http request", "error", err.Error())
		return nil, nil, err
	}

	for k, v := range api.cfg.Headers {
		req.Header.Add(utils.Format(k, parsedValue, index, input), utils.Format(v, parsedValue, index, input))
	}

	client := http_client.GetDefaultClient()
	if api.client != nil {
		client = api.client
	}

	if api.cfg.Proxy != nil {
		proxyUrl, errProxy := url.Parse(utils.Format(api.cfg.Proxy.Server, parsedValue, index, input))
		if errProxy != nil {
			api.logger.Errorw("unable to create proxy", "error", errProxy.Error())
			return nil, nil, err
		}

		if api.cfg.Proxy.Username != "" {
			if api.cfg.Proxy.Password != "" {
				proxyUrl.User = url.UserPassword(utils.Format(api.cfg.Proxy.Username, parsedValue, index, input), utils.Format(api.cfg.Proxy.Password, parsedValue, index, input))
			} else {
				proxyUrl.User = url.User(utils.Format(api.cfg.Proxy.Username, parsedValue, index, input))
			}
		}
		api.logger.Debugw("set proxy", "server", api.cfg.Proxy.Server, "username", api.cfg.Proxy.Username, "password", api.cfg.Proxy.Password)
		client.Transport = &http.Transport{Proxy: http.ProxyURL(proxyUrl)}
	}

	if hostLimit := limitter.HostLimiter(req.Host); hostLimit != nil {
		errHostLimit := hostLimit.Acquire(ctx, 1)
		if errHostLimit != nil {
			api.logger.Errorw("unable to acquire host limit semaphore", "method", api.cfg.Method, "url", formattedURL, "error", errHostLimit.Error(), "host", req.Host)
			return nil, nil, errHostLimit
		}
		defer hostLimit.Release(1)
	}

	tt := timeout
	if api.cfg.Timeout > 0 {
		tt = time.Duration(api.cfg.Timeout) * time.Second
	}
	reqCtx, cancel := context.WithTimeout(ctx, tt)
	defer cancel()

	api.logger.Infow("sending request to url", "url", formattedURL, "body", formattedBody)
	resp, err := client.Do(req.WithContext(reqCtx))
	if err != nil {
		api.logger.Errorw("unable to send http request", "method", api.cfg.Method, "url", formattedURL, "error", err.Error())
		return nil, nil, err
	}

	if resp != nil && resp.Body != nil {
		defer resp.Body.Close()
	}

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		api.logger.Errorw("unable to read http response", "error", err.Error())
		return nil, nil, err
	}

	api.logger.Debugw("returned response", "status_code", resp.Status, "body", string(bytes))
	return resp.Header, bytes, nil
}

func (api *apiConnector) Get(parsedValue builder.Interfacable, index *uint32, input builder.Interfacable) ([]byte, error) {
	_, body, err := api.get(parsedValue, index, input)
	return body, err
}
