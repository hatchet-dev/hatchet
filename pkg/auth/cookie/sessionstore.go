package cookie

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/jackc/pgx/v5"

	"github.com/hatchet-dev/hatchet/pkg/repository"
)

const UserSessionKey string = "user_id"

type sessionDataJSON struct {
	Data []byte `json:"data"`
}

type UserSessionStore struct {
	codecs     []securecookie.Codec
	options    *sessions.Options
	repo       repository.UserSessionRepository
	cookieName string
}

type UserSessionStoreOpts struct {
	// The max age of the cookie, in seconds.
	maxAge int

	repo          repository.UserSessionRepository
	cookieSecrets []string
	isInsecure    bool
	cookieDomain  string
	cookieName    string
}

type UserSessionStoreOpt func(*UserSessionStoreOpts)

func defaultUserSessionStoreOpts() *UserSessionStoreOpts {
	return &UserSessionStoreOpts{
		maxAge: 86400 * 30,
		cookieSecrets: []string{
			"secret1",
			"secret2",
		},
		isInsecure:   false,
		cookieDomain: "",
		cookieName:   "hatchet",
	}
}

func WithCookieSecrets(secrets ...string) UserSessionStoreOpt {
	return func(opts *UserSessionStoreOpts) {
		opts.cookieSecrets = secrets
	}
}

func WithCookieDomain(domain string) UserSessionStoreOpt {
	return func(opts *UserSessionStoreOpts) {
		opts.cookieDomain = domain
	}
}

func WithCookieName(name string) UserSessionStoreOpt {
	return func(opts *UserSessionStoreOpts) {
		opts.cookieName = name
	}
}

func WithCookieAllowInsecure(allow bool) UserSessionStoreOpt {
	return func(opts *UserSessionStoreOpts) {
		opts.isInsecure = allow
	}
}

func WithSessionRepository(repo repository.UserSessionRepository) UserSessionStoreOpt {
	return func(opts *UserSessionStoreOpts) {
		opts.repo = repo
	}
}

func NewUserSessionStore(fs ...UserSessionStoreOpt) (*UserSessionStore, error) {
	opts := defaultUserSessionStoreOpts()

	for _, f := range fs {
		f(opts)
	}

	// user session store is required
	if opts.repo == nil {
		return nil, errors.New("session repository is required. use WithSessionRepository.")
	}

	// cookie domain is required
	if opts.cookieDomain == "" {
		return nil, errors.New("cookie domain is required. use WithCookieDomain.")
	}

	if len(opts.cookieSecrets) == 0 || len(opts.cookieSecrets)%2 != 0 {
		return nil, errors.New("at least one cookie secret must be provided, and must provide an even number of secrets")
	}

	keyPairs := [][]byte{}

	for _, key := range opts.cookieSecrets {
		keyPairs = append(keyPairs, []byte(key))
	}

	res := &UserSessionStore{
		codecs: securecookie.CodecsFromPairs(keyPairs...),
		options: &sessions.Options{
			Path:     "/",
			Domain:   opts.cookieDomain,
			MaxAge:   86400 * 30,
			Secure:   !opts.isInsecure,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		},
		repo:       opts.repo,
		cookieName: opts.cookieName,
	}

	return res, nil
}

func (store *UserSessionStore) GetName() string {
	return store.cookieName
}

func (store *UserSessionStore) New(r *http.Request, name string) (*sessions.Session, error) {
	session := sessions.NewSession(store, name)

	if session == nil {
		return nil, fmt.Errorf("could not create new session")
	}

	opts := *store.options
	session.Options = &(opts)
	session.IsNew = true

	var err error
	if c, errCookie := r.Cookie(name); errCookie == nil {
		err = securecookie.DecodeMulti(name, c.Value, &session.ID, store.codecs...)
		if err == nil {
			err = store.load(r.Context(), session)

			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					err = nil
				} else if strings.Contains(err.Error(), "expired timestamp") {
					err = nil
					session.IsNew = false
				}
			} else {
				session.IsNew = false
			}
		} else if strings.Contains(err.Error(), "the value is not valid") {
			// this error occurs if the encryption keys have been rotated, in which case we'd like a new cookie
			err = nil
			session.IsNew = true
		}
	}

	store.MaxAge(store.options.MaxAge)

	return session, err
}

// Get Fetches a session for a given name after it has been added to the
// registry.
func (store *UserSessionStore) Get(r *http.Request, name string) (*sessions.Session, error) {
	return sessions.GetRegistry(r).Get(store, name)
}

// Save saves the given session into the database and deletes cookies if needed
func (store *UserSessionStore) Save(r *http.Request, w http.ResponseWriter, session *sessions.Session) error {
	repo := store.repo

	// Set delete if max-age is < 0
	if session.Options.MaxAge < 0 {
		if _, err := repo.Delete(r.Context(), session.ID); err != nil {
			return err
		}

		http.SetCookie(w, sessions.NewCookie(session.Name(), "", session.Options))
		return nil
	}

	if session.ID == "" {
		// Generate a random session UUID
		session.ID = uuid.New().String()
	}

	if err := store.save(r.Context(), session); err != nil {
		return err
	}

	// Keep the session ID key in a cookie so it can be looked up in DB later.
	encoded, err := securecookie.EncodeMulti(session.Name(), session.ID, store.codecs...)
	if err != nil {
		return err
	}

	http.SetCookie(w, sessions.NewCookie(session.Name(), encoded, session.Options))
	return nil
}

// save writes encoded session.Values to a database record.
// writes to http_sessions table by default.
func (store *UserSessionStore) save(ctx context.Context, session *sessions.Session) error {
	if session.ID == "" {
		return fmt.Errorf("session ID required but not set")
	}

	encoded, err := securecookie.EncodeMulti(session.Name(), session.Values, store.codecs...)
	if err != nil {
		return err
	}

	exOn := session.Values["expires_on"]

	var expiresOn time.Time

	if exOn == nil {
		expiresOn = time.Now().UTC().Add(time.Second * time.Duration(session.Options.MaxAge))
	} else {
		expiresOn = exOn.(time.Time)
		if expiresOn.Sub(time.Now().UTC().Add(time.Second*time.Duration(session.Options.MaxAge))) < 0 {
			expiresOn = time.Now().UTC().Add(time.Second * time.Duration(session.Options.MaxAge))
		}
	}

	var userId *string

	if userIDInt, exists := session.Values[UserSessionKey]; exists && userIDInt != nil {
		userIdStr := userIDInt.(string)
		userId = &userIdStr
	}

	jsonData := &sessionDataJSON{
		Data: []byte(encoded),
	}

	jsonBytes, err := json.Marshal(jsonData)

	if err != nil {
		return err
	}

	repo := store.repo

	_, err = repo.GetById(ctx, session.ID)

	if err != nil && errors.Is(err, pgx.ErrNoRows) {
		_, err := repo.Create(ctx, &repository.CreateSessionOpts{
			ID:        session.ID,
			Data:      jsonBytes,
			ExpiresAt: expiresOn,
			UserId:    userId,
		})

		return err
	}

	_, err = repo.Update(ctx, session.ID, &repository.UpdateSessionOpts{
		Data:   jsonBytes,
		UserId: userId,
	})

	return err
}

// load fetches a session by ID from the database and decodes its content
// into session.Values.
func (store *UserSessionStore) load(ctx context.Context, session *sessions.Session) error {
	res, err := store.repo.GetById(ctx, session.ID)
	if err != nil {
		return err
	}

	data := sessionDataJSON{}

	if len(res.Data) > 0 {
		err = json.Unmarshal(res.Data, &data)

		if err != nil {
			return err
		}
	}

	return securecookie.DecodeMulti(session.Name(), string(data.Data), &session.Values, store.codecs...)
}

// MaxLength restricts the maximum length of new sessions to l.
// If l is 0 there is no limit to the size of a session, use with caution.
// The default for a new PGStore is 4096. PostgreSQL allows for max
// value sizes of up to 1GB (http://www.postgresql.org/docs/current/interactive/datatype-character.html)
func (store *UserSessionStore) MaxLength(l int) {
	for _, c := range store.codecs {
		if codec, ok := c.(*securecookie.SecureCookie); ok {
			codec.MaxLength(l)
		}
	}
}

// MaxAge sets the maximum age for the store and the underlying cookie
// implementation. Individual sessions can be deleted by setting Options.MaxAge
// = -1 for that session.
func (store *UserSessionStore) MaxAge(age int) {
	store.options.MaxAge = age

	// Set the maxAge for each securecookie instance.
	for _, codec := range store.codecs {
		if sc, ok := codec.(*securecookie.SecureCookie); ok {
			sc.MaxAge(age)
		}
	}
}
