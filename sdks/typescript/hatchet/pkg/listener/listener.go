package listener

// import (
// 	"net/http"

// 	"github.com/go-chi/chi"
// 	"github.com/go-chi/chi/middleware"
// 	"github.com/hatchet-dev/hatchet/pkg/dispatcher"
// 	"github.com/hatchet-dev/hatchet/pkg/integrations"
// 	"github.com/hatchet-dev/hatchet/pkg/workflows/fileutils"
// 	"github.com/hatchet-dev/hatchet/pkg/workflows/types"
// )

// type listenerOptions struct {
// 	dispatcher     dispatcher.DispatcherInterface
// 	filesLoader    func() []*types.WorkflowFile
// 	webhooksLoader func(files []*types.WorkflowFile) []integrations.IntegrationWebhook
// }

// func defaultListenerOptions() *listenerOptions {
// 	return &listenerOptions{
// 		dispatcher:  dispatcher.NewDispatcher(),
// 		filesLoader: fileutils.DefaultLoader,
// 		webhooksLoader: func(_ []*types.WorkflowFile) []integrations.IntegrationWebhook {
// 			return []integrations.IntegrationWebhook{}
// 		},
// 	}
// }

// type listenerOptsFunc func(d *listenerOptions)

// func WithDispatcher(dispatcher dispatcher.DispatcherInterface) listenerOptsFunc {
// 	return func(opts *listenerOptions) {
// 		opts.dispatcher = dispatcher
// 	}
// }

// func WithWorkflowFiles(files []*types.WorkflowFile) listenerOptsFunc {
// 	return func(opts *listenerOptions) {
// 		opts.filesLoader = func() []*types.WorkflowFile {
// 			return files
// 		}
// 	}
// }

// // WithIntegrations registers all integrations with the listener. See [integrations.Integration] to see the interface
// // integrations must satisfy.
// func WithIntegrations(ints ...integrations.Integration) listenerOptsFunc {
// 	return func(opts *listenerOptions) {
// 		opts.webhooksLoader = func(files []*types.WorkflowFile) []integrations.IntegrationWebhook {
// 			webhooks := []integrations.IntegrationWebhook{}

// 			// flatten events into a lookup map
// 			events := map[string]bool{}

// 			for _, f := range files {
// 				for _, e := range f.On.Events {
// 					events[e] = true
// 				}
// 			}

// 			// only register the webhooks that are relevant to event triggers
// 			for _, i := range ints {
// 				for _, w := range i.GetWebhooks() {
// 					if _, exists := events[w.GetAction().String()]; exists {
// 						webhooks = append(webhooks, w)
// 					}
// 				}
// 			}

// 			return webhooks
// 		}
// 	}
// }

// type Listener struct {
// 	dispatcher dispatcher.DispatcherInterface

// 	files []*types.WorkflowFile

// 	webhooks []integrations.IntegrationWebhook
// }

// func NewListener(opts ...listenerOptsFunc) *Listener {
// 	options := defaultListenerOptions()

// 	for _, o := range opts {
// 		o(options)
// 	}

// 	return &Listener{
// 		dispatcher: options.dispatcher,
// 		files:      options.filesLoader(),
// 		webhooks:   options.webhooksLoader(options.filesLoader()),
// 	}
// }

// // Listen starts an HTTP server that listens for incoming webhooks and dispatches them to the appropriate workflow.
// // It uses chi under the hood.
// func (l *Listener) Listen() error {
// 	r := chi.NewRouter()

// 	r.Route("/", func(r chi.Router) {
// 		r.Use(middleware.Logger)

// 		for _, w := range l.webhooks {
// 			webhookCp := w

// 			r.Method(
// 				webhookCp.GetMethod(),
// 				webhookCp.GetDefaultPaths(),
// 				l.handlerFromWebhook(webhookCp),
// 			)
// 		}
// 	})

// 	// TODO: make port configurable
// 	return http.ListenAndServe(":3000", r)
// }

// func (l *Listener) handlerFromWebhook(webhook integrations.IntegrationWebhook) http.HandlerFunc {
// 	return func(w http.ResponseWriter, r *http.Request) {
// 		// validate the payload
// 		authErr := webhook.ValidatePayload(r)

// 		if authErr != nil {
// 			if code := authErr.StatusCode(); code != 0 {
// 				w.WriteHeader(code)
// 				w.Write([]byte("Unauthorized"))
// 				return
// 			}

// 			l.handleInternalError(w, authErr)
// 			return
// 		}

// 		// dispatch the action
// 		dispatchedData, err := webhook.GetData(r)

// 		if err != nil {
// 			l.handleInternalError(w, err)
// 			return
// 		}

// 		action := webhook.GetAction().String()

// 		// dispatch
// 		err = l.dispatcher.Trigger(action, dispatchedData)

// 		if err != nil {
// 			l.handleInternalError(w, err)
// 			return
// 		}

// 		w.WriteHeader(200)
// 	}
// }

// func (l *Listener) handleInternalError(w http.ResponseWriter, err error) {
// 	w.WriteHeader(500)
// 	w.Write([]byte("Internal Server Error"))

// 	// TODO: log an error here
// }
