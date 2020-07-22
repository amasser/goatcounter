// Copyright © 2019 Martin Tournoij – This file is part of GoatCounter and
// published under the terms of a slightly modified EUPL v1.2 license, which can
// be found in the LICENSE file or at https://license.goatcounter.com

package handlers

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"zgo.at/errors"
	"zgo.at/goatcounter"
	"zgo.at/guru"
	"zgo.at/zdb"
	"zgo.at/zhttp"
	"zgo.at/zlog"
)

var (
	redirect = func(w http.ResponseWriter, r *http.Request) error {
		zhttp.Flash(w, "Need to log in")
		return guru.Errorf(303, "/user/new")
	}

	loggedIn = zhttp.Filter(func(w http.ResponseWriter, r *http.Request) error {
		u := goatcounter.GetUser(r.Context())
		if u != nil && u.ID > 0 {
			return nil
		}
		return redirect(w, r)
	})

	loggedInOrPublic = zhttp.Filter(func(w http.ResponseWriter, r *http.Request) error {
		u := goatcounter.GetUser(r.Context())
		if (u != nil && u.ID > 0) || goatcounter.MustGetSite(r.Context()).Settings.Public {
			return nil
		}
		return redirect(w, r)
	})

	noSubSites = zhttp.Filter(func(w http.ResponseWriter, r *http.Request) error {
		if goatcounter.MustGetSite(r.Context()).Parent == nil ||
			*goatcounter.MustGetSite(r.Context()).Parent == 0 {
			return nil
		}
		zlog.FieldsRequest(r).Errorf("noSubSites")
		return guru.Errorf(403, "child sites can't access this")
	})

	adminOnly = zhttp.Filter(func(w http.ResponseWriter, r *http.Request) error {
		if goatcounter.MustGetSite(r.Context()).Admin() {
			return nil
		}
		return guru.Errorf(404, "")
	})

	keyAuth = zhttp.Auth(func(ctx context.Context, key string) (zhttp.User, error) {
		u := &goatcounter.User{}
		err := u.ByTokenAndSite(ctx, key)
		return u, err
	})
)

// Allow debugging frontend timing issues by setting a "debug-delay" cookie.
func delay() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if c, _ := r.Cookie("debug-delay"); c != nil {
				n, _ := strconv.ParseInt(c.Value, 10, 32)
				zlog.Module("debug-delay").Printf("%ds delay", n)
				time.Sleep(time.Duration(n) * time.Second)
			}
			next.ServeHTTP(w, r)
		})
	}
}

func addctx(db zdb.DB, loadSite bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// Add timeout on non-admin pages.
			if !strings.HasPrefix(r.URL.Path, "/admin") {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(r.Context(), 5*time.Second)
				defer func() {
					cancel()
					if ctx.Err() == context.DeadlineExceeded {
						w.WriteHeader(http.StatusGatewayTimeout)
					}
				}()
			}

			// Add database.
			*r = *r.WithContext(zdb.With(ctx, db))

			// Load site from subdomain.
			if loadSite {
				var s goatcounter.Site
				err := s.ByHost(r.Context(), r.Host)
				if err != nil {
					if zdb.ErrNoRows(err) {
						zhttp.ErrPage(w, r, 400, errors.Errorf("no site at this domain (%q)", r.Host))
						return
					}

					zlog.Error(err)
					return
				}

				*r = *r.WithContext(goatcounter.WithSite(r.Context(), &s))
			}

			next.ServeHTTP(w, r)
		})
	}
}
