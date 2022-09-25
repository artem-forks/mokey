package server

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/session"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"github.com/ubccr/goipa"
)

type Router struct {
	client       *ipa.Client
	sessionStore *session.Store
}

func NewRouter(storage fiber.Storage) (*Router, error) {
	r := &Router{}

	r.client = ipa.NewDefaultClient()

	err := r.client.LoginWithKeytab(viper.GetString("keytab"), viper.GetString("ktuser"))
	if err != nil {
		return nil, err
	}

	r.client.StickySession(false)

	r.sessionStore = session.New(session.Config{
		Storage:        storage,
		CookieSecure:   !viper.GetBool("develop"),
		CookieHTTPOnly: true,
	})

	return r, nil
}

func (r *Router) session(c *fiber.Ctx) (*session.Session, error) {
	sess, err := r.sessionStore.Get(c)
	if err != nil {
		log.WithFields(log.Fields{
			"path": c.Path(),
			"err":  err,
			"ip":   c.IP(),
		}).Error("Failed to fetch session from storage")

		return nil, err
	}

	return sess, nil
}

func (r *Router) sessionSave(c *fiber.Ctx, sess *session.Session) error {
	if err := sess.Save(); err != nil {
		log.WithFields(log.Fields{
			"path": c.Path(),
			"err":  err,
			"ip":   c.IP(),
		}).Error("Failed to save session to storage")

		return err
	}

	return nil
}

func (r *Router) SetupRoutes(app *fiber.App) {
	app.Get("/", r.RequireLogin, r.Index)

	// Auth
	app.Get("/auth/login", r.Login)
	app.Get("/auth/logout", r.Logout)
	app.Post("/auth/login", r.CheckUser)
	app.Post("/auth/authenticate", r.Authenticate)

	// Security
	app.Get("/security", r.RequireLogin, r.RequireHTMX, r.SecurityList)
	app.Post("/security/mfa/enable", r.RequireLogin, r.RequireHTMX, r.TwoFactorEnable)
	app.Post("/security/mfa/disable", r.RequireLogin, r.RequireHTMX, r.TwoFactorDisable)

	// SSH Keys
	app.Get("/sshkey/list", r.RequireLogin, r.RequireHTMX, r.SSHKeyList)
	app.Get("/sshkey/modal", r.RequireLogin, r.RequireHTMX, r.SSHKeyModal)
	app.Post("/sshkey/add", r.RequireLogin, r.RequireHTMX, r.SSHKeyAdd)
	app.Post("/sshkey/remove", r.RequireLogin, r.RequireHTMX, r.SSHKeyRemove)

	// OTP Tokens
	app.Get("/otptoken/list", r.RequireLogin, r.RequireHTMX, r.OTPTokenList)
	app.Get("/otptoken/modal", r.RequireLogin, r.RequireHTMX, r.OTPTokenModal)
	app.Post("/otptoken/add", r.RequireLogin, r.RequireHTMX, r.OTPTokenAdd)
	app.Post("/otptoken/verify", r.RequireLogin, r.RequireHTMX, r.OTPTokenVerify)
	app.Post("/otptoken/remove", r.RequireLogin, r.RequireHTMX, r.OTPTokenRemove)
	app.Post("/otptoken/enable", r.RequireLogin, r.RequireHTMX, r.OTPTokenEnable)
	app.Post("/otptoken/disable", r.RequireLogin, r.RequireHTMX, r.OTPTokenDisable)
}

func (r *Router) Index(c *fiber.Ctx) error {
	username := c.Locals(ContextKeyUser).(string)
	client := c.Locals(ContextKeyIPAClient).(*ipa.Client)

	user, err := client.UserShow(username)
	if err != nil {
		return err
	}

	vars := fiber.Map{
		"user": user,
	}

	return c.Render("index.html", vars)
}
