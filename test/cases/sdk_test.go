package otelchi_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/riandyrn/otelchi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	"go.opentelemetry.io/otel/trace"
)

var traceID trace.TraceID = [16]byte{1}

var spanCtxSampled = trace.NewSpanContext(trace.SpanContextConfig{
	TraceID:    traceID,
	TraceFlags: trace.FlagsSampled,
})

var spanCtxNotSampled = trace.NewSpanContext(trace.SpanContextConfig{
	TraceID: traceID,
})

func TestSDKIntegration(t *testing.T) {
	// prepare router and span recorder
	router, sr := newSDKTestRouter("foobar", false)

	// define routes
	router.HandleFunc("/user/{id:[0-9]+}", ok)
	router.HandleFunc("/book/{title}", ok)

	// execute requests
	reqs := []*http.Request{
		httptest.NewRequest("GET", "/user/123", nil),
		httptest.NewRequest("GET", "/book/foo", nil),
	}
	executeRequests(router, reqs)

	// get recorded spans
	recordedSpans := sr.Ended()

	// ensure that we have 2 recorded spans
	require.Len(t, recordedSpans, len(reqs))

	// ensure span values
	checkSpans(t, recordedSpans, []spanValueCheck{
		{
			Name: "/user/{id:[0-9]+}",
			Kind: trace.SpanKindServer,
			Attributes: getSemanticAttributes(
				"foobar",
				http.StatusOK,
				"GET",
				"/user/{id:[0-9]+}",
			),
		},
		{
			Name: "/book/{title}",
			Kind: trace.SpanKindServer,
			Attributes: getSemanticAttributes(
				"foobar",
				http.StatusOK,
				"GET",
				"/book/{title}",
			),
		},
	})
}

func TestSDKIntegrationWithFilter(t *testing.T) {
	// prepare test cases
	serviceName := "foobar"
	testCases := []struct {
		Name               string
		FilterFn           []otelchi.Filter
		LenSpans           int
		ExpectedRouteNames []string
	}{
		{
			Name: "One WithFilter",
			FilterFn: []otelchi.Filter{
				func(r *http.Request) bool {
					return r.URL.Path != "/live" && r.URL.Path != "/ready"
				},
			},
			LenSpans:           2,
			ExpectedRouteNames: []string{"/user/{id:[0-9]+}", "/book/{title}"},
		},
		{
			Name: "Multiple WithFilter",
			FilterFn: []otelchi.Filter{
				func(r *http.Request) bool {
					return r.URL.Path != "/ready"
				},
				func(r *http.Request) bool {
					return r.URL.Path != "/live"
				},
			},
			LenSpans:           2,
			ExpectedRouteNames: []string{"/user/{id:[0-9]+}", "/book/{title}"},
		},
		{
			Name: "All Routes are traced",
			FilterFn: []otelchi.Filter{
				func(r *http.Request) bool {
					return true
				},
			},
			LenSpans:           4,
			ExpectedRouteNames: []string{"/user/{id:[0-9]+}", "/book/{title}", "/live", "/ready"},
		},
		{
			Name: "All Routes are not traced",
			FilterFn: []otelchi.Filter{
				func(r *http.Request) bool {
					return false
				},
			},
			LenSpans:           0,
			ExpectedRouteNames: []string{},
		},
	}

	// execute test cases
	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			// prepare router and span recorder
			filters := []otelchi.Option{}
			for _, filter := range testCase.FilterFn {
				filters = append(filters, otelchi.WithFilter(filter))
			}
			router, sr := newSDKTestRouter(serviceName, false, filters...)

			// define router
			router.HandleFunc("/user/{id:[0-9]+}", ok)
			router.HandleFunc("/book/{title}", ok)
			router.HandleFunc("/health", ok)
			router.HandleFunc("/live", ok)
			router.HandleFunc("/ready", ok)

			// execute requests
			executeRequests(router, []*http.Request{
				httptest.NewRequest("GET", "/user/123", nil),
				httptest.NewRequest("GET", "/book/foo", nil),
				httptest.NewRequest("GET", "/live", nil),
				httptest.NewRequest("GET", "/ready", nil),
			})

			// check recorded spans
			recordedSpans := sr.Ended()
			require.Len(t, recordedSpans, testCase.LenSpans)

			// ensure span values
			spanValues := []spanValueCheck{}
			for _, routeName := range testCase.ExpectedRouteNames {
				spanValues = append(spanValues, spanValueCheck{
					Name: routeName,
					Kind: trace.SpanKindServer,
					Attributes: getSemanticAttributes(
						serviceName,
						http.StatusOK,
						"GET",
						routeName,
					),
				})
			}
			checkSpans(t, recordedSpans, spanValues)
		})
	}

}

func TestSDKIntegrationWithChiRoutes(t *testing.T) {
	// define router & span recorder
	router, sr := newSDKTestRouter("foobar", true)

	// define route
	router.HandleFunc("/user/{id:[0-9]+}", ok)
	router.HandleFunc("/book/{title}", ok)

	// execute requests
	reqs := []*http.Request{
		httptest.NewRequest("GET", "/user/123", nil),
		httptest.NewRequest("GET", "/book/foo", nil),
	}
	executeRequests(router, reqs)

	// get recorded spans
	recordedSpans := sr.Ended()

	// ensure that we have 2 recorded spans
	require.Len(t, recordedSpans, len(reqs))

	// ensure span values
	checkSpans(t, recordedSpans, []spanValueCheck{
		{
			Name: "/user/{id:[0-9]+}",
			Kind: trace.SpanKindServer,
			Attributes: getSemanticAttributes(
				"foobar",
				http.StatusOK,
				"GET",
				"/user/{id:[0-9]+}",
			),
		},
		{
			Name: "/book/{title}",
			Kind: trace.SpanKindServer,
			Attributes: getSemanticAttributes(
				"foobar",
				http.StatusOK,
				"GET",
				"/book/{title}",
			),
		},
	})
}

func TestSDKIntegrationOverrideSpanName(t *testing.T) {
	// prepare test router and span recorder
	router, sr := newSDKTestRouter("foobar", true)

	// define route
	router.HandleFunc("/user/{id:[0-9]+}", func(w http.ResponseWriter, r *http.Request) {
		span := trace.SpanFromContext(r.Context())
		span.SetName("overriden span name")
		w.WriteHeader(http.StatusOK)
	})
	router.HandleFunc("/book/{title}", ok)

	// execute requests
	reqs := []*http.Request{
		httptest.NewRequest("GET", "/user/123", nil),
		httptest.NewRequest("GET", "/book/foo", nil),
	}
	executeRequests(router, reqs)

	// get recorded spans
	recordedSpans := sr.Ended()

	// ensure the number of spans is correct
	require.Len(t, sr.Ended(), len(reqs))

	// check span values
	checkSpans(t, recordedSpans, []spanValueCheck{
		{
			Name: "overriden span name",
			Kind: trace.SpanKindServer,
			Attributes: getSemanticAttributes(
				"foobar",
				http.StatusOK,
				"GET",
				"/user/{id:[0-9]+}",
			),
		},
		{
			Name: "/book/{title}",
			Kind: trace.SpanKindServer,
			Attributes: getSemanticAttributes(
				"foobar",
				http.StatusOK,
				"GET",
				"/book/{title}",
			),
		},
	})
}

func TestSDKIntegrationWithRequestMethodInSpanName(t *testing.T) {
	// prepare router & span recorder
	router, sr := newSDKTestRouter("foobar", true, otelchi.WithRequestMethodInSpanName(true))

	// define handler
	router.HandleFunc("/user/{id:[0-9]+}", ok)
	router.HandleFunc("/book/{title}", ok)

	// execute requests
	reqs := []*http.Request{
		httptest.NewRequest("GET", "/user/123", nil),
		httptest.NewRequest("GET", "/book/foo", nil),
	}
	executeRequests(router, reqs)

	// get recorded spans & ensure the number is correct
	recordedSpans := sr.Ended()
	require.Len(t, sr.Ended(), len(reqs))

	// check span values
	checkSpans(t, recordedSpans, []spanValueCheck{
		{
			Name: "GET /user/{id:[0-9]+}",
			Kind: trace.SpanKindServer,
			Attributes: getSemanticAttributes(
				"foobar",
				http.StatusOK,
				"GET",
				"/user/{id:[0-9]+}",
			),
		},
		{
			Name: "GET /book/{title}",
			Kind: trace.SpanKindServer,
			Attributes: getSemanticAttributes(
				"foobar",
				http.StatusOK,
				"GET",
				"/book/{title}",
			),
		},
	})
}

func TestSDKIntegrationEmptyHandlerDefaultStatusCode(t *testing.T) {
	// prepare router and span recorder
	router, sr := newSDKTestRouter("foobar", false)

	// define routes
	router.HandleFunc("/user/{id:[0-9]+}", func(w http.ResponseWriter, r *http.Request) {})
	router.HandleFunc("/book/{title}", func(w http.ResponseWriter, r *http.Request) {})
	router.HandleFunc("/not-found", http.NotFound)

	// execute requests
	reqs := []*http.Request{
		httptest.NewRequest("GET", "/user/123", nil),
		httptest.NewRequest("GET", "/book/foo", nil),
		httptest.NewRequest("GET", "/not-found", nil),
	}
	executeRequests(router, reqs)

	// get recorded spans
	recordedSpans := sr.Ended()

	// ensure that we have 3 recorded spans
	require.Len(t, recordedSpans, len(reqs))

	// ensure span values
	checkSpans(t, recordedSpans, []spanValueCheck{
		{
			Name: "/user/{id:[0-9]+}",
			Kind: trace.SpanKindServer,
			Attributes: getSemanticAttributes(
				"foobar",
				http.StatusOK,
				"GET",
				"/user/{id:[0-9]+}",
			),
		},
		{
			Name: "/book/{title}",
			Kind: trace.SpanKindServer,
			Attributes: getSemanticAttributes(
				"foobar",
				http.StatusOK,
				"GET",
				"/book/{title}",
			),
		},
		{
			Name: "/not-found",
			Kind: trace.SpanKindServer,
			Attributes: getSemanticAttributes(
				"foobar",
				http.StatusNotFound,
				"GET",
				"/not-found",
			),
		},
	})
}

func TestSDKIntegrationRootHandler(t *testing.T) {
	// prepare router and span recorder
	router, sr := newSDKTestRouter("foobar", true)

	// define routes
	router.HandleFunc("/", ok)

	// execute requests
	reqs := []*http.Request{
		httptest.NewRequest("GET", "/", nil),
	}
	executeRequests(router, reqs)

	// get recorded spans
	recordedSpans := sr.Ended()

	// ensure that we have 1 recorded span
	require.Len(t, recordedSpans, 1)

	// ensure span values
	checkSpans(t, recordedSpans, []spanValueCheck{
		{
			Name: "/",
			Kind: trace.SpanKindServer,
			Attributes: getSemanticAttributes(
				"foobar",
				http.StatusOK,
				"GET",
				"/",
			),
		},
	})
}

func TestSDKIntegrationWithTraceIDResponseHeader(t *testing.T) {
	testCases := []struct {
		Name          string
		HeaderKeyFunc func() string
		ExpHeaderName string
	}{
		{
			Name:          "With Default Header Key",
			HeaderKeyFunc: nil,
			ExpHeaderName: otelchi.DefaultTraceResponseHeaderKey,
		},
		{
			Name:          "With Custom Header Key",
			HeaderKeyFunc: func() string { return "X-Custom-Trace-ID" },
			ExpHeaderName: "X-Custom-Trace-ID",
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			router, sr := newSDKTestRouter(
				"foobar",
				true,
				otelchi.WithTraceIDResponseHeader(testCase.HeaderKeyFunc),
			)
			router.HandleFunc("/user/{id:[0-9]+}", ok)
			router.HandleFunc("/book/{title}", ok)

			r0 := httptest.NewRequest("GET", "/user/123", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, r0)

			recordedSpans := sr.Ended()
			require.Len(t, recordedSpans, 1)
			checkSpans(t, recordedSpans, []spanValueCheck{
				{
					Name: "/user/{id:[0-9]+}",
					Kind: trace.SpanKindServer,
					Attributes: getSemanticAttributes(
						"foobar",
						http.StatusOK,
						"GET",
						"/user/{id:[0-9]+}",
					),
				},
			})
			require.Equalf(
				t,
				recordedSpans[0].SpanContext().TraceID().String(),
				w.Header().Get(testCase.ExpHeaderName),
				"the expected header key: `%v` is not found in the response header",
				testCase.ExpHeaderName,
			)
		})
	}
}

func TestSDKIntegrationWithoutWithTraceIDResponseHeader(t *testing.T) {
	router, sr := newSDKTestRouter("foobar", true)

	router.HandleFunc("/user/{id:[0-9]+}", ok)
	router.HandleFunc("/book/{title}", ok)

	r0 := httptest.NewRequest("GET", "/user/123", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, r0)

	recordedSpans := sr.Ended()
	require.Len(t, recordedSpans, 1)

	checkSpans(t, recordedSpans, []spanValueCheck{
		{
			Name: "/user/{id:[0-9]+}",
			Kind: trace.SpanKindServer,
			Attributes: getSemanticAttributes(
				"foobar",
				http.StatusOK,
				"GET",
				"/user/{id:[0-9]+}",
			),
		},
	})

	require.Empty(t, w.Header().Get(otelchi.DefaultTraceResponseHeaderKey))
}

func TestWithPublicEndpoint(t *testing.T) {
	// prepare router and span recorder
	router, spanRecorder := newSDKTestRouter("foobar", true, otelchi.WithPublicEndpoint())

	// prepare remote span context
	remoteSpanCtx := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: trace.TraceID{0x01},
		SpanID:  trace.SpanID{0x01},
		Remote:  true,
	})

	// prepare http request & inject remote span context into it
	endpointURL := "/with/public/endpoint"
	req := httptest.NewRequest(http.MethodGet, endpointURL, nil)
	ctx := trace.ContextWithSpanContext(context.Background(), remoteSpanCtx)
	(propagation.TraceContext{}).Inject(ctx, propagation.HeaderCarrier(req.Header))

	// configure router handler
	router.HandleFunc(endpointURL, func(w http.ResponseWriter, r *http.Request) {
		// get span from request context
		span := trace.SpanFromContext(r.Context())
		spanCtx := span.SpanContext()

		// ensure it is not equal to the remote span context
		require.False(t, spanCtx.Equal(remoteSpanCtx))
		require.True(t, spanCtx.IsValid())
		require.False(t, spanCtx.IsRemote())
	})

	// execute http request
	executeRequests(router, []*http.Request{req})

	// get recorded spans
	recordedSpans := spanRecorder.Ended()
	require.Len(t, recordedSpans, 1)

	links := recordedSpans[0].Links()
	require.Len(t, links, 1)
	require.True(t, remoteSpanCtx.Equal(links[0].SpanContext))
}

func TestWithPublicEndpointFn(t *testing.T) {
	// prepare remote span context
	remoteSpanCtx := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: trace.TraceID{0x01},
		SpanID:  trace.SpanID{0x01},
		Remote:  true,
	})

	// prepare test cases
	testCases := []struct {
		Name          string
		Fn            func(r *http.Request) bool
		HandlerAssert func(t *testing.T, spanCtx trace.SpanContext)
		SpansAssert   func(t *testing.T, spanCtx trace.SpanContext, spans []sdktrace.ReadOnlySpan)
	}{
		{
			Name: "Function Always Return True",
			Fn:   func(r *http.Request) bool { return true },
			HandlerAssert: func(t *testing.T, spanCtx trace.SpanContext) {
				// ensure it is not equal to the remote span context
				require.False(t, spanCtx.Equal(remoteSpanCtx))
				require.True(t, spanCtx.IsValid())

				// ensure it is not remote span
				require.False(t, spanCtx.IsRemote())
			},
			SpansAssert: func(t *testing.T, spanCtx trace.SpanContext, spans []sdktrace.ReadOnlySpan) {
				// ensure spans length
				require.Len(t, spans, 1)

				// ensure the span has been linked
				links := spans[0].Links()
				require.Len(t, links, 1)
				require.True(t, remoteSpanCtx.Equal(links[0].SpanContext))
			},
		},
		{
			Name: "Function Always Return False",
			Fn:   func(r *http.Request) bool { return false },
			HandlerAssert: func(t *testing.T, spanCtx trace.SpanContext) {
				// ensure the span is child of the remote span
				require.Equal(t, remoteSpanCtx.TraceID(), spanCtx.TraceID())
				require.True(t, spanCtx.IsValid())

				// ensure it is not remote span
				require.False(t, spanCtx.IsRemote())
			},
			SpansAssert: func(t *testing.T, spanCtx trace.SpanContext, spans []sdktrace.ReadOnlySpan) {
				// ensure spans length
				require.Len(t, spans, 1, "unexpected span length")

				// ensure the span has no links
				links := spans[0].Links()
				require.Len(t, links, 0)
			},
		},
	}

	// execute test cases
	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {

			// prepare router and span recorder
			router, spanRecorder := newSDKTestRouter(
				"foobar",
				true,
				otelchi.WithPublicEndpointFn(testCase.Fn),
			)

			// prepare http request & inject remote span context into it
			endpointURL := "/with/public/endpoint"
			req := httptest.NewRequest(http.MethodGet, endpointURL, nil)
			ctx := trace.ContextWithSpanContext(context.Background(), remoteSpanCtx)
			(propagation.TraceContext{}).Inject(ctx, propagation.HeaderCarrier(req.Header))

			// configure router handler
			router.HandleFunc(endpointURL, func(w http.ResponseWriter, r *http.Request) {
				// assert handler
				span := trace.SpanFromContext(r.Context())
				testCase.HandlerAssert(t, span.SpanContext())
			})

			// execute http request
			executeRequests(router, []*http.Request{req})

			// assert recorded spans
			testCase.SpansAssert(t, remoteSpanCtx, spanRecorder.Ended())
		})
	}
}

func TestSDKIntegrationResponseModifierTraceIDResponseHeader(t *testing.T) {

	testCases := []struct {
		Name           string
		Option         otelchi.ResponseModifierTraceIDResponseHeaderOption
		SpanContext    trace.SpanContext
		ExpHeaderName  string
		ExpHeaderValue string
	}{
		{
			Name:           "With Default Option - Trace Sampled",
			Option:         otelchi.ResponseModifierTraceIDResponseHeaderOption{},
			SpanContext:    spanCtxSampled,
			ExpHeaderName:  otelchi.DefaultTraceResponseHeaderKey,
			ExpHeaderValue: traceID.String(),
		},
		{
			Name:           "With Default Option - Trace Not Sampled",
			Option:         otelchi.ResponseModifierTraceIDResponseHeaderOption{},
			SpanContext:    spanCtxNotSampled,
			ExpHeaderName:  otelchi.DefaultTraceResponseHeaderKey,
			ExpHeaderValue: traceID.String(),
		},
		{
			Name: "With Custom Header Key - Trace Sampled",
			Option: otelchi.ResponseModifierTraceIDResponseHeaderOption{
				HeaderKeyFunc: func() string { return "X-Custom-Trace-ID" },
			},
			SpanContext:    spanCtxSampled,
			ExpHeaderName:  "X-Custom-Trace-ID",
			ExpHeaderValue: traceID.String(),
		},
		{
			Name: "With Custom Header Key - Trace Not Sampled",
			Option: otelchi.ResponseModifierTraceIDResponseHeaderOption{
				HeaderKeyFunc: func() string { return "X-Custom-Trace-ID" },
			},
			SpanContext:    spanCtxNotSampled,
			ExpHeaderName:  "X-Custom-Trace-ID",
			ExpHeaderValue: traceID.String(),
		},
		{
			Name: "With Include Sampled Status - Trace Sampled",
			Option: otelchi.ResponseModifierTraceIDResponseHeaderOption{
				IncludeSampledStatus: true,
			},
			SpanContext:    spanCtxSampled,
			ExpHeaderName:  otelchi.DefaultTraceResponseHeaderKey,
			ExpHeaderValue: traceID.String() + "; sampled=true",
		},
		{
			Name: "With Include Sampled Status - Trace Not Sampled",
			Option: otelchi.ResponseModifierTraceIDResponseHeaderOption{
				IncludeSampledStatus: true,
			},
			SpanContext:    spanCtxNotSampled,
			ExpHeaderName:  otelchi.DefaultTraceResponseHeaderKey,
			ExpHeaderValue: traceID.String() + "; sampled=false",
		},
		{
			Name: "With Custom Header Key & Include Sampled Status - Trace Sampled",
			Option: otelchi.ResponseModifierTraceIDResponseHeaderOption{
				HeaderKeyFunc:        func() string { return "X-Custom-Trace-ID" },
				IncludeSampledStatus: true,
			},
			SpanContext:    spanCtxSampled,
			ExpHeaderName:  "X-Custom-Trace-ID",
			ExpHeaderValue: traceID.String() + "; sampled=true",
		},
		{
			Name: "With Custom Header Key & Include Sampled Status - Trace Not Sampled",
			Option: otelchi.ResponseModifierTraceIDResponseHeaderOption{
				HeaderKeyFunc:        func() string { return "X-Custom-Trace-ID" },
				IncludeSampledStatus: true,
			},
			SpanContext:    spanCtxNotSampled,
			ExpHeaderName:  "X-Custom-Trace-ID",
			ExpHeaderValue: traceID.String() + "; sampled=false",
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			// define the router, here we are using global tracer provider
			// which provides "pass through" spans for any span context in the
			// incoming request context.
			router := chi.NewRouter()
			router.Use(
				otelchi.Middleware(
					"foobar",
					otelchi.WithChiRoutes(router),
					otelchi.WithResponseModifier(
						otelchi.ResponseModifierTraceIDResponseHeader(testCase.Option),
					),
				),
			)

			router.HandleFunc("/user/{id:[0-9]+}", ok)
			router.HandleFunc("/book/{title}", ok)

			r0 := httptest.NewRequest("GET", "/user/123", nil)
			r0 = r0.WithContext(trace.ContextWithSpanContext(context.Background(), testCase.SpanContext))

			w := httptest.NewRecorder()
			router.ServeHTTP(w, r0)

			// check if expected response header is set
			require.NotEmpty(t, w.Header().Get(testCase.ExpHeaderName))

			// check if the expected value is correct
			require.Equal(t, testCase.ExpHeaderValue, w.Header().Get(testCase.ExpHeaderName))
		})
	}
}

func TestSDKIntegrationWithResponseModifier(t *testing.T) {
	// In this test case we will exercise the response modifier feature
	// to simulate use case mentioned in this PR: https://github.com/riandyrn/otelchi/pull/56
	// which essentially don't add trace id to response header if the trace is not sampled.

	// define response modifier
	headerKey := otelchi.DefaultTraceResponseHeaderKey
	respMod := func(ctx context.Context, w http.ResponseWriter) {
		spanCtx := trace.SpanFromContext(ctx).SpanContext()
		if spanCtx.IsSampled() {
			w.Header().Set(headerKey, spanCtx.TraceID().String())
		}
	}

	// define next middleware, we should be able to access the response header
	// set by the response modifier.
	mdlwr := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			spanCtx := trace.SpanFromContext(r.Context()).SpanContext()
			headerVal := w.Header().Get(headerKey)
			if spanCtx.IsSampled() {
				// we should be able to access the response header
				require.NotEmpty(t, headerVal)
			} else {
				require.Empty(t, headerVal)
			}
			next.ServeHTTP(w, r)
		})
	}

	// define router, here we are using global tracer provider
	// which provides "pass through" spans for any span context in the
	// incoming request context.
	router := chi.NewRouter()
	router.Use(
		otelchi.Middleware(
			"foobar",
			otelchi.WithChiRoutes(router),
			otelchi.WithResponseModifier(respMod),
		),
		mdlwr,
	)
	router.HandleFunc("/user/{id:[0-9]+}", ok)

	// execute test cases
	testCases := []struct {
		Name            string
		SpanContext     trace.SpanContext
		ExpHeaderExists bool
	}{
		{
			Name:            "Trace Sampled",
			SpanContext:     spanCtxSampled,
			ExpHeaderExists: true,
		},
		{
			Name:            "Trace Not Sampled",
			SpanContext:     spanCtxNotSampled,
			ExpHeaderExists: false,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			r0 := httptest.NewRequest("GET", "/user/123", nil)
			r0 = r0.WithContext(trace.ContextWithSpanContext(context.Background(), testCase.SpanContext))

			w := httptest.NewRecorder()
			router.ServeHTTP(w, r0)

			if testCase.ExpHeaderExists {
				require.NotEmpty(t, w.Header().Get(headerKey))
			} else {
				require.Empty(t, w.Header().Get(headerKey))
			}
		})
	}
}

func assertSpan(t *testing.T, span sdktrace.ReadOnlySpan, name string, kind trace.SpanKind, attrs ...attribute.KeyValue) {
	assert.Equal(t, name, span.Name())
	assert.Equal(t, kind, span.SpanKind())

	got := make(map[attribute.Key]attribute.Value, len(span.Attributes()))
	for _, a := range span.Attributes() {
		got[a.Key] = a.Value
	}
	for _, want := range attrs {
		if !assert.Contains(t, got, want.Key) {
			continue
		}
		assert.Equal(t, want.Value, got[want.Key])
	}
}

func ok(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
}

func newSDKTestRouter(serverName string, withChiRoutes bool, opts ...otelchi.Option) (*chi.Mux, *tracetest.SpanRecorder) {
	spanRecorder := tracetest.NewSpanRecorder()
	tracerProvider := sdktrace.NewTracerProvider(
		// set the tracer provider to always sample trace, this is important
		// because if we don't set this, sometimes there are traces that
		// won't be sampled (recorded), so we need to set this option
		// to ensure every trace in this test is recorded.
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)
	tracerProvider.RegisterSpanProcessor(spanRecorder)

	opts = append(opts, otelchi.WithTracerProvider(tracerProvider))

	router := chi.NewRouter()
	if withChiRoutes {
		opts = append(opts, otelchi.WithChiRoutes(router))
	}
	router.Use(otelchi.Middleware(serverName, opts...))

	return router, spanRecorder
}

type spanValueCheck struct {
	Name       string
	Kind       trace.SpanKind
	Attributes []attribute.KeyValue
}

func getSemanticAttributes(serverName string, httpStatusCode int, httpMethod, httpRoute string) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("net.host.name", serverName),
		attribute.Int("http.status_code", httpStatusCode),
		attribute.String("http.method", httpMethod),
		attribute.String("http.route", httpRoute),
	}
}

func checkSpans(t *testing.T, spans []sdktrace.ReadOnlySpan, valueChecks []spanValueCheck) {
	for i := 0; i < len(spans); i++ {
		span := spans[i]
		valueCheck := valueChecks[i]
		assertSpan(t, span, valueCheck.Name, valueCheck.Kind, valueCheck.Attributes...)
	}
}

func executeRequests(router *chi.Mux, reqs []*http.Request) {
	w := httptest.NewRecorder()
	for _, r := range reqs {
		router.ServeHTTP(w, r)
	}
}
