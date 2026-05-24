// Lunex lang
// Created by David Dev · GitHub: https://github.com/Megamexlevi2
// (c) David Dev 2026. License.

package builtin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"lunex/internal/runtime"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"
	"golang.org/x/oauth2/google"
)

func oauth2ConfigObj(cfg *oauth2.Config) *runtime.Value {
	authURL := runtime.FuncVal(&runtime.Function{
		Name: "authURL",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			state := "state"
			if len(args) > 0 {
				state = args[0].ToString()
			}
			url := cfg.AuthCodeURL(state, oauth2.AccessTypeOnline)
			return runtime.StringVal(url), nil
		},
	})

	exchange := runtime.FuncVal(&runtime.Function{
		Name: "exchange",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 {
				return runtime.Null, fmt.Errorf("exchange(code)")
			}
			tok, err := cfg.Exchange(context.Background(), args[0].ToString())
			if err != nil {
				return runtime.Null, err
			}
			return runtime.ObjectVal(map[string]*runtime.Value{
				"accessToken":  runtime.StringVal(tok.AccessToken),
				"refreshToken": runtime.StringVal(tok.RefreshToken),
				"tokenType":    runtime.StringVal(tok.TokenType),
				"expiry":       runtime.StringVal(tok.Expiry.String()),
			}), nil
		},
	})

	refresh := runtime.FuncVal(&runtime.Function{
		Name: "refresh",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 || args[0].Tag != runtime.TypeObject {
				return runtime.Null, fmt.Errorf("refresh(token)")
			}
			opts := args[0].ObjVal
			tok := &oauth2.Token{}
			if v, ok := opts["accessToken"]; ok {
				tok.AccessToken = v.ToString()
			}
			if v, ok := opts["refreshToken"]; ok {
				tok.RefreshToken = v.ToString()
			}
			src := cfg.TokenSource(context.Background(), tok)
			newTok, err := src.Token()
			if err != nil {
				return runtime.Null, err
			}
			return runtime.ObjectVal(map[string]*runtime.Value{
				"accessToken":  runtime.StringVal(newTok.AccessToken),
				"refreshToken": runtime.StringVal(newTok.RefreshToken),
				"tokenType":    runtime.StringVal(newTok.TokenType),
			}), nil
		},
	})

	fetchUser := runtime.FuncVal(&runtime.Function{
		Name: "fetchUser",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) < 2 {
				return runtime.Null, fmt.Errorf("fetchUser(accessToken, userInfoURL)")
			}
			accessToken := args[0].ToString()
			userInfoURL := args[1].ToString()
			req, err := http.NewRequest("GET", userInfoURL, nil)
			if err != nil {
				return runtime.Null, err
			}
			req.Header.Set("Authorization", "Bearer "+accessToken)
			req.Header.Set("Accept", "application/json")
			client := &http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				return runtime.Null, err
			}
			defer resp.Body.Close()
			body, err := io.ReadAll(resp.Body)
			if err != nil {
				return runtime.Null, err
			}
			var raw interface{}
			if err := json.Unmarshal(body, &raw); err != nil {
				return runtime.StringVal(string(body)), nil
			}
			return jsonToValue(raw), nil
		},
	})

	return runtime.ObjectVal(map[string]*runtime.Value{
		"authURL":   authURL,
		"exchange":  exchange,
		"refresh":   refresh,
		"fetchUser": fetchUser,
	})
}

func OAuth2Module() *runtime.Value {
	create := runtime.FuncVal(&runtime.Function{
		Name: "create",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 || args[0].Tag != runtime.TypeObject {
				return runtime.Null, fmt.Errorf("create({clientId, clientSecret, redirectURL, scopes, provider?})")
			}
			opts := args[0].ObjVal
			cfg := &oauth2.Config{}
			if v, ok := opts["clientId"]; ok {
				cfg.ClientID = v.ToString()
			}
			if v, ok := opts["clientSecret"]; ok {
				cfg.ClientSecret = v.ToString()
			}
			if v, ok := opts["redirectURL"]; ok {
				cfg.RedirectURL = v.ToString()
			}
			if v, ok := opts["scopes"]; ok && v.Tag == runtime.TypeArray {
				for _, s := range v.ArrVal {
					cfg.Scopes = append(cfg.Scopes, s.ToString())
				}
			}
			provider := ""
			if v, ok := opts["provider"]; ok {
				provider = v.ToString()
			}
			switch provider {
			case "google":
				cfg.Endpoint = google.Endpoint
			case "github":
				cfg.Endpoint = github.Endpoint
			default:
				if v, ok := opts["authURL"]; ok {
					cfg.Endpoint.AuthURL = v.ToString()
				}
				if v, ok := opts["tokenURL"]; ok {
					cfg.Endpoint.TokenURL = v.ToString()
				}
			}
			return oauth2ConfigObj(cfg), nil
		},
	})

	google := runtime.FuncVal(&runtime.Function{
		Name: "google",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 || args[0].Tag != runtime.TypeObject {
				return runtime.Null, fmt.Errorf("google({clientId, clientSecret, redirectURL})")
			}
			opts := args[0].ObjVal
			cfg := &oauth2.Config{
				Endpoint: google.Endpoint,
				Scopes:   []string{"openid", "email", "profile"},
			}
			if v, ok := opts["clientId"]; ok {
				cfg.ClientID = v.ToString()
			}
			if v, ok := opts["clientSecret"]; ok {
				cfg.ClientSecret = v.ToString()
			}
			if v, ok := opts["redirectURL"]; ok {
				cfg.RedirectURL = v.ToString()
			}
			return oauth2ConfigObj(cfg), nil
		},
	})

	githubFn := runtime.FuncVal(&runtime.Function{
		Name: "github",
		Native: func(args []*runtime.Value, _ *runtime.Value) (*runtime.Value, error) {
			if len(args) == 0 || args[0].Tag != runtime.TypeObject {
				return runtime.Null, fmt.Errorf("github({clientId, clientSecret, redirectURL})")
			}
			opts := args[0].ObjVal
			cfg := &oauth2.Config{
				Endpoint: github.Endpoint,
				Scopes:   []string{"user:email"},
			}
			if v, ok := opts["clientId"]; ok {
				cfg.ClientID = v.ToString()
			}
			if v, ok := opts["clientSecret"]; ok {
				cfg.ClientSecret = v.ToString()
			}
			if v, ok := opts["redirectURL"]; ok {
				cfg.RedirectURL = v.ToString()
			}
			return oauth2ConfigObj(cfg), nil
		},
	})

	return runtime.ObjectVal(map[string]*runtime.Value{
		"create": create,
		"google": google,
		"github": githubFn,
	})
}
