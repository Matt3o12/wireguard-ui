package router

import (
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"reflect"

	rice "github.com/GeertJohan/go.rice"
	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/labstack/gommon/log"
)

// TemplateRegistry is a custom html/template renderer for Echo framework
type TemplateRegistry struct {
	templates map[string]*template.Template
	extraData map[string]string
}

// Render e.Renderer interface
func (t *TemplateRegistry) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	tmpl, ok := t.templates[name]
	if !ok {
		return fmt.Errorf("Template not found: %q", name)
	}

	// inject more app data information. E.g. appVersion
	if reflect.TypeOf(data).Kind() == reflect.Map {
		for k, v := range t.extraData {
			data.(map[string]interface{})[k] = v
		}
	}

	return tmpl.Execute(w, data)
}

// riceFS is an implementation of fs.FS for rice.Box
type riceFS struct {
	fs *rice.Box
}

func (f riceFS) Open(name string) (fs.File, error) {
	return f.fs.Open(name)
}

func loadTemplates(tmplBox *rice.Box) map[string]*template.Template {
	templates := make(map[string]*template.Template)
	// loadTempls loads the template with the given name and all sub templates
	type templatePaths []string
	loader := func(paths ...templatePaths) {
		for _, files := range paths {
			name := files[0]
			tmpl, err := template.New(name).ParseFS(riceFS{tmplBox}, files...)
			if err != nil {
				log.Fatal("Error loading template: %q: %v", name, err)
				return
			}

			templates[name] = tmpl
		}

	}

	loader(
		templatePaths{"login.html"},
		templatePaths{"clients.html", "base.html"},
		templatePaths{"server.html", "base.html"},
		templatePaths{"global_settings.html", "base.html"},
		templatePaths{"status.html", "base.html"},
	)

	return templates
}

// New function
func New(tmplBox *rice.Box, extraData map[string]string, secret []byte) *echo.Echo {
	e := echo.New()
	e.Use(session.Middleware(sessions.NewCookieStore(secret)))

	// create template list
	e.Logger.SetLevel(log.DEBUG)
	e.Pre(middleware.RemoveTrailingSlash())
	e.Use(middleware.Logger())
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
		AllowMethods: []string{echo.GET, echo.HEAD, echo.PUT, echo.PATCH, echo.POST, echo.DELETE},
	}))
	e.HideBanner = true
	e.Validator = NewValidator()
	e.Renderer = &TemplateRegistry{
		templates: loadTemplates(tmplBox),
		extraData: extraData,
	}

	return e
}
