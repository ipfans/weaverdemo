package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"

	"github.com/ServiceWeaver/weaver"
	"github.com/ipfans/weaverdemo/reverse"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func main() {
	if err := weaver.Run(context.Background(), serve); err != nil {
		log.Fatal(err)
	}
}

type app struct {
	weaver.Implements[weaver.Main]
	reverser weaver.Ref[reverse.Reverser]
	listener weaver.Listener
}

func serve(c context.Context, app *app) error {
	// The hello listener will listen on a random port chosen by the operating
	// system. This behavior can be changed in the config file.
	logger := app.Logger()
	logger.Info("Listener available", "addr", app.listener)

	// Serve the /hello endpoint.
	http.Handle("/hello",
		// Instrument the /hello endpoint to prometheus metrics.
		// The output `serviceweaver_http_request_bytes_received_bucket{host="127.0.0.1:12345",label="hello",serviceweaver_node="9a96cb5b",le="2000"} 1`
		weaver.InstrumentHandlerFunc("hello", func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()
			// OpenTelemetry trace evnet.
			trace.SpanFromContext(ctx).AddEvent("hello hit",
				trace.WithAttributes(
					attribute.String("name", r.URL.Query().Get("name")),
				),
			)
			name := r.URL.Query().Get("name")
			if name == "" {
				name = "World"
			}
			reversed, err := app.reverser.Get().Reverse(ctx, name)
			if err != nil {
				if errors.Is(err, weaver.RemoteCallError) {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			fmt.Fprintf(w, "Hello, %s!\n", reversed)
		}),
	)
	// Init handler
	otelHandler := otelhttp.NewHandler(http.DefaultServeMux, "http")
	return http.Serve(app.listener, otelHandler)
}

// // demo for gin, also can be extended to graphql etc.
// func serve(ctx context.Context, app *app) error {
// 	gin.SetMode(gin.ReleaseMode)
// 	r := gin.Default()
// 	r.GET("/hello", func(c *gin.Context) {
// 		name := c.Query("name")
// 		if name == "" {
// 			name = "World"
// 		}
// 		reversed, err := app.reverser.Get().Reverse(ctx, name)
// 		if err != nil {
// 			c.String(500, err.Error())
// 			return
// 		}
// 		c.String(200, "Hello, %s!\n", reversed)
// 	})
// 	return r.RunListener(app.listener)
// }
