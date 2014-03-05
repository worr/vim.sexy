package main

import (
	"bytes"
	"code.google.com/p/gcfg"
	"code.google.com/p/go-uuid/uuid"
	"fmt"
	"github.com/dpapathanasiou/go-recaptcha"
	"github.com/justinas/nosurf"
	"github.com/worr/chrooter"
	"github.com/worr/secstring"
	"html/template"
	"log"
	"math/rand"
	"net/http"
	"net/mail"
	"net/smtp"
	"time"
	txttemplate "text/template"
)

type Config struct {
	Mail struct {
		Email    string
		Username string
		Password string
		Hostname string
		password *secstring.SecString
	}

	Recaptcha struct {
		Private string
	}
}

var t = template.Must(template.ParseFiles("template/index.html"))
var emailTemplate = txttemplate.Must(txttemplate.New("email").Parse("Here is your exclusive Vim download link: http://www.vim.org/download.php?code={{.Code}}"))
var c = make(chan string)
var conf Config

// Default handler
func dispatch(w http.ResponseWriter, r *http.Request) {
	context := map[string]string{
		"token": nosurf.Token(r),
	}

	if r.Method == "POST" {
		context["email"] = r.FormValue("email")
		if !recaptcha.Confirm(r.RemoteAddr, r.FormValue("recaptcha_challenge_field"), r.FormValue("recaptcha_response_field")) {
			http.Error(w, "Failed captcha", http.StatusBadRequest)
			return
		}

		if context["email"] == "" {
			http.Error(w, "Empty email address", http.StatusBadRequest)
			return
		}

		c <- context["email"]
	}

	if err := t.Execute(w, context); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// Handler for all bad CSRF requests
func failedCSRF(w http.ResponseWriter, r *http.Request) {
	http.Error(w, nosurf.Reason(r).Error(), http.StatusBadRequest)
}

// Pulls email off of the channel and possibly sends download codes
func email() {
	auth := smtp.PlainAuth("", conf.Mail.Username, string(conf.Mail.password.String), conf.Mail.Hostname)

	for addr := range c {
		// Exclusivity
		if r := rand.Intn(3); r != 0 {
			continue
		}

		var emailAddr *mail.Address
		var err error
		if emailAddr, err = mail.ParseAddress(addr); err != nil {
			log.Printf("Failed to send email to %v: %v", emailAddr.Address, err)
			continue
		}

		fromAddr := mail.Address{Name:"Vim Download Czar", Address: conf.Mail.Email}
		headers := make(map[string]string)
		headers["To"] = emailAddr.String()
		headers["From"] = fromAddr.String()
		headers["Date"] = time.Now().String()
		headers["Subject"] = "Your Vim invite is ready"
		headers["MIME-Version"] = "1.0"
		headers["Content-Type"] = "text/plain; charset=\"utf-8\""

		msg := ""
		for key, val := range(headers) {
			msg += fmt.Sprintf("%v: %v\r\n", key, val)
		}

		buf := bytes.NewBuffer(make([]byte, 0))
		if err := emailTemplate.Execute(buf, struct{ Code string }{uuid.NewUUID().String()}); err != nil {
			log.Printf("Can't execute email template: %v", err)
			continue
		}

		msg += fmt.Sprintf("\r\n%s", buf)

		conf.Mail.password.Decrypt()
		if err = smtp.SendMail(conf.Mail.Hostname, auth, fromAddr.Address, []string{emailAddr.Address}, []byte(msg)); err != nil {
			log.Printf("Failed to send email to %v: %v", emailAddr.Address, err)
			continue
		}
		conf.Mail.password.Encrypt()
	}
}

func main() {
	if err := gcfg.ReadFileInto(&conf, "vim.sexy.ini"); err != nil {
		log.Fatalf("Can't read config file: %v", err)
	}

	if err := chrooter.Chroot("www", "/var/chroot/vim.sexy"); err != nil {
		log.Fatalf("Can't chroot: %v", err)
	}

	var err error
	if conf.Mail.password, err = secstring.FromString(&conf.Mail.Password); err != nil {
		log.Fatal(err)
	}

	recaptcha.Init(conf.Recaptcha.Private)

	go email()

	csrf := nosurf.New(http.HandlerFunc(dispatch))
	cookie := http.Cookie{HttpOnly: true}
	csrf.SetBaseCookie(cookie)
	csrf.SetFailureHandler(http.HandlerFunc(failedCSRF))
	if err = http.ListenAndServe("127.0.0.1:8000", csrf); err != nil {
		log.Fatalf("Cannot listen: %v", err)
	}
}
